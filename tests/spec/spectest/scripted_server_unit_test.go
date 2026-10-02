package spectest

import (
	"fmt"
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
	if len(fake.fatals) != 1 || fake.fatals[0] != harnessFault("sequence number wraparound is out of scope") {
		t.Fatalf("Retain failed with %q, want %q", fake.fatals, harnessFault("sequence number wraparound is out of scope"))
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
	if len(fake.fatals) != 1 || fake.fatals[0] != harnessFault("sequence number wraparound is out of scope") {
		t.Fatalf("AnswerWithSequenceNumber failed with %q, want %q", fake.fatals, harnessFault("sequence number wraparound is out of scope"))
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

func TestDeleteSubscriptionsMarksTheSubscriptionDeleted(t *testing.T) {
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
	marked, still := srv.subscriptions[9]
	if !still || !marked.deleted {
		t.Fatalf("DeleteSubscriptions left the subscription live or removed it from the harness set: still=%v deleted=%v", still, still && marked.deleted)
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

func TestWaitCreatedSubscriptionFailsWhenTheClientDeletedTheSubscription(t *testing.T) {
	fake := &fakeT{}
	recorder := newRecorder(fake)
	openAddress := addr("127.0.0.1:4305")
	recorder.observeConnection(0, openAddress)
	recorder.observeUpstream(0, "127.0.0.1:1")
	srv := &ScriptedServer{
		t:             fake,
		address:       "opc.tcp://127.0.0.1:1",
		heldWait:      time.Second,
		heldSignal:    make(chan struct{}, 1),
		subscriptions: make(map[uint32]*harnessSub),
		clientHandles: make(map[uint32]uint32),
		recorder:      recorder,
	}
	sub := newHarnessSub(3)
	srv.subscriptions[3] = sub
	recorder.appendService(1, 0, serverToClient, 10, Forwarded, &ua.CreateSubscriptionResponse{SubscriptionID: 3})
	srv.mu.Lock()
	sub.deleted = true
	srv.mu.Unlock()

	if !fatalPanics(func() { srv.WaitCreatedSubscription(Mark{}) }) {
		t.Fatalf("WaitCreatedSubscription returned although the client deleted the subscription it created")
	}
	if len(fake.fatals) != 1 || fake.fatals[0] != "client deleted the subscription it created (id 3)" {
		t.Fatalf("WaitCreatedSubscription failed with %q, want the plain deleted-subscription failure", fake.fatals)
	}
	for _, cleanup := range fake.cleanups {
		cleanup()
	}
}

func TestWaitCreatedSubscriptionLockProbeWithAConcurrentDeleteAndRestore(t *testing.T) {
	fake := &fakeT{}
	recorder := newRecorder(fake)
	openAddress := addr("127.0.0.1:4306")
	recorder.observeConnection(0, openAddress)
	recorder.observeUpstream(0, "127.0.0.1:1")
	srv := &ScriptedServer{
		t:             fake,
		address:       "opc.tcp://127.0.0.1:1",
		heldWait:      time.Second,
		heldSignal:    make(chan struct{}, 1),
		subscriptions: make(map[uint32]*harnessSub),
		clientHandles: make(map[uint32]uint32),
		recorder:      recorder,
	}
	sub := newHarnessSub(3)
	srv.subscriptions[3] = sub
	recorder.appendService(1, 0, serverToClient, 10, Forwarded, &ua.CreateSubscriptionResponse{SubscriptionID: 3})

	stop := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			select {
			case <-stop:
				return
			default:
				srv.mu.Lock()
				sub.deleted = true
				srv.mu.Unlock()
				srv.mu.Lock()
				sub.deleted = false
				srv.mu.Unlock()
			}
		}
	}()

	var waitedID uint32
	deleted := fatalPanics(func() { waitedID = srv.WaitCreatedSubscription(Mark{}).ID() })

	close(stop)
	<-done
	if deleted {
		if len(fake.fatals) != 1 || fake.fatals[0] != "client deleted the subscription it created (id 3)" {
			t.Fatalf("WaitCreatedSubscription failed with %q, want the plain deleted-subscription failure", fake.fatals)
		}
	} else if waitedID != 3 {
		t.Fatalf("WaitCreatedSubscription returned subscription %d, want 3, the subscription the recorded create answered", waitedID)
	}
	for _, cleanup := range fake.cleanups {
		cleanup()
	}
}

func TestCreateOverADeletedIdReplacesTheRecord(t *testing.T) {
	fake := &fakeT{}
	srv := &ScriptedServer{
		t:             fake,
		heldWait:      time.Second,
		heldSignal:    make(chan struct{}, 1),
		subscriptions: make(map[uint32]*harnessSub),
		clientHandles: make(map[uint32]uint32),
	}
	deleted := newHarnessSub(3)
	srv.subscriptions[3] = deleted
	Subscription{server: srv, sub: deleted}.Retain(4, 7001)
	Subscription{server: srv, sub: deleted}.FailRepublish(5, ua.StatusBadTimeout)

	deleteRequest := &ua.DeleteSubscriptionsRequest{RequestHeader: &ua.RequestHeader{}, SubscriptionIDs: []uint32{3}}
	deleteResponse, deleteErr := srv.deleteSubscriptions(nil, deleteRequest, 0)
	if deleteErr != nil {
		t.Fatalf("the DeleteSubscriptions failed: %v", deleteErr)
	}
	if deleteResults, isDelete := deleteResponse.(*ua.DeleteSubscriptionsResponse); !isDelete || !slices.Equal(deleteResults.Results, []ua.StatusCode{ua.StatusOK}) {
		t.Fatalf("DeleteSubscriptions answered %+v, want [Good] for the live id", deleteResponse)
	}

	if !srv.createSubscriptionRecord(&ua.CreateSubscriptionResponse{SubscriptionID: 3}) {
		t.Fatalf("CreateSubscription over the deleted id 3 faulted as a collision")
	}
	srv.mu.Lock()
	replacement, held := srv.subscriptions[3]
	srv.mu.Unlock()
	if !held || replacement == deleted || replacement.deleted {
		t.Fatalf("creating over the deleted id left the deleted record in place: held=%v same=%v", held, replacement == deleted)
	}
	if len(replacement.retained) != 0 || len(replacement.unansweredRetained) != 0 || len(replacement.failRepublish) != 0 {
		t.Fatalf("the replacement did not start with fresh scripts: %+v", replacement)
	}
	if scripts := srv.UnusedScripts(); !slices.Contains(scripts, "retained 4 for subscription 3") || !slices.Contains(scripts, "FailRepublish(5, BadTimeout) for subscription 3") {
		t.Fatalf("UnusedScripts returned %v, want the replaced record's staged retained message and FailRepublish script still listed", scripts)
	}
}

func TestRetainOnAReplacedHandleStoresAHarnessFault(t *testing.T) {
	fake := &fakeT{}
	srv := &ScriptedServer{
		t:             fake,
		heldWait:      time.Second,
		heldSignal:    make(chan struct{}, 1),
		subscriptions: make(map[uint32]*harnessSub),
		clientHandles: make(map[uint32]uint32),
	}
	old := newHarnessSub(3)
	srv.subscriptions[3] = old
	deleteRequest := &ua.DeleteSubscriptionsRequest{RequestHeader: &ua.RequestHeader{}, SubscriptionIDs: []uint32{3}}
	if _, deleteErr := srv.deleteSubscriptions(nil, deleteRequest, 0); deleteErr != nil {
		t.Fatalf("the DeleteSubscriptions failed: %v", deleteErr)
	}
	if !srv.createSubscriptionRecord(&ua.CreateSubscriptionResponse{SubscriptionID: 3}) {
		t.Fatalf("CreateSubscription over the deleted id 3 faulted as a collision")
	}

	Subscription{server: srv, sub: old}.Retain(5, 7002)

	srv.mu.Lock()
	faultErr := srv.faultErr
	replacement, held := srv.subscriptions[3]
	srv.mu.Unlock()
	if faultErr == nil || !strings.HasPrefix(faultErr.Error(), "spectest:") || !strings.Contains(faultErr.Error(), "no live subscription 3") {
		t.Fatalf("Retain on the replaced handle stored %v, want the prefixed no-live-subscription fault", faultErr)
	}
	if !held || replacement == old {
		t.Fatalf("Retain on the replaced handle disturbed the replacement record: held=%v same=%v", held, replacement == old)
	}
	if len(replacement.retained) != 0 || len(replacement.unansweredRetained) != 0 {
		t.Fatalf("Retain on the replaced handle staged its message on the replacement: %+v", replacement)
	}
}

func TestFailRepublishOnAReplacedHandleStoresAHarnessFault(t *testing.T) {
	fake := &fakeT{}
	srv := &ScriptedServer{
		t:             fake,
		heldWait:      time.Second,
		heldSignal:    make(chan struct{}, 1),
		subscriptions: make(map[uint32]*harnessSub),
		clientHandles: make(map[uint32]uint32),
	}
	old := newHarnessSub(3)
	srv.subscriptions[3] = old
	deleteRequest := &ua.DeleteSubscriptionsRequest{RequestHeader: &ua.RequestHeader{}, SubscriptionIDs: []uint32{3}}
	if _, deleteErr := srv.deleteSubscriptions(nil, deleteRequest, 0); deleteErr != nil {
		t.Fatalf("the DeleteSubscriptions failed: %v", deleteErr)
	}
	if !srv.createSubscriptionRecord(&ua.CreateSubscriptionResponse{SubscriptionID: 3}) {
		t.Fatalf("CreateSubscription over the deleted id 3 faulted as a collision")
	}

	Subscription{server: srv, sub: old}.FailRepublish(5, ua.StatusBadTimeout)

	srv.mu.Lock()
	faultErr := srv.faultErr
	replacement, held := srv.subscriptions[3]
	srv.mu.Unlock()
	if faultErr == nil || !strings.HasPrefix(faultErr.Error(), "spectest:") || !strings.Contains(faultErr.Error(), "no live subscription 3") {
		t.Fatalf("FailRepublish on the replaced handle stored %v, want the prefixed no-live-subscription fault", faultErr)
	}
	if !held || replacement == old {
		t.Fatalf("FailRepublish on the replaced handle disturbed the replacement record: held=%v same=%v", held, replacement == old)
	}
	if len(replacement.failRepublish) != 0 {
		t.Fatalf("FailRepublish on the replaced handle staged its script on the replacement: %+v", replacement)
	}
}

func TestHeldPublishOrderMatchesTheRecordedRequest(t *testing.T) {
	fake := &fakeT{}
	recorder := newRecorder(fake)
	recorder.appendService(7, 1, clientToServer, 42, Forwarded, &ua.PublishRequest{RequestHeader: &ua.RequestHeader{}})
	srv := &ScriptedServer{
		t:             fake,
		heldWait:      time.Second,
		heldSignal:    make(chan struct{}, 1),
		subscriptions: make(map[uint32]*harnessSub),
		clientHandles: make(map[uint32]uint32),
		recorder:      recorder,
	}
	entry := &heldEntry{requestID: 42, request: &ua.PublishRequest{RequestHeader: &ua.RequestHeader{}}, connection: 1}
	held := HeldPublish{server: srv, entry: entry}

	if order, found := held.Order(); !found || order != 7 {
		t.Fatalf("HeldPublish.Order() = %d, %v, want 7, true, the order of the recorded request it pairs with", order, found)
	}
	unrecorded := HeldPublish{server: srv, entry: &heldEntry{requestID: 43, request: &ua.PublishRequest{RequestHeader: &ua.RequestHeader{}}, connection: 1}}
	if order, found := unrecorded.Order(); found {
		t.Fatalf("HeldPublish.Order() = %d, true for a request the recorder never saw, want false", order)
	}
	for _, cleanup := range fake.cleanups {
		cleanup()
	}
}

func TestStaleSubscriptionHandlesStoreTheNoLiveSubscriptionFault(t *testing.T) {
	newServer := func() (*fakeT, *ScriptedServer) {
		fake := &fakeT{}
		return fake, &ScriptedServer{
			t:             fake,
			heldWait:      time.Second,
			heldSignal:    make(chan struct{}, 1),
			subscriptions: make(map[uint32]*harnessSub),
			clientHandles: make(map[uint32]uint32),
		}
	}
	assertNoLiveSubscriptionFault := func(fake *fakeT, srv *ScriptedServer, wantID uint32) {
		t.Helper()
		srv.mu.Lock()
		faultErr := srv.faultErr
		srv.mu.Unlock()
		if faultErr == nil || !strings.HasPrefix(faultErr.Error(), "spectest:") ||
			!strings.Contains(faultErr.Error(), fmt.Sprintf("no live subscription %d", wantID)) {
			t.Fatalf("the stale handle stored %v, want the prefixed no-live-subscription fault naming subscription %d", faultErr, wantID)
		}
	}

	fake, srv := newServer()
	deleted := newHarnessSub(3)
	srv.subscriptions[3] = deleted
	deleteRequest := &ua.DeleteSubscriptionsRequest{RequestHeader: &ua.RequestHeader{}, SubscriptionIDs: []uint32{3}}
	if _, deleteErr := srv.deleteSubscriptions(nil, deleteRequest, 0); deleteErr != nil {
		t.Fatalf("the DeleteSubscriptions failed: %v", deleteErr)
	}
	Subscription{server: srv, sub: deleted}.Retain(5, 7002)
	Subscription{server: srv, sub: deleted}.FailRepublish(5, ua.StatusBadTimeout)
	assertNoLiveSubscriptionFault(fake, srv, 3)
	srv.mu.Lock()
	undisturbed := len(deleted.retained) == 0 && len(deleted.unansweredRetained) == 0 && len(deleted.failRepublish) == 0
	srv.mu.Unlock()
	if !undisturbed {
		t.Fatalf("the stale-handle calls staged scripts on the deleted record: %+v", deleted)
	}

	fake, srv = newServer()
	neverHeld := newHarnessSub(9)
	Subscription{server: srv, sub: neverHeld}.Retain(5, 7002)
	Subscription{server: srv, sub: neverHeld}.FailRepublish(5, ua.StatusBadTimeout)
	assertNoLiveSubscriptionFault(fake, srv, 9)

	fake, srv = newServer()
	recorder := newRecorder(fake)
	openAddress := addr("127.0.0.1:4307")
	recorder.observeConnection(0, openAddress)
	srv.recorder = recorder
	pending := srv.QueueTransferSuccess(5, 6)
	entry := &heldEntry{request: &ua.PublishRequest{RequestHeader: &ua.RequestHeader{}}, connection: 0, remoteAddr: openAddress}
	held := HeldPublish{server: srv, entry: entry}
	if !fatalPanics(func() { held.Answer(pending, 7003) }) {
		t.Fatalf("Answer accepted the pending-transfer handle")
	}
	if len(fake.fatals) != 1 || !strings.HasPrefix(fake.fatals[0], "spectest:") || !strings.Contains(fake.fatals[0], "no live subscription") {
		t.Fatalf("Answer through the pending-transfer handle failed with %q, want the prefixed no-live-subscription fault", fake.fatals)
	}
	for _, cleanup := range fake.cleanups {
		cleanup()
	}
}

func TestTransferOverADeletedIdReplacesTheRecord(t *testing.T) {
	fake := &fakeT{}
	srv := &ScriptedServer{
		t:             fake,
		heldWait:      time.Second,
		heldSignal:    make(chan struct{}, 1),
		subscriptions: make(map[uint32]*harnessSub),
		clientHandles: make(map[uint32]uint32),
	}
	deleted := newHarnessSub(4)
	srv.subscriptions[4] = deleted
	Subscription{server: srv, sub: deleted}.Retain(4, valueRetained)

	deleteRequest := &ua.DeleteSubscriptionsRequest{RequestHeader: &ua.RequestHeader{}, SubscriptionIDs: []uint32{4}}
	if _, deleteErr := srv.deleteSubscriptions(nil, deleteRequest, 0); deleteErr != nil {
		t.Fatalf("the DeleteSubscriptions failed: %v", deleteErr)
	}

	srv.QueueTransferSuccess(5, 6)
	request := &ua.TransferSubscriptionsRequest{RequestHeader: &ua.RequestHeader{}, SubscriptionIDs: []uint32{4}, SendInitialValues: false}
	response, err := srv.transferSubscriptions(nil, request, 0)
	if err != nil {
		t.Fatalf("the TransferSubscriptions over the deleted id failed: %v", err)
	}
	transfer, isTransfer := response.(*ua.TransferSubscriptionsResponse)
	if !isTransfer || len(transfer.Results) != 1 || transfer.Results[0].StatusCode != ua.StatusOK {
		t.Fatalf("the transfer answered %+v, want one Good result over the deleted id", response)
	}
	srv.mu.Lock()
	replacement, held := srv.subscriptions[4]
	srv.mu.Unlock()
	if !held || replacement == deleted || replacement.deleted {
		t.Fatalf("the transfer over the deleted id left the deleted record in place: held=%v same=%v", held, replacement == deleted)
	}
	if scripts := srv.UnusedScripts(); !slices.Contains(scripts, "retained 4 for subscription 4") {
		t.Fatalf("UnusedScripts returned %v, want the replaced record's staged retained message still listed", scripts)
	}
}

func TestUnusedScriptsStillListsADeletedSubscription(t *testing.T) {
	fake := &fakeT{}
	srv := &ScriptedServer{
		t:             fake,
		heldSignal:    make(chan struct{}, 1),
		subscriptions: make(map[uint32]*harnessSub),
		clientHandles: make(map[uint32]uint32),
	}
	sub := Subscription{server: srv, sub: newHarnessSub(3)}
	srv.subscriptions[3] = sub.sub
	sub.Retain(4, 7001)
	sub.FailRepublish(5, ua.StatusBadTimeout)

	deleteRequest := &ua.DeleteSubscriptionsRequest{RequestHeader: &ua.RequestHeader{}, SubscriptionIDs: []uint32{3}}
	deleteResponse, deleteErr := srv.deleteSubscriptions(nil, deleteRequest, 0)
	if deleteErr != nil {
		t.Fatalf("the DeleteSubscriptions failed: %v", deleteErr)
	}
	deleteResults, isDelete := deleteResponse.(*ua.DeleteSubscriptionsResponse)
	if !isDelete {
		t.Fatalf("the DeleteSubscriptions answered with a %T, want a DeleteSubscriptionsResponse", deleteResponse)
	}
	if !slices.Equal(deleteResults.Results, []ua.StatusCode{ua.StatusOK}) {
		t.Fatalf("DeleteSubscriptions answered %v, want [Good]", deleteResults.Results)
	}

	scripts := srv.UnusedScripts()
	want := []string{
		"FailRepublish(5, BadTimeout) for subscription 3",
		"retained 4 for subscription 3",
	}
	if !slices.Equal(scripts, want) {
		t.Errorf("UnusedScripts on a deleted subscription returned %v, want %v", scripts, want)
	}
}

func TestUnusedScriptsRaisesAStoredServerFault(t *testing.T) {
	fake := &fakeT{}
	srv := &ScriptedServer{
		t:             fake,
		heldSignal:    make(chan struct{}, 1),
		subscriptions: make(map[uint32]*harnessSub),
		clientHandles: make(map[uint32]uint32),
	}
	srv.fault("the scripted server misused")

	if !fatalPanics(func() { srv.UnusedScripts() }) {
		t.Fatalf("UnusedScripts returned without failing although the server stored a fault")
	}
	if len(fake.fatals) != 1 || fake.fatals[0] != harnessFault("the scripted server misused") {
		t.Errorf("UnusedScripts failed with %v, want the stored fault raised once", fake.fatals)
	}
}
