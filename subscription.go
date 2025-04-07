package opcua

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gopcua/opcua/debug"
	"github.com/gopcua/opcua/errors"
	"github.com/gopcua/opcua/id"
	"github.com/gopcua/opcua/stats"
	"github.com/gopcua/opcua/ua"
	"github.com/gopcua/opcua/uasc"
)

const (
	DefaultSubscriptionMaxNotificationsPerPublish = 10000
	DefaultSubscriptionLifetimeCount              = 10000
	DefaultSubscriptionMaxKeepAliveCount          = 3000
	DefaultSubscriptionInterval                   = 100 * time.Millisecond
	DefaultSubscriptionPriority                   = 0
)

type Subscription struct {
	SubscriptionID            uint32
	RevisedPublishingInterval time.Duration
	RevisedLifetimeCount      uint32
	RevisedMaxKeepAliveCount  uint32
	Notifs                    chan<- *PublishNotificationData
	params                    *SubscriptionParameters
	paramsMu                  sync.Mutex
	items                     map[uint32]*monitoredItem
	itemsMu                   sync.Mutex
	lastSeq                   uint32
	nextSeq                   uint32
	c                         *Client
}

type SubscriptionParameters struct {
	Interval                   time.Duration
	LifetimeCount              uint32
	MaxKeepAliveCount          uint32
	MaxNotificationsPerPublish uint32
	Priority                   uint8
}

type MonitoredItemCreateRequestArgs struct {
	NodeID       *ua.NodeID
	AttributeID  ua.AttributeID
	ClientHandle uint32
	Filter       *ua.ExtensionObject
}

type monitoredItem struct {
	req *ua.MonitoredItemCreateRequest
	res *ua.MonitoredItemCreateResult
	ts  ua.TimestampsToReturn
}

// Deprectated: Use NewDefaultMonitoredItemCreateRequest instead. Will be removed with 0.8.0
func NewMonitoredItemCreateRequestWithDefaults(nodeID *ua.NodeID, attributeID ua.AttributeID, clientHandle uint32) *ua.MonitoredItemCreateRequest {
	return NewDefaultMonitoredItemCreateRequest(MonitoredItemCreateRequestArgs{
		NodeID:       nodeID,
		AttributeID:  attributeID,
		ClientHandle: clientHandle,
	})
}

func NewDefaultMonitoredItemCreateRequest(args MonitoredItemCreateRequestArgs) *ua.MonitoredItemCreateRequest {
	if args.AttributeID == 0 {
		args.AttributeID = ua.AttributeIDValue
	}
	return &ua.MonitoredItemCreateRequest{
		ItemToMonitor: &ua.ReadValueID{
			NodeID:       args.NodeID,
			AttributeID:  args.AttributeID,
			DataEncoding: &ua.QualifiedName{},
		},
		MonitoringMode: ua.MonitoringModeReporting,
		RequestedParameters: &ua.MonitoringParameters{
			ClientHandle:     args.ClientHandle,
			DiscardOldest:    true,
			Filter:           args.Filter,
			QueueSize:        10,
			SamplingInterval: 0.0,
		},
	}
}

type PublishNotificationData struct {
	SubscriptionID uint32
	Error          error
	Value          interface{}
}

// Cancel stops the subscription and removes it
// from the client and the server.
func (s *Subscription) Cancel(ctx context.Context) error {
	stats.Subscription().Add("Cancel", 1)
	s.c.forgetSubscription(ctx, s.SubscriptionID)
	return s.delete(ctx)
}

// delete removes the subscription from the server.
func (s *Subscription) delete(ctx context.Context) error {
	req := &ua.DeleteSubscriptionsRequest{
		SubscriptionIDs: []uint32{s.SubscriptionID},
	}

	var res *ua.DeleteSubscriptionsResponse
	err := s.c.Send(ctx, req, func(v ua.Response) error {
		return safeAssign(v, &res)
	})

	switch {
	case err != nil:
		return err
	case res.Results[0] == ua.StatusOK:
		s.itemsMu.Lock()
		s.items = make(map[uint32]*monitoredItem)
		s.itemsMu.Unlock()
		return nil
	default:
		return res.Results[0]
	}
}

func (s *Subscription) ModifySubscription(ctx context.Context, params SubscriptionParameters) (*ua.ModifySubscriptionResponse, error) {
	stats.Subscription().Add("ModifySubscription", 1)

	params.setDefaults()
	req := &ua.ModifySubscriptionRequest{
		SubscriptionID:              s.SubscriptionID,
		RequestedPublishingInterval: float64(params.Interval.Milliseconds()),
		RequestedLifetimeCount:      params.LifetimeCount,
		RequestedMaxKeepAliveCount:  params.MaxKeepAliveCount,
		MaxNotificationsPerPublish:  params.MaxNotificationsPerPublish,
		Priority:                    params.Priority,
	}

	var res *ua.ModifySubscriptionResponse
	err := s.c.Send(ctx, req, func(v ua.Response) error {
		return safeAssign(v, &res)
	})

	if err != nil {
		return nil, err
	}

	// update subscription parameters
	s.paramsMu.Lock()
	s.params = &params
	s.paramsMu.Unlock()
	// update revised subscription parameters
	s.RevisedPublishingInterval = time.Duration(res.RevisedPublishingInterval) * time.Millisecond
	s.RevisedLifetimeCount = res.RevisedLifetimeCount
	s.RevisedMaxKeepAliveCount = res.RevisedMaxKeepAliveCount

	return res, nil
}

func (s *Subscription) Monitor(ctx context.Context, ts ua.TimestampsToReturn, items ...*ua.MonitoredItemCreateRequest) (*ua.CreateMonitoredItemsResponse, error) {
	stats.Subscription().Add("Monitor", 1)
	stats.Subscription().Add("MonitoredItems", int64(len(items)))

	// Part 4, 5.12.2.2 CreateMonitoredItems Service Parameters
	req := &ua.CreateMonitoredItemsRequest{
		SubscriptionID:     s.SubscriptionID,
		TimestampsToReturn: ts,
		ItemsToCreate:      items,
	}

	var res *ua.CreateMonitoredItemsResponse
	err := s.c.Send(ctx, req, func(v ua.Response) error {
		return safeAssign(v, &res)
	})

	if err != nil {
		return nil, err
	}

	// store monitored items
	s.itemsMu.Lock()
	for i, item := range items {
		result := res.Results[i]
		s.items[result.MonitoredItemID] = &monitoredItem{
			req: item,
			res: result,
			ts:  ts,
		}
	}
	s.itemsMu.Unlock()

	return res, err
}

func (s *Subscription) Unmonitor(ctx context.Context, monitoredItemIDs ...uint32) (*ua.DeleteMonitoredItemsResponse, error) {
	stats.Subscription().Add("Unmonitor", 1)
	stats.Subscription().Add("UnmonitoredItems", int64(len(monitoredItemIDs)))

	req := &ua.DeleteMonitoredItemsRequest{
		MonitoredItemIDs: monitoredItemIDs,
		SubscriptionID:   s.SubscriptionID,
	}

	var res *ua.DeleteMonitoredItemsResponse
	err := s.c.Send(ctx, req, func(v ua.Response) error {
		return safeAssign(v, &res)
	})
	if err != nil {
		return nil, err
	}

	// remove monitored items
	s.itemsMu.Lock()
	for _, id := range monitoredItemIDs {
		delete(s.items, id)
	}
	s.itemsMu.Unlock()

	return res, nil
}

func (s *Subscription) ModifyMonitoredItems(ctx context.Context, ts ua.TimestampsToReturn, items ...*ua.MonitoredItemModifyRequest) (*ua.ModifyMonitoredItemsResponse, error) {
	stats.Subscription().Add("ModifyMonitoredItems", 1)
	stats.Subscription().Add("ModifiedMonitoredItems", int64(len(items)))

	var err error
	s.itemsMu.Lock()
	for _, item := range items {
		id := item.MonitoredItemID
		if _, exists := s.items[id]; !exists {
			err = fmt.Errorf("sub %d: cannot modify unknown monitored item id: %d", s.SubscriptionID, id)
			break
		}
	}
	s.itemsMu.Unlock()

	if err != nil {
		return nil, err
	}

	req := &ua.ModifyMonitoredItemsRequest{
		SubscriptionID:     s.SubscriptionID,
		TimestampsToReturn: ts,
		ItemsToModify:      items,
	}
	var res *ua.ModifyMonitoredItemsResponse

	err = s.c.Send(ctx, req, func(v ua.Response) error {

		return safeAssign(v, &res)
	})
	if err != nil {
		return nil, err
	}

	// update monitored items
	s.itemsMu.Lock()
	for i, res := range res.Results {
		if res.StatusCode != ua.StatusOK {
			continue
		}

		id := req.ItemsToModify[i].MonitoredItemID
		item := s.items[id]
		item.ts = req.TimestampsToReturn
		item.req.RequestedParameters = req.ItemsToModify[i].RequestedParameters
		item.res.StatusCode = res.StatusCode
		item.res.RevisedSamplingInterval = res.RevisedSamplingInterval
		item.res.RevisedQueueSize = res.RevisedQueueSize
		item.res.FilterResult = res.FilterResult
	}
	s.itemsMu.Unlock()

	return res, nil
}

func (s *Subscription) SetMonitoringMode(ctx context.Context, monitoringMode ua.MonitoringMode, monitoredItemIDs ...uint32) (*ua.SetMonitoringModeResponse, error) {
	stats.Subscription().Add("SetMonitoringMode", 1)
	stats.Subscription().Add("SetMonitoringModeMonitoredItems", int64(len(monitoredItemIDs)))

	var err error
	s.itemsMu.Lock()
	for _, id := range monitoredItemIDs {
		if _, exists := s.items[id]; !exists {
			err = fmt.Errorf("sub %d: cannot set monitoring mode for unknown monitored item id: %d", s.SubscriptionID, id)
			break
		}
	}
	s.itemsMu.Unlock()

	if err != nil {
		return nil, err
	}

	req := &ua.SetMonitoringModeRequest{
		SubscriptionID:   s.SubscriptionID,
		MonitoringMode:   monitoringMode,
		MonitoredItemIDs: monitoredItemIDs,
	}
	var res *ua.SetMonitoringModeResponse
	err = s.c.Send(ctx, req, func(v ua.Response) error {
		return safeAssign(v, &res)
	})
	if err != nil {
		return nil, err
	}

	return res, nil
}

// SetTriggering sends a request to the server to add and/or remove triggering links from a triggering item.
// To add links from a triggering item to an item to report provide the server assigned ID(s) in the `add` argument.
// To remove links from a triggering item to an item to report provide the server assigned ID(s) in the `remove` argument.
func (s *Subscription) SetTriggering(ctx context.Context, triggeringItemID uint32, add, remove []uint32) (*ua.SetTriggeringResponse, error) {
	stats.Subscription().Add("SetTriggering", 1)

	// Part 4, 5.12.5.2 SetTriggering Service Parameters
	req := &ua.SetTriggeringRequest{
		SubscriptionID:   s.SubscriptionID,
		TriggeringItemID: triggeringItemID,
		LinksToAdd:       add,
		LinksToRemove:    remove,
	}

	var res *ua.SetTriggeringResponse
	err := s.c.Send(ctx, req, func(v ua.Response) error {
		return safeAssign(v, &res)
	})
	return res, err
}

func (s *Subscription) publishTimeout() time.Duration {
	timeout := time.Duration(s.RevisedMaxKeepAliveCount) * s.RevisedPublishingInterval // expected keepalive interval
	if timeout > uasc.MaxTimeout {
		return uasc.MaxTimeout
	}
	if timeout < s.c.cfg.sechan.RequestTimeout {
		return s.c.cfg.sechan.RequestTimeout
	}
	return timeout
}

func (s *Subscription) notify(ctx context.Context, data *PublishNotificationData) {
	select {
	case <-ctx.Done():
		return
	case s.Notifs <- data:
	}
}

// Stats returns a diagnostic struct with metadata about the current subscription
func (s *Subscription) Stats(ctx context.Context) (*ua.SubscriptionDiagnosticsDataType, error) {
	// TODO(kung-foo): once browsing feature is merged, attempt to get direct access to the
	// diagnostics node. for example, Prosys lists them like:
	// i=2290/ns=1;g=918ee6f4-2d25-4506-980d-e659441c166d
	// maybe cache the nodeid to speed up future stats queries
	node := s.c.Node(ua.NewNumericNodeID(0, id.Server_ServerDiagnostics_SubscriptionDiagnosticsArray))
	v, err := node.Value(ctx)
	if err != nil {
		return nil, err
	}

	if v == nil {
		return nil, errors.Errorf("empty SubscriptionDiagnostics for sub=%d", s.SubscriptionID)
	}

	eos, ok := v.Value().([]*ua.ExtensionObject)
	if !ok {
		return nil, errors.Errorf("invalid type for SubscriptionDiagnosticsArray. Want []*ua.ExtensionObject. subID=%d nodeID=%s type=%T", s.SubscriptionID, node.String(), v.Value())
	}

	for _, eo := range eos {
		stat, ok := eo.Value.(*ua.SubscriptionDiagnosticsDataType)
		if !ok {
			continue
		}

		if stat.SubscriptionID == s.SubscriptionID {
			return stat, nil
		}
	}

	return nil, errors.Errorf("unable to find SubscriptionDiagnostics for sub=%d", s.SubscriptionID)
}

func (p *SubscriptionParameters) setDefaults() {
	if p.MaxNotificationsPerPublish == 0 {
		p.MaxNotificationsPerPublish = DefaultSubscriptionMaxNotificationsPerPublish
	}
	if p.LifetimeCount == 0 {
		p.LifetimeCount = DefaultSubscriptionLifetimeCount
	}
	if p.MaxKeepAliveCount == 0 {
		p.MaxKeepAliveCount = DefaultSubscriptionMaxKeepAliveCount
	}
	if p.Interval == 0 {
		p.Interval = DefaultSubscriptionInterval
	}
	if p.Priority == 0 {
		// DefaultSubscriptionPriority is 0 at the time of writing, so this redundant assignment is
		// made only to allow for a one-liner change of default priority should a need arise
		// and to explicitly expose the default priority as a constant
		p.Priority = DefaultSubscriptionPriority
	}
}

// recreate_delete is called by the client when it is trying to
// recreate an existing subscription. This function deletes the
// existing subscription from the server.
func (s *Subscription) recreate_delete(ctx context.Context) error {
	dlog := debug.NewPrefixLogger("sub %d: recreate_delete: ", s.SubscriptionID)
	req := &ua.DeleteSubscriptionsRequest{
		SubscriptionIDs: []uint32{s.SubscriptionID},
	}
	var res *ua.DeleteSubscriptionsResponse
	_ = s.c.Send(ctx, req, func(v ua.Response) error {
		return safeAssign(v, &res)
	})
	dlog.Print("subscription deleted")
	return nil
}

// recreate_create is called by the client when it is trying to
// recreate an existing subscription. This function creates a
// new subscription with the same parameters as the previous one.
func (s *Subscription) recreate_create(ctx context.Context) error {
	dlog := debug.NewPrefixLogger("sub %d: recreate_create: ", s.SubscriptionID)

	s.paramsMu.Lock()
	params := s.params
	s.paramsMu.Unlock()

	req := &ua.CreateSubscriptionRequest{
		RequestedPublishingInterval: float64(params.Interval / time.Millisecond),
		RequestedLifetimeCount:      params.LifetimeCount,
		RequestedMaxKeepAliveCount:  params.MaxKeepAliveCount,
		PublishingEnabled:           true,
		MaxNotificationsPerPublish:  params.MaxNotificationsPerPublish,
		Priority:                    params.Priority,
	}
	var res *ua.CreateSubscriptionResponse
	err := s.c.Send(ctx, req, func(v ua.Response) error {
		return safeAssign(v, &res)
	})
	if err != nil {
		dlog.Printf("failed to recreate subscription")
		return err
	}
	// todo (unknownet): check if necessary
	if status := res.ResponseHeader.ServiceResult; status != ua.StatusOK {
		return status
	}
	dlog.Printf("recreated as subscription %d", res.SubscriptionID)
	dlog.SetPrefix(fmt.Sprintf("sub %d: recreate: ", res.SubscriptionID))

	s.SubscriptionID = res.SubscriptionID
	s.RevisedPublishingInterval = time.Duration(res.RevisedPublishingInterval) * time.Millisecond
	s.RevisedLifetimeCount = res.RevisedLifetimeCount
	s.RevisedMaxKeepAliveCount = res.RevisedMaxKeepAliveCount
	s.lastSeq = 0
	s.nextSeq = 1

	if err := s.c.registerSubscription_NeedsSubMuxLock(s); err != nil {
		return err
	}
	dlog.Printf("subscription registered")

	// Sort by timestamp to return
	itemsByTimestamps := make(map[ua.TimestampsToReturn][]*ua.MonitoredItemCreateRequest)
	s.itemsMu.Lock()
	for _, mi := range s.items {
		itemsByTimestamps[mi.ts] = append(itemsByTimestamps[mi.ts], mi.req)
	}
	s.items = make(map[uint32]*monitoredItem, len(s.items))
	s.itemsMu.Unlock()

	for ts, items := range itemsByTimestamps {
		req := &ua.CreateMonitoredItemsRequest{
			SubscriptionID:     s.SubscriptionID,
			TimestampsToReturn: ts,
			ItemsToCreate:      items,
		}

		var res *ua.CreateMonitoredItemsResponse
		err := s.c.Send(ctx, req, func(v ua.Response) error {
			return safeAssign(v, &res)
		})
		if err != nil {
			dlog.Printf("failed to create monitored items: %v", err)
			return err
		}

		for _, result := range res.Results {
			if status := result.StatusCode; status != ua.StatusOK {
				return status
			}
		}

		s.itemsMu.Lock()
		for i, item := range items {
			s.items[res.Results[i].MonitoredItemID] = &monitoredItem{
				req: item,
				res: res.Results[i],
				ts:  ts,
			}
		}
		s.itemsMu.Unlock()
	}
	dlog.Printf("subscription successfully recreated")

	return nil
}
