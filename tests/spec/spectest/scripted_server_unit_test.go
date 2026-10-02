package spectest

import (
	"math"
	"slices"
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

func TestCreateSubscriptionCollisionStoresAServerFault(t *testing.T) {
	fake := &fakeT{}
	srv := &ScriptedServer{
		t:             fake,
		heldWait:      time.Second,
		heldSignal:    make(chan struct{}, 1),
		subscriptions: make(map[uint32]*harnessSub),
		clientHandles: make(map[uint32]uint32),
	}
	transferred := newHarnessSub(3)
	srv.subscriptions[3] = transferred

	response := &ua.CreateSubscriptionResponse{SubscriptionID: 3}
	if fatalPanics(func() { srv.createSubscriptionRecord(response) }) {
		t.Fatalf("the collision guard failed the server goroutine directly instead of storing a server fault: %v", fake.fatals)
	}
	srv.mu.Lock()
	faultErr := srv.faultErr
	srv.mu.Unlock()
	if faultErr == nil || !strings.Contains(faultErr.Error(), "subscription id collision") {
		t.Fatalf("the collision guard stored %v, want the subscription id collision fault", faultErr)
	}
	if !strings.HasPrefix(faultErr.Error(), "spectest:") {
		t.Fatalf("the stored collision fault is not a harness fault: %q", faultErr.Error())
	}
	if _, still := srv.subscriptions[3]; !still || srv.subscriptions[3] != transferred {
		t.Fatalf("the collision guard changed the live subscription %d", 3)
	}

	srv.heldWait = 20 * time.Millisecond
	if !fatalPanics(func() { srv.WaitCreatedSubscription(Mark{}) }) {
		t.Fatalf("the stored collision fault never surfaced from a later harness call")
	}
	if len(fake.fatals) != 1 || !strings.Contains(fake.fatals[0], "subscription id collision") || !strings.HasPrefix(fake.fatals[0], "spectest:") {
		t.Fatalf("the stored collision fault surfaced as %q, want the spectest: subscription id collision fault", fake.fatals)
	}
}

func TestTransferCollisionStoresAServerFault(t *testing.T) {
	fake := &fakeT{}
	srv := &ScriptedServer{
		t:             fake,
		heldWait:      time.Second,
		heldSignal:    make(chan struct{}, 1),
		subscriptions: make(map[uint32]*harnessSub),
		clientHandles: make(map[uint32]uint32),
	}
	live := newHarnessSub(4)
	srv.subscriptions[4] = live
	srv.QueueTransferSuccess(5, 6)

	request := &ua.TransferSubscriptionsRequest{RequestHeader: &ua.RequestHeader{}, SubscriptionIDs: []uint32{4}, SendInitialValues: false}
	response, err := srv.transferSubscriptions(nil, request, 0)
	if response != nil || err != ua.StatusBadInternalError {
		t.Fatalf("the colliding transfer answered with (%T, %v), want a Bad_InternalError ServiceFault", response, err)
	}
	srv.mu.Lock()
	faultErr := srv.faultErr
	srv.mu.Unlock()
	if faultErr == nil || !strings.Contains(faultErr.Error(), "subscription id collision") || !strings.HasPrefix(faultErr.Error(), "spectest:") {
		t.Fatalf("the collision guard stored %v, want the spectest: subscription id collision fault", faultErr)
	}
	if _, still := srv.subscriptions[4]; !still || srv.subscriptions[4] != live {
		t.Fatalf("the collision guard changed the live subscription %d", 4)
	}
}

func TestDeleteSubscriptionsRemovesTheIdFromTheLiveSet(t *testing.T) {
	fake := &fakeT{}
	recorder := newRecorder(fake)
	openAddress := addr("127.0.0.1:4303")
	recorder.observeConnection(0, openAddress)
	srv := &ScriptedServer{
		t:             fake,
		heldWait:      time.Second,
		heldSignal:    make(chan struct{}, 1),
		subscriptions: make(map[uint32]*harnessSub),
		clientHandles: make(map[uint32]uint32),
		recorder:      recorder,
	}
	deletedSub := newHarnessSub(9)
	srv.subscriptions[9] = deletedSub
	srv.clientHandles[9] = monitorClientHandle
	entry := &heldEntry{request: &ua.PublishRequest{RequestHeader: &ua.RequestHeader{}}, connection: 0, remoteAddr: openAddress}
	held := HeldPublish{server: srv, entry: entry}

	deleteRequest := &ua.DeleteSubscriptionsRequest{RequestHeader: &ua.RequestHeader{}, SubscriptionIDs: []uint32{9, 10}}
	response, err := srv.deleteSubscriptions(nil, deleteRequest, 0)
	if err != nil {
		t.Fatalf("the DeleteSubscriptions failed: %v", err)
	}
	deleteResponse, isDelete := response.(*ua.DeleteSubscriptionsResponse)
	if !isDelete {
		t.Fatalf("the DeleteSubscriptions answered with a %T, want a DeleteSubscriptionsResponse", response)
	}
	if !slices.Equal(deleteResponse.Results, []ua.StatusCode{ua.StatusOK, ua.StatusBadSubscriptionIDInvalid}) {
		t.Fatalf("DeleteSubscriptions answered %v, want Good for the live id and Bad_SubscriptionIDInvalid for the unknown one", deleteResponse.Results)
	}
	if _, still := srv.subscriptions[9]; still {
		t.Fatalf("DeleteSubscriptions left the id live in the harness set")
	}

	republishRequest := &ua.RepublishRequest{RequestHeader: &ua.RequestHeader{}, SubscriptionID: 9, RetransmitSequenceNumber: 1}
	if _, republishErr := srv.republish(nil, republishRequest, 0); republishErr != ua.StatusBadSubscriptionIDInvalid {
		t.Fatalf("the Republish for the deleted id failed with %v, want Bad_SubscriptionIDInvalid", republishErr)
	}

	if !fatalPanics(func() { held.Answer(Subscription{server: srv, sub: deletedSub}, 5) }) {
		t.Fatalf("Answer with the deleted subscription's handle did not fail")
	}
	if len(fake.fatals) != 1 || !strings.Contains(fake.fatals[0], "no live subscription 9") || !strings.HasPrefix(fake.fatals[0], "spectest:") {
		t.Fatalf("Answer with the deleted handle failed with %q, want the no-live-subscription harness fault", fake.fatals)
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
