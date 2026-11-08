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
// gopcua passes this row: Dialer.Dial hands the endpoint's host name to
// net.Dialer (uacp/conn.go:87), which tries each address in order, so a
// host name whose first address refuses is still tried on its next one.
//
// Neighbouring rows:
// handshake_gives_up_when_its_context_is_cancelled_test.go and
// handshake_gives_up_at_its_context_deadline_test.go score when Dial
// gives up on a peer that has the connection; this row scores which
// address Dial reaches a peer on, before any handshake is held open.
// They share the silent peer, not the HEL check - this row reads the
// peer's state through its own logic - and, in the library, nothing
// beyond Dial.
//
// The row runs against the silent peer from harness.go, bound by the
// row's own setup to the last address localhost resolves to. It
// resolves the name with net.Resolver.LookupIPAddr, the resolution
// net.Dialer makes inside the dial, so the row and the library agree
// on the order. It listens on the last address on a free
// port and dials opc.tcp://localhost:<the peer's own port, from this
// row's port helper>. See harness.go.
// awaitHello is the wrong helper for this row: its "accepted no
// connection" reason is an ordinary failure here, not a decline, so
// the row reads the peer's state through its own logic.
//
// The row declines, and does not score, when the environment takes the
// behaviour out of reach: when localhost resolves to a single address,
// there is no next address and the behaviour cannot be constructed; when
// the peer cannot listen on the last address; and when the row's setup
// check - a probe of the first address at the peer's port, from the
// peer's port helper again so probe and dial cannot disagree, bounded by
// unblock - does not see that address refuse:
// the probe requires refused(err), the build-tagged helper in
// refused_windows.go and refused_other.go. A probe that connects - the
// row closes the connection it opened before it declines - or that fails
// any other way - a timeout, a drop, a network unreachable - means the
// premise "the first address refuses" does not hold, so the row cannot
// pin a missing HEL on the library. All are declines, because a red
// the row cannot pin on the library is not a score. A failed lookup of
// localhost is a decline, for the same reason: the row can then neither
// probe nor dial the name's addresses.
//
// The row dials through the shared dial helper with a context whose
// deadline is 2×unblock - long enough that it never ends inside the
// scored window, while the teardown still releases a fixed library's
// blocked Dial - because on a fixed library Dial never returns: the
// silent peer never sends an Acknowledge, so a library that dials the
// right address still blocks in Handshake after sending its HEL. The row
// scores only whether a HEL arrives at the listener on the last address
// within unblock of the dial. It deliberately asserts nothing about the
// connection Dial returns, and nothing about the connection's state,
// which would score the uasc layer above, not the transport.
//
// The arms, in words: a HEL seen at the last-address listener at any
// point before the verdict is the fixed arm, whatever Dial returned;
// before scoring any arm other than the fixed one, the row re-checks
// the peer's HEL signal without blocking, so a HEL that raced a Dial
// error is still the fixed arm; any Dial return with no HEL arrived -
// an error, or a nil error - is an ordinary failure with t.Errorf: the
// silent peer never acknowledges, so a nil return cannot come from a
// dial that reached it. When no HEL arrives within unblock and Dial
// has not returned, that is an ordinary failure too, scored with
// t.Errorf: the message says still blocked when the peer read no
// header, and says what the peer did read when a connection reached the
// listener but did not send a HEL.
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
// addr. It is a helper one row calls, so it lives with that row
// (../README.md's placement rule).
func (p *silentPeer) port() string {
	_, port, _ := net.SplitHostPort(p.listener.Addr().String())
	return port
}

// readHeader returns the message header the peer read off the connection
// it accepted, or "" if it has read none yet. It is a helper one row
// calls, so it lives with that row (../README.md's placement rule).
func (p *silentPeer) readHeader() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.header
}

func TestDialReachesAHostNameOnItsNextAddress(t *testing.T) {
	// The row resolves the name itself, on a zero-valued net.Resolver, as
	// net.Dialer does inside the dial, so the row and the library agree
	// on the order of the addresses.
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
	// reached the last address after the first one refused. It is not
	// evidence of in-order iteration: a Dialer that tried only the last
	// address, or raced them, lands in the same arm, and the row cannot
	// tell these apart with the two addresses localhost gives it.
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
		// One more non-blocking read before scoring "still blocked": a
		// Dial return can land with the timer.
		select {
		case o = <-out:
			returned = true
		default:
		}
	}
	// One more non-blocking re-check before any arm other than the fixed
	// one scores: a HEL that raced the Dial return or the timer is still
	// the fixed arm, whatever Dial returned. A header the peer has already
	// recorded as HELF counts too, whether or not it has closed p.hello
	// yet - the shape awaitHello scores by.
	select {
	case <-peer.hello:
		hello = true
	default:
	}
	if peer.readHeader() == "HELF" {
		hello = true
	}

	switch {
	case hello:
		t.Logf("a HEL arrived at the listener on %s", peer.addr())
	case !returned:
		// A library that reached the right listener but sent a header that
		// is not HELF leaves its header behind; the peer read it, so the
		// row says reached rather than still blocked.
		if hdr := peer.readHeader(); hdr != "" {
			t.Errorf("a connection reached the listener on %s but its first "+
				"message header was %q, not HELF", peer.addr(), hdr)
		} else {
			t.Errorf("no HEL arrived at the listener on %s and Dial was still "+
				"blocked %v after the dial", peer.addr(), unblock)
		}
	case o.err != nil:
		t.Errorf("Dial returned %v and no HEL arrived at the listener on %s",
			o.err, peer.addr())
	default:
		t.Errorf("Dial returned a nil error and no HEL arrived at the "+
			"listener on %s - the silent peer never acknowledges, so such a "+
			"return cannot come from a dial that reached it", peer.addr())
	}
}
