package opcua

import (
	"context"
	"math"
	"time"

	"github.com/gopcua/opcua/ua"
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
	RevisedPublishingInterval float64
	RevisedLifetimeCount      uint32
	RevisedMaxKeepAliveCount  uint32
	Notifs                    chan *PublishNotificationData
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
	delete(s.c.subscriptions, s.SubscriptionID)

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

func publishOnceForSubscription(s *Subscription, acks []*ua.SubscriptionAcknowledgement) (*ua.PublishResponse, error) {
	if acks == nil {
		acks = []*ua.SubscriptionAcknowledgement{}
	}
	req := &ua.PublishRequest{
		SubscriptionAcknowledgements: acks,
	}

	var res *ua.PublishResponse
	err := s.c.sendWithTimeout(req, s.getPublishTimeout(), func(v interface{}) error {
		return safeAssign(v, &res)
	})
	return res, err
}

func (s *Subscription) getPublishTimeout() time.Duration {
	keepaliveIntervalMs := uint64(s.RevisedPublishingInterval) *
		uint64(s.RevisedMaxKeepAliveCount)
	if keepaliveIntervalMs > math.MaxUint32 {
		keepaliveIntervalMs = math.MaxUint32
	}
	requestedTimeout := time.Duration(keepaliveIntervalMs) * time.Millisecond
	if requestedTimeout < s.c.cfg.RequestTimeout {
		return s.c.cfg.RequestTimeout
	}
	return requestedTimeout
}

// Run() starts an infinite loop that sends PublishRequests and delivers received
// notifications to registered Subscriptions.
func (s *Subscription) Run(ctx context.Context) {
	var acks []*ua.SubscriptionAcknowledgement

	for {
		select {
		case <-ctx.Done():
			return
		default:
			res, err := publishOnceForSubscription(s, acks)
			if err != nil {
				// StatusBadSequenceNumberUnknown means at least one ack has been submitted repeatedly
				// Ignore the error. Acks will be cleared below
				if err == ua.StatusBadSequenceNumberUnknown ||
					// Ignore StatusBadTimeout. No need to do anything except continue the loop
					err == ua.StatusBadTimeout ||
					// Ignore StatusBadNoSubscription as probably the cause is that all subscriptions
					// are already deleted, but the publishing loop is still running and will be stopped shortly
					err == ua.StatusBadNoSubscription {
				} else {
					s.c.notifySubscriptions(ctx, err)
				}
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
