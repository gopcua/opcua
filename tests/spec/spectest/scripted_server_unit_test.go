package spectest

import (
	"math"
	"strings"
	"testing"
	"time"

	"github.com/gopcua/opcua/ua"
)

func TestRetainFailsOnSequenceNumberWraparound(t *testing.T) {
	fake := &fakeT{}
	recorder := newRecorder(fake)
	openAddress := addr("127.0.0.1:4300")
	recorder.observeConnection(0, openAddress)
	srv := &ScriptedServer{
		t:             fake,
		heldWait:      time.Second,
		heldSignal:    make(chan struct{}, 1),
		subscriptions: make(map[uint32]*harnessSub),
		clientHandles: make(map[uint32]uint32),
		recorder:      recorder,
	}
	sub := Subscription{server: srv, sub: newHarnessSub(1)}
	srv.subscriptions[sub.ID()] = sub.sub
	srv.clientHandles[sub.ID()] = monitorClientHandle

	if !fatalPanics(func() { sub.Retain(math.MaxUint32, 5) }) {
		t.Fatalf("Retain did not fail for the largest uint32 sequence number")
	}
	if len(fake.fatals) != 1 || fake.fatals[0] != wraparoundFailure {
		t.Fatalf("Retain failed with %q, want %q", fake.fatals, wraparoundFailure)
	}
	if sub.sub.next != 1 {
		t.Fatalf("Retain moved the sequence counter to %d, want it unchanged at 1", sub.sub.next)
	}
	if len(sub.sub.retained) != 0 || len(sub.sub.unansweredRetained) != 0 {
		t.Fatalf("Retain stored a message for the largest uint32 sequence number, want nothing stored: retained %v, unanswered %v", sub.sub.retained, sub.sub.unansweredRetained)
	}
	for _, cleanup := range fake.cleanups {
		cleanup()
	}
}

func TestAnswerWithSequenceNumberFailsOnSequenceNumberWraparound(t *testing.T) {
	fake := &fakeT{}
	recorder := newRecorder(fake)
	openAddress := addr("127.0.0.1:4301")
	recorder.observeConnection(0, openAddress)
	srv := &ScriptedServer{
		t:             fake,
		heldWait:      time.Second,
		heldSignal:    make(chan struct{}, 1),
		subscriptions: make(map[uint32]*harnessSub),
		clientHandles: make(map[uint32]uint32),
		recorder:      recorder,
	}
	sub := Subscription{server: srv, sub: newHarnessSub(7)}
	srv.subscriptions[sub.ID()] = sub.sub
	srv.clientHandles[sub.ID()] = monitorClientHandle
	entry := &heldEntry{request: &ua.PublishRequest{RequestHeader: &ua.RequestHeader{}}, connection: 0, remoteAddr: openAddress}
	held := HeldPublish{server: srv, entry: entry}

	if !fatalPanics(func() { held.AnswerWithSequenceNumber(sub, math.MaxUint32, 5) }) {
		t.Fatalf("AnswerWithSequenceNumber did not fail for the largest uint32 sequence number")
	}
	if len(fake.fatals) != 1 || fake.fatals[0] != wraparoundFailure {
		t.Fatalf("AnswerWithSequenceNumber failed with %q, want %q", fake.fatals, wraparoundFailure)
	}
	if sub.sub.next != 1 {
		t.Fatalf("AnswerWithSequenceNumber moved the sequence counter to %d, want it unchanged at 1", sub.sub.next)
	}
	for _, cleanup := range fake.cleanups {
		cleanup()
	}
}

func TestAnswerFailsWhenTheSubscriptionBelongsToAnotherServer(t *testing.T) {
	fake := &fakeT{}
	recorder := newRecorder(fake)
	openAddress := addr("127.0.0.1:4302")
	recorder.observeConnection(0, openAddress)
	owning := &ScriptedServer{
		t:             fake,
		address:       "opc.tcp://127.0.0.1:1",
		heldWait:      time.Second,
		heldSignal:    make(chan struct{}, 1),
		subscriptions: make(map[uint32]*harnessSub),
		clientHandles: make(map[uint32]uint32),
		recorder:      recorder,
	}
	answering := &ScriptedServer{
		t:             fake,
		address:       "opc.tcp://127.0.0.1:2",
		heldWait:      time.Second,
		heldSignal:    make(chan struct{}, 1),
		subscriptions: make(map[uint32]*harnessSub),
		clientHandles: make(map[uint32]uint32),
		recorder:      recorder,
	}
	ownedSub := Subscription{server: owning, sub: newHarnessSub(3)}
	owning.subscriptions[ownedSub.ID()] = ownedSub.sub
	answering.clientHandles[ownedSub.ID()] = monitorClientHandle
	entry := &heldEntry{request: &ua.PublishRequest{RequestHeader: &ua.RequestHeader{}}, connection: 0, remoteAddr: openAddress}
	held := HeldPublish{server: answering, entry: entry}

	if !fatalPanics(func() { held.Answer(ownedSub, 5) }) {
		t.Fatalf("Answer did not fail for a subscription of another scripted server")
	}
	if len(fake.fatals) != 1 {
		t.Fatalf("Answer failed %d times, want the cross-server guard once", len(fake.fatals))
	}
	if !strings.HasPrefix(fake.fatals[0], "spectest:") ||
		!strings.Contains(fake.fatals[0], owning.address) ||
		!strings.Contains(fake.fatals[0], answering.address) {
		t.Fatalf("Answer failed with %q, want the harness fault naming both scripted servers", fake.fatals[0])
	}
	for _, cleanup := range fake.cleanups {
		cleanup()
	}
}
