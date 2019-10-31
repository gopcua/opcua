package opcua

import (
	"context"
	"io"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gopcua/opcua/debug"
	"github.com/gopcua/opcua/errors"
	"github.com/gopcua/opcua/ua"
	"github.com/gopcua/opcua/uasc"
)

var (
	publishRequestCountInPipeline uint32 = 5
	minTimeoutHint                uint32 = 0x7FFFFFF
)

type publishEngine struct {
	s                                    *Session
	sessionMux                           sync.RWMutex
	TimeoutHint                          uint32
	ActiveSubscriptionCount              uint32
	NbPendingPublishRequests             uint32
	NbMaxPublishRequestsAcceptedByServer uint32
	isSuspended                          atomic.Value
	SubscriptionAcknowledgements         []*ua.SubscriptionAcknowledgement

	// map of active subscriptions managed by this client. key is SubscriptionID
	// access guarded by subMux
	subscriptions map[uint32]*Subscription
	subMux        sync.RWMutex

	chanRunning chan bool
}

func newPublishEngine(session *Session) *publishEngine {
	p := &publishEngine{
		s:                                    session,
		TimeoutHint:                          10000,
		ActiveSubscriptionCount:              0,
		NbPendingPublishRequests:             0,
		NbMaxPublishRequestsAcceptedByServer: 1000,
		SubscriptionAcknowledgements:         []*ua.SubscriptionAcknowledgement{},
		subscriptions:                        make(map[uint32]*Subscription),
		chanRunning:                          make(chan bool, 1),
	}
	p.isSuspended.Store(false)
	return p
}

func (p *publishEngine) IsSuspended() bool {
	return p.isSuspended.Load().(bool)
}

func (p *publishEngine) setIsSuspended(val bool) {
	p.isSuspended.Store(val)
}

func (p *publishEngine) PublishRequestCountInPipeline() uint32 {
	return publishRequestCountInPipeline
}

func (p *publishEngine) SubscriptionCount() uint32 {
	p.subMux.Lock()
	defer p.subMux.Unlock()
	return uint32(len(p.subscriptions))
}

func (p *publishEngine) RegisterSubscription(sub *Subscription) error {
	if sub.SubscriptionID == 0 {
		return ua.StatusBadSubscriptionIDInvalid
	}
	p.subMux.Lock()
	if _, ok := p.subscriptions[sub.SubscriptionID]; ok {
		p.subMux.Unlock()
		return errors.Errorf("SubscriptionID (%d) already registered", sub.SubscriptionID)
	}

	p.ActiveSubscriptionCount++
	p.subscriptions[sub.SubscriptionID] = sub
	if sub.TimeoutHint > p.TimeoutHint {
		p.TimeoutHint = sub.TimeoutHint
	}
	if p.TimeoutHint < minTimeoutHint {
		p.TimeoutHint = minTimeoutHint
	}
	p.subMux.Unlock()

	return nil
}

func (p *publishEngine) UnregisterSubscription(subID uint32) {
	p.ActiveSubscriptionCount--
	p.subMux.Lock()
	_, found := p.subscriptions[subID]
	if !found {
		return
	}
	delete(p.subscriptions, subID)
	p.subMux.Unlock()
}

func (p *publishEngine) SubscriptionIDs() []uint32 {
	subscriptionIDs := []uint32{}
	p.subMux.Lock()
	for key := range p.subscriptions {
		subscriptionIDs = append(subscriptionIDs, key)
	}
	p.subMux.Unlock()
	return subscriptionIDs
}

func (p *publishEngine) GetSubscription(subscriptionID uint32) *Subscription {
	p.subMux.Lock()
	val, found := p.subscriptions[subscriptionID]
	p.subMux.Unlock()
	if !found {
		return nil
	}
	return val
}

func (p *publishEngine) HasSubscription(subscriptionID uint32) bool {
	p.subMux.Lock()
	_, found := p.subscriptions[subscriptionID]
	p.subMux.Unlock()
	if !found {
		return false
	}
	return true
}

func (p *publishEngine) Suspend() {
	p.chanRunning <- false
}

func (p *publishEngine) Resume() {
	p.chanRunning <- true
}

func (p *publishEngine) Terminate() {
	p.s = nil
}

func (p *publishEngine) Run(ctx context.Context) {

	closePublishedNotif := func(notif chan error) {
		// drain the channel
		defer close(notif)
		timer := time.NewTimer(uasc.MaxTimeout)
		defer timer.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-notif:
				timer.Reset(uasc.MaxTimeout)
			case <-timer.C:
				return
			}
		}
	}

	publishedNotif := make(chan error, 1)
	p.replenishPublishRequestQueue(ctx, publishedNotif)
	for {
		select {
		case <-ctx.Done():
			return
		case run := <-p.chanRunning:
			if run {
				continue
			}

			debug.Printf("opcua: publish engine is suspended")
			p.setIsSuspended(true)
			for !run {
				select {
				case <-ctx.Done():
					return
				case run = <-p.chanRunning:
					if run {
						debug.Printf("opcua: publish engine is resumed")
						// Flush the channel to avoid previous error on restarting
						go closePublishedNotif(publishedNotif)
						publishedNotif = make(chan error, 1)

						p.setIsSuspended(false)
						p.replenishPublishRequestQueue(ctx, publishedNotif)
					}
				}
			}
		case err := <-publishedNotif:

			if err != nil {
				debug.Printf("opcua: suspending publish engine. %s", err.Error())
				p.Suspend()
			}

			if p.ActiveSubscriptionCount > 0 && !p.IsSuspended() {
				go p.attemptToFetchPublish(ctx, publishedNotif)
			}
		}
	}
}

// Fill in the Publish request Queue
func (p *publishEngine) replenishPublishRequestQueue(ctx context.Context, publishedNotif chan error) {
	// Spec 1.03 part 4 5.13.5 Publish
	// [..] in high latency networks, the Client may wish to pipeline Publish requests
	// to ensure cyclic reporting from the Server. Pipe-lining involves sending more than one Publish
	// request for each Subscription before receiving a response. For example, if the network introduces a
	// delay between the Client and the Server of 5 seconds and the publishing interval for a Subscription
	// is one second, then the Client will have to issue Publish requests every second instead of waiting for
	// a response to be received before sending the next request.

	// send more than one publish request to server to cope with latency
	for i := uint32(0); i < publishRequestCountInPipeline+1; i++ {
		select {
		case <-ctx.Done():
			return
		default:
			go p.attemptToFetchPublish(ctx, publishedNotif)
		}
	}
}

func (p *publishEngine) attemptToFetchPublish(ctx context.Context, publishedNotif chan error) {
	if p.IsSuspended() {
		debug.Printf("opcua: publish Engine should not publish while suspended")
		publishedNotif <- errors.Errorf("Publish Engine should not publish while suspended")
		return
	}

	if p.NbPendingPublishRequests >= p.NbMaxPublishRequestsAcceptedByServer {
		publishedNotif <- errors.Errorf("Publish Engine NbMaxPublishRequestsAcceptedByServer has been reached")
		return
	}

	p.sessionMux.Lock()
	if p.s != nil && !p.s.isChannelValid() {
		p.sessionMux.Unlock()

		// wait for channel  to be valid
		timer := time.NewTimer(100 * time.Millisecond)

		select {
		case <-ctx.Done():
		case <-timer.C:
			go p.attemptToFetchPublish(ctx, publishedNotif)
		}
		return
	}
	if p.s == nil || p.IsSuspended() {
		p.sessionMux.Unlock()
		publishedNotif <- errors.Errorf("Session has been terminated or suspended")
		return
	}
	p.sessionMux.Unlock()
	p.fetchPublish(ctx, publishedNotif)
}

func (p *publishEngine) fetchPublish(ctx context.Context, publishedNotif chan error) {
	if p.s == nil {
		publishedNotif <- errors.Errorf("Publish Engine terminated ?")
		return
	}
	if p.IsSuspended() {
		debug.Printf("opcua: publish Engine should not publish while suspended")
		publishedNotif <- errors.Errorf("Publish Engine should not publish while suspended")
		return
	}
	p.NbPendingPublishRequests++
	debug.Printf("opcua: sending publish request")

	acks := p.SubscriptionAcknowledgements
	p.SubscriptionAcknowledgements = []*ua.SubscriptionAcknowledgement{}

	// as started in the spec (Spec 1.02 part 4 page 81 5.13.2.2 Function DequeuePublishReq())
	// the server will dequeue the PublishRequest  in first-in first-out order
	// and will validate if the publish request is still valid by checking the timeoutHint in the RequestHeader.
	// If the request timed out, the server will send a Bad_Timeout service result for the request and de-queue
	// another publish request.
	//
	// in Part 4. page 144 Request Header the timeoutHint is described this way.
	// timeoutHint UInt32 This timeout in milliseconds is used in the Client side Communication Stack to
	//                    set the timeout on a per-call base.
	//                    For a Server this timeout is only a hint and can be used to cancel long running
	//                    operations to free resources. If the Server detects a timeout, he can cancel the
	//                    operation by sending the Service result Bad_Timeout. The Server should wait
	//                    at minimum the timeout after he received the request before cancelling the operation.
	//                    The value of 0 indicates no timeout.
	// In issue#40 (MonitoredItem on changed not fired), we have found that some server might wrongly interpret
	// the timeoutHint of the request header ( and will bang a Bad_Timeout regardless if client send timeoutHint=0)
	// as a work around here , we force the timeoutHint to be set to a suitable value.
	//
	// see https://github.com/node-opcua/node-opcua/issues/141
	// This suitable value shall be at least the time between two keep alive signal that the server will send.
	// (i.e revisedLifetimeCount * revisedPublishingInterval)

	// also ( part 3 - Release 1.03 page 140)
	// The Server shall check the timeoutHint parameter of a PublishRequest before processing a PublishResponse.
	// If the request timed out, a Bad_Timeout Service result is sent and another PublishRequest is used.
	// The value of 0 indicates no timeout

	// in our case:

	timeout := time.Duration(p.NbPendingPublishRequests*p.TimeoutHint) * time.Microsecond
	res, err := p.s.publish(acks, timeout)
	p.NbPendingPublishRequests--
	debug.Printf("opcua: publish request in queue: %d.", p.NbPendingPublishRequests)
	if p.NbPendingPublishRequests == 0 {
		debug.Printf("opcua: publish request queue is empty.")
		publishedNotif <- errors.Errorf("The Publish Request Queue is empty")
		return
	}

	if res != nil {
		debug.Printf("opcua: receive publish response.")
		p.prepareNextSubscriptionAcknowledgements(res)
	}

	if err != nil {
		debug.Printf("opcua: publish engine fetchPublish: %s", err.Error())
		switch {
		case err == ua.StatusBadSequenceNumberUnknown:
			// At least one ack has been submitted repeatedly
			// Ignore the error. Acks will be cleared below
			publishedNotif <- nil
		case err == ua.StatusBadTimeout:
			// ignore and continue the loop
			debug.Printf("opcua: publish request timeout, because server had no data to send")
			publishedNotif <- nil
		case err == ua.StatusBadNoSubscription:
			// All subscriptions have been deleted, but the publishing loop is still running
			// The user will stop the loop or create subscriptions at his discretion
			debug.Printf("opcua: subscription has been deleted by the server")
			publishedNotif <- nil

		case err == ua.StatusBadNoSubscription && p.ActiveSubscriptionCount >= 1:
			// there is something wrong happening here.
			// the server tells us that there is no subscription for this session
			// but the client have some active subscription left.
			// This could happen if the client has missed or not received the StatusChange Notification
			debug.Printf("opcua: server tells that is has no subscription, but client disagree")
			debug.Printf("opcua: activeSubscriptionCount = %d", p.ActiveSubscriptionCount)
			publishedNotif <- nil

		case err == ua.StatusBadSessionClosed || err == ua.StatusBadSessionIDInvalid:
			// server has closed the session ....
			// may be the session timeout is shorted than the subscription life time
			// and the client does not send intermediate keepAlive request to keep the connection working.
			debug.Printf("opcua: server tells that the session has closed ...")
			debug.Printf("opcua: the publish engine shall now be disabled as server will reject any further request")
			publishedNotif <- err

		case err == ua.StatusBadTooManyPublishRequests:
			// preventing queue overflow
			// -------------------------
			//   if the client send too many publish requests that the server can queue, the server returns
			//   a Service result of BadTooManyPublishRequests.
			//
			//   let adjust the nbMaxPublishRequestsAcceptedByServer value so we never overflow the server
			//   with extraneous publish requests in the future.
			//

			if p.NbMaxPublishRequestsAcceptedByServer > p.NbPendingPublishRequests {
				p.NbMaxPublishRequestsAcceptedByServer = p.NbPendingPublishRequests
			}
			debug.Printf("opcua: server tells that that too many publish request has been sent ...")
			debug.Printf("opcua: on Client side nbPendingPublishRequests = %d (max: %d)",
				p.NbPendingPublishRequests,
				p.NbMaxPublishRequestsAcceptedByServer,
			)
			publishedNotif <- nil

		default:
			if err == io.EOF {
				// the previous publish request has ended up with an error because
				// the connection has failed ...
				// There is no need to send more publish request for the time being until reconnection is completed
				debug.Printf("opcua: client is not connected: May be reconnection is in progress")
				debug.Printf("opcua: activeSubscriptionCount = %d", p.ActiveSubscriptionCount)
			}
			p.notifySubscriptionsOfError(ctx, res, err)
		}
	} else {
		err := p.notifySubscription(ctx, res)
		if err == ua.StatusBadSequenceNumberUnknown {
			publishedNotif <- nil
		} else {
			publishedNotif <- err
		}
	}
}

func (p *publishEngine) notifySubscriptionsOfError(ctx context.Context, res *ua.PublishResponse, err error) {
	p.subMux.RLock()
	defer p.subMux.RUnlock()

	subsToNotify := p.subscriptions
	if res != nil && res.SubscriptionID != 0 {
		subsToNotify = map[uint32]*Subscription{
			res.SubscriptionID: p.subscriptions[res.SubscriptionID],
		}
	}
	for _, sub := range subsToNotify {
		go func(s *Subscription) {
			s.sendNotification(ctx, &PublishNotificationData{Error: err})
		}(sub)
	}
}

func (p *publishEngine) notifySubscription(ctx context.Context, response *ua.PublishResponse) error {
	p.subMux.RLock()
	sub, ok := p.subscriptions[response.SubscriptionID]
	p.subMux.RUnlock()
	if !ok {
		debug.Printf("opcua: unknown subscription: %v", response.SubscriptionID)
		return errors.Errorf("Unknown subscription: %v", response.SubscriptionID)
	}

	// Check for errors
	status := ua.StatusOK
	for _, res := range response.Results {
		if res != ua.StatusOK {
			status = res
			break
		}
	}

	if status == ua.StatusBadSequenceNumberUnknown {
		// the session was suspended for too long...
		// the publish request couldn't find the server node value
		// for the server Publish Request Queue dropped the value
		debug.Printf("opcua: value was dropped by the server")
	}

	if status != ua.StatusOK {
		sub.sendNotification(ctx, &PublishNotificationData{
			SubscriptionID: response.SubscriptionID,
			Error:          status,
		})
		return status
	}

	if response.NotificationMessage == nil {
		err := errors.Errorf("empty NotificationMessage")
		sub.sendNotification(ctx, &PublishNotificationData{
			SubscriptionID: response.SubscriptionID,
			Error:          err,
		})
		return err
	}

	// Part 4, 7.21 NotificationMessage
	for _, data := range response.NotificationMessage.NotificationData {
		// Part 4, 7.20 NotificationData parameters
		if data == nil || data.Value == nil {
			sub.sendNotification(ctx, &PublishNotificationData{
				SubscriptionID: response.SubscriptionID,
				Error:          errors.Errorf("missing NotificationData parameter"),
			})
			continue
		}

		switch data.Value.(type) {
		// Part 4, 7.20.2 DataChangeNotification parameter
		// Part 4, 7.20.3 EventNotificationList parameter
		// Part 4, 7.20.4 StatusChangeNotification parameter
		case *ua.DataChangeNotification,
			*ua.EventNotificationList,
			*ua.StatusChangeNotification:
			sub.sendNotification(ctx, &PublishNotificationData{
				SubscriptionID: response.SubscriptionID,
				Value:          data.Value,
			})

		// Error
		default:
			sub.sendNotification(ctx, &PublishNotificationData{
				SubscriptionID: response.SubscriptionID,
				Error:          errors.Errorf("unknown NotificationData parameter: %T", data.Value),
			})
		}
	}
	return nil
}

func (p *publishEngine) RepairSubscriptions(subscriptionIDs ...uint32) error {

	if subscriptionIDs == nil {
		subscriptionIDs = p.SubscriptionIDs()
	}

	p.subMux.RLock()
	defer p.subMux.RUnlock()

	for _, subID := range subscriptionIDs {
		sub, ok := p.subscriptions[subID]
		if !ok {
			return errors.Errorf("Invalid SubscriptionID")
		}
		if err := p.repairSubscription(sub); err != nil {
			return err
		}
	}

	return nil
}

func (p *publishEngine) repairSubscription(subscription *Subscription) error {
	debug.Printf("opcua: repairSubscription  for SubscriptionId %d", subscription.SubscriptionID)
	if status, err := p.republish(subscription); err != nil {
		switch status {
		case ua.StatusBadSessionIDInvalid: /*BadSessionInvalid*/
			return err
		case ua.StatusBadSubscriptionIDInvalid:
			debug.Printf("opcua: republish failed, subscriptionId is not valid anymore on server side.")
			return subscription.recreateSubscriptionAndMonitoredItem()
		}
	}
	return nil
}

func (p *publishEngine) republish(subscription *Subscription) (ua.StatusCode, error) {
	isDone := false

	for !isDone {
		request := &ua.RepublishRequest{
			SubscriptionID:           subscription.SubscriptionID,
			RetransmitSequenceNumber: subscription.lastSequenceNumber + 1,
		}

		debug.Printf("opcua: republish Request for subscription %d retransmitSequenceNumber=%d",
			request.SubscriptionID,
			request.RetransmitSequenceNumber,
		)

		if p.s == nil || p.s.c.sessionIsClosed() /*|| closeEventHasBeenEmitted*/ {
			debug.Printf("opcua: publish engine republish aborted")
			isDone = true
			continue
		}
		var res *ua.RepublishResponse
		var err error

		res, err = p.s.republish(request)
		status := ua.StatusBad
		if res != nil {
			status = res.ResponseHeader.ServiceResult
		}
		if err != nil && status == ua.StatusOK {
			// reprocess notification message and keep going
		} else {
			if err == nil {
				err = errors.Errorf(res.ResponseHeader.ServiceResult.Error())
			}
			debug.Printf("opcua: republish request ends with: %s", err.Error())
			return status, err
		}
	}
	return ua.StatusOK, nil
}

func (p *publishEngine) prepareNextSubscriptionAcknowledgements(res *ua.PublishResponse) {
	availableSequenceNumbers := []uint32{}
	if res.AvailableSequenceNumbers != nil {
		availableSequenceNumbers = res.AvailableSequenceNumbers
	}

	subscriptionID := res.SubscriptionID
	notificationMessage := res.NotificationMessage
	if len(notificationMessage.NotificationData) != 0 {
		sequenceNumber := notificationMessage.SequenceNumber
		contains := false
		for _, sn := range availableSequenceNumbers {
			if sequenceNumber == sn {
				contains = true
				break
			}
		}
		if !contains {
			availableSequenceNumbers = append(availableSequenceNumbers, sequenceNumber)
		}
	}

	for _, sequenceNumber := range availableSequenceNumbers {
		p.acknowledgeNotification(
			subscriptionID,
			sequenceNumber,
		)
	}
}

func (p *publishEngine) acknowledgeNotification(subscriptionID uint32, sequenceNumber uint32) {
	p.SubscriptionAcknowledgements = append(
		p.SubscriptionAcknowledgements,
		&ua.SubscriptionAcknowledgement{
			SubscriptionID: subscriptionID,
			SequenceNumber: sequenceNumber,
		},
	)
}

func (p *publishEngine) cleanupAcknowledgementForSubscription(subscriptionID uint32) {
	filtered := []*ua.SubscriptionAcknowledgement{}
	for _, subAck := range p.SubscriptionAcknowledgements {
		if subAck.SubscriptionID != subscriptionID {
			filtered = append(filtered, subAck)
		}
	}
	p.SubscriptionAcknowledgements = filtered
}
