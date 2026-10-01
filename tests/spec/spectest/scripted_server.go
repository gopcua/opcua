package spectest

import (
	"cmp"
	"context"
	"fmt"
	"maps"
	"math"
	"net"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/gopcua/opcua/id"
	"github.com/gopcua/opcua/server"
	"github.com/gopcua/opcua/ua"
	"github.com/gopcua/opcua/uasc"
)

const (
	serverStartAttempts = 5
	scriptedNamespace   = "spectest.scripted"
	scriptedNode        = "spectest_variable"
	wraparoundFailure   = "spectest: sequence number wraparound is out of scope"
)

// ScriptedServer is an OPC UA server that holds every Publish request
// arriving on an open relay connection for the harness to answer,
// drops the rest, and records the subscriptions and the client handle
// of the last monitored item created per subscription.
type ScriptedServer struct {
	t             T
	srv           *server.Server
	address       string
	node          *ua.NodeID
	recorder      *Recorder
	heldWait      time.Duration
	mu            sync.Mutex
	subscriptions map[uint32]*harnessSub
	first         *harnessSub
	clientHandles map[uint32]uint32
	held          []*heldEntry
	heldSignal    chan struct{}
	faultErr      error
}

type harnessSub struct {
	id                 uint32
	next               uint32
	retained           map[uint32]*ua.NotificationMessage
	failRepublish      map[uint32]ua.StatusCode
	unansweredRetained map[uint32]bool
}

func newHarnessSub(id uint32) *harnessSub {
	return &harnessSub{
		id:                 id,
		next:               1,
		retained:           make(map[uint32]*ua.NotificationMessage),
		failRepublish:      make(map[uint32]ua.StatusCode),
		unansweredRetained: make(map[uint32]bool),
	}
}

type heldEntry struct {
	channel    *uasc.SecureChannel
	requestID  uint32
	request    *ua.PublishRequest
	connection int
	remoteAddr net.Addr
	answered   bool
}

func newScriptedServer(t T) *ScriptedServer {
	s := &ScriptedServer{
		t:             t,
		heldWait:      startTimeout,
		subscriptions: make(map[uint32]*harnessSub),
		clientHandles: make(map[uint32]uint32),
		heldSignal:    make(chan struct{}, 1),
	}
	var lastErr error
	for range serverStartAttempts {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatalf("spectest: reserving a free port failed: %v", err)
		}
		port := listener.Addr().(*net.TCPAddr).Port
		if err := listener.Close(); err != nil {
			t.Fatalf("spectest: releasing the reserved port failed: %v", err)
		}

		srv := server.New(
			server.EndPoint("127.0.0.1", port),
			server.EnableSecurity("None", ua.MessageSecurityModeNone),
			server.EnableAuthMode(ua.UserTokenTypeAnonymous),
		)
		namespace := server.NewNodeNameSpace(srv, scriptedNamespace)
		srv.AddNamespace(namespace)
		node := namespace.AddNewVariableStringNode(scriptedNode, int32(0)).ID()

		srv.RegisterHandler(id.CreateSubscriptionRequest_Encoding_DefaultBinary, s.createSubscription)
		srv.RegisterHandler(id.PublishRequest_Encoding_DefaultBinary, s.holdPublish)
		srv.RegisterHandler(id.CreateMonitoredItemsRequest_Encoding_DefaultBinary, s.createMonitoredItems)
		srv.RegisterHandler(id.RepublishRequest_Encoding_DefaultBinary, s.republish)

		if err := srv.Start(context.Background()); err != nil {
			lastErr = err
			_ = srv.Close()
			continue
		}
		s.srv = srv
		s.address = fmt.Sprintf("opc.tcp://127.0.0.1:%d", port)
		s.node = node
		t.Cleanup(func() { _ = srv.Close() })
		return s
	}
	t.Fatalf("spectest: the scripted server could not start on a free port in %d attempts, last error: %v", serverStartAttempts, lastErr)
	return nil
}

// Address returns the OPC UA endpoint URL the scripted server listens
// on.
func (s *ScriptedServer) Address() string {
	return s.address
}

// WaitHeldPublish returns the oldest Publish request the client sent on
// the newest open relay connection and that the harness has not
// answered yet.
func (s *ScriptedServer) WaitHeldPublish() HeldPublish {
	timer := time.NewTimer(s.heldWait)
	defer timer.Stop()
	discarded := 0
	for {
		s.mu.Lock()
		discarded += s.pruneHeldLocked()
		if s.faultErr != nil {
			err := s.faultErr
			s.mu.Unlock()
			s.t.Fatalf("%v", err)
			return HeldPublish{}
		}
		entry, found := s.oldestOnNewestLocked()
		if found {
			s.removeHeldLocked(entry)
			s.mu.Unlock()
			return HeldPublish{server: s, entry: entry}
		}
		s.mu.Unlock()
		select {
		case <-s.heldSignal:
		case <-timer.C:
			if discarded > 0 {
				s.t.Fatalf("client sent no Publish request on an open connection; %d held on closed connections were discarded", discarded)
			} else {
				s.t.Fatalf("client sent no Publish request")
			}
			return HeldPublish{}
		}
	}
}

func (s *ScriptedServer) fault(format string, args ...any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.faultErr == nil {
		s.faultErr = fmt.Errorf(format, args...)
	}
	select {
	case s.heldSignal <- struct{}{}:
	default:
	}
}

func (s *ScriptedServer) pruneHeldLocked() int {
	var kept []*heldEntry
	discarded := 0
	for _, entry := range s.held {
		switch s.entryStateLocked(entry) {
		case Open:
			kept = append(kept, entry)
		case Closed:
			discarded++
		}
	}
	s.held = kept
	return discarded
}

func (s *ScriptedServer) entryStateLocked(entry *heldEntry) ConnectionState {
	_, state := s.recorder.ConnectionOf(entry.remoteAddr)
	return state
}

func (s *ScriptedServer) oldestOnNewestLocked() (*heldEntry, bool) {
	newest := -1
	for _, entry := range s.held {
		if entry.connection > newest {
			newest = entry.connection
		}
	}
	if newest < 0 {
		return nil, false
	}
	for _, entry := range s.held {
		if entry.connection == newest {
			return entry, true
		}
	}
	return nil, false
}

func (s *ScriptedServer) removeHeldLocked(entry *heldEntry) {
	for i, held := range s.held {
		if held == entry {
			s.held = append(s.held[:i], s.held[i+1:]...)
			return
		}
	}
}

func (s *ScriptedServer) createSubscription(sc *uasc.SecureChannel, r ua.Request, reqID uint32) (ua.Response, error) {
	response, err := s.srv.SubscriptionService.CreateSubscription(sc, r, reqID)
	if err != nil {
		return nil, err
	}
	if create, isCreate := response.(*ua.CreateSubscriptionResponse); isCreate {
		s.mu.Lock()
		sub := newHarnessSub(create.SubscriptionID)
		s.subscriptions[sub.id] = sub
		if s.first == nil {
			s.first = sub
		}
		s.mu.Unlock()
	}
	return response, nil
}

func (s *ScriptedServer) holdPublish(sc *uasc.SecureChannel, r ua.Request, reqID uint32) (ua.Response, error) {
	request, isPublish := r.(*ua.PublishRequest)
	if !isPublish {
		return nil, ua.StatusBadRequestTypeInvalid
	}
	connection, state := s.recorder.ConnectionOf(sc.RemoteAddr())
	switch state {
	case Closed:
		return nil, nil
	case Unknown:
		s.fault("spectest: the scripted server received a Publish request from %s, an address the relay never accepted", sc.RemoteAddr())
		return nil, nil
	}
	s.mu.Lock()
	s.held = append(s.held, &heldEntry{channel: sc, requestID: reqID, request: request, connection: connection, remoteAddr: sc.RemoteAddr()})
	s.mu.Unlock()
	select {
	case s.heldSignal <- struct{}{}:
	default:
	}
	return nil, nil
}

func (s *ScriptedServer) createMonitoredItems(sc *uasc.SecureChannel, r ua.Request, reqID uint32) (ua.Response, error) {
	response, err := s.srv.MonitoredItemService.CreateMonitoredItems(sc, r, reqID)
	if err != nil {
		return nil, err
	}
	if request, isCreate := r.(*ua.CreateMonitoredItemsRequest); isCreate {
		s.mu.Lock()
		for _, item := range request.ItemsToCreate {
			if item != nil && item.RequestedParameters != nil {
				s.clientHandles[request.SubscriptionID] = item.RequestedParameters.ClientHandle
			}
		}
		s.mu.Unlock()
	}
	return response, nil
}

func (s *ScriptedServer) republish(sc *uasc.SecureChannel, r ua.Request, reqID uint32) (ua.Response, error) {
	request, isRepublish := r.(*ua.RepublishRequest)
	if !isRepublish {
		return nil, ua.StatusBadRequestTypeInvalid
	}
	sequenceNumber := request.RetransmitSequenceNumber
	s.mu.Lock()
	sub, live := s.subscriptions[request.SubscriptionID]
	if !live {
		s.mu.Unlock()
		return nil, ua.StatusBadSubscriptionIDInvalid
	}
	if status, scripted := sub.failRepublish[sequenceNumber]; scripted {
		delete(sub.failRepublish, sequenceNumber)
		s.mu.Unlock()
		return nil, status
	}
	message, retained := sub.retained[sequenceNumber]
	if !retained {
		s.mu.Unlock()
		return nil, ua.StatusBadMessageNotAvailable
	}
	delete(sub.unansweredRetained, sequenceNumber)
	s.mu.Unlock()
	return &ua.RepublishResponse{
		ResponseHeader:      responseHeader(request.RequestHeader.RequestHandle),
		NotificationMessage: message,
	}, nil
}

func counterAfterRetain(counter, sequenceNumber uint32) uint32 {
	return max(counter, sequenceNumber+1)
}

// Subscription identifies one subscription the client created on a
// scripted server.
type Subscription struct {
	server *ScriptedServer
	sub    *harnessSub
}

// ID returns the subscription id the server handed the client.
func (s Subscription) ID() uint32 {
	return s.sub.id
}

// Retain stores a NotificationMessage carrying v at seq in the
// subscription's retransmission queue, where it stays until a Publish
// acknowledges seq. It also moves the subscription's sequence counter
// to the larger of the counter and seq+1 (the counterAfterRetain
// rule), so a later Answer reuses no number at or below seq. Retain
// fails when seq is the largest uint32, because the counter cannot
// move past it.
func (s Subscription) Retain(seq uint32, v int32) {
	s.server.mu.Lock()
	if seq == math.MaxUint32 {
		s.server.mu.Unlock()
		s.server.t.Fatalf("%s", wraparoundFailure)
		return
	}
	handle := s.server.clientHandles[s.sub.id]
	s.sub.retained[seq] = dataChangeNotificationMessage(seq, handle, v)
	s.sub.unansweredRetained[seq] = true
	s.sub.next = counterAfterRetain(s.sub.next, seq)
	s.server.mu.Unlock()
}

// FailRepublish scripts a ServiceFault with status for the next
// Republish that names seq, instead of the retransmission queue's
// answer. The script answers one Republish and is then gone.
func (s Subscription) FailRepublish(seq uint32, status ua.StatusCode) {
	s.server.mu.Lock()
	s.sub.failRepublish[seq] = status
	s.server.mu.Unlock()
}

// HeldPublish is one Publish request the client sent that the harness
// has not answered yet.
type HeldPublish struct {
	server *ScriptedServer
	entry  *heldEntry
}

// Connection returns the relay connection index the held Publish
// request arrived on.
func (h HeldPublish) Connection() int {
	return h.entry.connection
}

// Answer answers the held Publish request with one data change
// notification carrying v for sub, at sub's next sequence number. Each
// acknowledgement the request carries gets one Good result, removes the
// acknowledged sequence number from sub's retransmission queue, and the
// response's AvailableSequenceNumbers holds the retained numbers left.
func (h HeldPublish) Answer(sub Subscription, v int32) {
	h.answer(sub, 0, v, true)
}

// AnswerWithSequenceNumber answers the held Publish request with one
// data change notification carrying v for sub, with the sequence number
// seq given explicitly, so a test can send a duplicate sequence number.
// It also moves the subscription's sequence counter to the larger of
// the counter and seq+1 (the counterAfterRetain rule), so a later Answer
// reuses no number at or below seq. AnswerWithSequenceNumber fails when
// seq is the largest uint32, because the counter cannot move past it.
func (h HeldPublish) AnswerWithSequenceNumber(sub Subscription, seq uint32, v int32) {
	h.answer(sub, seq, v, false)
}

func (h HeldPublish) answer(sub Subscription, sequenceNumber uint32, v int32, useCounter bool) {
	s := h.server
	if sub.server != s {
		s.t.Fatalf("spectest: answering a held Publish request of the scripted server at %s with a subscription of the scripted server at %s", s.address, sub.server.address)
		return
	}
	s.mu.Lock()
	if s.faultErr != nil {
		err := s.faultErr
		s.mu.Unlock()
		s.t.Fatalf("%v", err)
		return
	}
	if h.entry.answered {
		s.mu.Unlock()
		s.t.Fatalf("spectest: the held Publish request on connection %d was already answered", h.entry.connection)
		return
	}
	if state := s.entryStateLocked(h.entry); state != Open {
		s.mu.Unlock()
		s.t.Fatalf("spectest: the connection the held Publish request arrived on (index %d) is not open", h.entry.connection)
		return
	}
	if !useCounter && sequenceNumber == math.MaxUint32 {
		s.mu.Unlock()
		s.t.Fatalf("%s", wraparoundFailure)
		return
	}
	handle, recorded := s.clientHandles[sub.sub.id]
	if !recorded {
		s.mu.Unlock()
		s.t.Fatalf("spectest: the client created no monitored item for subscription %d", sub.sub.id)
		return
	}
	if useCounter {
		sequenceNumber = sub.sub.next
	}
	for _, acknowledgement := range h.entry.request.SubscriptionAcknowledgements {
		if acknowledgement != nil && acknowledgement.SubscriptionID == sub.sub.id {
			delete(sub.sub.retained, acknowledgement.SequenceNumber)
		}
	}
	sub.sub.next = counterAfterRetain(sub.sub.next, sequenceNumber)
	h.entry.answered = true
	results := make([]ua.StatusCode, len(h.entry.request.SubscriptionAcknowledgements))
	for i := range results {
		results[i] = ua.StatusOK
	}
	response := &ua.PublishResponse{
		ResponseHeader:           responseHeader(h.entry.request.RequestHeader.RequestHandle),
		SubscriptionID:           sub.sub.id,
		MoreNotifications:        false,
		NotificationMessage:      dataChangeNotificationMessage(sequenceNumber, handle, v),
		AvailableSequenceNumbers: slices.Sorted(maps.Keys(sub.sub.retained)),
		Results:                  results,
		DiagnosticInfos:          []*ua.DiagnosticInfo{},
	}
	s.mu.Unlock()

	sendErr := h.entry.channel.SendResponseWithContext(context.Background(), h.entry.requestID, response)
	if sendErr != nil {
		s.t.Fatalf("spectest: the connection of the held Publish request closed while answering: %v", sendErr)
	}
}

// UnusedScripts returns one readable description per FailRepublish
// script no Republish has answered with and per retained message no
// Republish has answered from. Acknowledging a retained message removes
// it from the retransmission queue but does not mark it used.
func (s *ScriptedServer) UnusedScripts() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	var scripts []string
	for _, sub := range slices.SortedFunc(maps.Values(s.subscriptions), func(a, b *harnessSub) int {
		return cmp.Compare(a.id, b.id)
	}) {
		for _, seq := range slices.Sorted(maps.Keys(sub.failRepublish)) {
			scripts = append(scripts, fmt.Sprintf("FailRepublish(%d, %s) for subscription %d", seq, statusName(sub.failRepublish[seq]), sub.id))
		}
		for _, seq := range slices.Sorted(maps.Keys(sub.unansweredRetained)) {
			scripts = append(scripts, fmt.Sprintf("retained %d for subscription %d", seq, sub.id))
		}
	}
	return scripts
}

func statusName(status ua.StatusCode) string {
	if details, known := ua.StatusCodes[status]; known {
		return strings.TrimPrefix(details.Name, "Status")
	}
	return fmt.Sprintf("0x%X", uint32(status))
}

func responseHeader(requestHandle uint32) *ua.ResponseHeader {
	return &ua.ResponseHeader{
		Timestamp:          time.Now(),
		RequestHandle:      requestHandle,
		ServiceResult:      ua.StatusOK,
		ServiceDiagnostics: &ua.DiagnosticInfo{},
		StringTable:        []string{},
		AdditionalHeader:   ua.NewExtensionObject(nil),
	}
}

func dataChangeNotificationMessage(sequenceNumber, handle uint32, v int32) *ua.NotificationMessage {
	change := &ua.DataChangeNotification{
		MonitoredItems: []*ua.MonitoredItemNotification{{
			ClientHandle: handle,
			Value:        server.DataValueFromValue(v),
		}},
		DiagnosticInfos: []*ua.DiagnosticInfo{},
	}
	return &ua.NotificationMessage{
		SequenceNumber:   sequenceNumber,
		PublishTime:      time.Now(),
		NotificationData: []*ua.ExtensionObject{ua.NewExtensionObject(change)},
	}
}
