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
// made to the cancellation path: a change that fixed the cancellation
// while breaking the deadline would pass
// handshake_gives_up_when_its_context_is_cancelled_test.go and fail
// this one.
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
// is one second, through the shared dial helper, and awaits the HEL
// through the shared setup check, passing it time.Until(deadline) and
// not unblock, so the HEL must arrive by the deadline itself: if it has
// not, the run cannot construct "the deadline expires while the HEL is
// held", and the row declines. The deadline is the row's own, so the
// peer's silence outlives it and the expiry lands while the HEL is held
// unanswered. The setup check is the shared awaitHello, and its decline
// and unscoreable split applies here as documented there. The row registers
// the peer's teardown with t.Cleanup, so a Dial still blocked on main is
// released when the test ends. See harness.go.
//
// The row asserts that Dial returns within unblock after its context's
// deadline: the verdict window ends at deadline + unblock on the wall
// clock. Outcomes come off the dial helper's buffered channel, and each
// return is placed on the wall clock by the helper's stamp, not by when
// the row reads the channel, so the row compares the stamp with the
// deadline: a return stamped before the deadline, while the HEL is still
// unanswered, is a failure with t.Errorf - the library gave up before
// the context did - after which the row returns; so is a Dial not
// returned by the bound; a return stamped after the deadline, within
// unblock of it, is the fixed arm. It deliberately asserts nothing
// about the error Dial
// returns: a fixed implementation may surface context.DeadlineExceeded
// or the read failure the expired connection deadline induces, and the
// contract fixes that the call returns, not the wording. It also asserts
// nothing about how long inside the context deadline the return lands -
// the connection deadline fires at it, and the margin is net's, not
// gopcua's.
//
// No reference stack has been read for this behaviour.
package transport

import (
	"context"
	"testing"
	"time"
)

func TestHandshakeGivesUpAtItsContextDeadline(t *testing.T) {
	// The deadline is the row's own and is set at creation: the verdict
	// compares every return against this moment on the wall clock, so the
	// row needs no stamp beside it the way the cancellation row needs
	// cancelledAt beside its cancel.
	deadline := time.Now().Add(time.Second)
	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()

	peer, teardown, why := startSilentPeer("127.0.0.1")
	t.Cleanup(teardown)
	if why != "" {
		t.Fatalf("could not set the test up: %s", why)
	}

	out := dial(ctx, "opc.tcp://"+peer.addr())

	// The HEL must arrive by the deadline itself: if it has not, the run
	// cannot construct "the deadline expires while the HEL is held", and
	// the row declines.
	if why, fromGopcua := peer.awaitHello(time.Until(deadline)); why != "" {
		if fromGopcua {
			t.Errorf("gopcua's own behaviour made this run unscoreable: %s", why)
			return
		}
		t.Fatalf("could not set the test up: %s", why)
	}

	// bound is where the verdict window ends on the wall clock: unblock
	// after the context's deadline, not unblock after this line.
	bound := deadline.Add(unblock)

	var o dialOutcome
	returned := false
	select {
	case o = <-out:
		returned = true
	case <-time.After(time.Until(bound)):
		// One more non-blocking read before scoring "still blocked": the
		// return and the timer can land together, and the helper's stamp,
		// not this read, places the return on the wall clock.
		select {
		case o = <-out:
			returned = true
		default:
		}
	}

	switch {
	case !returned:
		t.Errorf("the context's deadline expired and Dial was still blocked "+
			"%v after it", unblock)
	case o.returned.Before(deadline):
		// A return stamped before the deadline, while the HEL is still
		// unanswered, is an early return wherever the row reads it.
		t.Errorf("Dial returned (%v) before the context's deadline expired - "+
			"the library gave up before the context did", o.err)
	case o.returned.Sub(deadline) <= unblock:
		t.Logf("Dial returned %v within %v of the deadline", o.err, unblock)
	default:
		// A return stamped past the bound is a Dial that stayed blocked
		// past it: the stamp, not the read, places the return.
		t.Errorf("Dial stayed blocked %v past the deadline, returning only "+
			"then (%v)", o.returned.Sub(deadline), o.err)
	}
}
