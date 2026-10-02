package spectest

import (
	"context"
	"errors"
	"fmt"
	"net"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/server"
	"github.com/gopcua/opcua/ua"
	"github.com/gopcua/opcua/uacp"
)

func TestResolveIntervalsPrecedence(t *testing.T) {
	cases := []struct {
		name          string
		option        time.Duration
		publishingEnv string
		reconnectEnv  string
		wantPublish   time.Duration
		wantReconnect time.Duration
		wantErr       bool
	}{
		{
			name:          "defaults when the option and both env vars are absent",
			wantPublish:   100 * time.Millisecond,
			wantReconnect: 50 * time.Millisecond,
		},
		{
			name:          "the option beats the publishing env var",
			option:        250 * time.Millisecond,
			publishingEnv: "300ms",
			wantPublish:   250 * time.Millisecond,
			wantReconnect: 50 * time.Millisecond,
		},
		{
			name:          "the publishing env var beats the default",
			publishingEnv: "300ms",
			wantPublish:   300 * time.Millisecond,
			wantReconnect: 50 * time.Millisecond,
		},
		{
			name:          "the reconnect env var beats the default",
			reconnectEnv:  "400ms",
			wantPublish:   100 * time.Millisecond,
			wantReconnect: 400 * time.Millisecond,
		},
		{
			name:          "an unparsable publishing env var fails",
			publishingEnv: "not-a-duration",
			wantErr:       true,
		},
		{
			name:         "an unparsable reconnect env var fails",
			reconnectEnv: "not-a-duration",
			wantErr:      true,
		},
		{
			name:         "a zero reconnect env var fails",
			reconnectEnv: "0s",
			wantErr:      true,
		},
		{
			name:         "a negative reconnect env var fails",
			reconnectEnv: "-1s",
			wantErr:      true,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			values := map[string]string{
				publishingIntervalEnv: c.publishingEnv,
				reconnectIntervalEnv:  c.reconnectEnv,
			}
			getenv := func(name string) string { return values[name] }
			publishing, reconnect, err := resolveIntervals(options{publishingInterval: c.option, publishingIntervalSet: c.option != 0}, getenv)
			if c.wantErr {
				if err == nil {
					t.Fatalf("resolveIntervals(%q, %q) did not fail", c.publishingEnv, c.reconnectEnv)
				}
				return
			}
			if err != nil {
				t.Fatalf("resolveIntervals(%q, %q) failed: %v", c.publishingEnv, c.reconnectEnv, err)
			}
			if publishing != c.wantPublish {
				t.Fatalf("resolveIntervals publishing = %s, want %s", publishing, c.wantPublish)
			}
			if reconnect != c.wantReconnect {
				t.Fatalf("resolveIntervals reconnect = %s, want %s", reconnect, c.wantReconnect)
			}
		})
	}
}

func TestStartRejectsDegeneratePublishingInterval(t *testing.T) {
	fake := &fakeT{}
	if !fatalPanics(func() { Start(fake, WithPublishingInterval(time.Millisecond)) }) {
		t.Fatalf("Start did not fail for a degenerate publishing interval")
	}
	if len(fake.fatals) != 1 {
		t.Fatalf("Start failed %d times for a degenerate publishing interval, want exactly the lifetime guard", len(fake.fatals))
	}
	if !strings.HasPrefix(fake.fatals[0], "spectest:") {
		t.Fatalf("Start's failure for a degenerate publishing interval is not a harness fault: %q", fake.fatals[0])
	}
	if !strings.Contains(fake.fatals[0], "stays below one hour") {
		t.Fatalf("Start's failure for a degenerate publishing interval is not the lifetime guard: %q", fake.fatals[0])
	}
}

func TestStartRejectsZeroPublishingIntervalOption(t *testing.T) {
	fake := &fakeT{}
	if !fatalPanics(func() { Start(fake, WithPublishingInterval(0)) }) {
		t.Fatalf("Start did not fail for a zero publishing interval")
	}
	if len(fake.fatals) != 1 {
		t.Fatalf("Start failed %d times for a zero publishing interval, want exactly the lifetime guard", len(fake.fatals))
	}
	if !strings.HasPrefix(fake.fatals[0], "spectest:") {
		t.Fatalf("Start's failure for a zero publishing interval is not a harness fault: %q", fake.fatals[0])
	}
	if !strings.Contains(fake.fatals[0], "stays below one hour") {
		t.Fatalf("Start resolved a zero publishing interval instead of failing the lifetime guard: %q", fake.fatals[0])
	}
}

func TestStartAppliesThePublishingIntervalEnvVar(t *testing.T) {
	t.Setenv(publishingIntervalEnv, "250ms")
	env := Start(t)
	for _, r := range env.Recorder.Requests() {
		m, ok := r.Message()
		if !ok {
			continue
		}
		req, isCreate := m.(*ua.CreateSubscriptionRequest)
		if !isCreate {
			continue
		}
		if req.RequestedPublishingInterval != 250 {
			t.Fatalf("the CreateSubscription request carries publishing interval %v, want 250ms from %s", req.RequestedPublishingInterval, publishingIntervalEnv)
		}
		return
	}
	t.Fatalf("the recorder saw no CreateSubscription request")
}

func TestAnswerIncrementsTheSequenceCounter(t *testing.T) {
	env := Start(t)
	if env.LastSequenceNumber() != 1 {
		t.Fatalf("the sequence counter does not start at 1: %d", env.LastSequenceNumber())
	}
	sub := env.Subscription()

	for _, v := range []int32{7001, 7002} {
		env.Server.WaitHeldPublish().Answer(sub, v)
	}

	want := []answeredPublish{
		{sequenceNumber: 1, value: valueBeforeCut},
		{sequenceNumber: 2, value: 7001},
		{sequenceNumber: 3, value: 7002},
	}
	answered := waitAnsweredPublishes(env, want)
	got, wantSequences := answeredSequences(answered), answeredSequences(want)
	if !slices.Equal(got, wantSequences) {
		t.Fatalf("the sequence numbers of the answered Publish responses are %v, want %v", got, wantSequences)
	}
	for i := range answered {
		if answered[i].value != want[i].value {
			t.Fatalf("the Publish response with sequence number %d carries %d, want %d", answered[i].sequenceNumber, answered[i].value, want[i].value)
		}
	}
	for _, response := range answered {
		if response.sequenceNumber != 2 && response.sequenceNumber != 3 {
			continue
		}
		if len(response.results) != 1 {
			t.Fatalf("the Publish response with sequence %d carries %d results, want one Good per acknowledgement", response.sequenceNumber, len(response.results))
		}
		if response.results[0] != ua.StatusOK {
			t.Fatalf("the Publish response with sequence %d carries results %v, want [Good]", response.sequenceNumber, response.results)
		}
	}
}

func TestWithClientOptionsApplyAfterStartOptions(t *testing.T) {
	env := Start(t, WithClientOptions(opcua.RequestTimeout(777*time.Millisecond)))

	for _, r := range env.Recorder.Requests() {
		m, ok := r.Message()
		if !ok {
			continue
		}
		if req, isCreate := m.(*ua.CreateSessionRequest); isCreate {
			if hint := req.RequestHeader.TimeoutHint; hint != 777 {
				t.Fatalf("the CreateSession request carries timeout hint %d, want 777 from WithClientOptions, not Start's own timeout", hint)
			}
			return
		}
	}
	t.Fatalf("the recorder saw no CreateSession request")
}

func TestAcceptRoutesErrorNotifications(t *testing.T) {
	env := &Environment{receivedSignal: make(chan struct{}, 1)}

	env.accept(&opcua.PublishNotificationData{Error: errors.New("connection cut")})
	if len(env.ReceivedErrors()) != 1 {
		t.Fatalf("accept put %d errors in receivedErrors, want 1", len(env.ReceivedErrors()))
	}
	if len(env.Received()) != 0 {
		t.Fatalf("an error notification also reached received: %v", env.Received())
	}

	item := &ua.MonitoredItemNotification{
		ClientHandle: monitorClientHandle,
		Value:        server.DataValueFromValue(int32(4242)),
	}
	env.accept(&opcua.PublishNotificationData{
		Value: &ua.DataChangeNotification{MonitoredItems: []*ua.MonitoredItemNotification{item}},
	})
	if len(env.Received()) != 1 || env.Received()[0] != 4242 {
		t.Fatalf("accept put %v in received, want [4242]", env.Received())
	}
}

func TestAcceptRecordsUndeliverableNotifications(t *testing.T) {
	env := &Environment{receivedSignal: make(chan struct{}, 1)}

	uint32Item := &ua.MonitoredItemNotification{
		ClientHandle: monitorClientHandle,
		Value:        server.DataValueFromValue(uint32(1)),
	}
	env.accept(&opcua.PublishNotificationData{
		Value: &ua.DataChangeNotification{MonitoredItems: []*ua.MonitoredItemNotification{uint32Item}},
	})
	if len(env.Received()) != 0 {
		t.Fatalf("accept put a non-int32 value in received: %v", env.Received())
	}
	if len(env.ReceivedErrors()) != 1 {
		t.Fatalf("accept put %d errors in receivedErrors for a mis-decoded value, want 1", len(env.ReceivedErrors()))
	}

	env.accept(&opcua.PublishNotificationData{
		Value: &ua.StatusChangeNotification{},
	})
	if len(env.Received()) != 0 {
		t.Fatalf("accept put a status change in received: %v", env.Received())
	}
	if len(env.ReceivedErrors()) != 2 {
		t.Fatalf("accept put %d errors in receivedErrors after a status change, want 2", len(env.ReceivedErrors()))
	}
	if err := env.ReceivedErrors()[1]; err == nil || !strings.Contains(err.Error(), "data change") {
		t.Fatalf("accept recorded %v for a status change, want an error naming the wanted data change", err)
	}

	env.accept(&opcua.PublishNotificationData{
		Value: &ua.DataChangeNotification{MonitoredItems: []*ua.MonitoredItemNotification{{
			ClientHandle: monitorClientHandle,
			Value:        &ua.DataValue{},
		}}},
	})
	if len(env.Received()) != 0 {
		t.Fatalf("accept put a value-less data change in received: %v", env.Received())
	}
	if len(env.ReceivedErrors()) != 3 {
		t.Fatalf("accept put %d errors in receivedErrors after a value-less data change, want 3", len(env.ReceivedErrors()))
	}
	if err := env.ReceivedErrors()[2]; err == nil || !strings.Contains(err.Error(), "no value") {
		t.Fatalf("accept recorded %v for a value-less data change, want an error naming the missing value", err)
	}
}

func TestRecorderSinceReturnsOnlyRecordsAfterTheMark(t *testing.T) {
	ft := &fakeT{}
	recorder := newRecorder(ft)
	for _, id := range []uint32{1, 2} {
		requestWire, err := readRequestWire(id)
		if err != nil {
			t.Fatalf("encoding a Read request failed: %v", err)
		}
		recorder.observe(0, clientToServer, requestWire)
		responseWire, err := readResponseWire(id)
		if err != nil {
			t.Fatalf("encoding a Read response failed: %v", err)
		}
		recorder.observe(0, serverToClient, responseWire)
	}
	mark := Mark{order: recorder.position()}
	requestWire, err := readRequestWire(3)
	if err != nil {
		t.Fatalf("encoding a Read request failed: %v", err)
	}
	recorder.observe(0, clientToServer, requestWire)
	responseWire, err := readResponseWire(3)
	if err != nil {
		t.Fatalf("encoding a Read response failed: %v", err)
	}
	recorder.observe(0, serverToClient, responseWire)

	sinceRequests := recorder.RequestsSince(mark)
	if len(sinceRequests) != 1 || sinceRequests[0].RequestID != 3 {
		t.Fatalf("RequestsSince returned %v, want only the request with id 3", sinceRequests)
	}
	if sinceRequests[0].Order <= mark.order {
		t.Fatalf("RequestsSince returned order %d, want one above the mark's order %d", sinceRequests[0].Order, mark.order)
	}
	sinceResponses := recorder.ResponsesSince(mark)
	if len(sinceResponses) != 1 || sinceResponses[0].RequestID != 3 {
		t.Fatalf("ResponsesSince returned %v, want only the response with id 3", sinceResponses)
	}
	if requests := recorder.Requests(); len(requests) != 3 {
		t.Fatalf("Requests returned %d records, want all 3 including the pre-mark ones", len(requests))
	}
}

func TestEnvironmentSinceScope(t *testing.T) {
	ft := &fakeT{}
	env := &Environment{
		receivedSignal: make(chan struct{}, 1),
		Recorder:       newRecorder(ft),
		Relay:          &Relay{},
	}
	env.accept(&opcua.PublishNotificationData{
		Value: &ua.DataChangeNotification{MonitoredItems: []*ua.MonitoredItemNotification{{
			ClientHandle: monitorClientHandle,
			Value:        server.DataValueFromValue(int32(100)),
		}}},
	})
	env.accept(&opcua.PublishNotificationData{Error: errors.New("before the mark")})
	before := env.Mark()
	env.accept(&opcua.PublishNotificationData{
		Value: &ua.DataChangeNotification{MonitoredItems: []*ua.MonitoredItemNotification{{
			ClientHandle: monitorClientHandle,
			Value:        server.DataValueFromValue(int32(200)),
		}}},
	})
	env.accept(&opcua.PublishNotificationData{Error: errors.New("after the mark")})

	if got := env.ReceivedSince(before); !slices.Equal(got, []int32{200}) {
		t.Fatalf("ReceivedSince returned %v, want [200]", got)
	}
	if errs := env.ReceivedErrorsSince(before); len(errs) != 1 || errs[0].Error() != "after the mark" {
		t.Fatalf("ReceivedErrorsSince returned %v, want only the post-mark error", errs)
	}
	if got := env.Received(); !slices.Equal(got, []int32{100, 200}) {
		t.Fatalf("Received returned %v, want the full stream [100 200]", got)
	}
	if env.ConnectionsSince(before) != 0 {
		t.Fatalf("ConnectionsSince returned %d, want 0 for a relay that accepted no connection", env.ConnectionsSince(before))
	}

	env.mu.Lock()
	env.states = append(env.states, opcua.Connected)
	env.mu.Unlock()
	statesMark := env.Mark()
	env.mu.Lock()
	env.states = append(env.states, opcua.Reconnecting)
	env.mu.Unlock()
	if got := env.StatesSince(statesMark); !slices.Equal(got, []opcua.ConnState{opcua.Reconnecting}) {
		t.Fatalf("StatesSince returned %v, want [Reconnecting]", got)
	}
	if got := env.States(); !slices.Equal(got, []opcua.ConnState{opcua.Connected, opcua.Reconnecting}) {
		t.Fatalf("States returned %v, want the full stream [Connected Reconnecting]", got)
	}
}

func TestTeardownClosesTheClientBeforeTheServers(t *testing.T) {
	fake := &fakeT{}
	env := Start(fake)
	second := env.StartServer()
	serversOpenWhenClientClosed := false
	env.onClientClosed = func() {
		serversOpenWhenClientClosed = dialSucceeds(env.Server.Address()) && dialSucceeds(second.Address())
	}

	for i := len(fake.cleanups) - 1; i >= 0; i-- {
		fake.cleanups[i]()
	}

	if state := env.Client.State(); state != opcua.Closed {
		t.Fatalf("teardown left the client in state %v, want Closed", state)
	}
	if !serversOpenWhenClientClosed {
		t.Fatalf("teardown closed a scripted server before the client")
	}
	if dialSucceeds(env.Server.Address()) || dialSucceeds(second.Address()) {
		t.Fatalf("teardown left a scripted server open")
	}
}

func TestTeardownSkipsTheConnectedWaitWhenTheClientIsAlreadyClosed(t *testing.T) {
	fake := &fakeT{}
	env := Start(fake)
	ctx, cancel := context.WithTimeout(context.Background(), specWait)
	defer cancel()
	if err := env.Client.Close(ctx); err != nil {
		t.Fatalf("closing the client failed: %v", err)
	}

	started := time.Now()
	fatalPanics(func() {
		for i := len(fake.cleanups) - 1; i >= 0; i-- {
			fake.cleanups[i]()
		}
	})
	if elapsed := time.Since(started); elapsed >= time.Second {
		t.Fatalf("teardown took %s although the client was already closed, want under one second", elapsed)
	}
}

func TestTeardownRaisesAStoredServerFault(t *testing.T) {
	fake := &fakeT{}
	env := Start(fake)
	env.Server.mu.Lock()
	env.Server.faultErr = fmt.Errorf("spectest: stored server fault")
	env.Server.mu.Unlock()

	raised := fatalPanics(func() {
		for i := len(fake.cleanups) - 1; i >= 0; i-- {
			fake.cleanups[i]()
		}
	})
	if !raised {
		t.Fatalf("teardown did not raise the stored server fault")
	}
	if len(fake.fatals) == 0 || !strings.HasPrefix(fake.fatals[0], "spectest: ") {
		t.Fatalf("teardown raised %q, want the stored server fault with the spectest: prefix", fake.fatals)
	}
}

func TestTeardownRaisesEveryStoredFaultUnderOnePrefix(t *testing.T) {
	fake := &fakeT{}
	env := Start(fake)
	second := env.StartServer()
	env.Server.mu.Lock()
	env.Server.faultErr = fmt.Errorf("spectest: stored fault one")
	env.Server.mu.Unlock()
	second.mu.Lock()
	second.faultErr = fmt.Errorf("spectest: stored fault two")
	second.mu.Unlock()

	raised := fatalPanics(func() {
		for i := len(fake.cleanups) - 1; i >= 0; i-- {
			fake.cleanups[i]()
		}
	})
	if !raised {
		t.Fatalf("teardown did not raise the stored server faults")
	}
	if len(fake.fatals) != 1 {
		t.Fatalf("teardown failed %d times for two stored faults, want one joined message", len(fake.fatals))
	}
	if !strings.HasPrefix(fake.fatals[0], "spectest: ") {
		t.Fatalf("the joined teardown failure %q does not start with the spectest: prefix the known-defect gate reads", fake.fatals[0])
	}
	if !strings.Contains(fake.fatals[0], "stored fault one") || !strings.Contains(fake.fatals[0], "stored fault two") {
		t.Fatalf("the joined teardown failure %q does not name both stored faults", fake.fatals[0])
	}
}

func dialSucceeds(address string) bool {
	conn, err := net.Dial("tcp", strings.TrimPrefix(address, "opc.tcp://"))
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

func TestWaitHeldPublishPicksTheOldestOnTheNewestOfTwoOpenConnections(t *testing.T) {
	fake := &fakeT{}
	recorder := newRecorder(fake)
	firstAddress := addr("127.0.0.1:4200")
	secondAddress := addr("127.0.0.1:4201")
	recorder.observeConnection(0, firstAddress)
	recorder.observeConnection(1, secondAddress)
	srv := &ScriptedServer{
		t:          fake,
		heldWait:   time.Second,
		heldSignal: make(chan struct{}, 1),
		recorder:   recorder,
	}
	olderOnFirst := &heldEntry{request: &ua.PublishRequest{}, connection: 0, remoteAddr: firstAddress}
	olderOnSecond := &heldEntry{request: &ua.PublishRequest{}, connection: 1, remoteAddr: secondAddress}
	newestOnSecond := &heldEntry{request: &ua.PublishRequest{}, connection: 1, remoteAddr: secondAddress}
	srv.held = []*heldEntry{olderOnFirst, olderOnSecond, newestOnSecond}

	held := srv.WaitHeldPublish()

	if held.entry != olderOnSecond {
		t.Fatalf("WaitHeldPublish picked the held request on connection %d, want the older of the held requests on connection 1, the newest open connection", held.Connection())
	}
	if held.Connection() != 1 {
		t.Fatalf("WaitHeldPublish returned a held request on connection %d, want connection 1", held.Connection())
	}
	for _, cleanup := range fake.cleanups {
		cleanup()
	}
}

func TestWaitHeldPublishPicksTheOldestOnTheNewestOpenConnection(t *testing.T) {
	fake := &fakeT{}
	recorder := newRecorder(fake)
	closedAddress := addr("127.0.0.1:4000")
	openAddress := addr("127.0.0.1:4001")
	recorder.observeConnection(0, closedAddress)
	recorder.observeConnectionClosed(0)
	recorder.observeConnection(1, openAddress)
	srv := &ScriptedServer{
		t:          fake,
		heldWait:   time.Second,
		heldSignal: make(chan struct{}, 1),
		recorder:   recorder,
	}
	closedEntry := &heldEntry{request: &ua.PublishRequest{}, connection: 0, remoteAddr: closedAddress}
	oldestOpen := &heldEntry{request: &ua.PublishRequest{}, connection: 1, remoteAddr: openAddress}
	newestOpen := &heldEntry{request: &ua.PublishRequest{}, connection: 1, remoteAddr: openAddress}
	srv.held = []*heldEntry{closedEntry, oldestOpen, newestOpen}

	held := srv.WaitHeldPublish()

	if held.entry != oldestOpen {
		t.Fatalf("WaitHeldPublish picked the held request on connection %d, want the oldest held request on the newest open connection", held.Connection())
	}
	if held.Connection() != 1 {
		t.Fatalf("WaitHeldPublish returned a held request on connection %d, want connection 1", held.Connection())
	}

	srv.heldWait = 20 * time.Millisecond
	srv.held = []*heldEntry{closedEntry}
	if !fatalPanics(func() { srv.WaitHeldPublish() }) {
		t.Fatalf("WaitHeldPublish did not fail when every held request sits on a closed connection")
	}
	if len(fake.fatals) != 1 || fake.fatals[0] != "client sent no Publish request on an open connection; 1 held on closed connections were discarded and 0 arrived on already closed connections" {
		t.Fatalf("WaitHeldPublish failed with %q, want the message naming the discarded closed-connection request", fake.fatals)
	}
	for _, cleanup := range fake.cleanups {
		cleanup()
	}
}

func TestAnswerFailsWithoutAMonitoredItem(t *testing.T) {
	fake := &fakeT{}
	recorder := newRecorder(fake)
	openAddress := addr("127.0.0.1:4100")
	recorder.observeConnection(0, openAddress)
	srv := &ScriptedServer{
		t:             fake,
		heldWait:      time.Second,
		heldSignal:    make(chan struct{}, 1),
		subscriptions: make(map[uint32]*harnessSub),
		clientHandles: make(map[uint32]uint32),
		recorder:      recorder,
	}
	sub := &harnessSub{id: 7, next: 1}
	srv.subscriptions[7] = sub
	entry := &heldEntry{request: &ua.PublishRequest{RequestHeader: &ua.RequestHeader{}}, connection: 0, remoteAddr: openAddress}
	held := HeldPublish{server: srv, entry: entry}
	if !fatalPanics(func() { held.Answer(Subscription{sub: sub, server: srv}, 5) }) {
		t.Fatalf("Answer did not fail for a subscription with no monitored item")
	}
	if len(fake.fatals) != 1 || !strings.Contains(fake.fatals[0], "no monitored item for subscription 7") {
		t.Fatalf("Answer failed with %q, want the no-monitored-item message", fake.fatals)
	}
	for _, cleanup := range fake.cleanups {
		cleanup()
	}
}

func TestCheckLastSequenceNumberFailsWithoutAnAnsweredPublishResponse(t *testing.T) {
	fake := &fakeT{}
	env := &Environment{t: fake, Recorder: newRecorder(fake)}
	if !fatalPanics(func() { env.checkLastSequenceNumber() }) {
		t.Fatalf("checkLastSequenceNumber did not fail without an answered Publish response")
	}
	if !strings.HasPrefix(fake.fatals[0], "spectest:") || !strings.Contains(fake.fatals[0], "no answered Publish response") {
		t.Fatalf("checkLastSequenceNumber failed with %q, want the harness fault naming the missing Publish response", fake.fatals)
	}
	for _, cleanup := range fake.cleanups {
		cleanup()
	}
}

func publishResponseWire(requestID uint32, sequence uint32, value int32) ([]byte, error) {
	typeID := ua.ServiceTypeID(&ua.PublishResponse{})
	if typeID == 0 {
		return nil, fmt.Errorf("ua.ServiceTypeID returned 0 for *ua.PublishResponse, want its registered type id")
	}
	body := ua.NewBuffer(nil)
	body.WriteStruct(ua.NewFourByteExpandedNodeID(0, typeID))
	body.WriteStruct(&ua.PublishResponse{
		ResponseHeader: &ua.ResponseHeader{
			ServiceDiagnostics: &ua.DiagnosticInfo{},
			StringTable:        []string{},
			AdditionalHeader:   ua.NewExtensionObject(nil),
		},
		SubscriptionID: 5,
		NotificationMessage: &ua.NotificationMessage{
			SequenceNumber: sequence,
			NotificationData: []*ua.ExtensionObject{ua.NewExtensionObject(&ua.DataChangeNotification{
				MonitoredItems: []*ua.MonitoredItemNotification{{
					ClientHandle: monitorClientHandle,
					Value:        server.DataValueFromValue(value),
				}},
				DiagnosticInfos: []*ua.DiagnosticInfo{},
			})},
		},
		AvailableSequenceNumbers: []uint32{},
		Results:                  []ua.StatusCode{},
		DiagnosticInfos:          []*ua.DiagnosticInfo{},
	})
	if body.Error() != nil {
		return nil, body.Error()
	}
	return messageChunk(uacp.ChunkTypeFinal, requestID, body.Bytes())
}

func TestCheckLastSequenceNumberFailsOnAValueMismatch(t *testing.T) {
	fake := &fakeT{}
	recorder := newRecorder(fake)
	wire, err := publishResponseWire(1, 1, 111)
	if err != nil {
		t.Fatalf("encoding the Publish response failed: %v", err)
	}
	recorder.observe(0, serverToClient, wire)
	env := &Environment{t: fake, Recorder: recorder}
	env.mu.Lock()
	env.received = append(env.received, 222)
	env.mu.Unlock()

	if !fatalPanics(func() { env.checkLastSequenceNumber() }) {
		t.Fatalf("checkLastSequenceNumber did not fail on a value mismatch")
	}
	if !strings.HasPrefix(fake.fatals[0], "spectest:") || !strings.Contains(fake.fatals[0], "carries") {
		t.Fatalf("checkLastSequenceNumber failed with %q, want the harness fault naming the carried value", fake.fatals)
	}
	for _, cleanup := range fake.cleanups {
		cleanup()
	}
}

func TestWaitUntilReconnectedTimesOutWhenTheClientNeverReconnects(t *testing.T) {
	fake := &fakeT{}
	e := &Environment{t: fake, waitTimeout: time.Millisecond, receivedSignal: make(chan struct{}, 1)}

	if !fatalPanics(func() { e.WaitUntilReconnected() }) {
		t.Fatalf("WaitUntilReconnected returned without failing, want the timeout Fatalf")
	}
	if len(fake.fatals) != 1 || !strings.Contains(fake.fatals[0], "client never entered Reconnecting within") {
		t.Errorf("WaitUntilReconnected failed with %v, want the never-entered-Reconnecting message", fake.fatals)
	}
}

func TestWaitUntilReconnectedTimesOutWhenTheClientNeverReachesConnected(t *testing.T) {
	fake := &fakeT{}
	e := &Environment{t: fake, waitTimeout: time.Millisecond, receivedSignal: make(chan struct{}, 1)}
	e.states = []opcua.ConnState{opcua.Reconnecting}

	if !fatalPanics(func() { e.WaitUntilReconnected() }) {
		t.Fatalf("WaitUntilReconnected returned without failing, want the timeout Fatalf")
	}
	if len(fake.fatals) != 1 || !strings.Contains(fake.fatals[0], "client entered Reconnecting but did not reach Connected within") {
		t.Errorf("WaitUntilReconnected failed with %v, want the never-reached-Connected message", fake.fatals)
	}
}
