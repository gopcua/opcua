// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package server

import (
	"context"
	"time"

	"github.com/gopcua/opcua/id"
	"github.com/gopcua/opcua/ua"
	"github.com/gopcua/opcua/uasc"
)

type Handler func(*uasc.SecureChannel, ua.Request, uint32) (ua.Response, error)

func (srv *Server) initHandlers() {
	// s.registerHandlerFunc(id.ServiceFault_Encoding_DefaultBinary, handleServiceFault)

	discovery := &DiscoveryService{srv}
	srv.RegisterHandler(id.FindServersRequest_Encoding_DefaultBinary, discovery.FindServers)
	srv.RegisterHandler(id.FindServersOnNetworkRequest_Encoding_DefaultBinary, discovery.FindServersOnNetwork)
	srv.RegisterHandler(id.GetEndpointsRequest_Encoding_DefaultBinary, discovery.GetEndpoints)
	srv.RegisterHandler(id.RegisterServerRequest_Encoding_DefaultBinary, discovery.RegisterServer)
	srv.RegisterHandler(id.RegisterServer2Request_Encoding_DefaultBinary, discovery.RegisterServer2)

	// SecureChannel service (handled in the uasc stack)
	// s.registerHandlerFunc(id.OpenSecureChannelRequest_Encoding_DefaultBinary, handleOpenSecureChannel)
	// s.registerHandlerFunc(id.CloseSecureChannelRequest_Encoding_DefaultBinary, handleCloseSecureChannel)

	session := &SessionService{srv}
	srv.RegisterHandler(id.CreateSessionRequest_Encoding_DefaultBinary, session.CreateSession)
	srv.RegisterHandler(id.ActivateSessionRequest_Encoding_DefaultBinary, session.ActivateSession)
	srv.RegisterHandler(id.CloseSessionRequest_Encoding_DefaultBinary, session.CloseSession)
	srv.RegisterHandler(id.CancelRequest_Encoding_DefaultBinary, session.Cancel)

	node := &NodeManagementService{srv}
	srv.RegisterHandler(id.AddNodesRequest_Encoding_DefaultBinary, node.AddNodes)
	srv.RegisterHandler(id.AddReferencesRequest_Encoding_DefaultBinary, node.AddReferences)
	srv.RegisterHandler(id.DeleteNodesRequest_Encoding_DefaultBinary, node.DeleteNodes)
	srv.RegisterHandler(id.DeleteReferencesRequest_Encoding_DefaultBinary, node.DeleteReferences)

	view := &ViewService{srv}
	srv.RegisterHandler(id.BrowseRequest_Encoding_DefaultBinary, view.Browse)
	srv.RegisterHandler(id.BrowseNextRequest_Encoding_DefaultBinary, view.BrowseNext)
	srv.RegisterHandler(id.TranslateBrowsePathsToNodeIDsRequest_Encoding_DefaultBinary, view.TranslateBrowsePathsToNodeIDs)
	srv.RegisterHandler(id.RegisterNodesRequest_Encoding_DefaultBinary, view.RegisterNodes)
	srv.RegisterHandler(id.UnregisterNodesRequest_Encoding_DefaultBinary, view.UnregisterNodes)

	query := &QueryService{srv}
	srv.RegisterHandler(id.QueryFirstRequest_Encoding_DefaultBinary, query.QueryFirst)
	srv.RegisterHandler(id.QueryNextRequest_Encoding_DefaultBinary, query.QueryNext)

	attr := &AttributeService{srv}
	srv.RegisterHandler(id.ReadRequest_Encoding_DefaultBinary, attr.Read)
	srv.RegisterHandler(id.HistoryReadRequest_Encoding_DefaultBinary, attr.HistoryRead)
	srv.RegisterHandler(id.WriteRequest_Encoding_DefaultBinary, attr.Write)
	srv.RegisterHandler(id.HistoryUpdateRequest_Encoding_DefaultBinary, attr.HistoryUpdate)

	method := &MethodService{srv}
	// s.registerHandler(id.CallMethodRequest_Encoding_DefaultBinary, method.CallMethod) // todo(fs): I think this is bogus
	srv.RegisterHandler(id.CallRequest_Encoding_DefaultBinary, method.Call)

	sub := &SubscriptionService{
		srv:  srv,
		Subs: make(map[uint32]*Subscription),
	}
	srv.SubscriptionService = sub
	srv.RegisterHandler(id.CreateSubscriptionRequest_Encoding_DefaultBinary, sub.CreateSubscription)
	srv.RegisterHandler(id.ModifySubscriptionRequest_Encoding_DefaultBinary, sub.ModifySubscription)
	srv.RegisterHandler(id.SetPublishingModeRequest_Encoding_DefaultBinary, sub.SetPublishingMode)
	srv.RegisterHandler(id.PublishRequest_Encoding_DefaultBinary, sub.Publish)
	srv.RegisterHandler(id.RepublishRequest_Encoding_DefaultBinary, sub.Republish)
	srv.RegisterHandler(id.TransferSubscriptionsRequest_Encoding_DefaultBinary, sub.TransferSubscriptions)
	srv.RegisterHandler(id.DeleteSubscriptionsRequest_Encoding_DefaultBinary, sub.DeleteSubscriptions)

	item := &MonitoredItemService{
		SubService: sub,
		Items:      make(map[uint32]*MonitoredItem),
		Nodes:      make(map[string][]*MonitoredItem),
		Subs:       make(map[uint32][]*MonitoredItem),
	}
	srv.MonitoredItemService = item
	// s.registerHandler(id.MonitoredItemCreateRequest_Encoding_DefaultBinary, item.MonitoredItemCreate)
	srv.RegisterHandler(id.CreateMonitoredItemsRequest_Encoding_DefaultBinary, item.CreateMonitoredItems)
	//s.RegisterHandler(id.CreateMonitoredItemsRequest_Encoding_DefaultBinary, s.CreateMonitoredItems)
	// s.registerHandler(id.MonitoredItemModifyRequest_Encoding_DefaultBinary, item.MonitoredItemModify)
	srv.RegisterHandler(id.ModifyMonitoredItemsRequest_Encoding_DefaultBinary, item.ModifyMonitoredItems)
	srv.RegisterHandler(id.SetMonitoringModeRequest_Encoding_DefaultBinary, item.SetMonitoringMode)
	srv.RegisterHandler(id.SetTriggeringRequest_Encoding_DefaultBinary, item.SetTriggering)
	srv.RegisterHandler(id.DeleteMonitoredItemsRequest_Encoding_DefaultBinary, item.DeleteMonitoredItems)
}

// This function allows you to overwrite a handler before you call start.
func (srv *Server) RegisterHandler(typeID uint16, h Handler) {
	_, ok := srv.handlers[typeID]
	if !ok {
		srv.handlers[typeID] = h
	}
}

func (srv *Server) handleService(ctx context.Context, sc *uasc.SecureChannel, reqID uint32, req ua.Request) {
	srv.cfg.logger.Debug("handleService: Got: %T\n", req)

	var resp ua.Response
	var err error

	typeID := ua.ServiceTypeID(req)
	h, ok := srv.handlers[typeID]
	if ok {
		resp, err = h(sc, req, reqID)
	} else {
		if typeID == 0 {
			srv.cfg.logger.Warn("unknown service %T. Did you call register?", req)
		}
		err = ua.StatusBadServiceUnsupported
	}

	if err != nil {
		if statusCode, ok := err.(ua.StatusCode); ok {
			resp = &ua.ServiceFault{ResponseHeader: responseHeader(0, statusCode)}
		} else {
			resp = &ua.ServiceFault{ResponseHeader: responseHeader(0, ua.StatusBadUnexpectedError)}
		}
	}

	if resp == nil {
		return
	}

	err = sc.SendResponseWithContext(ctx, reqID, resp)
	if err != nil {
		srv.cfg.logger.Warn("Error sending response: %s\n", err)
	}
}

func responseHeader(reqID uint32, statusCode ua.StatusCode) *ua.ResponseHeader {
	return &ua.ResponseHeader{
		Timestamp:          time.Now(),
		RequestHandle:      reqID,
		ServiceResult:      statusCode,
		ServiceDiagnostics: &ua.DiagnosticInfo{},
		StringTable:        []string{},
		AdditionalHeader:   ua.NewExtensionObject(nil),
	}
}

func serviceUnsupported(hdr *ua.RequestHeader) ua.Response {
	return &ua.ServiceFault{
		ResponseHeader: responseHeader(hdr.RequestHandle, ua.StatusBadServiceUnsupported),
	}
}

func safeReq[T ua.Request](r ua.Request) (T, error) {
	var t T
	req, ok := r.(T)
	if !ok {
		//debug.Printf("expected %T, got %T", t, r)
		return t, ua.StatusBadRequestTypeInvalid
	}
	return req, nil
}

// func handleServiceFault(s *Server, sc *uasc.SecureChannel, r ua.Request) (ua.Response, error) {
// 	debug.Printf("Handling %T", r)

// 	req, ok := r.(*ua.ServiceFault)
// 	if !ok {
// 		debug.Printf("handleServiceFault: Expected *ua.ServiceFault, got %T", r)
// 		return nil, ua.StatusBadRequestTypeInvalid
// 	}
// 	debug.Printf("Got ServiceFault: %s", req.ResponseHeader.ServiceResult)

// 	// No response required
// 	return nil, nil
// }
