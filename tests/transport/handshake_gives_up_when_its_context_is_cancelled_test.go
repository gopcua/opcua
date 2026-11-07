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
// should stop." (go/src/context/context.go:15). gopcua does not have the
// behaviour: Handshake sets the connection's deadline only from
// ctx.Deadline() (uacp/conn.go:236-237), and with no deadline its
// Receive (uacp/conn.go:359) reads the connection directly and never
// selects on the context, so a cancellation after the HEL does nothing
// to the blocked read and Dial stays blocked until the peer closes.
//
// Neighbouring rows: handshake_gives_up_at_its_context_deadline_test.go
// is the same property on the deadline half of the same contract, and
// gopcua passes that one today, because the deadline path sets the
// connection deadline the cancellation path never does.
// dial_reaches_a_host_name_on_its_next_address_test.go scores which
// address of a name Dial dials; it shares no code path with this row
// beyond ResolveEndpoint.
//
// The row runs against the scripted silent peer - the shared fixture in
// harness.go - driven through uacp.Dialer.Dial directly: the row starts
// the peer, dials opc.tcp://<peer address> on its cancellable context
// through the shared dial helper, and awaits the HEL through the shared
// setup check, within unblock, before it cancels. The setup check is the
// shared awaitHello, and its decline and unscoreable split applies here
// as documented there. The row registers the peer's teardown with
// t.Cleanup, so a Dial still blocked on main is released when the test
// ends. See harness.go.
//
// The row asserts that Dial returns within unblock of the cancel. It
// deliberately asserts nothing about the error Dial returns: the Go
// contract fixes that the call returns, not the wording of the error,
// and a fixed implementation may report the cancellation directly or
// wrapped around the read failure the teardown close induces. It also
// asserts nothing about what the peer sees after the cancel - a fixed
// library may close the connection or leave it to the teardown.
//
// The knownBug constant is declared in the test function's body. The
// known-bug outcome is an absence - on main, Dial is still blocked at
// the bound, and there is nothing present to key an arm on. What makes
// the absence scoreable: the setup check proves the HEL reached the peer
// before the cancel, so a library that never reaches the stimulus - no
// connection, or no HEL on it - declines instead of landing in the
// known-bug arm. The arms, in words: a return stamped before the cancel
// is the early-return failure wherever the row reads it, and the fixed
// arm must not credit it; the fixed arm is a return stamped after the
// cancel and within unblock of it - each return is placed on the wall
// clock by the helper's stamp, not by when the row reads the channel -
// and it demands the constant's deletion. A Dial still blocked
// at the bound is the known bug: the row logs a `known bug:` line with
// t.Logf while the constant stands, and the build stays green; once the
// constant is deleted, the same outcome is scored with a plain t.Errorf.
// The arms read the dial helper's buffered channel, so a blocked Dial
// cannot wedge the test past the bound.
//
// No reference stack has been read for this behaviour.
package transport

import (
	"context"
	"testing"
	"time"
)

func TestHandshakeGivesUpWhenItsContextIsCancelled(t *testing.T) {
	// knownBug records that gopcua does not do this yet; the text is logged
	// in place of a failure. ../README.md says when to delete it. It carries
	// no issue link.
	const knownBug = "Handshake sets the connection's deadline only from " +
		"ctx.Deadline() (uacp/conn.go:236-237) and its Receive reads the " +
		"connection directly (uacp/conn.go:359), so a cancellation after " +
		"the HEL does nothing to the blocked read and Dial stays blocked"

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

	// Stamped before the cancel: a return the cancel itself causes is
	// stamped after this moment, so the early-return arm cannot mis-score it.
	cancelledAt := time.Now()
	cancel()

	var o dialOutcome
	returned := false
	select {
	case o = <-out:
		returned = true
	case <-time.After(unblock):
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
		if knownBug != "" {
			t.Logf("known bug: %s (Dial was still blocked %v after the cancel)",
				knownBug, unblock)
		} else {
			t.Errorf("the context was cancelled and Dial was still blocked "+
				"%v later", unblock)
		}
	case o.returned.Before(cancelledAt):
		// A return stamped before the cancel is an early return wherever
		// the row reads it, and the fixed arm must not credit it.
		t.Errorf("Dial returned (%v) before the context was cancelled - the "+
			"library gave up without being asked", o.err)
	case o.returned.Sub(cancelledAt) <= unblock && knownBug != "":
		t.Errorf("this now passes (Dial returned %v within %v of the cancel) - "+
			"delete the knownBug constant, every arm that reads it, and anything "+
			"only those arms used, in the commit that fixed it", o.err, unblock)
	case o.returned.Sub(cancelledAt) <= unblock:
		t.Logf("Dial returned %v within %v of the cancel", o.err, unblock)
	default:
		// A return stamped past the bound is a Dial that stayed blocked
		// past it: the stamp, not the read, places the return.
		if knownBug != "" {
			t.Logf("known bug: %s (Dial stayed blocked %v past the cancel, "+
				"returning only then)", knownBug, unblock)
		} else {
			t.Errorf("Dial stayed blocked %v past the cancel, returning only "+
				"then (%v)", o.returned.Sub(cancelledAt), o.err)
		}
	}
}
