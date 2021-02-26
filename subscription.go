package opcua

import (
	"context"
	"fmt"
	"time"

	"github.com/liuxgo/opcua/debug"
	"github.com/liuxgo/opcua/errors"
	"github.com/liuxgo/opcua/id"
	"github.com/liuxgo/opcua/ua"
	"github.com/liuxgo/opcua/uasc"
)

const (
	DefaultSubscriptionMaxNotificationsPerPublish = 10000
	DefaultSubscriptionLifetimeCount              = 10000
	DefaultSubscriptionMaxKeepAliveCount          = 3000
	DefaultSubscriptionInterval                   = 100 * time.Millisecond
	DefaultSubscriptionPriority                   = 0
)

const terminatedSubscriptionID uint32 = 0xC0CAC01B

type Subscription struct {
	SubscriptionID            uint32
	RevisedPublishingInterval time.Duration
	RevisedLifetimeCount      uint32
	RevisedMaxKeepAliveCount  uint32
	Notifs                    chan *PublishNotificationData
	params                    *SubscriptionParameters
	items                     []*monitoredItem
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

type monitoredItem struct {
	ItemToMonitor        *ua.ReadValueID
	MonitoringParameters *ua.MonitoringParameters
	MonitoringMode       ua.MonitoringMode
	TimestampsToReturn   ua.TimestampsToReturn
	createResult         *ua.MonitoredItemCreateResult
}

func NewMonitoredItemCreateRequestWithDefaults(nodeID *ua.NodeID, attributeID ua.AttributeID, clientHandle uint32) *ua.MonitoredItemCreateRequest {
	if attributeID == 0 {
		attributeID = ua.AttributeIDValue
	}
	return &ua.MonitoredItemCreateRequest{
		ItemToMonitor: &ua.ReadValueID{
			NodeID:       nodeID,
			AttributeID:  attributeID,
			DataEncoding: &ua.QualifiedName{},
		},
		MonitoringMode: ua.MonitoringModeReporting,
		RequestedParameters: &ua.MonitoringParameters{
			ClientHandle:     clientHandle,
			DiscardOldest:    true,
			Filter:           nil,
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
func (s *Subscription) Cancel() error {
	s.c.forgetSubscription(s.SubscriptionID)
	return s.delete()
}

// delete removes the subscription from the server.
func (s *Subscription) delete() error {
	req := &ua.DeleteSubscriptionsRequest{
		SubscriptionIDs: []uint32{s.SubscriptionID},
	}
	var res *ua.DeleteSubscriptionsResponse
	err := s.c.Send(req, func(v interface{}) error {
		return safeAssign(v, &res)
	})
	switch {
	case err == ua.StatusOK:
		return nil
	case err != nil:
		return err
	default:
		return res.ResponseHeader.ServiceResult
	}
}

func (s *Subscription) Monitor(ts ua.TimestampsToReturn, items ...*ua.MonitoredItemCreateRequest) (*ua.CreateMonitoredItemsResponse, error) {
	// Part 4, 5.12.2.2 CreateMonitoredItems Service Parameters
	req := &ua.CreateMonitoredItemsRequest{
		SubscriptionID:     s.SubscriptionID,
		TimestampsToReturn: ts,
		ItemsToCreate:      items,
	}

	var res *ua.CreateMonitoredItemsResponse
	err := s.c.Send(req, func(v interface{}) error {
		return safeAssign(v, &res)
	})

	if err != nil {
		return nil, err
	}

	for _, result := range res.Results {
		if status := result.StatusCode; status != ua.StatusOK {
			return nil, status
		}
	}

	// store monitored items
	for i, item := range items {
		result := res.Results[i]

		mi := &monitoredItem{
			ItemToMonitor:        item.ItemToMonitor,
			MonitoringParameters: item.RequestedParameters,
			MonitoringMode:       item.MonitoringMode,
			TimestampsToReturn:   ts,
			createResult:         result,
		}
		s.items = append(s.items, mi)
	}

	return res, err
}

func (s *Subscription) Unmonitor(monitoredItemIDs ...uint32) (*ua.DeleteMonitoredItemsResponse, error) {
	req := &ua.DeleteMonitoredItemsRequest{
		MonitoredItemIDs: monitoredItemIDs,
		SubscriptionID:   s.SubscriptionID,
	}
	var res *ua.DeleteMonitoredItemsResponse
	err := s.c.Send(req, func(v interface{}) error {
		return safeAssign(v, &res)
	})
	return res, err
}

// SetTriggering sends a request to the server to add and/or remove triggering links from a triggering item.
// To add links from a triggering item to an item to report provide the server assigned ID(s) in the `add` argument.
// To remove links from a triggering item to an item to report provide the server assigned ID(s) in the `remove` argument.
func (s *Subscription) SetTriggering(triggeringItemID uint32, add, remove []uint32) (*ua.SetTriggeringResponse, error) {
	// Part 4, 5.12.5.2 SetTriggering Service Parameters
	req := &ua.SetTriggeringRequest{
		SubscriptionID:   s.SubscriptionID,
		TriggeringItemID: triggeringItemID,
		LinksToAdd:       add,
		LinksToRemove:    remove,
	}

	var res *ua.SetTriggeringResponse
	err := s.c.Send(req, func(v interface{}) error {
		return safeAssign(v, &res)
	})
	return res, err
}

func (s *Subscription) publishTimeout() time.Duration {
	timeout := time.Duration(s.RevisedMaxKeepAliveCount) * s.RevisedPublishingInterval // expected keepalive interval
	if timeout > uasc.MaxTimeout {
		return uasc.MaxTimeout
	}
	if timeout < s.c.cfg.RequestTimeout {
		return s.c.cfg.RequestTimeout
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
func (s *Subscription) Stats() (*ua.SubscriptionDiagnosticsDataType, error) {
	// TODO(kung-foo): once browsing feature is merged, attempt to get direct access to the
	// diagnostics node. for example, Prosys lists them like:
	// i=2290/ns=1;g=918ee6f4-2d25-4506-980d-e659441c166d
	// maybe cache the nodeid to speed up future stats queries
	node := s.c.Node(ua.NewNumericNodeID(0, id.Server_ServerDiagnostics_SubscriptionDiagnosticsArray))
	v, err := node.Value()
	if err != nil {
		return nil, err
	}

	for _, eo := range v.Value().([]*ua.ExtensionObject) {
		stat := eo.Value.(*ua.SubscriptionDiagnosticsDataType)

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

// recreate creates a new subscription based on the previous subscription
// parameters and monitored items.
func (s *Subscription) recreate() error {
	dlog := debug.NewPrefixLogger("sub %d: recreate: ", s.SubscriptionID)

	if s.SubscriptionID == terminatedSubscriptionID {
		dlog.Printf("subscription is not in a valid state")
		return nil
	}

	params := s.params
	{
		req := &ua.DeleteSubscriptionsRequest{
			SubscriptionIDs: []uint32{s.SubscriptionID},
		}
		var res *ua.DeleteSubscriptionsResponse
		_ = s.c.Send(req, func(v interface{}) error {
			return safeAssign(v, &res)
		})
		dlog.Print("subscription deleted")
	}
	s.c.forgetSubscription(s.SubscriptionID)
	dlog.Printf("subscription forgotton")

	req := &ua.CreateSubscriptionRequest{
		RequestedPublishingInterval: float64(params.Interval / time.Millisecond),
		RequestedLifetimeCount:      params.LifetimeCount,
		RequestedMaxKeepAliveCount:  params.MaxKeepAliveCount,
		PublishingEnabled:           true,
		MaxNotificationsPerPublish:  params.MaxNotificationsPerPublish,
		Priority:                    params.Priority,
	}
	var res *ua.CreateSubscriptionResponse
	err := s.c.Send(req, func(v interface{}) error {
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

	if err := s.c.registerSubscription(s); err != nil {
		return err
	}
	dlog.Printf("subscription registered")

	// Sort by timestamp to return
	itemsByTs := make(map[ua.TimestampsToReturn][]*ua.MonitoredItemCreateRequest)
	for _, m := range s.items {
		cr := &ua.MonitoredItemCreateRequest{
			ItemToMonitor:       m.ItemToMonitor,
			MonitoringMode:      m.MonitoringMode,
			RequestedParameters: m.MonitoringParameters,
		}
		itemsByTs[m.TimestampsToReturn] = append(itemsByTs[m.TimestampsToReturn], cr)
	}

	for ts, items := range itemsByTs {
		req := &ua.CreateMonitoredItemsRequest{
			SubscriptionID:     s.SubscriptionID,
			TimestampsToReturn: ts,
			ItemsToCreate:      items,
		}

		var res *ua.CreateMonitoredItemsResponse
		err := s.c.Send(req, func(v interface{}) error {
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

		for i, m := range s.items {
			m.createResult = res.Results[i]
		}
	}
	dlog.Printf("subscription successfully recreated")

	return nil
}
