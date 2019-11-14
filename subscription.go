package opcua

import (
	"context"
	"time"

	"github.com/gopcua/opcua/debug"
	"github.com/gopcua/opcua/errors"
	"github.com/gopcua/opcua/id"
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

const (
	terminatedSubscriptionID uint32 = 0xC0CAC01B
)

type Subscription struct {
	SubscriptionID            uint32
	RevisedPublishingInterval time.Duration
	RevisedLifetimeCount      uint32
	RevisedMaxKeepAliveCount  uint32
	Notifs                    chan *PublishNotificationData
	lastSequenceNumber        uint32
	monitoredItems            []*MonitoredItem
	params                    *SubscriptionParameters
	running                   chan bool
	c                         *Client
}

type SubscriptionParameters struct {
	Interval                   time.Duration
	LifetimeCount              uint32
	MaxKeepAliveCount          uint32
	MaxNotificationsPerPublish uint32
	Priority                   uint8
	Notifs                     chan *PublishNotificationData
}

type MonitoredItem struct {
	MonitoredItemID           uint32
	ItemToMonitor             *ua.ReadValueID
	MonitoringParameters      *ua.MonitoringParameters
	MonitoringMode            ua.MonitoringMode
	TimestampsToReturn        ua.TimestampsToReturn
	monitoredItemCreateResult *ua.MonitoredItemCreateResult
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

// Cancel() deletes the Subscription from Server and makes the Client forget it so that publishing
// loops cannot deliver notifications to it anymore
func (s *Subscription) Cancel() error {
	s.c.forgetSubscription(s.SubscriptionID)

	req := &ua.DeleteSubscriptionsRequest{
		SubscriptionIDs: []uint32{s.SubscriptionID},
	}
	var res *ua.DeleteSubscriptionsResponse
	err := s.c.Send(req, func(v interface{}) error {
		return safeAssign(v, &res)
	})
	if err != nil {
		return err
	}
	if res.ResponseHeader.ServiceResult != ua.StatusOK {
		return res.ResponseHeader.ServiceResult
	}

	return nil
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

	// store Monitored items
	monitoredItems := make([]*MonitoredItem, len(items))
	for i, item := range items {
		result := res.Results[i]

		monitoredItems[i] = &MonitoredItem{
			MonitoredItemID:           result.MonitoredItemID,
			ItemToMonitor:             item.ItemToMonitor,
			MonitoringParameters:      item.RequestedParameters,
			MonitoringMode:            item.MonitoringMode,
			TimestampsToReturn:        ts,
			monitoredItemCreateResult: result,
		}
	}

	s.monitoredItems = append(s.monitoredItems, monitoredItems...)
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

func (s *Subscription) republish(req *ua.RepublishRequest) (*ua.RepublishResponse, error) {
	var res *ua.RepublishResponse
	err := s.c.sechan.SendRequest(req, s.c.Session().resp.AuthenticationToken, func(v interface{}) error {
		if err := safeAssign(v, &res); err != nil {
			return err
		}
		if res.ResponseHeader.ServiceResult != ua.StatusOK {
			return errors.Errorf(res.ResponseHeader.ServiceResult.Error())
		}
		return nil
	})
	return res, err
}

func (s *Subscription) publish(acks []*ua.SubscriptionAcknowledgement) (*ua.PublishResponse, error) {
	if acks == nil {
		acks = []*ua.SubscriptionAcknowledgement{}
	}
	req := &ua.PublishRequest{
		SubscriptionAcknowledgements: acks,
	}

	var res *ua.PublishResponse
	err := s.c.sendWithTimeout(req, s.publishTimeout(), func(v interface{}) error {
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

func (s *Subscription) Resume() {
	s.running <- true
}

func (s *Subscription) Suspend() {
	s.running <- false
}

// Run() starts an infinite loop that sends PublishRequests and delivers received
// notifications to registered Subscriptions.
// It is the responsibility of the user to stop no longer needed Run() loops by cancelling ctx
// Note that Run() may return before ctx is cancelled in case of an irrecoverable communication error
func (s *Subscription) Run(ctx context.Context) {
	var acks []*ua.SubscriptionAcknowledgement

	for {
		select {
		case <-ctx.Done():
			return
		case run := <-s.running:
			// run indicate if the publishing should be interrupted
			if run {
				// If run is active then continue publishing
				continue
			}
			// Suspended mode
			for !run {
				select {
				case <-ctx.Done():
					return
				case run = <-s.running:
					// Block until running has been triggered by Resume
				}
			}

		default:
			// send the next publish request
			// note that res contains data even if an error was returned
			res, err := s.publish(acks)
			switch {
			case err == ua.StatusBadSequenceNumberUnknown:
				// At least one ack has been submitted repeatedly
				// Ignore the error. Acks will be cleared below
			case err == ua.StatusBadTimeout:
				// ignore and continue the loop
			case err == ua.StatusBadNoSubscription:
				// All subscriptions have been deleted, but the publishing loop is still running
				// The user will stop the loop or create subscriptions at his discretion
			case err == ua.StatusBadSessionClosed || err == ua.StatusBadSessionIDInvalid:
				// The session is no longer opened on the server side

				if s.c.cfg.AutoReconnect {
					if err := s.c.repairSession(); err != nil {
						return
					}
					acks = make([]*ua.SubscriptionAcknowledgement, 0)
					continue
				}

			case err != nil:
				// irrecoverable error
				s.c.notifySubscriptionsOfError(ctx, res, err)

				if !s.c.cfg.AutoReconnect {
					return
				}
				s.Suspend()
			}

			if res != nil {
				// Prepare SubscriptionAcknowledgement for next PublishRequest
				acks = make([]*ua.SubscriptionAcknowledgement, 0)
				if res.AvailableSequenceNumbers != nil {
					for _, i := range res.AvailableSequenceNumbers {
						ack := &ua.SubscriptionAcknowledgement{
							SubscriptionID: res.SubscriptionID,
							SequenceNumber: i,
						}
						acks = append(acks, ack)
					}
				}
			}

			if err == nil {
				s.c.notifySubscription(ctx, res)
			}
		}
	}
}

func (s *Subscription) sendNotification(ctx context.Context, data *PublishNotificationData) {
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
	if p.Notifs == nil {
		p.Notifs = make(chan *PublishNotificationData)
	}
}

// recreateSubscriptionAndMonitoredItem recreate a new subscription base of a previous subscription
// parameters
func (s *Subscription) recreateSubscriptionAndMonitoredItem() error {
	if s.SubscriptionID == terminatedSubscriptionID {
		debug.Printf("Subscription is not in a valid state")
		return nil
	}

	params := s.params
	s.c.forgetSubscription(s.SubscriptionID)

	subReq := &ua.CreateSubscriptionRequest{
		RequestedPublishingInterval: float64(params.Interval / time.Millisecond),
		RequestedLifetimeCount:      params.LifetimeCount,
		RequestedMaxKeepAliveCount:  params.MaxKeepAliveCount,
		PublishingEnabled:           true,
		MaxNotificationsPerPublish:  params.MaxNotificationsPerPublish,
		Priority:                    params.Priority,
	}

	var subRes *ua.CreateSubscriptionResponse
	err := s.c.Send(subReq, func(v interface{}) error {
		return safeAssign(v, &subRes)
	})
	if err != nil {
		return err
	}
	if status := subRes.ResponseHeader.ServiceResult; status != ua.StatusOK {
		return status
	}

	s.SubscriptionID = subRes.SubscriptionID
	s.RevisedPublishingInterval = time.Duration(subRes.RevisedPublishingInterval) * time.Millisecond
	s.RevisedLifetimeCount = subRes.RevisedLifetimeCount
	s.RevisedMaxKeepAliveCount = subRes.RevisedMaxKeepAliveCount
	s.lastSequenceNumber = 0

	if err := s.c.registerSubscription(s); err != nil {
		return err
	}

	// Sort by timestamp to return
	itemsByTs := make(map[ua.TimestampsToReturn][]*ua.MonitoredItemCreateRequest)
	for _, m := range s.monitoredItems {

		if _, ok := itemsByTs[m.TimestampsToReturn]; !ok {
			itemsByTs[m.TimestampsToReturn] = []*ua.MonitoredItemCreateRequest{}
		}

		itemsByTs[m.TimestampsToReturn] = append(
			itemsByTs[m.TimestampsToReturn],
			&ua.MonitoredItemCreateRequest{
				ItemToMonitor:       m.ItemToMonitor,
				MonitoringMode:      m.MonitoringMode,
				RequestedParameters: m.MonitoringParameters,
			},
		)
	}

	for ts, items := range itemsByTs {

		monReq := &ua.CreateMonitoredItemsRequest{
			SubscriptionID:     s.SubscriptionID,
			TimestampsToReturn: ts,
			ItemsToCreate:      items,
		}

		var monRes *ua.CreateMonitoredItemsResponse
		err = s.c.Send(monReq, func(v interface{}) error {
			return safeAssign(v, &monRes)
		})
		if err != nil {
			return err
		}
		for _, result := range monRes.Results {
			if status := result.StatusCode; status != ua.StatusOK {
				return status
			}
		}

		for i, m := range s.monitoredItems {
			result := monRes.Results[i]
			m.MonitoredItemID = result.MonitoredItemID
			m.monitoredItemCreateResult = result
		}
	}

	return nil
}
