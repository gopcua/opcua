package opcua

import (
	"context"
	"fmt"
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

type Subscription struct {
	SubscriptionID            uint32
	RevisedPublishingInterval time.Duration
	RevisedLifetimeCount      uint32
	RevisedMaxKeepAliveCount  uint32
	Notifs                    chan *PublishNotificationData
	params                    *SubscriptionParameters
	items                     []*monitoredItem
	lastSequenceNumber        uint32 // use atomic.Load/Store
	pausech                   chan struct{}
	resumech                  chan struct{}
	stopch                    chan struct{}
	c                         *Client

	pendingAcks    []uint32
	pendingAcksMux sync.RWMutex
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

func (s *Subscription) publish() (*ua.PublishResponse, error) {
	s.pendingAcksMux.RLock()
	req := &ua.PublishRequest{
		SubscriptionAcknowledgements: make([]*ua.SubscriptionAcknowledgement, len(s.pendingAcks)),
	}
	for i := range s.pendingAcks {
		req.SubscriptionAcknowledgements[i] = &ua.SubscriptionAcknowledgement{
			SubscriptionID: s.SubscriptionID,
			SequenceNumber: s.pendingAcks[i],
		}
	}
	s.pendingAcksMux.RUnlock()

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
	plog := log.New(log.Writer(), fmt.Sprintf("sub %d: ", s.SubscriptionID), log.Flags()|log.Lmsgprefix)

	cctx, cancel := context.WithCancel(ctx)
	defer cancel()
	defer plog.Print("done")

publish:
	for {
		plog.Print("select")
		select {
		case <-ctx.Done():
			plog.Println("ctx.Done()")
			cancel()
			return

		case <-s.stopch:
			plog.Println("stop")
			cancel()
			return

		case <-s.pausech:
			plog.Print("pause")
		paused:
			for {
				select {
				case <-ctx.Done():
					plog.Print("pause: ctx.Done()")
					cancel()
					return
				case <-s.stopch:
					plog.Print("pause: stop")
					cancel()
					return
				case <-s.pausech:
					plog.Print("pause: pause")
					// ignore since already paused
					continue paused
				case <-s.resumech:
					plog.Print("pause: resume")
					continue publish
				}
			}

		case <-s.resumech:
			plog.Print("resume")
			// ignore since not paused
			continue

		default:
			plog.Print("publish")
			s.runPublish(cctx)
		}
	}
}

func (s *Subscription) runPublish(ctx context.Context) {
	plog := log.New(log.Writer(), fmt.Sprintf("sub %d: publish: ", s.SubscriptionID), log.Flags()|log.Lmsgprefix)

	defer plog.Print("done")

	for {
		plog.Print("select")
		select {
		case <-ctx.Done():
			plog.Print("ctx.Done()")
			return

		case <-s.stopch:
			plog.Printf("stop")
			return

		default:
			plog.Printf("default")

			// send the next publish request
			// note that res contains data even if an error was returned
			res, err := s.publish()
			switch {
			case err == nil && res.SubscriptionID != s.SubscriptionID:
				plog.Printf("Got notifs for other subscription %d. Skipping", res.SubscriptionID)
				continue

			case err == ua.StatusBadSequenceNumberUnknown:
				// todo(fs): this should only happen per in the status codes
				// todo(fs): lets log this here to see
				plog.Printf("Got error %s which should only happen in the ACK results", err)

			case err == ua.StatusBadTooManyPublishRequests:
				// todo(fs): we have sent too many publish requests
				// todo(fs): we need to slow down
				plog.Printf("got %s. Sleeping for one second", err)
				time.Sleep(time.Second) // does this make sense

			case err == ua.StatusBadTimeout:
				// ignore and continue the loop
				plog.Print("Timeout. ignoring")

			case err == ua.StatusBadNoSubscription:
				// All subscriptions have been deleted, but the publishing loop is still running
				// The user will stop the loop or create subscriptions at his discretion
				plog.Print("Subscription invalid. Waiting for publishing loop to stop")

			case err != nil:
				// irrecoverable error
				s.c.notifySubscriptionsOfError(ctx, s.SubscriptionID, err)
				plog.Printf("Notify error %s. Stopping publish loop", err)
				return
			}

			if res != nil {
				// clean up pending acks
				s.pendingAcksMux.RLock()
				pendingAcks := s.pendingAcks
				s.pendingAcksMux.RUnlock()

				// we assume that the number of results in the response match
				// the number of pending acks from the previous PublishRequest.
				if len(pendingAcks) != len(res.Results) {
					plog.Printf("Got %d results for pending ACKs, want %d", len(res.Results), len(pendingAcks))
					// todo(fs): what should we do here?
					pendingAcks = nil
				}

				// find the messages which we have received but which we have not acked.
				var notAcked []uint32
				for i, seqnr := range pendingAcks {
					err := res.Results[i]
					switch err {
					case ua.StatusOK:
					// publish response ack'ed -> skip
					case ua.StatusBadSubscriptionIDInvalid:
						// old subscription id -> skip
						plog.Printf("Old subscription id: %s", err)
					case ua.StatusBadSequenceNumberUnknown:
						// server does not have the message in its retransmission queue anymore
						plog.Printf("Server does not have notif %d anymore: %s", seqnr, err)
					default:
						// otherwise, we try to ack again
						notAcked = append(notAcked, seqnr)
						plog.Printf("Retrying to ACK notif %d: %s", seqnr, err)
					}
				}
				pendingAcks = notAcked

				// check if we have missed a notification
				// we assume that the sequence numbers are increasing monotonically
				// and if we missed one then we should ask the server to republish
				// it, if it is still in its retransmission queue

				var (
					lastSeq   = s.lastSequenceNumber
					nextSeq   = lastSeq + 1
					thisSeq   = res.NotificationMessage.SequenceNumber
					availSeqs = res.AvailableSequenceNumbers
				)

				if thisSeq > nextSeq {
					for seqnr := nextSeq; seqnr < thisSeq; seqnr++ {
						if !uint32SliceContains(seqnr, availSeqs) {
							plog.Printf("Missed notif %d but server no longer has it. Data loss", seqnr)
							continue
						}

						plog.Printf("Requesting republish of missed notif %d", seqnr)
						rpres, rperr := s.republish(&ua.RepublishRequest{
							SubscriptionID:           res.SubscriptionID,
							RetransmitSequenceNumber: seqnr,
						})
						switch rperr {
						case ua.StatusOK:
							lastSeq = seqnr
							pendingAcks = append(pendingAcks, seqnr)
							s.c.notifySubscription(ctx, res.SubscriptionID, rpres.NotificationMessage)
							plog.Printf("Received missed notif %d", seqnr)
						default:
							lastSeq = seqnr
							plog.Printf("Republish request for missed notif %d failed. Data loss: %s", seqnr, err)
						}
					}
				}

				if err == nil {
					s.c.notifySubscription(ctx, res.SubscriptionID, res.NotificationMessage)
					plog.Printf("notif %d", res.NotificationMessage.SequenceNumber)
					lastSeq = res.NotificationMessage.SequenceNumber
					pendingAcks = append(pendingAcks, res.NotificationMessage.SequenceNumber)
				}

				s.lastSequenceNumber = lastSeq
				s.pendingAcksMux.Lock()
				s.pendingAcks = pendingAcks
				s.pendingAcksMux.Unlock()
			}
		}
	}
}

func uint32SliceContains(n uint32, a []uint32) bool {
	for _, v := range a {
		if v == n {
			return true
		}
	}
	return false
}

func (s *Subscription) notify(ctx context.Context, data *PublishNotificationData) {
	if s == nil {
		panic("s == nil")
	}
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
