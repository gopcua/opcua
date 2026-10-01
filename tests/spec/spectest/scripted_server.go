package spectest

import (
	"context"
	"fmt"
	"net"
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
	id   uint32
	next uint32
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
		sub := &harnessSub{id: create.SubscriptionID, next: 1}
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

// Subscription identifies one subscription the client created on a
// scripted server.
type Subscription struct {
	sub *harnessSub
}

// ID returns the subscription id the server handed the client.
func (s Subscription) ID() uint32 {
	return s.sub.id
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
// notification carrying v for sub, at sub's next sequence number.
func (h HeldPublish) Answer(sub Subscription, v int32) {
	s := h.server
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
	handle, recorded := s.clientHandles[sub.sub.id]
	if !recorded {
		s.mu.Unlock()
		s.t.Fatalf("the client created no monitored item for subscription %d", sub.sub.id)
		return
	}
	sequence := sub.sub.next
	sub.sub.next++
	h.entry.answered = true
	change := &ua.DataChangeNotification{
		MonitoredItems: []*ua.MonitoredItemNotification{{
			ClientHandle: handle,
			Value:        server.DataValueFromValue(v),
		}},
		DiagnosticInfos: []*ua.DiagnosticInfo{},
	}
	results := make([]ua.StatusCode, len(h.entry.request.SubscriptionAcknowledgements))
	for i := range results {
		results[i] = ua.StatusOK
	}
	response := &ua.PublishResponse{
		ResponseHeader: &ua.ResponseHeader{
			Timestamp:          time.Now(),
			RequestHandle:      h.entry.request.RequestHeader.RequestHandle,
			ServiceResult:      ua.StatusOK,
			ServiceDiagnostics: &ua.DiagnosticInfo{},
			StringTable:        []string{},
			AdditionalHeader:   ua.NewExtensionObject(nil),
		},
		SubscriptionID:    sub.sub.id,
		MoreNotifications: false,
		NotificationMessage: &ua.NotificationMessage{
			SequenceNumber:   sequence,
			PublishTime:      time.Now(),
			NotificationData: []*ua.ExtensionObject{ua.NewExtensionObject(change)},
		},
		AvailableSequenceNumbers: []uint32{},
		Results:                  results,
		DiagnosticInfos:          []*ua.DiagnosticInfo{},
	}
	s.mu.Unlock()

	sendErr := h.entry.channel.SendResponseWithContext(context.Background(), h.entry.requestID, response)
	if sendErr != nil {
		s.t.Fatalf("spectest: the connection of the held Publish request closed while answering: %v", sendErr)
	}
}
