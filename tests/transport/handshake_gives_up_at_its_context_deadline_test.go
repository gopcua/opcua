//go:build conformance

// A caller whose context has a deadline is not left blocked in Dial past
// it: the row dials the silent peer with a context whose deadline expires
// while the peer holds the HEL unanswered, and Dial returns.
//
// The clause is OPC 10000-6 §7.1.3, "The Client sends the
// OpenSecureChannel request once it receives the Acknowledge back from
// the Server." It is descriptive: it fixes the order - the client's HEL,
// the server's Acknowledge, the OpenSecureChannel after - so a peer that
// never answers has nothing further to send, and no sentence in the
// section says when or whether a client gives up waiting. No OPC UA
// clause binds this behaviour. What the row scores is the Go contract's
// deadline half: "A Context with a deadline is canceled after the
// deadline passes." (go/src/context/context.go:16). gopcua passes this
// row as it stands: with a deadline present, Handshake sets the
// connection's read deadline from the context (uacp/conn.go:236-237) and
// the blocked Receive (uacp/conn.go:244) fails when it passes, so Dial
// returns. The row exists to guard that deadline path against a change
// made to the cancellation path: the fix
// handshake_gives_up_when_its_context_is_cancelled_test.go waits for is
// naturally a select on the context beside the read, and a change that
// broke the deadline while fixing the cancellation would pass that row
// and fail this one.
//
// Neighbouring rows:
// handshake_gives_up_when_its_context_is_cancelled_test.go is the same
// property on the cancellation half of the same
// contract, and it carries the knownBug constant this row does not.
// dial_reaches_a_host_name_on_its_next_address_test.go scores which
// address of a name Dial dials; it shares no code path with this row
// beyond ResolveEndpoint.
//
// The row runs against the scripted silent peer - the shared fixture in
// harness.go - driven through uacp.Dialer.Dial directly: the row starts
// the peer, dials opc.tcp://<peer address> with a context whose deadline
// is one second, on a goroutine it spawns, and awaits the HEL through
// the shared setup check bounded by that deadline, not by unblock: if
// the HEL has not reached the peer by the time the deadline passes, the
// run cannot construct "the deadline expires while the HEL is held", and
// the row declines. The deadline is the row's own, so the peer's silence
// outlives it and the expiry lands while the HEL is held unanswered. The
// setup check's decline and unscoreable split is awaitHello's: a failed
// check either side could have caused - no connection accepted, or no
// header read in time on a loaded machine - is a decline ("could not set
// the test up: %s"), and a header that is not HELF is gopcua's own
// behaviour, scored "gopcua's own behaviour made this run unscoreable:
// %s" with t.Errorf and a return. The row registers the peer's teardown
// with t.Cleanup, so a Dial still blocked on main is released when the
// test ends. See harness.go.
//
// The row asserts that Dial returns by its context's deadline plus
// unblock, on the wall clock. A Dial that returns before the deadline,
// while the HEL is still unanswered, is a failure with t.Errorf - the
// library gave up before the context did - and so is a Dial not returned
// by the bound; a return after the deadline, within unblock of it, is
// the fixed arm. It deliberately asserts nothing about the error Dial
// returns: a fixed implementation may surface context.DeadlineExceeded
// or the read failure the expired connection deadline induces, and the
// contract fixes that the call returns, not the wording. It also asserts
// nothing about how long inside the context deadline the return lands -
// the connection deadline fires at it, and the margin is net's, not
// gopcua's.
//
// No reference stack has been read for this behaviour.
package transport

import "testing"

func TestHandshakeGivesUpAtItsContextDeadline(t *testing.T) {
	t.Skip("body not written yet")
}
