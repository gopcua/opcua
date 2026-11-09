//go:build conformance

// Package scriptedpeer holds the core the transport and securitypolicy
// packages' scripted TCP peers share: the listen on a free port, the
// exactly one accepted connection, the teardown that closes the listener
// and that connection, and the address. It is a helper two packages
// call, so per ../README.md it lives here, in tests/internal/scriptedpeer,
// under the same plain-values rule a helper in a harness.go answers to:
// nothing here calls t.Errorf or t.Fatalf, and the harness that calls it
// words the reason a failure leaves the run unscoreable. The script a
// peer plays once its connection is accepted - and the state it records
// of the exchange - is each package's own and stays in its harness.go.
package scriptedpeer

import (
	"net"
	"sync"
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
