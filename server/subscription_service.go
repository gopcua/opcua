package server

import (
	"context"
	"sync"
	"time"

	"github.com/gopcua/opcua/ua"
	"github.com/gopcua/opcua/uasc"
)

// SubscriptionService implements the Subscription Service Set.
//
// https://reference.opcfoundation.org/Core/Part4/v105/docs/5.13
type SubscriptionService struct {
	srv *Server
	// pub sub stuff
	Mu   sync.Mutex
	Subs map[uint32]*Subscription
}

// get rid of all references to a subscription and all monitored items that are pointed at this subscription.
func (s *SubscriptionService) DeleteSubscription(id uint32) {
	s.Mu.Lock()
	defer s.Mu.Unlock()

	sub, ok := s.Subs[id]
	if ok {
		sub.Mu.Lock()
		if sub.running {
			sub.running = false
			close(sub.shutdown)
		}
		sub.Mu.Unlock()
	}

	delete(s.Subs, id)

	// ask the monitored item service to purge out any items that use this subscription
	s.srv.MonitoredItemService.DeleteSub(id)

}

// https://reference.opcfoundation.org/Core/Part4/v105/docs/5.13.2
func (s *SubscriptionService) CreateSubscription(sc *uasc.SecureChannel, r ua.Request, reqID uint32) (ua.Response, error) {
	if s.srv.cfg.logger != nil {
		s.srv.cfg.logger.Debug("Handling %T", r)
	}

	req, err := safeReq[*ua.CreateSubscriptionRequest](r)
	if err != nil {
		return nil, err
	}

	s.Mu.Lock()
	defer s.Mu.Unlock()

	newsubid := uint32(len(s.Subs)) + 1

	if s.srv.cfg.logger != nil {
		s.srv.cfg.logger.Info("New Sub %d for %v", newsubid, sc.RemoteAddr())
	}

	sub := NewSubscription()
	sub.srv = s
	sub.Session = s.srv.Session(r.Header())
	sub.Channel = sc
	sub.ID = newsubid
	sub.RevisedPublishingInterval = req.RequestedPublishingInterval
	sub.RevisedLifetimeCount = req.RequestedLifetimeCount
	sub.RevisedMaxKeepAliveCount = req.RequestedMaxKeepAliveCount
	sub.MaxNotificationsPerPublish = req.MaxNotificationsPerPublish

	s.Subs[newsubid] = sub
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
	if s.srv.cfg.logger != nil {
		s.srv.cfg.logger.Debug("Handling %T", r)
	}

	req, err := safeReq[*ua.ModifySubscriptionRequest](r)
	if err != nil {
		return nil, err
	}

	// When this gets implemented, be sure to check the subscription session vs the request session!

	return serviceUnsupported(req.RequestHeader), nil
}

// https://reference.opcfoundation.org/Core/Part4/v105/docs/5.13.4
func (s *SubscriptionService) SetPublishingMode(sc *uasc.SecureChannel, r ua.Request, reqID uint32) (ua.Response, error) {
	if s.srv.cfg.logger != nil {
		s.srv.cfg.logger.Debug("Handling %T", r)
	}

	req, err := safeReq[*ua.SetPublishingModeRequest](r)
	if err != nil {
		return nil, err
	}
	// When this gets implemented, be sure to check the subscription session vs the request session!
	return serviceUnsupported(req.RequestHeader), nil
}

// https://reference.opcfoundation.org/Core/Part4/v105/docs/5.13.5
func (s *SubscriptionService) Publish(sc *uasc.SecureChannel, r ua.Request, reqID uint32) (ua.Response, error) {
	if s.srv.cfg.logger != nil {
		s.srv.cfg.logger.Debug("Raw Publish req")
	}

	req, err := safeReq[*ua.PublishRequest](r)
	if err != nil {
		if s.srv.cfg.logger != nil {
			s.srv.cfg.logger.Error("ERROR: bad PublishRequest Struct")
		}
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
		if s.srv.cfg.logger != nil {
			s.srv.cfg.logger.Warn("Too many publish reqs.")
		}
	}

	// per opcua spec, we don't respond now.  When data is available on the subscription,
	// the Subscription will respond in the background.
	return nil, nil
}

// https://reference.opcfoundation.org/Core/Part4/v105/docs/5.13.6
func (s *SubscriptionService) Republish(sc *uasc.SecureChannel, r ua.Request, reqID uint32) (ua.Response, error) {
	if s.srv.cfg.logger != nil {
		s.srv.cfg.logger.Debug("Handling %T", r)
	}

	req, err := safeReq[*ua.RepublishRequest](r)
	if err != nil {
		return nil, err
	}
	return serviceUnsupported(req.RequestHeader), nil
}

// https://reference.opcfoundation.org/Core/Part4/v105/docs/5.13.7
func (s *SubscriptionService) TransferSubscriptions(sc *uasc.SecureChannel, r ua.Request, reqID uint32) (ua.Response, error) {
	if s.srv.cfg.logger != nil {
		s.srv.cfg.logger.Debug("Handling %T", r)
	}

	req, err := safeReq[*ua.TransferSubscriptionsRequest](r)
	if err != nil {
		return nil, err
	}
	// When this gets implemented, be sure to check the subscription session vs the request session!
	return serviceUnsupported(req.RequestHeader), nil
}

// https://reference.opcfoundation.org/Core/Part4/v105/docs/5.13.8
func (s *SubscriptionService) DeleteSubscriptions(sc *uasc.SecureChannel, r ua.Request, reqID uint32) (ua.Response, error) {
	if s.srv.cfg.logger != nil {
		s.srv.cfg.logger.Debug("Handling %T", r)
	}

	req, err := safeReq[*ua.DeleteSubscriptionsRequest](r)
	if err != nil {
		return nil, err
	}
	session := s.srv.Session(req.Header())

	s.Mu.Lock()
	defer s.Mu.Unlock()

	results := make([]ua.StatusCode, len(req.SubscriptionIDs))
	for i := range req.SubscriptionIDs {

		subid := req.SubscriptionIDs[i]
		if s.srv.cfg.logger != nil {
			s.srv.cfg.logger.Info("Subscription %d deleted by client", subid)
		}
		sub, ok := s.Subs[subid]
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
// MonitoredItems queue their sampled values in the subscription. Each MonitoredItem has its own
// queue; the background task sends the queued values with the next publish responses.
type Subscription struct {
	srv                       *SubscriptionService
	Session                   *session
	ID                        uint32
	RevisedPublishingInterval float64
	RevisedLifetimeCount      uint32
	RevisedMaxKeepAliveCount  uint32
	// MaxNotificationsPerPublish is the largest number of notifications
	// the client accepts in one publish response. Zero means no limit.
	MaxNotificationsPerPublish uint32
	Channel                    *uasc.SecureChannel

	// send sends a publish response. It is nil outside of tests, where the
	// response goes to Channel.
	send       func(reqID uint32, resp ua.Response) error
	SequenceID uint32
	//SeqNums                   map[uint32]struct{}
	T *time.Ticker

	ModifyChannel chan *ua.ModifySubscriptionRequest

	// queues holds the notification queue of each MonitoredItem of the
	// subscription by MonitoredItem id. queueSeq numbers the queued values
	// in sample order across all queues.
	queueMu  sync.Mutex
	queues   map[uint32]*notificationQueue
	queueSeq uint64

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
		ModifyChannel: make(chan *ua.ModifySubscriptionRequest, 2),
		queues:        map[uint32]*notificationQueue{},
		shutdown:      make(chan struct{}),
	}
}

func (s *Subscription) Update(req *ua.ModifySubscriptionRequest) {
	s.RevisedPublishingInterval = req.RequestedPublishingInterval
	s.RevisedLifetimeCount = req.RequestedLifetimeCount
	s.RevisedMaxKeepAliveCount = req.RequestedMaxKeepAliveCount
	s.MaxNotificationsPerPublish = req.MaxNotificationsPerPublish
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
	return s.sendResponse(pubreq.ID, response)
}

// sendResponse sends a publish response for the request with the given id.
func (s *Subscription) sendResponse(reqID uint32, resp ua.Response) error {
	if s.send != nil {
		return s.send(reqID, resp)
	}
	return s.Channel.SendResponseWithContext(context.Background(), reqID, resp)
}

// notificationLimit returns the largest number of notifications for the next
// publish response: the smaller of the client's maxNotificationsPerPublish
// (Part 4 §5.14.2.2 Table 82, §5.14.3.2 Table 84) and the server's limit,
// where zero means no limit.
func (s *Subscription) notificationLimit() uint32 {
	limit := s.MaxNotificationsPerPublish
	if srvLimit := s.srv.srv.cfg.cap.MaxNotificationsPerPublish; srvLimit > 0 && (limit == 0 || srvLimit < limit) {
		limit = srvLimit
	}
	return limit
}

// this function should be run as a go-routine and will handle sending data out
// to the client at the correct rate assuming there are publish requests queued up.
// if the function returns it deletes the subscription
func (s *Subscription) run() {
	// if this go routine dies, we need to delete ourselves.
	defer func() {
		if s.srv.srv.cfg.logger != nil {
			s.srv.srv.cfg.logger.Info("Subscription %d shutting down.", s.ID)
		}
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
	// while waiting are held in the queues of their MonitoredItems.  Once the interval expires, we'll move on to L2 if we've got notifications.
	// In L2 we wait for a publish request.  If we get one, we'll publish the queued notifications.  If we don't
	// get a publish request, we'll continue to count intervals without a publish request.
	//
	// In L0 and L2, If we get to the lifetime count without a publish request, we'll kill the subscription.
	//
	// When a publish response could not carry every queued notification (more is set), the rest is sent
	// with the next publish request without waiting for the interval, so we skip L0 (Part 4 §5.14.1.2,
	// Table 79 rows 5 and 10: a Publish request received while MoreNotifications is TRUE returns notifications;
	// §5.14.1.4 Table 81, ReturnNotifications).
	more := false
	for {
		// Collect notifications until our publication interval is ready
	L0:
		for !more {
			select {
			case <-s.shutdown:
				return
			case <-s.T.C:
				if !s.hasQueued() {
					// nothing to publish, increment the keepalive counter and send a keepalive if it
					// has been enough intervals.
					keepalive_counter++
					if keepalive_counter > int(s.RevisedMaxKeepAliveCount) {
						keepalive_counter = 0
						select {
						case pubreq := <-s.Session.PublishRequests:
							err := s.keepalive(pubreq)
							if err != nil {
								if s.srv.srv.cfg.logger != nil {
									s.srv.srv.cfg.logger.Warn("problem sending keepalive to subscription #%d: %v", s.ID, err)
								}
								return
							}
						default:
							lifetime_counter++
							if lifetime_counter > int(s.RevisedLifetimeCount) {
								if s.srv.srv.cfg.logger != nil {
									s.srv.srv.cfg.logger.Warn("Subscription #%d timed out.", s.ID)
								}
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
			case <-s.T.C:
				// we had another tick without a publish request.
				lifetime_counter++
				if lifetime_counter > int(s.RevisedLifetimeCount) {
					if s.srv.srv.cfg.logger != nil {
						s.srv.srv.cfg.logger.Warn("Subscription %d timed out.", s.ID)
					}
					return
				}
			}
		}
		lifetime_counter = 0
		keepalive_counter = 0

		// Take the values queued since the last publish response, in sample order, as many as
		// the client and the server allow in one response.
		var final_items []*ua.MonitoredItemNotification
		final_items, more = s.drainQueues(s.notificationLimit())
		if len(final_items) == 0 {
			// The items that held the queued values were deleted in the meantime: answer with a
			// keep-alive (Part 4 §5.14.1.2, Table 79 row 11: no notifications available). Table 79
			// has no row for this while MoreNotifications is TRUE; a keep-alive is sent then too.
			if err := s.keepalive(pubreq); err != nil {
				if s.srv.srv.cfg.logger != nil {
					s.srv.srv.cfg.logger.Warn("problem sending keepalive to subscription #%d: %v", s.ID, err)
				}
				return
			}
			continue
		}

		s.SequenceID++
		if s.SequenceID == 0 {
			// per the spec, the sequence ID cannot be 0
			s.SequenceID = 1
		}
		if s.srv.srv.cfg.logger != nil {
			s.srv.srv.cfg.logger.Debug("Got publish req on sub #%d.  Sequence %d", s.ID, s.SequenceID)
		}
		// then get all the tags and send them back to the client

		//for x := range pubreq.Req.SubscriptionAcknowledgements {
		//a := pubreq.Req.SubscriptionAcknowledgements[x]
		//delete(s.SeqNums, a.SequenceNumber)
		//}

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
			MoreNotifications:        more,
			NotificationMessage:      &msg,
			AvailableSequenceNumbers: []uint32{}, // an empty array indicates taht we don't support retransmission of messages
			Results:                  []ua.StatusCode{},
			DiagnosticInfos:          []*ua.DiagnosticInfo{},
		}
		err := s.sendResponse(pubreq.ID, response)
		if err != nil {
			if s.srv.srv.cfg.logger != nil {
				s.srv.srv.cfg.logger.Error("problem sending channel response: %v", err)
				s.srv.srv.cfg.logger.Error("Killing subscription %d", s.ID)
			}
			return
		}
		if s.srv.srv.cfg.logger != nil {
			s.srv.srv.cfg.logger.Debug("Published %d items OK for %d", len(final_items), s.ID)
		}
		// wait till we've got a publish request.
	}
}

//PublishRequest_Encoding_DefaultBinary
