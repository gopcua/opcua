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
// is the same property on the deadline half of the same contract - "A
// Context with a deadline is canceled after the deadline passes."
// (go/src/context/context.go:16) - and gopcua passes that one today,
// because the deadline path sets the connection deadline the
// cancellation path never does.
// dial_reaches_a_host_name_on_its_next_address_test.go scores which address of a name Dial dials; it shares
// no code path with this row beyond ResolveEndpoint.
//
// The row runs against the scripted silent peer - the shared fixture in
// harness.go - driven through uacp.Dialer.Dial directly: the row starts
// the peer, dials opc.tcp://<peer address> on its cancellable context on
// a goroutine it spawns, and awaits the HEL through the shared setup
// check before it cancels. The row registers the peer's teardown with
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
// The knownBug constant is written with the body. The known-bug outcome
// is an absence - on main, Dial is still blocked at the bound, and there
// is nothing present to key an arm on. What makes the absence scoreable:
// the setup check proves the HEL reached the peer before the cancel, so
// a library that never reaches the stimulus - no connection, or no HEL
// on it - declines instead of landing in the known-bug arm. The arms, in
// words: Dial returning within unblock of the cancel is the fixed arm
// and demands the constant's deletion; Dial still blocked at the bound
// logs the known bug. The arms read the dial goroutine's outcome off a
// channel, so a blocked Dial cannot wedge the test past the bound.
//
// No reference stack has been read for this behaviour.
package transport

import "testing"

func TestHandshakeGivesUpWhenItsContextIsCancelled(t *testing.T) {
	t.Skip("body not written yet")
}
