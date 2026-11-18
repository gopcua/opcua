//go:build conformance

// Package transport holds the gopcua transport tests: one behaviour per
// file, with the clause it checks in the file's opening comment. This file
// holds the fixture they all run against; it is not a row.
//
// The fixture is a scripted silent TCP peer, built on net and not on
// gopcua's uacp server: it accepts one connection, reads and records that
// a HEL message header arrived, and never answers. The rows score what
// uacp.Dialer.Dial does while the peer withholds its Acknowledge, and the
// silence is the stimulus; gopcua's uacp server would answer, so a
// scripted peer is what the behaviours need.
//
// Every helper here returns plain values. A helper does not call t.Errorf
// or t.Fatalf; a row's test function turns a non-empty reason string into
// the failure or the decline, and it, not the fixture, registers the
// teardown with t.Cleanup. The peer's listen on a free port, its one
// accepted connection, its teardown, and its address are the core it
// shares with the securitypolicy package's peer; they live in
// tests/internal/scriptedpeer, and the script the peer plays once a
// connection is accepted is this package's own, below.
//
// The package name is transport, as the directory is; the production code
// it drives lives in github.com/gopcua/opcua/uacp.
package transport

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/gopcua/opcua/tests/internal/scriptedpeer"
	"github.com/gopcua/opcua/uacp"
)

// unblock is how long a row waits for Dial to return once its stimulus
// has fired - the cancel or the deadline - and how long a setup check
// waits for the HEL to reach the peer. README.md's "A wall clock scores
// a blocked dial" bullet says why the bound is not a timing assumption.
const unblock = 5 * time.Second

// dialOutcome is what dial delivers on its channel: err is the error,
// and returned the wall-clock time stamped immediately after Dial
// returned.
type dialOutcome struct {
	err      error
	returned time.Time
}

// dial runs uacp.Dialer.Dial - with the zero-valued &uacp.Dialer{}, whose
// fields are nil, so the library's defaults stand - on its own goroutine
// and delivers the call's outcome on a one-buffered channel, so the
// goroutine never blocks on a row that has stopped reading. A row scores
// a return by the helper's stamp wherever it reads it, so a return
// stamped before the cancel or the deadline is the early-return failure
// even when the row reads it later. When a handshake row's window ends
// with nothing on the channel, the two handshake rows' shared
// awaitOutcome helper takes one more non-blocking read before the row
// scores "still blocked", and the row scores what it finds by the stamp.
// A *uacp.Conn the library did
// return is closed in this goroutine, so it leaks nothing. Like every
// helper here it returns plain values; the row turns the outcome it
// reads into the failure or the decline.
func dial(ctx context.Context, endpoint string) <-chan dialOutcome {
	out := make(chan dialOutcome, 1)
	go func() {
		conn, err := (&uacp.Dialer{}).Dial(ctx, endpoint)
		o := dialOutcome{err: err, returned: time.Now()}
		if conn != nil {
			_ = conn.Close()
		}
		out <- o
	}()
	return out
}

// silentPeer is the scripted silent peer. It accepts one connection,
// reads the message header off it, and never writes anything.
type silentPeer struct {
	core     *scriptedpeer.Core
	listener net.Listener // the core's listener, named here for the address row's port helper, which lives with its row

	mu     sync.Mutex
	header string // the first four bytes read off the connection, once read

	accepted chan struct{} // closed once a connection has been accepted
	hello    chan struct{} // closed once a HEL message header has been read
}

// startSilentPeer brings the silent peer up on the host it is given, on
// the shared core's listener (tests/internal/scriptedpeer). Rows that
// need the peer on the loopback interface pass 127.0.0.1; the address
// row passes the last address localhost resolves to, which may be ::1
// or 127.0.0.1, so an IPv6 host must come out bracketed - the shared
// core's Listen joins the host with the port. It returns the peer, a
// teardown for the listener and the accepted connection, and "" - or,
// when the peer could not be started, the reason the run cannot be
// scored.
func startSilentPeer(host string) (*silentPeer, func(), string) {
	l, err := scriptedpeer.Listen(host)
	if err != nil {
		return nil, func() {}, fmt.Sprintf("listen on %s: %v", host, err)
	}
	p := &silentPeer{
		core:     scriptedpeer.New(l),
		listener: l,
		accepted: make(chan struct{}),
		hello:    make(chan struct{}),
	}
	go p.serve()
	return p, p.core.Teardown, ""
}

// addr is the host:port the peer listens on; the shared core holds it.
func (p *silentPeer) addr() string {
	return p.core.Addr()
}

// serve accepts the peer's one connection - through the shared core's
// Accept, which owns the accept and the teardown race - records its
// first message header, and then discards everything else and never
// answers, until the connection or the listener is closed.
func (p *silentPeer) serve() {
	c := p.core.Accept()
	if c == nil {
		return // the listener was closed in teardown, or teardown won the race
	}
	close(p.accepted)

	var hdr [4]byte
	if _, err := io.ReadFull(c, hdr[:]); err != nil {
		return // the peer went away before a whole header arrived
	}
	p.mu.Lock()
	p.header = string(hdr[:])
	isHello := p.header == "HELF"
	p.mu.Unlock()
	if isHello {
		close(p.hello)
	}

	_, _ = io.Copy(io.Discard, c) // hold the connection open, never answer
}

// awaitOutcome is the verdict-window wait the two handshake rows share:
// it waits up to `within` for the outcome dial delivered, and returns it
// with true. Once the bound passes it takes one more non-blocking read -
// the return and the timer can land together, and dial's stamp, not this
// read, places the return on the wall clock - and returns a zero
// dialOutcome with false when there is nothing there. It is a helper two
// rows call, so it lives here.
func awaitOutcome(
	out <-chan dialOutcome, within time.Duration,
) (dialOutcome, bool) {
	var o dialOutcome
	select {
	case o = <-out:
		return o, true
	case <-time.After(within):
		select {
		case o = <-out:
			return o, true
		default:
			return dialOutcome{}, false
		}
	}
}

// awaitHello is the setup check the handshake rows share: it waits up to
// `within` for the peer to read a HEL message header off the connection
// Dial opened, which proves the stimulus - a Dial blocked in Receive
// (uacp/conn.go:267) - has been reached before the row cancels its
// context or its deadline passes. It returns "", false once the HEL has
// been read. A HEL that lands with the timer is caught by the
// fallback's re-read of the peer's recorded header, not blamed on
// either side. Otherwise it returns a reason and which side caused the
// failed check, and the row prints the reason with the wording below:
//
//   - fromGopcua false, a decline: a failed check either side could have
//     caused - the peer accepted no connection within `within`, so Dial
//     never reached it; or it accepted the connection but read no header
//     within `within`, which can close on a loaded machine. The row prints
//     "could not set the test up: %s".
//
//   - fromGopcua true, unscoreable: a failed check only the library
//     could have caused - the peer read a message header that is not
//     HELF, which is determinately gopcua's bytes. The row prints
//     "gopcua's own behaviour made this run unscoreable: %s" with
//     t.Errorf, then returns.
//
// dial_reaches_a_host_name_on_its_next_address_test.go does not call
// this helper: for that row "accepted no connection" is an ordinary
// failure, not a decline, and it reads the peer's state through its own
// logic.
func (p *silentPeer) awaitHello(within time.Duration) (string, bool) {
	select {
	case <-p.hello:
		return "", false
	case <-time.After(within):
	}

	select {
	case <-p.accepted:
	default:
		return fmt.Sprintf("the peer accepted no connection within %v, "+
			"so Dial never reached it", within), false
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.header == "HELF" {
		// The header says the HEL arrived, whether or not the peer has
		// closed p.hello yet: the check passed.
		return "", false
	}
	if p.header == "" {
		return fmt.Sprintf("the peer accepted the connection but read no "+
			"message header within %v", within), false
	}
	return fmt.Sprintf("the peer read message header %q, not HELF", p.header), true
}
