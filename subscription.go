package opcua

import (
	"context"
	"time"

	"github.com/gopcua/opcua/ua"
)

type Subscription struct {
	SubscriptionID            uint32
	RevisedPublishingInterval float64
	RevisedLifetimeCount      uint32
	RevisedMaxKeepAliveCount  uint32
	Notifs                    chan PublishNotificationData
	c                         *Client
}

type SubscriptionParameters struct {
	Interval                   time.Duration
	LifetimeCount              uint32
	MaxKeepAliveCount          uint32
	MaxNotificationsPerPublish uint32
	Priority                   uint8
	Notifs                     chan PublishNotificationData
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

// Run() starts an infinite loop that sends PublishRequests and delivers received
// notifications to registered Subscriptions.
func (s *Subscription) Run(ctx context.Context) {
	// Empty SubscriptionAcknowledgements for first PublishRequest
	var acks = make([]*ua.SubscriptionAcknowledgement, 0)

	for {
		select {
		case <-ctx.Done():
			return
		default:
			res, err := s.c.Publish(acks)
			if err != nil {
				if err == ua.StatusBadTimeout {
					continue
				} else if err == ua.StatusBadNoSubscription {
					// ignore it as probably the cause is that all subscriptions are already deleted,
					// but the publishing loop is still running and will be stopped shortly
					continue
				}
				s.c.notifySubscriptions(ctx, err)
				continue
			}
			// Prepare SubscriptionAcknowledgement for next PublishRequest
			acks = make([]*ua.SubscriptionAcknowledgement, 0)
			for _, i := range res.AvailableSequenceNumbers {
				ack := &ua.SubscriptionAcknowledgement{
					SubscriptionID: res.SubscriptionID,
					SequenceNumber: i,
				}
				acks = append(acks, ack)
			}
			s.c.notifySubscription(res)
		}
	}
}
