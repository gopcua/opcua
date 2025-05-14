package server

import (
	"context"
	"sync"
	"time"

	"github.com/gopcua/opcua/internal/ualog"
	"github.com/gopcua/opcua/ua"
	"github.com/gopcua/opcua/uasc"
)

// SubscriptionService implements the Subscription Service Set.
//
// https://reference.opcfoundation.org/Core/Part4/v105/docs/5.13
type SubscriptionService struct {
	srv *Server

	// mu guards subs
	mu sync.Mutex

	// subs stores the active subscriptions by id
	subs map[uint32]*Subscription
}

// get rid of all references to a subscription and all monitored items that are pointed at this subscription.
func (s *SubscriptionService) DeleteSubscription(id uint32) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sub, ok := s.subs[id]
	if ok {
		sub.Mu.Lock()
		if sub.running {
			sub.running = false
			close(sub.shutdown)
		}
		sub.Mu.Unlock()
	}

	delete(s.subs, id)

	// ask the monitored item service to purge out any items that use this subscription
	s.srv.MonitoredItemService.DeleteSub(id)

}

// https://reference.opcfoundation.org/Core/Part4/v105/docs/5.13.2
func (s *SubscriptionService) CreateSubscription(sc *uasc.SecureChannel, r ua.Request, reqID uint32) (ua.Response, error) {
	dlog := s.srv.logger.With("func", "SubscriptionService.CreateSubscription")
	dlog.Debug("Handling", "type", ualog.TypeOf(r))

	req, err := safeReq[*ua.CreateSubscriptionRequest](r)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	newsubid := uint32(len(s.subs)) + 1

	dlog.Info("New subscription", "sub_id", newsubid, "remote_addr", sc.RemoteAddr())

	sub := NewSubscription()
	sub.srv = s
	sub.Session = s.srv.Session(r.Header())
	sub.Channel = sc
	sub.ID = newsubid
	sub.RevisedPublishingInterval = req.RequestedPublishingInterval
	sub.RevisedLifetimeCount = req.RequestedLifetimeCount
	sub.RevisedMaxKeepAliveCount = req.RequestedMaxKeepAliveCount

	s.subs[newsubid] = sub
	sub.running = true
	sub.Start()

	resp := &ua.CreateSubscriptionResponse{
		ResponseHeader: &ua.ResponseHeader{
			Timestamp:          time.Now(),
			RequestHandle:      req.RequestHeader.RequestHandle,
			ServiceResult:      ua.StatusOK,
			ServiceDiagnostics: &ua.DiagnosticInfo{},
			StringTable:        []string{},
			AdditionalHeader:   ua.NewExtensionObject(nil),
		},
		SubscriptionID:            uint32(newsubid),
		RevisedPublishingInterval: req.RequestedPublishingInterval,
		RevisedLifetimeCount:      req.RequestedLifetimeCount,
		RevisedMaxKeepAliveCount:  req.RequestedMaxKeepAliveCount,
	}
	return resp, nil
}

// https://reference.opcfoundation.org/Core/Part4/v105/docs/5.13.3
func (s *SubscriptionService) ModifySubscription(sc *uasc.SecureChannel, r ua.Request, reqID uint32) (ua.Response, error) {
	dlog := s.srv.logger.With("func", "SubscriptionService.ModifySubscription")
	dlog.Debug("Handling", "type", ualog.TypeOf(r))

	req, err := safeReq[*ua.ModifySubscriptionRequest](r)
	if err != nil {
		return nil, err
	}

	// When this gets implemented, be sure to check the subscription session vs the request session!
	return serviceUnsupported(req.RequestHeader), nil
}

// https://reference.opcfoundation.org/Core/Part4/v105/docs/5.13.4
func (s *SubscriptionService) SetPublishingMode(sc *uasc.SecureChannel, r ua.Request, reqID uint32) (ua.Response, error) {
	dlog := s.srv.logger.With("func", "SubscriptionService.SetPublishingMode")
	dlog.Debug("Handling", "type", ualog.TypeOf(r))

	req, err := safeReq[*ua.SetPublishingModeRequest](r)
	if err != nil {
		return nil, err
	}
	// When this gets implemented, be sure to check the subscription session vs the request session!
	return serviceUnsupported(req.RequestHeader), nil
}

// https://reference.opcfoundation.org/Core/Part4/v105/docs/5.13.5
func (s *SubscriptionService) Publish(sc *uasc.SecureChannel, r ua.Request, reqID uint32) (ua.Response, error) {
	dlog := s.srv.logger.With("func", "SubscriptionService.Publish")
	dlog.Debug("Handling", "type", ualog.TypeOf(r))

	req, err := safeReq[*ua.PublishRequest](r)
	if err != nil {
		dlog.Error("Bad publish request", "error", err)
		return nil, err
	}

	session := s.srv.Session(req.RequestHeader)
	if session == nil {
		response := &ua.PublishResponse{
			ResponseHeader: &ua.ResponseHeader{
				Timestamp:          time.Now(),
				RequestHandle:      req.RequestHeader.RequestHandle,
				ServiceResult:      ua.StatusBadSessionIDInvalid,
				ServiceDiagnostics: &ua.DiagnosticInfo{},
				StringTable:        []string{},
				AdditionalHeader:   ua.NewExtensionObject(nil),
			},
			SubscriptionID:           0,
			MoreNotifications:        false,
			NotificationMessage:      &ua.NotificationMessage{NotificationData: []*ua.ExtensionObject{}},
			AvailableSequenceNumbers: []uint32{}, // an empty array indicates taht we don't support retransmission of messages
			Results:                  []ua.StatusCode{},
			DiagnosticInfos:          []*ua.DiagnosticInfo{},
		}

		return response, nil
	}

	select {
	case session.PublishRequests <- PubReq{Req: req, ID: reqID}:
	default:
		s.srv.logger.Warn("Too many publish reqs.")
	}

	// per opcua spec, we don't respond now.  When data is available on the subscription,
	// the Subscription will respond in the background.
	return nil, nil
}

// https://reference.opcfoundation.org/Core/Part4/v105/docs/5.13.6
func (s *SubscriptionService) Republish(sc *uasc.SecureChannel, r ua.Request, reqID uint32) (ua.Response, error) {
	dlog := s.srv.logger.With("func", "SubscriptionService.Republish")
	dlog.Debug("Handling", "type", ualog.TypeOf(r))

	req, err := safeReq[*ua.RepublishRequest](r)
	if err != nil {
		return nil, err
	}
	return serviceUnsupported(req.RequestHeader), nil
}

// https://reference.opcfoundation.org/Core/Part4/v105/docs/5.13.7
func (s *SubscriptionService) TransferSubscriptions(sc *uasc.SecureChannel, r ua.Request, reqID uint32) (ua.Response, error) {
	dlog := s.srv.logger.With("func", "SubscriptionService.TransferSubscription")
	dlog.Debug("Handling", "type", ualog.TypeOf(r))

	req, err := safeReq[*ua.TransferSubscriptionsRequest](r)
	if err != nil {
		return nil, err
	}
	// When this gets implemented, be sure to check the subscription session vs the request session!
	return serviceUnsupported(req.RequestHeader), nil
}

// https://reference.opcfoundation.org/Core/Part4/v105/docs/5.13.8
func (s *SubscriptionService) DeleteSubscriptions(sc *uasc.SecureChannel, r ua.Request, reqID uint32) (ua.Response, error) {
	dlog := s.srv.logger.With("func", "SubscriptionService.DeleteSubscription")
	dlog.Debug("Handling", "type", ualog.TypeOf(r))

	req, err := safeReq[*ua.DeleteSubscriptionsRequest](r)
	if err != nil {
		return nil, err
	}
	session := s.srv.Session(req.Header())

	s.mu.Lock()
	defer s.mu.Unlock()

	results := make([]ua.StatusCode, len(req.SubscriptionIDs))
	for i := range req.SubscriptionIDs {
		subid := req.SubscriptionIDs[i]
		dlog.Info("Subscription deleted by client", "sub_id", subid)
		sub, ok := s.subs[subid]
		if !ok {
			results[i] = ua.StatusBadSubscriptionIDInvalid
			continue
		}
		if session.AuthTokenID.String() != sub.Session.AuthTokenID.String() {
			results[i] = ua.StatusBadSessionIDInvalid
			continue
		}
		// delete subscription gets the lock so we set them up to run in the background
		// once this function releases its lock
		go s.DeleteSubscription(subid)
		results[i] = ua.StatusOK
	}

	return &ua.DeleteSubscriptionsResponse{
		ResponseHeader: &ua.ResponseHeader{
			Timestamp:          time.Now(),
			RequestHandle:      req.RequestHeader.RequestHandle,
			ServiceResult:      ua.StatusOK,
			ServiceDiagnostics: &ua.DiagnosticInfo{},
			StringTable:        []string{},
			AdditionalHeader:   ua.NewExtensionObject(nil),
		},
		Results:         results,                //                  []StatusCode
		DiagnosticInfos: []*ua.DiagnosticInfo{}, //          []*DiagnosticInfo
	}, nil
}

type PubReq struct {
	// The data of the publish request
	Req *ua.PublishRequest

	// The request ID (from the header) of the publish request.  This has to be used when replying.
	ID uint32
}

// This is the type that with its run() function will work in the bakground fullfilling subscription
// publishes.
//
// MonitoredItems will send updates on the NotifyChannel to let the background task know that
// an event has occured that needs to be published.
type Subscription struct {
	srv                       *SubscriptionService
	Session                   *session
	ID                        uint32
	RevisedPublishingInterval float64
	RevisedLifetimeCount      uint32
	RevisedMaxKeepAliveCount  uint32
	Channel                   *uasc.SecureChannel
	SequenceID                uint32
	//SeqNums                   map[uint32]struct{}
	T *time.Ticker

	NotifyChannel chan *ua.MonitoredItemNotification
	ModifyChannel chan *ua.ModifySubscriptionRequest

	// the running flag and shutdown channel are used to signal the background task that it should stop.
	// multiple places can kill the subscription so make sure you check the running flag using the mutex
	// before closing the shutdown channel.
	Mu       sync.Mutex
	running  bool
	shutdown chan struct{}
}

func NewSubscription() *Subscription {
	return &Subscription{
		//SeqNums:       map[uint32]struct{}{},
		NotifyChannel: make(chan *ua.MonitoredItemNotification, 100),
		ModifyChannel: make(chan *ua.ModifySubscriptionRequest, 2),
		shutdown:      make(chan struct{}),
	}
}

func (s *Subscription) Update(req *ua.ModifySubscriptionRequest) {
	s.RevisedPublishingInterval = req.RequestedPublishingInterval
	s.RevisedLifetimeCount = req.RequestedLifetimeCount
	s.RevisedMaxKeepAliveCount = req.RequestedMaxKeepAliveCount
}

func (s *Subscription) Start() {
	go s.run()

}

func (s *Subscription) keepalive(pubreq PubReq) error {
	eo := make([]*ua.ExtensionObject, 0)

	msg := ua.NotificationMessage{
		SequenceNumber:   s.SequenceID + 1, // not sure why but ua expert wants the next sequence number on keepalives.
		PublishTime:      time.Now(),
		NotificationData: eo,
	}

	response := &ua.PublishResponse{
		ResponseHeader: &ua.ResponseHeader{
			Timestamp:          time.Now(),
			RequestHandle:      pubreq.Req.RequestHeader.RequestHandle,
			ServiceResult:      ua.StatusOK,
			ServiceDiagnostics: &ua.DiagnosticInfo{},
			StringTable:        []string{},
			AdditionalHeader:   ua.NewExtensionObject(nil),
		},
		SubscriptionID:           s.ID,
		MoreNotifications:        false,
		NotificationMessage:      &msg,
		AvailableSequenceNumbers: []uint32{}, // an empty array indicates taht we don't support retransmission of messages
		Results:                  []ua.StatusCode{},
		DiagnosticInfos:          []*ua.DiagnosticInfo{},
	}
	err := s.Channel.SendResponseWithContext(context.Background(), pubreq.ID, response)
	if err != nil {
		return err
	}
	return nil
}

// this function should be run as a go-routine and will handle sending data out
// to the client at the correct rate assuming there are publish requests queued up.
// if the function returns it deletes the subscription
func (s *Subscription) run() {
	// if this go routine dies, we need to delete ourselves.
	defer func() {
		s.srv.srv.logger.Info("Subscription shutting down", "sub_id", s.ID)
		s.srv.DeleteSubscription(s.ID)
	}()

	keepalive_counter := 0
	lifetime_counter := 0
	//TODO: if a sub is modified, this ticker time may need to change.
	s.T = time.NewTicker(time.Millisecond * time.Duration(s.RevisedPublishingInterval))
	defer s.T.Stop()

	// This is the master run event loop.  It has effectively 3 states that it can be in.  The first two are designated with
	// the labels L0, and L2.  Everything after the L2 loop is the third state where we send any pending notifications.
	// The states always go L0 -> L2 -> Sending -> L0.  L0 and L2 are both places where we wait so they are done as for loops with
	// breaks to go to the next state.
	// The sending state always runs to completion.
	//
	// L0 waits for our notification interval to expire.  Any notifications that come in
	// while waiting will be stored in the publishQueue.  Once the interval expires, we'll move on to L2 if we've got notifications.
	// In L2 we wait for a publish request.  If we get one, we'll publish the notifications in the publishQueue.  If we don't
	// get a publish request, we'll continue to count intervals without a publish request.
	//
	// In L0 and L2, If we get to the lifetime count without a publish request, we'll kill the subscription.
	for {
		// we don't need to do anything if we don't have at least one thing to publish so lets get that first
		publishQueue := make(map[uint32]*ua.MonitoredItemNotification)

		// Collect notifications until our publication interval is ready
	L0:
		for {
			select {
			case <-s.shutdown:
				return
			case newNotification := <-s.NotifyChannel:
				publishQueue[newNotification.ClientHandle] = newNotification
			case <-s.T.C:
				if len(publishQueue) == 0 {
					// nothing to publish, increment the keepalive counter and send a keepalive if it
					// has been enough intervals.
					keepalive_counter++
					if keepalive_counter > int(s.RevisedMaxKeepAliveCount) {
						keepalive_counter = 0
						select {
						case pubreq := <-s.Session.PublishRequests:
							err := s.keepalive(pubreq)
							if err != nil {
								s.srv.srv.logger.Warn("problem sending keepalive to subscription", "sub_id", s.ID, "error", err)
								return
							}
						default:
							lifetime_counter++
							if lifetime_counter > int(s.RevisedLifetimeCount) {
								s.srv.srv.logger.Warn("Subscription timed out", "sub_id", s.ID)
								return
							}
						}
					}
					continue // nothing to publish this interval
				}
				// we have things to publish so we'll break out to do that.
				break L0
			case update := <-s.ModifyChannel:
				s.Update(update)
			}
		}
		var pubreq PubReq

		// now we need to continue to collect notifications until we've got a publish request
	L2:
		for {
			select {
			case <-s.shutdown:
				return
			case pubreq = <-s.Session.PublishRequests:
				// once we get a publish request, we should move on to publish them back
				break L2
			case newNotification := <-s.NotifyChannel:
				publishQueue[newNotification.ClientHandle] = newNotification

			case <-s.T.C:
				// we had another tick without a publish request.
				lifetime_counter++
				if lifetime_counter > int(s.RevisedLifetimeCount) {
					s.srv.srv.logger.Warn("Subscription timed out", "sub_id", s.ID)
					return
				}
			}
		}
		lifetime_counter = 0
		keepalive_counter = 0

		s.SequenceID++
		if s.SequenceID == 0 {
			// per the spec, the sequence ID cannot be 0
			s.SequenceID = 1
		}
		s.srv.srv.logger.Debug("Got publish req", "sub_id", s.ID, "seq_nr", s.SequenceID)
		// then get all the tags and send them back to the client

		//for x := range pubreq.Req.SubscriptionAcknowledgements {
		//a := pubreq.Req.SubscriptionAcknowledgements[x]
		//delete(s.SeqNums, a.SequenceNumber)
		//}

		final_items := make([]*ua.MonitoredItemNotification, len(publishQueue))
		i := 0
		for k := range publishQueue {
			final_items[i] = publishQueue[k]
			i++
		}

		dcn := ua.DataChangeNotification{
			MonitoredItems:  final_items,
			DiagnosticInfos: []*ua.DiagnosticInfo{},
		}
		eo := make([]*ua.ExtensionObject, 1)
		eo[0] = ua.NewExtensionObject(&dcn)
		eo[0].UpdateMask()

		msg := ua.NotificationMessage{
			SequenceNumber:   s.SequenceID,
			PublishTime:      time.Now(),
			NotificationData: eo,
		}
		//s.SeqNums[s.SequenceID] = struct{}{}

		response := &ua.PublishResponse{
			ResponseHeader: &ua.ResponseHeader{
				Timestamp:          time.Now(),
				RequestHandle:      pubreq.Req.RequestHeader.RequestHandle,
				ServiceResult:      ua.StatusOK,
				ServiceDiagnostics: &ua.DiagnosticInfo{},
				StringTable:        []string{},
				AdditionalHeader:   ua.NewExtensionObject(nil),
			},
			SubscriptionID:           s.ID,
			MoreNotifications:        false,
			NotificationMessage:      &msg,
			AvailableSequenceNumbers: []uint32{}, // an empty array indicates taht we don't support retransmission of messages
			Results:                  []ua.StatusCode{},
			DiagnosticInfos:          []*ua.DiagnosticInfo{},
		}
		err := s.Channel.SendResponseWithContext(context.Background(), pubreq.ID, response)
		if err != nil {
			s.srv.srv.logger.Error("problem sending channel response", "error", err)
			s.srv.srv.logger.Error("Killing subscription", "sub_id", s.ID)
			return
		}
		s.srv.srv.logger.Debug("Published OK items", "sub_id", s.ID, "item_count", len(publishQueue))
		// wait till we've got a publish request.
	}
}

//PublishRequest_Encoding_DefaultBinary
