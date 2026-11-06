// The server's own write path is MapNamespace.SetAttribute, where a
// client's Write request lands (server/attribute_service.go:90): called
// on a node a client is subscribed to, it returns, and the subscriber
// is notified of the value that was written.
//
// The clause is the same Go one the SetValue row scores: the package
// comment of server/namespace_map.go:19, "To notify subscribers of
// changes, be sure to call ChangeNotification(key) after changing the
// value". SetAttribute
// (server/namespace_map.go:339) holds the namespace lock across the
// write and the notification (server/namespace_map.go:357), and the
// notification is delivered by MonitoredItemService.ChangeNotification
// (server/monitored_item_service.go:100), which reads the new value back
// through the namespace's own Attribute method on the calling goroutine
// (server/monitored_item_service.go:130). Any lock a fix puts in
// Attribute is therefore a lock SetAttribute's caller already holds, and
// sync.RWMutex forbids the re-entry: "Note that this prohibits recursive
// read-locking" (go/src/sync/rwmutex.go). The specification's nearest
// clause is OPC 10000-3 §5.6.2 8.58, "In general Atomicity is expected of
// OPC UA read and write operations" - descriptive rather than a shall,
// and not checkable in this tree.
//
// Neighbouring rows: setvalue_on_a_subscribed_tag_returns_test.go is the
// same property through the application's documented write path
// (SetValue); this row is separate because the two are separate call
// sites, and a fix applied to SetValue alone passes that row and fails
// this one - SetAttribute has its own lock-and-notify body, and a fix
// that only removes the re-entrancy from one of them leaves the wedge in
// the other. read_while_writing_is_race_free_test.go is the other half
// of the property and pulls the fix in the opposite direction, as its
// header says.
//
// The row runs against the in-tree gopcua server holding one
// MapNamespace, and a real gopcua client connected over the wire - the
// shared fixture in harness.go, with the monitored item registered
// through the real CreateSubscription and CreateMonitoredItems
// services; why the registration must be real is in harness.go.
// See harness.go.
//
// The row asserts that the client's Write returns within respond and is
// accepted, and that a notification carrying the written literal
// reaches the subscription. The
// notification only has to arrive within respond; how late inside that
// window it arrives is not scored. A Write that errors or is refused is
// a scored failure, not a decline: the server answering it is gopcua's
// own server, so it is the library's behaviour, and nothing after a
// Write that wrote nothing can be scored. It deliberately asserts
// nothing about the wire the value crossed - the server encoded it and
// the client decoded it, both gopcua, so the notification check scores
// delivery and not bytes, and no row anywhere holds the literal bytes
// for an Int32 inside a Variant that this one could name; README.md
// says what would change that. It also asserts nothing about
// concurrent readers: that is read_while_writing_is_race_free_test.go.
// gopcua passes this row as it stands: measured 2026-11-06 with go test
// -count=1 -race -run TestSetAttributeOnASubscribedTagReturns
// ./tests/server/, which passed in 1.45s because Attribute takes no
// lock today; the same measurement with an RLock added to
// MapNamespace.Attribute wedges this call site too - measured
// 2026-11-06, the row failed in 36.82s with "the client's Write on a
// subscribed tag did not return within 5s", the deadline's failure line
// plus the ~30s cleanup lag README.md measures - which is the
// regression this row exists to catch.
//
// No reference stack has been read for this behaviour; what one would
// mean here - another server's source - is in README.md. The instrument
// the verdict rests on is a wall clock on the goroutine that waits - the
// Write itself runs on a goroutine the row spawns, as in the SetValue
// row - not wire bytes.

package server

import (
	"context"
	"testing"
	"time"

	"github.com/gopcua/opcua/ua"
)

func TestSetAttributeOnASubscribedTagReturns(t *testing.T) {
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

	// The header says why this is a client Write and not a SetAttribute
	// call from the row.
	req := &ua.WriteRequest{
		NodesToWrite: []*ua.WriteValue{{
			NodeID:      ua.NewStringNodeID(h.ns.ID(), tag),
			AttributeID: ua.AttributeIDValue,
			Value: &ua.DataValue{
				EncodingMask: ua.DataValueValue,
				Value:        ua.MustVariant(written),
			},
		}},
	}

	// As in the SetValue row, the write's return is scored by a deadline
	// and not by a value, and the deadline needs the Write off the test
	// goroutine: a wedged Write called directly would hang the test
	// before it ever reached the deadline. The channel is buffered on
	// purpose - when the deadline arm wins, the goroutine is still inside
	// Write, and it must not then block on a send to a reader that has
	// stopped waiting.
	type writeResult struct {
		resp *ua.WriteResponse
		err  error
	}
	wrote := make(chan writeResult, 1)
	start := time.Now()
	go func() {
		resp, err := h.client.Write(context.Background(), req)
		wrote <- writeResult{resp, err}
	}()

	// The client's own request timeout (10s by default) is longer than
	// respond, so on a wedged server the deadline below wins; were it
	// shorter, the Write would come back as an error instead and land in
	// the scored failure below - the same red verdict, a different line.
	var wr writeResult
	var elapsed time.Duration
	select {
	case wr = <-wrote:
		elapsed = time.Since(start)
	case <-time.After(respond):
		// The verdict the row exists for. Not a decline: the run was
		// scored and the library failed it. Errorf followed by return,
		// because there is nothing left to score after a Write that did
		// not return - the checks below read the response and the
		// notification, and neither is coming.
		t.Errorf("the client's Write on a subscribed tag did not return within %v",
			respond)
		return
	}

	// A Write that did not complete, or one the server refused, wrote
	// nothing, so the notification the row scores could not happen. The
	// header says why that is a scored failure and not a decline.
	if wr.err != nil {
		t.Errorf("gopcua's own behaviour made this run unscoreable: "+
			"the client's Write did not complete: %v", wr.err)
		return
	}
	if len(wr.resp.Results) != 1 || wr.resp.Results[0] != ua.StatusOK {
		t.Errorf("gopcua's own behaviour made this run unscoreable: "+
			"the server did not accept the client's Write: %v", wr.resp.Results)
		return
	}

	if why := h.expectNotification(written); why != "" {
		t.Errorf("%s", why)
	} else {
		t.Logf("the client's Write returned after %s and the subscription "+
			"was notified of %d", elapsed, written)
	}
}
