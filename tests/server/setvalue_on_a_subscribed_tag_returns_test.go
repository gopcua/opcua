// An application's documented write path is MapNamespace.SetValue:
// called on a key a client is subscribed to, it returns, and the
// subscriber is notified of the value that was written.
//
// The clause is the package comment of server/namespace_map.go:19 - "To
// notify subscribers of changes, be sure to call ChangeNotification(key)
// after changing the value" - which makes an application writing while a
// client holds a subscription the documented case, not an exotic one.
// SetValue (server/namespace_map.go:46) holds the namespace lock across
// the write and the notification, and the notification is delivered by
// MonitoredItemService.ChangeNotification
// (server/monitored_item_service.go:100), which reads the new value back
// through the namespace's own Attribute method on the calling goroutine
// (server/monitored_item_service.go:130). Any lock a fix puts in
// Attribute is therefore a lock SetValue's caller already holds, and
// sync.RWMutex forbids the re-entry: "Note that this prohibits recursive
// read-locking" (go/src/sync/rwmutex.go). The specification's nearest
// clause is OPC 10000-3 §5.6.2 8.58, "In general Atomicity is expected of
// OPC UA read and write operations" - descriptive rather than a shall,
// and not checkable in this tree; the authority the row scores is the Go
// one.
//
// Neighbouring rows: setattribute_on_a_subscribed_tag_returns_test.go is
// the same property through the server's own write path -
// MapNamespace.SetAttribute, where a client Write lands
// (server/attribute_service.go:90) - kept separate because a fix applied
// to SetValue alone passes this row and fails that one.
// read_while_writing_is_race_free_test.go is the other half of the same
// property, and it pulls the fix in the opposite direction: that row goes
// red until Attribute locks the map, this one goes red if the lock is
// taken the obvious way. A change that passes all three is the only
// shape a fix can take.
//
// The row runs against the in-tree gopcua server holding one
// MapNamespace, and a real gopcua client connected over the wire - the
// shared fixture in harness.go, which registers the monitored item
// through the real CreateSubscription and CreateMonitoredItems services;
// why the registration must be real is in harness.go. See harness.go.
//
// The row asserts that the write returns within respond, and that a
// notification carrying the written literal reaches the subscription: a
// change that makes SetValue return by not notifying has not fixed
// anything, and the notification check is what catches that. The
// notification only has to arrive within respond; how late inside that
// window it arrives is not scored. It deliberately asserts nothing
// about the wire the value
// crossed - the server encoded it and the client decoded it, both
// gopcua, so the check scores delivery and not bytes. No row anywhere
// holds the literal bytes for an Int32 inside a Variant that this one
// could name; README.md says what would change that. It also asserts
// nothing about concurrent readers: that is
// read_while_writing_is_race_free_test.go. gopcua passes this row as it
// stands: measured 2026-11-06 with go test -count=1 -race -run
// TestSetValueOnASubscribedTagReturns ./tests/server/, which passed in
// 1.41s because Attribute takes no lock today; and measured again with
// the obvious race fix in place - an RLock added to
// MapNamespace.Attribute - where the same run failed in 6.35s with
// "SetValue on a subscribed tag did not return within 5s", which is the
// regression this row exists to catch.
//
// No reference stack has been read for this behaviour; what one would
// mean here - another server's source - is in README.md. The instrument
// the verdict rests on is a wall clock on the write's return - SetValue
// returns nothing, so the return is scored by outliving the deadline,
// with the write on a goroutine the row spawns and the clock on the
// goroutine that waits - not wire bytes.

package server

import (
	"testing"
	"time"
)

func TestSetValueOnASubscribedTagReturns(t *testing.T) {
	h, teardown, why := startServerAndClient()
	defer teardown()
	if why != "" {
		t.Fatalf("could not set the test up: %s", why)
	}

	subTeardown, why := h.subscribeTo(tag)
	if why != "" {
		t.Fatalf("could not set the test up: %s", why)
	}
	defer subTeardown()

	// The literal comes from outside gopcua; the header says what the
	// notification check scores.
	const written = int32(-4242)

	// The deadline needs the write off the test goroutine: a wedged
	// SetValue called directly would hang the test before it ever reached
	// the deadline. The channel is unbuffered on purpose - the only
	// signal is the close after the call returns.
	returned := make(chan struct{})
	start := time.Now()
	go func() {
		h.ns.SetValue(tag, written)
		close(returned)
	}()

	var elapsed time.Duration
	select {
	case <-returned:
		elapsed = time.Since(start)
	case <-time.After(respond):
		// The verdict the row exists for. Not a decline: the run was
		// scored and the library failed it. Errorf followed by return,
		// because there is nothing left to score after a write that did
		// not return - expectNotification would only add a second failure
		// line five seconds later.
		t.Errorf("SetValue on a subscribed tag did not return within %v", respond)
		return
	}

	if why := h.expectNotification(written); why != "" {
		t.Errorf("%s", why)
	} else {
		t.Logf("SetValue returned after %s and the subscription was notified of %d", elapsed, written)
	}
}
