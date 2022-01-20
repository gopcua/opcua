package opcua

import (
	"context"
	"io"
	"log"
	"time"

	"github.com/gopcua/opcua/debug"
	"github.com/gopcua/opcua/errors"
	"github.com/gopcua/opcua/stats"
	"github.com/gopcua/opcua/ua"
	"github.com/gopcua/opcua/uasc"
)

// Subscribe creates a Subscription with given parameters.
// Parameters that have not been set are set to their default values.
// See opcua.DefaultSubscription* constants
//
// Note: Starting with v0.5 this method will require a context
// and the corresponding XXXWithContext(ctx) method will be removed.
func (c *Client) Subscribe(params *SubscriptionParameters, notifyCh chan<- *PublishNotificationData) (*Subscription, error) {
	return c.SubscribeWithContext(context.Background(), params, notifyCh)
}

// Note: Starting with v0.5 this method is superseded by the non 'WithContext' method.
func (c *Client) SubscribeWithContext(ctx context.Context, params *SubscriptionParameters, notifyCh chan<- *PublishNotificationData) (*Subscription, error) {
	stats.Client().Add("Subscribe", 1)

	if params == nil {
		params = &SubscriptionParameters{}
	}

	params.setDefaults()
	req := &ua.CreateSubscriptionRequest{
		RequestedPublishingInterval: float64(params.Interval / time.Millisecond),
		RequestedLifetimeCount:      params.LifetimeCount,
		RequestedMaxKeepAliveCount:  params.MaxKeepAliveCount,
		PublishingEnabled:           true,
		MaxNotificationsPerPublish:  params.MaxNotificationsPerPublish,
		Priority:                    params.Priority,
	}

	var res *ua.CreateSubscriptionResponse
	err := c.SendWithContext(ctx, req, func(v interface{}) error {
		return safeAssign(v, &res)
	})
	if err != nil {
		return nil, err
	}
	if res.ResponseHeader.ServiceResult != ua.StatusOK {
		return nil, res.ResponseHeader.ServiceResult
	}

	stats.Subscription().Add("Count", 1)

	// start the publish loop if it isn't already running
	c.resumech <- struct{}{}

	sub := &Subscription{
		SubscriptionID:            res.SubscriptionID,
		RevisedPublishingInterval: time.Duration(res.RevisedPublishingInterval) * time.Millisecond,
		RevisedLifetimeCount:      res.RevisedLifetimeCount,
		RevisedMaxKeepAliveCount:  res.RevisedMaxKeepAliveCount,
		Notifs:                    notifyCh,
		items:                     make(map[uint32]*monitoredItem),
		params:                    params,
		nextSeq:                   1,
		c:                         c,
	}

	c.subMux.Lock()
	defer c.subMux.Unlock()

	if sub.SubscriptionID == 0 || c.subs[sub.SubscriptionID] != nil {
		// this should not happen and is usually indicative of a server bug
		// see: Part 4 Section 5.13.2.2, Table 88 – CreateSubscription Service Parameters
		return nil, ua.StatusBadSubscriptionIDInvalid
	}

	c.subs[sub.SubscriptionID] = sub
	c.updatePublishTimeout()
	return sub, nil
}

// SubscriptionIDs gets a list of subscriptionIDs
func (c *Client) SubscriptionIDs() []uint32 {
	c.subMux.RLock()
	defer c.subMux.RUnlock()

	var ids []uint32
	for id := range c.subs {
		ids = append(ids, id)
	}
	return ids
}

// recreateSubscriptions creates new subscriptions
// with the same parameters to replace the previous ones
func (c *Client) recreateSubscription(ctx context.Context, id uint32) error {
	sub, ok := c.subs[id]
	if !ok {
		return ua.StatusBadSubscriptionIDInvalid
	}
	return sub.recreate(ctx)
}

// transferSubscriptions ask the server to transfer the given subscriptions
// of the previous session to the current one.
func (c *Client) transferSubscriptions(ctx context.Context, ids []uint32) (*ua.TransferSubscriptionsResponse, error) {
	req := &ua.TransferSubscriptionsRequest{
		SubscriptionIDs:   ids,
		SendInitialValues: false,
	}

	var res *ua.TransferSubscriptionsResponse
	err := c.SendWithContext(ctx, req, func(v interface{}) error {
		return safeAssign(v, &res)
	})
	return res, err
}

// republishSubscriptions sends republish requests for the given subscription id.
func (c *Client) republishSubscription(ctx context.Context, id uint32, availableSeq []uint32) error {
	c.subMux.RLock()
	defer c.subMux.RUnlock()

	sub, ok := c.subs[id]
	if !ok {
		return errors.Errorf("invalid subscription id %d", id)
	}

	debug.Printf("republishing subscription %d", sub.SubscriptionID)
	if err := c.sendRepublishRequests(ctx, sub, availableSeq); err != nil {
		status, ok := err.(ua.StatusCode)
		if !ok {
			return err
		}

		switch status {
		case ua.StatusBadSessionIDInvalid:
			return nil
		case ua.StatusBadSubscriptionIDInvalid:
			// todo(fs): do we need to forget the subscription id in this case?
			debug.Printf("republish failed since subscription %d is invalid", sub.SubscriptionID)
			return errors.Errorf("republish failed since subscription %d is invalid", sub.SubscriptionID)
		}
	}
	return nil
}

// sendRepublishRequests sends republish requests for the given subscription
// until it gets a BadMessageNotAvailable which implies that there are no
// more messages to restore.
func (c *Client) sendRepublishRequests(ctx context.Context, sub *Subscription, availableSeq []uint32) error {
	// todo(fs): check if sub.nextSeq is in the available sequence numbers
	// todo(fs): if not then we need to decide whether we fail b/c of data loss
	// todo(fs): or whether we log it and continue.
	if len(availableSeq) > 0 && !uint32SliceContains(sub.nextSeq, availableSeq) {
		log.Printf("sub %d: next sequence number %d not in retransmission buffer %v", sub.SubscriptionID, sub.nextSeq, availableSeq)
	}

	for {
		req := &ua.RepublishRequest{
			SubscriptionID:           sub.SubscriptionID,
			RetransmitSequenceNumber: sub.nextSeq,
		}

		debug.Printf("Republishing subscription %d and sequence number %d",
			req.SubscriptionID,
			req.RetransmitSequenceNumber,
		)

		if c.sessionClosed() {
			debug.Printf("Republishing subscription %d aborted", req.SubscriptionID)
			return ua.StatusBadSessionClosed
		}

		debug.Printf("RepublishRequest: req=%s", debug.ToJSON(req))
		var res *ua.RepublishResponse
		err := c.SecureChannel().SendRequestWithContext(ctx, req, c.Session().resp.AuthenticationToken, func(v interface{}) error {
			return safeAssign(v, &res)
		})
		debug.Printf("RepublishResponse: res=%s err=%v", debug.ToJSON(res), err)

		switch {
		case err == ua.StatusBadMessageNotAvailable:
			// No more message to restore
			debug.Printf("Republishing subscription %d OK", req.SubscriptionID)
			return nil

		case err != nil:
			debug.Printf("Republishing subscription %d failed: %v", req.SubscriptionID, err)
			return err

		default:
			status := ua.StatusBad
			if res != nil {
				status = res.ResponseHeader.ServiceResult
			}

			if status != ua.StatusOK {
				debug.Printf("Republishing subscription %d failed: %v", req.SubscriptionID, status)
				return status
			}
		}
		time.Sleep(time.Second)
	}
}

// registerSubscription register a subscription
func (c *Client) registerSubscription(sub *Subscription) error {
	c.subMux.Lock()
	defer c.subMux.Unlock()

	if sub.SubscriptionID == 0 {
		return ua.StatusBadSubscriptionIDInvalid
	}

	if _, ok := c.subs[sub.SubscriptionID]; ok {
		return errors.Errorf("SubscriptionID %d already registered", sub.SubscriptionID)
	}

	c.subs[sub.SubscriptionID] = sub
	return nil
}

func (c *Client) forgetSubscription(ctx context.Context, id uint32) {
	c.subMux.Lock()
	delete(c.subs, id)
	c.updatePublishTimeout()
	c.subMux.Unlock()
	stats.Subscription().Add("Count", -1)

	if len(c.subs) == 0 {
		c.pauseSubscriptions(ctx)
	}
}

func (c *Client) updatePublishTimeout() {
	// we need to hold the subMux lock already

	maxTimeout := uasc.MaxTimeout
	for _, s := range c.subs {
		if d := s.publishTimeout(); d < maxTimeout {
			maxTimeout = d
		}
	}
	c.setPublishTimeout(maxTimeout)
}

func (c *Client) notifySubscriptionsOfError(ctx context.Context, subID uint32, err error) {
	// we need to hold the subMux lock already

	subsToNotify := c.subs
	if subID != 0 {
		subsToNotify = map[uint32]*Subscription{
			subID: c.subs[subID],
		}
	}
	for _, sub := range subsToNotify {
		go func(s *Subscription) {
			s.notify(ctx, &PublishNotificationData{Error: err})
		}(sub)
	}
}

func (c *Client) notifySubscription(ctx context.Context, sub *Subscription, notif *ua.NotificationMessage) {
	// todo(fs): response.Results contains the status codes of which messages were
	// todo(fs): were successfully removed from the transmission queue on the server.
	// todo(fs): The client sent the list of ids in the *previous* PublishRequest.
	// todo(fs): If we want to handle them then we probably need to keep track
	// todo(fs): of the message ids we have ack'ed.
	// todo(fs): see discussion in https://github.com/gopcua/opcua/issues/337

	if notif == nil {
		sub.notify(ctx, &PublishNotificationData{
			SubscriptionID: sub.SubscriptionID,
			Error:          errors.Errorf("empty NotificationMessage"),
		})
		return
	}

	// Part 4, 7.21 NotificationMessage
	for _, data := range notif.NotificationData {
		// Part 4, 7.20 NotificationData parameters
		if data == nil || data.Value == nil {
			sub.notify(ctx, &PublishNotificationData{
				SubscriptionID: sub.SubscriptionID,
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
			sub.notify(ctx, &PublishNotificationData{
				SubscriptionID: sub.SubscriptionID,
				Value:          data.Value,
			})

		// Error
		default:
			sub.notify(ctx, &PublishNotificationData{
				SubscriptionID: sub.SubscriptionID,
				Error:          errors.Errorf("unknown NotificationData parameter: %T", data.Value),
			})
		}
	}
}

// pauseSubscriptions suspends the publish loop by signalling the pausech.
// It has no effect if the publish loop is already paused.
func (c *Client) pauseSubscriptions(ctx context.Context) {
	select {
	case <-ctx.Done():
	case c.pausech <- struct{}{}:
	}
}

// resumeSubscriptions restarts the publish loop by signalling the resumech.
// It has no effect if the publish loop is not paused.
func (c *Client) resumeSubscriptions(ctx context.Context) {
	select {
	case <-ctx.Done():
	case c.resumech <- struct{}{}:
	}
}

// monitorSubscriptions sends publish requests and handles publish responses
// for all active subscriptions.
func (c *Client) monitorSubscriptions(ctx context.Context) {
	dlog := debug.NewPrefixLogger("sub: ")
	defer dlog.Print("done")

publish:
	for {
		select {
		case <-ctx.Done():
			dlog.Println("ctx.Done()")
			return

		case <-c.resumech:
			dlog.Print("resume")
			// ignore since not paused

		case <-c.pausech:
			dlog.Print("pause")
			for {
				select {
				case <-ctx.Done():
					dlog.Print("pause: ctx.Done()")
					return

				case <-c.resumech:
					dlog.Print("pause: resume")
					continue publish

				case <-c.pausech:
					dlog.Print("pause: pause")
					// ignore since already paused
				}
			}

		default:
			// send publish request and handle response
			if err := c.publish(ctx); err != nil {
				dlog.Print("error: ", err.Error())
				c.pauseSubscriptions(ctx)
			}
		}
	}
}

// publish sends a publish request and handles the response.
func (c *Client) publish(ctx context.Context) error {
	dlog := debug.NewPrefixLogger("publish: ")

	c.subMux.RLock()
	dlog.Printf("pendingAcks=%s", debug.ToJSON(c.pendingAcks))
	c.subMux.RUnlock()

	// send the next publish request
	// note that res contains data even if an error was returned
	res, err := c.sendPublishRequest(ctx)
	stats.RecordError(err)
	switch {
	case err == io.EOF:
		dlog.Printf("eof: pausing publish loop")
		return err

	case err == ua.StatusBadSessionNotActivated:
		dlog.Printf("error: session not active. pausing publish loop")
		return err

	case err == ua.StatusBadServerNotConnected:
		dlog.Printf("error: no connection. pausing publish loop")
		return err

	case err == ua.StatusBadSequenceNumberUnknown:
		// todo(fs): this should only happen per in the status codes
		// todo(fs): lets log this here to see
		dlog.Printf("error: this should only happen when ACK'ing results: %s", err)

	case err == ua.StatusBadTooManyPublishRequests:
		// todo(fs): we have sent too many publish requests
		// todo(fs): we need to slow down
		dlog.Printf("error: sleeping for one second: %s", err)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second):
		}

	case err == ua.StatusBadTimeout:
		// ignore and continue the loop
		dlog.Printf("error: ignoring: %s", err)

	case err == ua.StatusBadNoSubscription:
		// All subscriptions have been deleted, but the publishing loop is still running
		// We should pause publishing until a subscription has been created
		dlog.Printf("error: no subscriptions but the publishing loop is still running: %s", err)
		return err

	case err != nil && res != nil:
		// irrecoverable error
		// todo(fs): do we need to stop and forget the subscription?
		c.notifySubscriptionsOfError(ctx, res.SubscriptionID, err)
		dlog.Printf("error: %s", err)
		return err

	case err != nil:
		dlog.Printf("error: unexpected error. Do we need to stop the publish loop?: %s", err)
		return err

	default:
		c.subMux.Lock()
		// handle pending acks for all subscriptions
		c.handleAcks(res.Results)

		sub, ok := c.subs[res.SubscriptionID]
		if !ok {
			// todo(fs): should we return an error here?
			dlog.Printf("error: unknown subscription %d", res.SubscriptionID)
			return nil
		}

		// handle the publish response for a specific subscription
		c.handleNotification(ctx, sub, res)
		c.subMux.Unlock()

		c.notifySubscription(ctx, sub, res.NotificationMessage)
		dlog.Printf("notif: %d", res.NotificationMessage.SequenceNumber)
	}

	return nil
}

func (c *Client) handleAcks(res []ua.StatusCode) {
	dlog := debug.NewPrefixLogger("publish: ")

	// we assume that the number of results in the response match
	// the number of pending acks from the previous PublishRequest.
	if len(c.pendingAcks) != len(res) {
		dlog.Printf("error: got %d results for pending ACKs but want %d", len(res), len(c.pendingAcks))
		c.pendingAcks = []*ua.SubscriptionAcknowledgement{}
	}

	// find the messages which we have received but which we have not acked.
	var notAcked []*ua.SubscriptionAcknowledgement
	for i, ack := range c.pendingAcks {
		err := res[i]
		switch err {
		case ua.StatusOK:
			// message ack'ed
		case ua.StatusBadSubscriptionIDInvalid:
			// old subscription id -> skip
			dlog.Printf("error: subscription id invalid. skipping: %s", err)
		case ua.StatusBadSequenceNumberUnknown:
			// server does not have the message in its retransmission queue anymore
			dlog.Printf("error: notif %d/%d not on server anymore: %s", ack.SubscriptionID, ack.SequenceNumber, err)
		default:
			// otherwise, we try to ack again
			notAcked = append(notAcked, ack)
			dlog.Printf("retrying to ACK notif %d/%d: %s", ack.SubscriptionID, ack.SequenceNumber, err)
		}
	}
	c.pendingAcks = notAcked
	dlog.Printf("notAcked=%v", notAcked)
}

func (c *Client) handleNotification(ctx context.Context, sub *Subscription, res *ua.PublishResponse) {
	dlog := debug.NewPrefixLogger("publish: sub %d: ", res.SubscriptionID)

	// keep-alive message
	if len(res.NotificationMessage.NotificationData) == 0 {
		// todo(fs): do we care about the next sequence number?
		sub.nextSeq = res.NotificationMessage.SequenceNumber
		return
	}

	if res.NotificationMessage.SequenceNumber != sub.nextSeq {
		dlog.Printf("error: got notif %d but was expecting notif %d. Data loss?", res.NotificationMessage.SequenceNumber, sub.nextSeq)
	}

	sub.lastSeq = res.NotificationMessage.SequenceNumber
	sub.nextSeq = sub.lastSeq + 1
	c.pendingAcks = append(c.pendingAcks, &ua.SubscriptionAcknowledgement{
		SubscriptionID: res.SubscriptionID,
		SequenceNumber: res.NotificationMessage.SequenceNumber,
	})
}

func (c *Client) sendPublishRequest(ctx context.Context) (*ua.PublishResponse, error) {
	dlog := debug.NewPrefixLogger("publish: ")

	c.subMux.RLock()
	req := &ua.PublishRequest{
		SubscriptionAcknowledgements: c.pendingAcks,
	}
	if req.SubscriptionAcknowledgements == nil {
		req.SubscriptionAcknowledgements = []*ua.SubscriptionAcknowledgement{}
	}
	c.subMux.RUnlock()

	dlog.Printf("PublishRequest: %s", debug.ToJSON(req))
	var res *ua.PublishResponse
	err := c.sendWithTimeout(ctx, req, c.publishTimeout(), func(v interface{}) error {
		return safeAssign(v, &res)
	})
	stats.RecordError(err)
	dlog.Printf("PublishResponse: %s", debug.ToJSON(res))
	return res, err
}
