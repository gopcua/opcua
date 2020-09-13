package opcua

import (
	"context"
	"log"
	"sync"
	"sync/atomic"
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

const terminatedSubscriptionID uint32 = 0xC0CAC01B

var (
	publishID   uint32 = 0
	publishIDMu sync.Mutex
)

type Subscription struct {
	SubscriptionID            uint32
	RevisedPublishingInterval time.Duration
	RevisedLifetimeCount      uint32
	RevisedMaxKeepAliveCount  uint32
	Notifs                    chan *PublishNotificationData
	params                    *SubscriptionParameters
	items                     []*monitoredItem
	lastSequenceNumber        uint32
	publishch                 chan publishReq
	reqs                      []uint32
	reqsMu                    sync.Mutex
	acks                      []*ua.SubscriptionAcknowledgement
	acksMu                    sync.Mutex
	pausech                   chan struct{}
	resumech                  chan struct{}
	stopch                    chan struct{}
	c                         *Client
}

type publishReq struct {
	ID  uint32
	Res *ua.PublishResponse
	Err error
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
	close(s.stopch)
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
	case err != nil:
		return err
	case res.ResponseHeader.ServiceResult != ua.StatusOK:
		return res.ResponseHeader.ServiceResult
	default:
		return nil
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

// republish executes a synchronous republish request.
func (s *Subscription) republish(req *ua.RepublishRequest) (*ua.RepublishResponse, error) {
	var res *ua.RepublishResponse
	err := s.c.sechan.SendRequest(req, s.c.Session().resp.AuthenticationToken, func(v interface{}) error {
		return safeAssign(v, &res)
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

// pause suspends the run loop by signalling the pausech.
// Since the channel is unbuffered we wait until the
// run loop has completed the current publish message.
func (s *Subscription) pause(ctx context.Context) {
	select {
	case <-ctx.Done():
	case <-s.stopch:
	case s.pausech <- struct{}{}:
	}
}

// resume restarts the run loop by signalling the resumech.
// It has no effect if the run loop isn't paused.
func (s *Subscription) resume(ctx context.Context) {
	select {
	case <-ctx.Done():
	case <-s.stopch:
	case s.resumech <- struct{}{}:
	}
}

// Run starts an infinite loop which sends PublishRequests and delivers received
// notifications to registered subcribers.
//
// It is the responsibility of the caller to stop the run loops by
// cancelling the context.
//
// Note that Run may return before the context is cancelled
// in case of an irrecoverable communication error.
func (s *Subscription) Run(ctx context.Context) {
	defer log.Print("sub: done")

	// start publish
	s.sendPublish(ctx)
publish:
	for {
		log.Print("sub: select")
		select {
		case <-ctx.Done():
			log.Println("sub: ctx.Done()")
			return

		case <-s.stopch:
			log.Println("sub: stop")
			return

		case <-s.pausech:
			log.Print("sub: pause")

			// ignore previous requests
			s.reqsMu.Lock()
			s.reqs = []uint32{}
			s.reqsMu.Unlock()

		paused:
			for {
				select {
				case <-ctx.Done():
					log.Print("sub: pause: ctx.Done()")
					return
				case <-s.stopch:
					log.Print("sub: pause: stop")
					return
				case <-s.pausech:
					log.Print("sub: pause: pause")
					// ignore since already paused
					continue paused
				case <-s.resumech:
					log.Print("sub: pause: resume")
					s.sendPublish(ctx)
					continue publish
				}
			}

		case <-s.resumech:
			log.Print("sub: resume")
			// ignore since not paused
			continue

		case req := <-s.publishch:

			switch {
			case req.Err == ua.StatusBadSequenceNumberUnknown:
				// At least one ack has been submitted repeatedly
				// Ignore the error. Acks will be cleared below
			case req.Err == ua.StatusBadTimeout:
				// ignore and continue the loop
			case req.Err == ua.StatusBadNoSubscription:
				// All subscriptions have been deleted, but the publishing loop is still running
				// The user will stop the loop or create subscriptions at his discretion
			case req.Err != nil:
				// irrecoverable error
				s.c.notifySubscriptionsOfError(ctx, req.Res, req.Err)
				debug.Printf("subscription %v Run loop stopped", s.SubscriptionID)
				log.Print("publish: notify error")

				// stop sendPublish until pause/resume
				continue publish
			}

			if req.Err == nil {
				s.c.notifySubscription(ctx, req.Res)
				log.Print("publish: notify")
			}

			// if a request end send a new one
			if ok := s.popReq(req.ID); ok {
				s.sendPublish(ctx)
			}
		}
	}
}

// popReq remove a req
func (s *Subscription) popReq(reqID uint32) (ok bool) {
	s.reqsMu.Lock()
	defer s.reqsMu.Unlock()
	ok = false
	for idx, id := range s.reqs {
		if id == reqID {
			ok = true
			s.reqs = append(s.reqs[:idx], s.reqs[idx+1:]...)
		}
	}
	return
}

func nextPublishID() uint32 {
	publishIDMu.Lock()
	defer publishIDMu.Unlock()

	publishID++
	return publishID
}

func (s *Subscription) sendPublish(ctx context.Context) {
	id := nextPublishID()

	s.reqsMu.Lock()
	s.reqs = append(s.reqs, id)
	s.reqsMu.Unlock()

	go func() {
		// send the next publish request
		// note that res contains data even if an error was returned
		s.acksMu.Lock()
		acks := s.acks
		s.acks = []*ua.SubscriptionAcknowledgement{}
		s.acksMu.Unlock()

		res, err := s.publish(acks)

		if res != nil {
			s.acksMu.Lock()
			s.acks = append(
				s.acks,
				&ua.SubscriptionAcknowledgement{
					SubscriptionID: res.SubscriptionID,
					SequenceNumber: res.NotificationMessage.SequenceNumber,
				},
			)
			s.acksMu.Unlock()
		}

		select {
		case <-ctx.Done():
			log.Print("sendPublish: ctx.Done()")
			return
		case <-s.stopch:
			log.Printf("sendPublish: stop")
			return
		case s.publishch <- publishReq{
			ID:  id,
			Res: res,
			Err: err,
		}:
		}
	}()
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

// restore creates a new subscription based on the previous subscription
// parameters and monitored items.
func (s *Subscription) restore() error {
	if s.SubscriptionID == terminatedSubscriptionID {
		debug.Printf("Subscription is not in a valid state")
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
	}
	s.c.forgetSubscription(s.SubscriptionID)

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
		return err
	}
	// todo (unknownet): check if necessary
	if status := res.ResponseHeader.ServiceResult; status != ua.StatusOK {
		return status
	}

	s.SubscriptionID = res.SubscriptionID
	s.RevisedPublishingInterval = time.Duration(res.RevisedPublishingInterval) * time.Millisecond
	s.RevisedLifetimeCount = res.RevisedLifetimeCount
	s.RevisedMaxKeepAliveCount = res.RevisedMaxKeepAliveCount
	atomic.StoreUint32(&s.lastSequenceNumber, 0)

	if err := s.c.registerSubscription(s); err != nil {
		return err
	}

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

	return nil
}
