//go:build conformance

// Package securitypolicy holds the gopcua security policy tests: one
// behaviour per file, with the clause it checks in the file's opening
// comment. This file holds the fixture they all run against; it is not
// a row.
//
// The fixture is a scripted TCP peer, built on net and not on gopcua's
// uacp server: it accepts one connection, reads the client's HEL, answers
// the ACK, reads the next frame - the OpenSecureChannel request the rows
// score - records its bytes, and closes the connection without ever
// answering it. The rows score what the opcua.Client's chosen
// SecurityPolicy hands the secure channel inside that frame. The peer
// uses gopcua's uacp types to frame the ACK and to read whole messages
// off the wire, which is not the library grading itself: the bytes the
// rows score are decrypted and counted by the standard library, and the
// expected numbers come from the clauses, never from gopcua.
//
// Two outcomes of the exchange are worth naming here, because the
// setup check reports both. Bytes the peer's uacp framing rejects - a
// bad header, an out-of-range size, an ERR frame - can only be the
// client's own, so awaitOpenSecureChannel reports them as gopcua's
// behaviour, not as a decline. And an exchange that ends before the
// OPN - the client closing, going away - is a
// decline the check reports at once, when serve returns, without
// waiting out its bound.
//
// Every helper here returns plain values. A helper does not call
// t.Errorf or t.Fatalf; a row's test function turns a non-empty reason
// string into the failure or the decline, and it, not the fixture,
// registers the teardown with t.Cleanup. The teardown closes the
// listener and the accepted connection, so a Connect left blocked by
// the library under test is released when the test ends.
//
// The package name is securitypolicy, as the directory is; the
// production code it drives lives in github.com/gopcua/opcua and its
// uasc and uapolicy packages.
package securitypolicy

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"syscall"
	"time"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/uacp"
)

// handshake bounds the opening exchange: how long the peer waits for the
// client's frames, and the deadline on the context launchClient gives
// Connect. The bound is not a timing assumption: a correct client sends
// its HEL and its OPN in milliseconds, so the bound exists to let a slow
// machine finish, not to wait out a broken one; the teardown releases a
// client that never sends, and the context's deadline releases a Connect
// that stays blocked. README.md states this bound's purpose the way
// ../transport/README.md states `unblock`'s.
const handshake = 10 * time.Second

// peerAck is the Acknowledge the scripted peer answers the HEL with:
// protocol version 0, 65535-byte buffers in both directions, and no
// message-size or chunk-count limit.
var peerAck = &uacp.Acknowledge{
	Version:        0,
	ReceiveBufSize: 65535,
	SendBufSize:    65535,
}

// scriptedPeer is the scripted TCP peer: it accepts one connection, reads
// the HEL, answers the ACK, records the next frame - the OPN - and
// closes the connection without answering it. The recorded frame is the
// whole secure-conversation chunk as the client wrote it, including its
// first eight bytes, so uasc.MessageChunk.Decode takes it whole, with
// no slicing.
type scriptedPeer struct {
	listener net.Listener

	mu                sync.Mutex
	conn              net.Conn
	closed            bool   // set by teardown; serve closes the connection itself when set
	offScript         string // the message type of a frame that arrived where the script expected another
	offScriptExpected string // the message type the script expected where that frame arrived
	recvErr           string // a read error other than a connection that merely ended: the client's bytes did not frame

	opn  chan []byte   // the recorded OPN frame, delivered once
	done chan struct{} // closed when serve returns, so awaitOpenSecureChannel need not wait out its bound
}

// startScriptedPeer brings the peer up on the loopback interface, on a
// free port the kernel chooses - never port 4840, which other suites
// bind. It returns the peer, a teardown for the listener and the
// accepted connection, and "" - or, when the peer could not be started,
// the reason the run cannot be scored.
func startScriptedPeer() (*scriptedPeer, func(), string) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, func() {}, fmt.Sprintf("listen on the loopback interface: %v", err)
	}
	p := &scriptedPeer{
		listener: l,
		opn:      make(chan []byte, 1),
		done:     make(chan struct{}),
	}
	go func() {
		p.serve()
		close(p.done)
	}()
	teardown := func() {
		_ = l.Close()
		p.mu.Lock()
		defer p.mu.Unlock()
		p.closed = true
		if p.conn != nil {
			_ = p.conn.Close()
		}
	}
	return p, teardown, ""
}

// addr is the host:port the peer listens on.
func (p *scriptedPeer) addr() string {
	return p.listener.Addr().String()
}

// serve plays the script: accept one connection, read the HEL, answer the
// ACK, read the OPN, deliver it on the channel, and close. A frame that
// arrives where the script expected another - the HEL at the first read,
// the OPN at the second - is recorded in offScript together with the
// type the script expected at that point: it is determinately the
// client's bytes, and awaitOpenSecureChannel reports it as gopcua's own
// behaviour. A read error other than a connection that merely ended - a
// bad header, an out-of-range size, an ERR frame - is recorded by
// noteReceiveError and reported the same way, for the same reason. Any
// other failure - no connection, a client that goes away mid-exchange -
// leaves the channel empty, and the check declines. If teardown runs
// between the Accept and this goroutine taking the mutex, teardown
// never saw the connection, so serve closes it here.
func (p *scriptedPeer) serve() {
	c, err := p.listener.Accept()
	if err != nil {
		return // the listener was closed in teardown
	}
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		_ = c.Close()
		return
	}
	p.conn = c
	p.mu.Unlock()

	uc, err := uacp.NewConn(c.(*net.TCPConn), peerAck)
	if err != nil {
		return // the peer cannot frame the connection; the check declines
	}

	hel, err := uc.Receive()
	if err != nil {
		// A connection that merely ended stays a decline; anything else
		// can only be the client's bytes, so it is recorded.
		p.noteReceiveError(err)
		return
	}
	if string(hel[:4]) != "HELF" {
		p.noteOffScript(hel[:4], []byte("HELF"))
		return
	}

	if err := uc.Send("ACKF", peerAck); err != nil {
		return // the client went away; the check declines
	}

	frame, err := uc.Receive()
	if err != nil {
		// Same split as the HEL read above.
		p.noteReceiveError(err)
		return
	}
	if string(frame[:4]) != "OPNF" {
		p.noteOffScript(frame[:4], []byte("OPNF"))
		return
	}

	// The OPN is the row's evidence. The channel is buffered, so the
	// deliver never blocks; the close is what makes the client's Connect
	// return its expected error.
	p.opn <- frame
	_ = uc.Close()
}

// noteOffScript records the message type of a frame that arrived in
// place of `expected` - a HEL at the first read, an OPN at
// the second - so awaitOpenSecureChannel can report both in its reason
// instead of naming every type the script ever reads.
func (p *scriptedPeer) noteOffScript(msgType, expected []byte) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.offScript = string(msgType)
	p.offScriptExpected = string(expected)
}

// noteReceiveError records a read error from the peer's uacp framing,
// unless the connection merely ended: a client that goes away
// mid-exchange stays a decline, and so does a client whose frame is cut
// short mid-flight (io.ErrUnexpectedEOF from a truncated frame) - both
// count as the connection ending - but any other read error - a bad
// header, an out-of-range size, an ERR frame - can only come from the
// client's own bytes, so awaitOpenSecureChannel reports it as gopcua's
// behaviour. A reset - a client that hit its context deadline with the
// peer's ACK still unread closes the connection with one - and the
// fixture's own closed connection (net.ErrClosed, from teardown) also
// count as the connection ending. The reset's error is ECONNRESET
// here and WSAECONNRESET's 10054 on Windows, where syscall.ECONNRESET
// is an invented value that no reset error ever equals, so the check
// compares against the literal syscall.Errno(10054), which compiles
// on every platform.
func (p *scriptedPeer) noteReceiveError(err error) {
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) ||
		errors.Is(err, syscall.ECONNRESET) || errors.Is(err, syscall.Errno(10054)) ||
		errors.Is(err, net.ErrClosed) {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.recvErr = err.Error()
}

// awaitOpenSecureChannel is the setup check the rows share: it waits up
// to `within` for the peer to record the OPN frame the client sent - or
// until the peer's exchange ends, whichever comes first - and returns
// the frame with "". Otherwise it returns the reason the run cannot be
// scored, and which side caused the failed check: a frame that arrives
// where the script expected another, or bytes the peer's uacp framing
// rejects, can only be the client's own (true), and the row prints the
// reason with "gopcua's own behaviour made this run unscoreable: %s"
// and t.Errorf, then returns; everything else - no connection, no HEL,
// an exchange that ends before the OPN, no OPN within `within` - can
// close on either side, and is a decline the row prints with "could not
// set the test up: %s", reported at once when the exchange has already
// ended rather than after the bound.
func (p *scriptedPeer) awaitOpenSecureChannel(within time.Duration) ([]byte, string, bool) {
	select {
	case frame := <-p.opn:
		return frame, "", false
	case <-p.done:
	case <-time.After(within):
	}
	// An OPN that lands with the timer or the close is caught here, not
	// blamed on either side; serve delivers the OPN before it returns, so
	// the re-read below catches it when both land together.
	select {
	case frame := <-p.opn:
		return frame, "", false
	default:
	}
	p.mu.Lock()
	offScript := p.offScript
	offScriptExpected := p.offScriptExpected
	recvErr := p.recvErr
	p.mu.Unlock()
	if offScript != "" {
		return nil, fmt.Sprintf("a frame arrived with message type %q, "+
			"where the script expected %q", offScript, offScriptExpected), true
	}
	if recvErr != "" {
		return nil, fmt.Sprintf("the peer's uacp framing rejected the client's frame: %s", recvErr), true
	}
	select {
	case <-p.done:
		return nil, "the peer's exchange ended without an OPN frame", false
	default:
	}
	return nil, fmt.Sprintf("no OPN frame arrived within %v", within), false
}

// launchClient builds the opcua.Client under test and runs its Connect on
// its own goroutine, under a context bounded by handshake. The row passes
// the security policy options it scores; the harness adds
// AutoReconnect(false), so a failed connect is a returned error and not
// a retry loop. The peer never answers the OPN and closes the connection
// once it has recorded it, so Connect is expected to return an error;
// no row asserts it. The goroutine closes the client after Connect
// returns, so a future peer that lets Connect succeed leaves no client
// running past the row. It returns "" - or, when the client could not be
// built, the reason the run cannot be scored. A "build the client" reason
// can come from either side - a certificate the row minted, or the
// library rejecting valid options - so per ../README.md the row declines
// on it, printing it with "could not set the test up".
func launchClient(endpoint string, opts ...opcua.Option) string {
	client, err := opcua.NewClient(endpoint,
		append([]opcua.Option{opcua.AutoReconnect(false)}, opts...)...)
	if err != nil {
		return fmt.Sprintf("build the client: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), handshake)
	go func() {
		_ = client.Connect(ctx)
		_ = client.Close(ctx)
		cancel()
	}()
	return ""
}
