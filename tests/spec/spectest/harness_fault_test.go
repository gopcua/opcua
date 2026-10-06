package spectest

import (
	"errors"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/tests/spec/faults"
	"github.com/gopcua/opcua/ua"
	"github.com/gopcua/opcua/uacp"
)

func assertSinglePrefix(t *testing.T, fake *fakeT, path string) {
	t.Helper()
	if len(fake.fatals) != 1 {
		t.Fatalf("%s failed with %d Fatalf calls, want 1: %v", path, len(fake.fatals), fake.fatals)
	}
	if count := strings.Count(fake.fatals[0], "spectest: "); count != 1 {
		t.Errorf("%s failed with %q, want exactly one spectest: prefix, got %d", path, fake.fatals[0], count)
	}
}

func assertPrefixedFault(t *testing.T, fake *fakeT, path string) {
	t.Helper()
	assertSinglePrefix(t, fake, path)
	if !strings.HasPrefix(fake.fatals[0], "spectest: ") {
		t.Errorf("%s failed with %q, want the message to start with the spectest: prefix", path, fake.fatals[0])
	}
}

func TestHarnessFaultPathsCarryExactlyOnePrefix(t *testing.T) {
	if count := strings.Count(harnessFault("the harness fault %v", 1), "spectest: "); count != 1 {
		t.Errorf("harnessFault(\"the harness fault %%v\", 1) = %q, want exactly one spectest: prefix, got %d", harnessFault("the harness fault %v", 1), count)
	}

	fake := &fakeT{}
	relay := &Relay{t: fake, draining: map[int]bool{}}
	if !fatalPanics(func() { relay.CutAt(Moment(9), faults.Read) }) {
		t.Fatalf("CutAt with an unknown Moment returned without failing")
	}
	assertSinglePrefix(t, fake, "CutAt with an unknown Moment")

	fake = &fakeT{}
	relay = &Relay{t: fake, draining: map[int]bool{}}
	if !fatalPanics(func() { relay.CutAt(BeforeRequestReachesServer, faults.Message(0)) }) {
		t.Fatalf("CutAt with an unknown Message returned without failing")
	}
	assertSinglePrefix(t, fake, "CutAt with an unknown Message")

	fake = &fakeT{}
	srv := &ScriptedServer{t: fake, heldSignal: make(chan struct{}, 1), subscriptions: map[uint32]*harnessSub{}, clientHandles: map[uint32]uint32{}}
	sub := Subscription{server: srv, sub: newHarnessSub(1)}
	srv.subscriptions[1] = sub.sub
	if !fatalPanics(func() { sub.Retain(4294967295, 7001) }) {
		t.Fatalf("Retain for the largest uint32 sequence number returned without failing")
	}
	assertSinglePrefix(t, fake, "Retain for the largest uint32 sequence number")

	fake = &fakeT{}
	recorder := newRecorder(fake)
	openAddress := addr("127.0.0.1:4301")
	recorder.observeConnection(0, openAddress)
	srv = &ScriptedServer{
		t:             fake,
		heldWait:      0,
		heldSignal:    make(chan struct{}, 1),
		subscriptions: map[uint32]*harnessSub{},
		clientHandles: map[uint32]uint32{7: monitorClientHandle},
		recorder:      recorder,
	}
	sub = Subscription{server: srv, sub: newHarnessSub(7)}
	srv.subscriptions[7] = sub.sub
	entry := &heldEntry{request: &ua.PublishRequest{RequestHeader: &ua.RequestHeader{}}, connection: 0, remoteAddr: openAddress}
	held := HeldPublish{server: srv, entry: entry}
	if !fatalPanics(func() { held.AnswerWithSequenceNumber(sub, 4294967295, 7002) }) {
		t.Fatalf("AnswerWithSequenceNumber for the largest uint32 sequence number returned without failing")
	}
	assertSinglePrefix(t, fake, "AnswerWithSequenceNumber for the largest uint32 sequence number")

	fake = &fakeT{}
	srv = &ScriptedServer{t: fake, address: "opc.tcp://127.0.0.1:1", heldSignal: make(chan struct{}, 1), subscriptions: map[uint32]*harnessSub{}, clientHandles: map[uint32]uint32{}}
	other := &ScriptedServer{t: fake, address: "opc.tcp://127.0.0.1:2", heldSignal: make(chan struct{}, 1), subscriptions: map[uint32]*harnessSub{}, clientHandles: map[uint32]uint32{}}
	foreign := Subscription{server: other, sub: newHarnessSub(1)}
	entry = &heldEntry{}
	held = HeldPublish{server: srv, entry: entry}
	if !fatalPanics(func() { held.Answer(foreign, 7003) }) {
		t.Fatalf("answering with another server's subscription returned without failing")
	}
	assertSinglePrefix(t, fake, "answering with another server's subscription")

	fake = &fakeT{}
	srv = &ScriptedServer{t: fake, heldSignal: make(chan struct{}, 1), subscriptions: map[uint32]*harnessSub{}, clientHandles: map[uint32]uint32{}}
	entry = &heldEntry{answered: true}
	held = HeldPublish{server: srv, entry: entry}
	sub = Subscription{server: srv, sub: newHarnessSub(1)}
	srv.subscriptions[1] = sub.sub
	if !fatalPanics(func() { held.Answer(sub, 7004) }) {
		t.Fatalf("answering a held Publish twice returned without failing")
	}
	assertSinglePrefix(t, fake, "answering a held Publish twice")

	fake = &fakeT{}
	recorder = newRecorder(fake)
	closedAddress := addr("127.0.0.1:4302")
	recorder.observeConnection(0, closedAddress)
	recorder.observeConnectionClosed(0)
	srv = &ScriptedServer{
		t:             fake,
		heldSignal:    make(chan struct{}, 1),
		subscriptions: map[uint32]*harnessSub{},
		clientHandles: map[uint32]uint32{7: monitorClientHandle},
		recorder:      recorder,
	}
	sub = Subscription{server: srv, sub: newHarnessSub(7)}
	srv.subscriptions[7] = sub.sub
	entry = &heldEntry{request: &ua.PublishRequest{RequestHeader: &ua.RequestHeader{}}, connection: 0, remoteAddr: closedAddress}
	held = HeldPublish{server: srv, entry: entry}
	if !fatalPanics(func() { held.Answer(sub, 7005) }) {
		t.Fatalf("answering a held Publish on a closed connection returned without failing")
	}
	assertSinglePrefix(t, fake, "answering a held Publish on a closed connection")

	fake = &fakeT{}
	recorder = newRecorder(fake)
	openAddress = addr("127.0.0.1:4303")
	recorder.observeConnection(0, openAddress)
	srv = &ScriptedServer{
		t:             fake,
		heldSignal:    make(chan struct{}, 1),
		subscriptions: map[uint32]*harnessSub{},
		clientHandles: map[uint32]uint32{7: monitorClientHandle},
		recorder:      recorder,
	}
	sub = Subscription{server: srv, sub: newHarnessSub(9)}
	entry = &heldEntry{request: &ua.PublishRequest{RequestHeader: &ua.RequestHeader{}}, connection: 0, remoteAddr: openAddress}
	held = HeldPublish{server: srv, entry: entry}
	if !fatalPanics(func() { held.Answer(sub, 7006) }) {
		t.Fatalf("answering with a subscription the server does not hold returned without failing")
	}
	assertSinglePrefix(t, fake, "answering with a subscription the server does not hold")
	if !strings.Contains(fake.fatals[0], "holds no live subscription") {
		t.Errorf("answering with a subscription the server does not hold failed with %q, want the no-live-subscription fault", fake.fatals[0])
	}

	fake = &fakeT{}
	recorder = newRecorder(fake)
	openAddress = addr("127.0.0.1:4304")
	recorder.observeConnection(0, openAddress)
	srv = &ScriptedServer{
		t:             fake,
		heldSignal:    make(chan struct{}, 1),
		subscriptions: map[uint32]*harnessSub{},
		clientHandles: map[uint32]uint32{},
		recorder:      recorder,
	}
	sub = Subscription{server: srv, sub: newHarnessSub(7)}
	srv.subscriptions[7] = sub.sub
	entry = &heldEntry{request: &ua.PublishRequest{RequestHeader: &ua.RequestHeader{}}, connection: 0, remoteAddr: openAddress}
	held = HeldPublish{server: srv, entry: entry}
	if !fatalPanics(func() { held.Answer(sub, 7007) }) {
		t.Fatalf("answering a subscription with no recorded client handle returned without failing")
	}
	assertSinglePrefix(t, fake, "answering a subscription with no recorded client handle")

	fake = &fakeT{}
	recorder = newRecorder(fake)
	recorder.setRelayError(errors.New("the relay stopped accepting client connections: connection refused"))
	if !fatalPanics(func() { recorder.Transport() }) {
		t.Fatalf("a stored recorder error returned without failing")
	}
	assertSinglePrefix(t, fake, "a stored recorder error raised at an accessor")

	fake = &fakeT{}
	srv = &ScriptedServer{t: fake, heldWait: time.Second, heldSignal: make(chan struct{}, 1), subscriptions: map[uint32]*harnessSub{}, clientHandles: map[uint32]uint32{}}
	srv.fault("the scripted server misused while waiting")
	if !fatalPanics(func() { srv.WaitHeldPublish() }) {
		t.Fatalf("WaitHeldPublish with a stored server fault returned without failing")
	}
	assertPrefixedFault(t, fake, "WaitHeldPublish with a stored server fault")

	fake = &fakeT{}
	srv = &ScriptedServer{t: fake, heldSignal: make(chan struct{}, 1), subscriptions: map[uint32]*harnessSub{}, clientHandles: map[uint32]uint32{}}
	srv.fault("the scripted server misused while listing scripts")
	if !fatalPanics(func() { srv.UnusedScripts() }) {
		t.Fatalf("UnusedScripts with a stored server fault returned without failing")
	}
	assertPrefixedFault(t, fake, "UnusedScripts with a stored server fault")
}

// Start's subscribe, monitor and first-delivery failures have no
// deterministic trigger through Start's options, so only the connect
// failure and Subscription() are covered.
func TestClientFailuresCarryNoPrefix(t *testing.T) {
	fake := &fakeT{}
	if !fatalPanics(func() {
		Start(fake, WithClientOptions(opcua.Dialer(&uacp.Dialer{Dialer: &net.Dialer{Timeout: time.Nanosecond}})))
	}) {
		t.Fatalf("Start returned without failing although its client cannot connect")
	}
	for i := len(fake.cleanups) - 1; i >= 0; i-- {
		fake.cleanups[i]()
	}
	if len(fake.fatals) != 1 {
		t.Fatalf("a client that cannot connect failed %d times, want exactly the connect failure: %v", len(fake.fatals), fake.fatals)
	}
	if strings.HasPrefix(fake.fatals[0], "spectest:") {
		t.Errorf("Start's connect failure %q carries the spectest: prefix, want a plain client failure", fake.fatals[0])
	}

	fake = &fakeT{}
	srv := &ScriptedServer{t: fake, heldSignal: make(chan struct{}, 1), subscriptions: map[uint32]*harnessSub{}, clientHandles: map[uint32]uint32{}}
	env := &Environment{t: fake, Server: srv}
	if !fatalPanics(func() { env.Subscription() }) {
		t.Fatalf("Subscription returned without failing although the client created no subscription")
	}
	if len(fake.fatals) != 1 || strings.HasPrefix(fake.fatals[0], "spectest:") {
		t.Errorf("Subscription failed with %q, want one plain client failure", fake.fatals)
	}
}
