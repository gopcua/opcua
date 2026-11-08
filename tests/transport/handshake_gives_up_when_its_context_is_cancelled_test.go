//go:build conformance

// A caller whose context is cancelled is not left blocked in Dial: the
// row dials the silent peer with a context that has no deadline, cancels
// the context after the peer has received the HEL, and Dial returns.
//
// The clause is OPC 10000-6 §7.1.3, "The Client sends the
// OpenSecureChannel request once it receives the Acknowledge back from
// the Server." It is descriptive: it fixes the order - the client's HEL,
// the server's Acknowledge, the OpenSecureChannel after - so a peer that
// never answers has nothing further to send, and no sentence in the
// section says when or whether a client gives up waiting. No OPC UA
// clause binds this behaviour. What the row scores is the Go contract:
// "A Context may be canceled to indicate that work done on its behalf
// should stop." (go/src/context/context.go:15). gopcua passes this row:
// Handshake arms context.AfterFunc on the context to set a past deadline
// on the connection when it is cancelled (uacp/conn.go:235-238), which
// unblocks the read the HEL is waiting on, and Handshake then returns
// ctx.Err() (uacp/conn.go:243), so Dial returns.
//
// Neighbouring rows: handshake_gives_up_at_its_context_deadline_test.go
// is the same property on the deadline half of the same contract, and
// gopcua passes that one too.
// dial_reaches_a_host_name_on_its_next_address_test.go scores which
// address of a name Dial dials; it shares no code path with this row
// beyond Dial.
//
// The row runs against the scripted silent peer - the shared fixture in
// harness.go - driven through uacp.Dialer.Dial directly: the row starts
// the peer, dials opc.tcp://<peer address> on its cancellable context
// through the shared dial helper, and awaits the HEL through the shared
// setup check, within unblock, before it cancels. The setup check is the
// shared awaitHello, and its decline and unscoreable split applies here
// as documented there. The row registers the peer's teardown with
// t.Cleanup, so a Dial left blocked is released when the test ends.
// See harness.go.
//
// The row asserts that Dial returns within unblock of the cancel. It
// deliberately asserts nothing about the error Dial returns: the Go
// contract fixes that the call returns, not the wording of the error,
// and a fixed implementation may report the cancellation directly or
// wrapped around the read failure the teardown close induces. It also
// asserts nothing about what the peer sees after the cancel - a fixed
// library may close the connection or leave it to the teardown.
//
// The setup check proves the HEL reached the peer before the cancel, so
// a library that never reaches the stimulus - no connection, or no HEL
// on it - declines instead of being scored. The arms, in words: a
// return stamped before the cancel is the early-return failure wherever
// the row reads the return, and the fixed arm must not credit it; the
// fixed arm is a return stamped after the cancel and within unblock of
// it - each return is placed on the wall clock by the helper's stamp,
// not by when the row reads the channel. A Dial still blocked at the
// bound is scored with a plain t.Errorf. The arms read the dial
// helper's buffered channel, so a blocked Dial cannot wedge the test
// past the bound.
//
// No reference stack has been read for this behaviour.
package transport

import (
	"context"
	"testing"
	"time"
)

func TestHandshakeGivesUpWhenItsContextIsCancelled(t *testing.T) {
	peer, teardown, why := startSilentPeer("127.0.0.1")
	t.Cleanup(teardown)
	if why != "" {
		t.Fatalf("could not set the test up: %s", why)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	out := dial(ctx, "opc.tcp://"+peer.addr())

	if why, fromGopcua := peer.awaitHello(unblock); why != "" {
		if fromGopcua {
			t.Errorf("gopcua's own behaviour made this run unscoreable: %s", why)
			return
		}
		t.Fatalf("could not set the test up: %s", why)
	}

	// cancelledAt is stamped before the cancel: a return the cancel
	// itself causes is stamped after this moment, so the early-return
	// arm cannot mis-score it.
	cancelledAt := time.Now()
	cancel()

	// awaitOutcome bounds the wait by unblock and takes the one more
	// non-blocking read documented on the helper before it reports the
	// window as ended with nothing on the channel.
	o, returned := awaitOutcome(out, unblock)

	switch {
	case !returned:
		t.Errorf("the context was cancelled and Dial was still blocked "+
			"%v later", unblock)
	case o.returned.Before(cancelledAt):
		// A return stamped before the cancel is an early return wherever
		// the row reads it, and the fixed arm must not credit it.
		t.Errorf("Dial returned (%v) before the context was cancelled - the "+
			"library gave up without being asked", o.err)
	case o.returned.Sub(cancelledAt) <= unblock:
		t.Logf("Dial returned %v within %v of the cancel", o.err, unblock)
	default:
		// A return stamped past the bound is a Dial that stayed blocked
		// past it: the stamp, not the read, places the return.
		t.Errorf("Dial stayed blocked %v past the cancel, returning only "+
			"then (%v)", o.returned.Sub(cancelledAt), o.err)
	}
}
