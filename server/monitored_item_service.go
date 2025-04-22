package server

import (
	"errors"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gopcua/opcua/ua"
	"github.com/gopcua/opcua/uasc"
)

// MonitoredItemService implements the MonitoredItem Service Set.
//
// https://reference.opcfoundation.org/Core/Part4/v105/docs/5.12
type MonitoredItemService struct {
	SubService *SubscriptionService
	Mu         sync.Mutex

	// items tracked by ID
	Items map[uint32]*MonitoredItem
	// items tracked by node
	Nodes map[string][]*MonitoredItem
	// items tracked by subscription
	Subs map[uint32][]*MonitoredItem

	id uint32
}

// function to get rid of all references to a specific Monitored Item (by ID number)
func (s *MonitoredItemService) DeleteMonitoredItem(id uint32) {
	s.Mu.Lock()
	defer s.Mu.Unlock()
	item, ok := s.Items[id]
	if !ok {
		// id does not exist.
		return
	}

	if item == nil || item.Req == nil || item.Req.ItemToMonitor == nil || item.Req.ItemToMonitor.NodeID == nil {
		return
	}
	nodeid := item.Req.ItemToMonitor.NodeID.String()

	if s == nil || s.Nodes == nil || s.Nodes[nodeid] == nil {
		return
	}

	// delete the monitored item from all nodes
	// was using slices.DeleteFunc but that is from a newer go version so we'll do it manually with /exp/slices
	// we've got to go backwards because we're deleting from the slice as we go.
	// I'm guessing this loop is less efficient than slices.DeleteFunc but it's what we've got.
	delete(s.Items, id)
	for i := len(s.Nodes[nodeid]) - 1; i >= 0; i-- {
		n := s.Nodes[nodeid][i]
		if n == nil {
			continue
		}
		if n.ID == id {
			s.Nodes[nodeid] = slices.Delete(s.Nodes[nodeid], i, i+1)
		}
	}
	//slices.DeleteFunc(s.Nodes[nodeid], func(i *MonitoredItem) bool { return i.ID == item.ID })
	if len(s.Nodes[nodeid]) == 0 {
		delete(s.Nodes, nodeid)
	}

	for i := len(s.Subs[item.Sub.ID]) - 1; i >= 0; i-- {
		n := s.Subs[item.Sub.ID][i]
		if n == nil {
			continue
		}
		if n.ID == id {
			s.Subs[item.Sub.ID] = slices.Delete(s.Subs[item.Sub.ID], i, i+1)
		}
	}
	//slices.DeleteFunc(s.Subs[item.Sub.ID], func(i *MonitoredItem) bool { return i.ID == item.ID })
	if len(s.Subs[item.Sub.ID]) == 0 {
		delete(s.Subs, item.Sub.ID)
	}
}

// function to delete all monitored items associated with a specific sub (as indicated by id number)
func (s *MonitoredItemService) DeleteSub(id uint32) {
	s.Mu.Lock()
	items, ok := s.Subs[id]
	delete(s.Subs, id)
	s.Mu.Unlock()
	if !ok {
		return
	}
	for i := range items {
		if items[i] != nil {
			s.DeleteMonitoredItem(items[i].ID)
		}
	}
}

func (s *MonitoredItemService) ChangeNotification(n *ua.NodeID) {

	s.Mu.Lock()
	defer s.Mu.Unlock()
	items, ok := s.Nodes[n.String()]

	if !ok {
		// this node isn't monitored - don't have to do anything.
		return
	}

	ns, err := s.SubService.srv.Namespace(int(n.Namespace()))

	for i := range items {
		item := items[i]
		if item == nil {
			continue
		}
		val := new(ua.MonitoredItemNotification)
		val.ClientHandle = item.Req.RequestedParameters.ClientHandle
		if err != nil {
			if s.SubService.srv.cfg.logger != nil {
				s.SubService.srv.cfg.logger.Warn("error getting namespace %d: %v", n.Namespace(), err)
			}
			val.Value = &ua.DataValue{}
			val.Value.Status = ua.StatusBad
			val.Value.EncodingMask |= ua.DataValueStatusCode
			item.Sub.NotifyChannel <- val
			continue
		}
		dv := ns.Attribute(n, item.Req.ItemToMonitor.AttributeID)
		val.Value = dv
		item.Sub.NotifyChannel <- val
	}

}

func (s *MonitoredItemService) NextID() uint32 {
	i := atomic.AddUint32(&s.id, 1)
	if i == 0 {
		i = atomic.AddUint32(&s.id, 1)
	}
	return i
}

type MonitoredItem struct {
	ID  uint32
	Sub *Subscription
	Req *ua.MonitoredItemCreateRequest

	//TODO: use this
	Mode ua.MonitoringMode
}

// https://reference.opcfoundation.org/Core/Part4/v105/docs/5.12.2
func (s *MonitoredItemService) CreateMonitoredItems(sc *uasc.SecureChannel, r ua.Request, reqID uint32) (ua.Response, error) {
	if s.SubService.srv.cfg.logger != nil {
		s.SubService.srv.cfg.logger.Debug("Handling %T", r)
	}

	req, err := safeReq[*ua.CreateMonitoredItemsRequest](r)
	if err != nil {
		return nil, err
	}
	s.Mu.Lock()
	defer s.Mu.Unlock()

	count := len(req.ItemsToCreate)

	res := make([]*ua.MonitoredItemCreateResult, count)

	subID := req.SubscriptionID
	if s.SubService.srv.cfg.logger != nil {
		s.SubService.srv.cfg.logger.Debug("Creating monitored items for sub #%d", subID)
	}
	s.SubService.Mu.Lock()
	sub, ok := s.SubService.Subs[subID]
	s.SubService.Mu.Unlock()
	if !ok {
		return nil, errors.New("sub doesn't exist")
	}

	sess := s.SubService.srv.Session(req.RequestHeader)
	if sub.Session.AuthTokenID.String() != sess.AuthTokenID.String() {
		return nil, errors.New("not your subscription, bro")
	}

	for i := range req.ItemsToCreate {
		itemreq := req.ItemsToCreate[i]
		nodeid := itemreq.ItemToMonitor.NodeID
		item := MonitoredItem{
			ID:  s.NextID(),
			Sub: sub,
			Req: itemreq,
		}

		// book keeping of the new item
		s.Items[item.ID] = &item
		list, ok := s.Nodes[item.Req.ItemToMonitor.NodeID.String()]
		if !ok {
			list = make([]*MonitoredItem, 0, 1)
		}
		s.Nodes[item.Req.ItemToMonitor.NodeID.String()] = append(list, &item)

		list, ok = s.Subs[item.Sub.ID]
		if !ok {
			list = make([]*MonitoredItem, 0, 1)
		}
		s.Subs[item.Sub.ID] = append(list, &item)

		if s.SubService.srv.cfg.logger != nil {
			s.SubService.srv.cfg.logger.Debug("Adding monitored item '%s' to sub #%d as %d->%d",
				nodeid.String(),
				subID,
				item.ID,
				itemreq.RequestedParameters.ClientHandle)
		}
		res[i] = &ua.MonitoredItemCreateResult{
			StatusCode:              ua.StatusOK,
			MonitoredItemID:         item.ID,
			RevisedSamplingInterval: sub.RevisedPublishingInterval,
			RevisedQueueSize:        1,
			FilterResult:            ua.NewExtensionObject(nil),
		}
		// do an initial update for the nodeids in the background.
		// These lock the mutex so we can't do them inline here.
		// This will cause them to happen once we unlock.
		go s.ChangeNotification(nodeid)

	}

	resp := &ua.CreateMonitoredItemsResponse{
		ResponseHeader: &ua.ResponseHeader{
			Timestamp:          time.Now(),
			RequestHandle:      req.RequestHeader.RequestHandle,
			ServiceResult:      ua.StatusOK,
			ServiceDiagnostics: &ua.DiagnosticInfo{},
			StringTable:        []string{},
			AdditionalHeader:   ua.NewExtensionObject(nil),
		},
		Results:         res,                    //                  []StatusCode
		DiagnosticInfos: []*ua.DiagnosticInfo{}, //          []*DiagnosticInfo
	}

	return resp, nil

}

// https://reference.opcfoundation.org/Core/Part4/v105/docs/5.12.3
func (s *MonitoredItemService) ModifyMonitoredItems(sc *uasc.SecureChannel, r ua.Request, reqID uint32) (ua.Response, error) {
	if s.SubService.srv.cfg.logger != nil {
		s.SubService.srv.cfg.logger.Debug("Handling %T", r)
	}

	req, err := safeReq[*ua.ModifyMonitoredItemsRequest](r)
	if err != nil {
		return nil, err
	}
	return serviceUnsupported(req.RequestHeader), nil
}

// https://reference.opcfoundation.org/Core/Part4/v105/docs/5.12.4
func (s *MonitoredItemService) SetMonitoringMode(sc *uasc.SecureChannel, r ua.Request, reqID uint32) (ua.Response, error) {
	if s.SubService.srv.cfg.logger != nil {
		s.SubService.srv.cfg.logger.Debug("Handling %T", r)
	}

	req, err := safeReq[*ua.SetMonitoringModeRequest](r)
	if err != nil {
		return nil, err
	}
	s.Mu.Lock()
	defer s.Mu.Unlock()

	results := make([]ua.StatusCode, len(req.MonitoredItemIDs))

	sess := s.SubService.srv.Session(req.RequestHeader)

	for i := range req.MonitoredItemIDs {
		id := req.MonitoredItemIDs[i]
		item, ok := s.Items[id]

		if item.Sub.Session.AuthTokenID.String() != sess.AuthTokenID.String() {
			results[i] = ua.StatusBadSessionIDInvalid
		}

		if !ok {
			results[i] = ua.StatusBadMonitoredItemIDInvalid
			continue
		}
		item.Mode = req.MonitoringMode
		results[i] = ua.StatusOK
	}

	return &ua.SetMonitoringModeResponse{
		ResponseHeader: &ua.ResponseHeader{
			Timestamp:          time.Now(),
			RequestHandle:      req.RequestHeader.RequestHandle,
			ServiceResult:      ua.StatusOK,
			ServiceDiagnostics: &ua.DiagnosticInfo{},
			StringTable:        []string{},
			AdditionalHeader:   ua.NewExtensionObject(nil),
		},
		Results:         results,
		DiagnosticInfos: []*ua.DiagnosticInfo{},
	}, nil

}

// https://reference.opcfoundation.org/Core/Part4/v105/docs/5.12.5
func (s *MonitoredItemService) SetTriggering(sc *uasc.SecureChannel, r ua.Request, reqID uint32) (ua.Response, error) {
	if s.SubService.srv.cfg.logger != nil {
		s.SubService.srv.cfg.logger.Debug("Handling %T", r)
	}

	req, err := safeReq[*ua.SetTriggeringRequest](r)
	if err != nil {
		return nil, err
	}
	return serviceUnsupported(req.RequestHeader), nil
}

// https://reference.opcfoundation.org/Core/Part4/v105/docs/5.12.6
func (s *MonitoredItemService) DeleteMonitoredItems(sc *uasc.SecureChannel, r ua.Request, reqID uint32) (ua.Response, error) {
	if s.SubService.srv.cfg.logger != nil {
		s.SubService.srv.cfg.logger.Debug("Handling %T", r)
	}

	req, err := safeReq[*ua.DeleteMonitoredItemsRequest](r)
	if err != nil {
		return nil, err
	}

	s.Mu.Lock()
	defer s.Mu.Unlock()

	sess := s.SubService.srv.Session(req.RequestHeader)

	results := make([]ua.StatusCode, len(req.MonitoredItemIDs))
	for i := range req.MonitoredItemIDs {
		id := req.MonitoredItemIDs[i]
		item, ok := s.Items[id]
		if !ok {
			results[i] = ua.StatusBadMonitoredItemIDInvalid
		}

		if item.Sub.Session.AuthTokenID.String() != sess.AuthTokenID.String() {
			results[i] = ua.StatusBadSessionIDInvalid
		}

		// this function gets the lock so we need to do it in the background so it can happen after our lock is released.
		go s.DeleteMonitoredItem(id)
		results[i] = ua.StatusOK
	}

	response := &ua.DeleteMonitoredItemsResponse{
		ResponseHeader: &ua.ResponseHeader{
			Timestamp:          time.Now(),
			RequestHandle:      req.RequestHeader.RequestHandle,
			ServiceResult:      ua.StatusOK,
			ServiceDiagnostics: &ua.DiagnosticInfo{},
			StringTable:        []string{},
			AdditionalHeader:   ua.NewExtensionObject(nil),
		},
		Results:         results,
		DiagnosticInfos: []*ua.DiagnosticInfo{},
	}
	return response, nil

}
