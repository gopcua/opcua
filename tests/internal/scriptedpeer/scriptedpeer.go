//go:build conformance

// Package scriptedpeer holds the core the transport, securitypolicy and
// securechannel packages' scripted TCP peers share: the listen on a free
// port, the exactly one accepted connection, the teardown that closes the
// listener and that connection, and the address; and, for the peers that
// answer where transport's stays silent, the HEL/ACK opening of the
// accepted connection and the split of a failed read into one that merely
// ended and one that only the client's own bytes can explain. They are
// helpers two packages call, so per ../README.md they live here, in
// tests/internal/scriptedpeer, under the same plain-values rule a helper in
// a harness.go answers to: nothing here calls t.Errorf or t.Fatalf, and the
// harness that calls it words the reason a failure leaves the run
// unscoreable. HelloAck's error tells the three ways the opening exchange
// can fail apart without any string parsing: its own failure wraps
// ErrPeerSide, a frame that is not a HEL is an *OffScriptError, and a
// receive error is returned unwrapped for the caller's ConnectionEnded
// split. The script a peer plays once its connection is accepted - and
// the state it records of the exchange - is each package's own and stays
// in its harness.go.
package scriptedpeer

import (
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"syscall"

	"github.com/gopcua/opcua/uacp"
)

// Core is the shared half of a scripted peer: a listener on a free port,
// the one connection it accepts, and the flag teardown sets. The half a
// package does not share - the script, and what it records - lives in
// that package's own peer type, which holds a Core beside its own state.
type Core struct {
	listener net.Listener

	mu     sync.Mutex
	conn   net.Conn
	closed bool // set by Teardown; Accept closes the connection itself when set
}

// Listen listens on host, on a free port the kernel chooses - never port
// 4840, which other suites bind. It returns the listener or the error;
// each harness words the error into the reason the run cannot be scored,
// because why the peer was listening - on the loopback interface, on the
// last address a row resolved - is the calling package's own story.
func Listen(host string) (net.Listener, error) {
	return net.Listen("tcp", net.JoinHostPort(host, "0"))
}

// New wraps a listener Listen returned in the core.
func New(l net.Listener) *Core {
	return &Core{listener: l}
}

// Accept accepts the core's one connection and returns it, or nil when
// the listener was closed in teardown. A second connection is never
// accepted. If teardown runs between the Accept and this goroutine
// taking the mutex, teardown never saw the connection, so Accept closes
// it here and returns nil; the caller's script never starts.
func (c *Core) Accept() net.Conn {
	conn, err := c.listener.Accept()
	if err != nil {
		return nil // the listener was closed in teardown
	}
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		_ = conn.Close()
		return nil
	}
	c.conn = conn
	c.mu.Unlock()
	return conn
}

// Teardown closes the listener and the accepted connection, so a dial or
// a connect left blocked by the library under test is released when the
// test ends. The row's test function, not the fixture, registers it with
// t.Cleanup.
func (c *Core) Teardown() {
	_ = c.listener.Close()
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closed = true
	if c.conn != nil {
		_ = c.conn.Close()
	}
}

// Addr is the host:port the core listens on.
func (c *Core) Addr() string {
	return c.listener.Addr().String()
}

// ErrPeerSide marks a failure of the peer's own half of the opening
// exchange - framing the accepted connection, or the ACK not going out
// because the client went away - which the calling harness reports as a
// decline, never as the client's behaviour.
var ErrPeerSide = errors.New("the scripted peer's own half of the exchange failed")

// OffScriptError is a frame that arrived where the script expected
// another: a client that did not send the HEL the opening exchange
// waits for. It can only be the client's own bytes, and the calling
// harness reports it as gopcua's behaviour, not as a decline.
type OffScriptError struct {
	Got      string // the message type of the frame that arrived
	Expected string // the message type the script expected
}

func (e *OffScriptError) Error() string {
	return fmt.Sprintf("a frame arrived with message type %q, "+
		"where the script expected %q", e.Got, e.Expected)
}

// HelloAck opens the accepted connection the way a scripted server half
// of OPC 10000-6's handshake does: it frames the connection with uacp,
// reads the client's HEL, and answers it with the ACK it is given. Its
// error tells the three ways the exchange can fail apart, so the
// caller never parses the message: an error wrapping ErrPeerSide is
// the peer's own failure - framing the connection, or the ACK not
// going out - which the caller reports as a decline without blaming
// the client; an *OffScriptError is a frame that is not a HEL, which
// can only be the client's own bytes; any other error is a receive
// error the caller splits with ConnectionEnded - a connection that
// merely ended, which is a decline, or the client's own bytes.
func HelloAck(c *net.TCPConn, ack *uacp.Acknowledge) (*uacp.Conn, error) {
	uc, err := uacp.NewConn(c, ack)
	if err != nil {
		return nil, fmt.Errorf("%w: frame the accepted connection with uacp: %v", ErrPeerSide, err)
	}
	hel, err := uc.Receive()
	if err != nil {
		return nil, err
	}
	if string(hel[:4]) != "HELF" {
		return nil, &OffScriptError{Got: string(hel[:4]), Expected: "HELF"}
	}
	if err := uc.Send("ACKF", ack); err != nil {
		return nil, fmt.Errorf("%w: send the ACK: %v", ErrPeerSide, err)
	}
	return uc, nil
}

// ConnectionEnded reports whether err is a connection that merely ended:
// a client that closes, goes away mid-exchange, or whose frame is cut
// short mid-flight (io.ErrUnexpectedEOF from a truncated frame, which
// counts as the connection ending) - or the fixture's own closed
// connection (net.ErrClosed, from teardown). A reset - a client that hit
// its context deadline with the peer's ACK still unread closes the
// connection with one - also counts: the reset's error is ECONNRESET here
// and WSAECONNRESET's 10054 on Windows, where syscall.ECONNRESET is an
// invented value that no reset error ever equals, so the comparison is
// against the literal syscall.Errno(10054), which compiles on every
// platform. Anything else - a bad header, an out-of-range size, an ERR
// frame - can only come from the client's own bytes, and the caller
// reports it as gopcua's behaviour.
func ConnectionEnded(err error) bool {
	return errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) ||
		errors.Is(err, syscall.ECONNRESET) || errors.Is(err, syscall.Errno(10054)) ||
		errors.Is(err, net.ErrClosed)
}
