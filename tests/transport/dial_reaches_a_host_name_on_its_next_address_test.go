//go:build conformance

// A host name whose first address refuses is still reached on its next
// one: the endpoint names localhost, the row resolves the name itself,
// listens only on the last address it resolves to, and dials
// opc.tcp://localhost:<port>; a HEL arrives at that listener.
//
// The clause is OPC 10000-6 §7.1.3, "If the EndpointUrl specifies a
// domain name then the initiator requires access to a domain name
// resolution service" - it says a client may be handed a name, and says
// nothing about what a client that is handed one may do with the several
// addresses the name resolves to. No OPC UA clause binds this behaviour.
// What the row scores is net.Dialer's documented behaviour, which is
// what the default dialer inside uacp.Dialer is: "When using TCP, and
// the host resolves to multiple IP addresses, Dial will try each IP
// address in order until one succeeds." (go/src/net/dial.go:438-439).
// gopcua does not have the behaviour: ResolveEndpoint resolves the name
// and keeps only addrs[0] (uacp/endpoint.go:40 and :50), and Dialer.Dial
// dials that one address (uacp/conn.go:83), so a host name whose first
// address refuses is never tried on its next one.
//
// Neighbouring rows: handshake_gives_up_when_its_context_is_cancelled_
// test.go and handshake_gives_up_at_its_context_deadline_test.go score
// when Dial gives up on a peer that has the connection; this row scores
// which address Dial reaches a peer on, before any handshake is held
// open. They share the silent peer and the HEL check, not a code path in
// the library beyond ResolveEndpoint.
//
// The row runs against the silent peer from harness.go, bound by the
// row's own setup to the last address localhost resolves to: the row
// resolves the name with net.Resolver.LookupIPAddr - the same call
// ResolveEndpoint makes (uacp/endpoint.go:40), so the row and the
// library agree on the order - listens on the last address on a free
// port, and dials opc.tcp://localhost:<that port>. See harness.go.
//
// The row declines, and does not score, when the environment takes the
// behaviour out of reach: when localhost resolves to a single address,
// there is no next address and the behaviour cannot be constructed; and
// when anything accepts a connection on the first address at that port -
// the row dials the first address itself as its setup check - a HEL at
// the last address would not be evidence about the first address's
// refusal. Both are declines, because a red the row cannot pin on the
// library is not a score.
//
// The row asserts that a HEL arrives at the listener on the last address
// within unblock of the dial. It deliberately asserts nothing about what
// Dial returns when the behaviour is fixed - a fixed library returns a
// live connection, and asserting more about the connection's state would
// score the uasc layer above, not the transport.
//
// The knownBug constant is written with the body. The known-bug outcome
// is a presence: Dial returned an error and no HEL arrived at the
// listener. The arms, in words: a HEL arriving at the listener is the
// fixed arm and demands the constant's deletion; Dial returning an error
// with no HEL at the listener logs the known bug. A Dial still blocked
// at the bound is neither arm, and the row scores it as a failure - the
// proxy the known-bug arm keys on is gone and the outcome is still
// wrong.
//
// No reference stack has been read for this behaviour.
package transport

import "testing"

func TestDialReachesAHostNameOnItsNextAddress(t *testing.T) {
	t.Skip("body not written yet")
}
