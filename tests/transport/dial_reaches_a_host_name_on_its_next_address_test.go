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
// Neighbouring rows:
// handshake_gives_up_when_its_context_is_cancelled_test.go and
// handshake_gives_up_at_its_context_deadline_test.go score when Dial
// gives up on a peer that has the connection; this row scores which
// address Dial reaches a peer on, before any handshake is held open.
// They share the silent peer, not the HEL check - this row reads the
// peer's state through its own logic - and no code path in the library
// beyond ResolveEndpoint.
//
// The row runs against the silent peer from harness.go, bound by the
// row's own setup to the last address localhost resolves to: the row
// resolves the name with net.Resolver.LookupIPAddr on a zero-valued
// net.Resolver, as ResolveEndpoint does - the same call, at
// uacp/endpoint.go:40, on the zero-valued net.Resolver declared at
// uacp/endpoint.go:38 - so the row and the library agree on the order -
// listens on the last address on a free port, and dials
// opc.tcp://localhost:<the peer's own port, from the peer's port
// helper>. See harness.go.
// awaitHello is the wrong helper for this row: its "accepted no
// connection" reason is the known-bug outcome here, not a decline, so
// the row reads the peer's state through its own logic.
//
// The row declines, and does not score, when the environment takes the
// behaviour out of reach: when localhost resolves to a single address,
// there is no next address and the behaviour cannot be constructed; and
// when the row's probe of the first address at the peer's port - from
// the peer's port helper again, so probe and dial cannot disagree - its
// setup check, a dial bounded by unblock - does not see that address
// refuse:
// the probe requires refused(err), the build-tagged helper in
// refused_windows.go and refused_other.go (errors.Is against Winsock's
// WSAECONNREFUSED on Windows, against syscall.ECONNREFUSED elsewhere).
// A probe that
// connects - the row closes the connection it opened before it declines
// - or that fails any other way - a timeout, a drop, a network
// unreachable - means the premise "the first address refuses" does not
// hold, so the row cannot pin a missing HEL on the library. All are
// declines, because a red the row cannot pin on the library is not a
// score. A failed lookup of localhost is a decline, for the same
// reason: the row can then neither probe nor dial the name's addresses.
//
// The row dials through the shared dial helper with a context whose
// deadline is 2×unblock - long enough that it never ends inside the
// scored window, while the teardown still releases a fixed library's
// blocked Dial - because on a fixed library Dial never returns: the
// silent peer never sends an Acknowledge, so a library that dials the
// right address still blocks in Handshake after sending its HEL. The row
// scores only whether a HEL arrives at the listener on the last address
// within unblock of the dial. It deliberately asserts nothing about
// Dial's return value, and nothing about the connection's state, which
// would score the uasc layer above, not the transport.
//
// The knownBug constant is declared in the test function's body. The
// known-bug outcome is a presence: Dial returned an error and no HEL
// arrived at the listener. That error is the presence: on main,
// ResolveEndpoint keeps addrs[0] (uacp/endpoint.go:50) and the dial to
// it is refused, so Dial returns an error and no HEL ever reaches the
// last-address listener.
// The arms, in words: a HEL seen at the last-address listener at any
// point before the verdict is the fixed arm, whatever Dial returned, and
// it demands the constant's deletion; before scoring any arm other than
// the fixed one, the row re-checks the peer's HEL signal without
// blocking, so a HEL that raced a Dial error is still the fixed arm;
// Dial returning an error with no
// HEL at the listener logs the known bug; Dial returning with a nil
// error and no HEL arrived is an ordinary failure with t.Errorf, because
// the silent peer never acknowledges, so such a return cannot come from
// a dial that reached it. When no HEL arrives within unblock and Dial
// has not returned, that is an ordinary failure too, scored with
// t.Errorf, and the failure message says so.
//
// No reference stack has been read for this behaviour.
package transport

import (
	"context"
	"net"
	"testing"
	"time"
)

// port is the port the address row's peer listens on, split from its
// listener's address, so the row's probe of the first address and its
// localhost dial use the peer's own port without re-deriving it from
// addr. It is a helper one row calls, so it lives with that row: the
// placement rule in ../README.md moved it here from harness.go when this
// row's body landed and called it.
func (p *silentPeer) port() string {
	_, port, _ := net.SplitHostPort(p.listener.Addr().String())
	return port
}

func TestDialReachesAHostNameOnItsNextAddress(t *testing.T) {
	// knownBug records that gopcua does not do this yet; the text is logged
	// in place of a failure. ../README.md says when to delete it. It carries
	// no issue link.
	const knownBug = "ResolveEndpoint keeps only addrs[0] (uacp/endpoint.go:50) " +
		"and Dialer.Dial dials that one address (uacp/conn.go:83), so a host " +
		"name whose first address refuses is never tried on its next one"

	// The row resolves the name itself, on a zero-valued net.Resolver, as
	// ResolveEndpoint does (uacp/endpoint.go:40), so the row and the
	// library agree on the order of the addresses.
	var resolver net.Resolver
	addrs, err := resolver.LookupIPAddr(context.Background(), "localhost")
	if err != nil {
		t.Fatalf("could not set the test up: resolving localhost: %v", err)
	}
	if len(addrs) < 2 {
		t.Fatalf("could not set the test up: localhost resolved to %d address(es) "+
			"(%v); there is no next address, so the behaviour cannot be constructed",
			len(addrs), addrs)
	}

	// The peer goes on the last address: a HEL there is evidence Dial
	// tried the addresses in order and passed the first one's refusal.
	peer, teardown, why := startSilentPeer(addrs[len(addrs)-1].String())
	t.Cleanup(teardown)
	if why != "" {
		t.Fatalf("could not set the test up: %s", why)
	}

	// The probe dials the first address at the peer's own port - from the
	// peer's port helper, so probe and dial cannot disagree - bounded by
	// unblock, and must see it refuse. Anything else takes the premise
	// "the first address refuses" out of the row's reach.
	c, err := net.DialTimeout("tcp",
		net.JoinHostPort(addrs[0].String(), peer.port()), unblock)
	if err == nil {
		_ = c.Close()
		t.Fatalf("could not set the test up: a connection to the first address "+
			"%s at the peer's port succeeded, so a HEL at the last address would "+
			"not be evidence about the first address's refusal", addrs[0])
	}
	if !refused(err) {
		t.Fatalf("could not set the test up: the first address %s at the "+
			"peer's port failed with %v, not a refusal", addrs[0], err)
	}

	// On a fixed library Dial never returns - the silent peer never sends
	// an Acknowledge, so a library that dials the right address still
	// blocks in Handshake after sending its HEL - so the context's
	// deadline is only a bound past the scored window, for the teardown
	// to release.
	ctx, cancel := context.WithTimeout(context.Background(), 2*unblock)
	defer cancel()

	out := dial(ctx, "opc.tcp://localhost:"+peer.port())

	var o dialOutcome
	returned := false
	hello := false
	select {
	case <-peer.hello:
		hello = true
	case o = <-out:
		returned = true
	case <-time.After(unblock):
	}
	if !hello {
		// One more non-blocking re-check before any arm other than the
		// fixed one scores: a HEL that raced the Dial return or the timer
		// is still the fixed arm, whatever Dial returned.
		select {
		case <-peer.hello:
			hello = true
		default:
		}
	}

	switch {
	case hello:
		if knownBug != "" {
			t.Errorf("this now passes (a HEL arrived at the listener on %s) - "+
				"delete the knownBug constant, every arm that reads it, and anything "+
				"only those arms used, in the commit that fixed it", peer.addr())
		} else {
			t.Logf("a HEL arrived at the listener on %s", peer.addr())
		}
	case !returned:
		t.Errorf("no HEL arrived at the listener on %s and Dial was still "+
			"blocked %v after the dial", peer.addr(), unblock)
	case o.err != nil:
		if knownBug != "" {
			t.Logf("known bug: %s (Dial returned %v and no HEL arrived at the "+
				"listener on %s)", knownBug, o.err, peer.addr())
		} else {
			t.Errorf("Dial returned %v and no HEL arrived at the listener on %s",
				o.err, peer.addr())
		}
	default:
		t.Errorf("Dial returned a nil error (%v) and no HEL arrived at the "+
			"listener on %s - the silent peer never acknowledges, so such a "+
			"return cannot come from a dial that reached it", o.err, peer.addr())
	}
}
