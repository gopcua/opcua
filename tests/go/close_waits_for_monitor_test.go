//go:build integration
// +build integration

package uatest2

import (
	"context"
	"io"
	"net"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/ua"
	"github.com/stretchr/testify/require"
)

// TestCloseWaitsForTheReconnectMonitor: Close cancels the reconnect
// monitor and the publish loop, and returns only once both have ended.
//
// Close used to cancel them and return at once. A monitor caught in the
// middle of a step (a restore, a handshake, a callback of the caller's)
// went on after Close had returned, and could still call back into the
// caller — OnSessionAbandoned handing over the session its cancelled
// restore gave up, StateChangedFunc — when the caller had already torn
// its own side down. Here the monitor is held for a moment inside the
// caller's StateChangedFunc when Close is called: Close must wait for it.
func TestCloseWaitsForTheReconnectMonitor(t *testing.T) {
	ctx := context.Background()

	srv := startServer()
	defer srv.Close()
	time.Sleep(2 * time.Second)

	proxy := newLinkCutter(t, "localhost:4840")

	inMonitor := make(chan struct{})
	var held atomic.Bool
	c, err := opcua.NewClient("opc.tcp://"+proxy.addr(),
		opcua.SecurityMode(ua.MessageSecurityModeNone),
		opcua.AutoReconnect(true),
		opcua.ReconnectInterval(200*time.Millisecond),
		opcua.StateChangedFunc(func(s opcua.ConnState) {
			// Set by the monitor when the link drops: hold it there once.
			if s == opcua.Disconnected && held.CompareAndSwap(false, true) {
				close(inMonitor)
				time.Sleep(300 * time.Millisecond)
			}
		}),
	)
	require.NoError(t, err)
	require.NoError(t, c.Connect(ctx))

	proxy.cutAll()
	select {
	case <-inMonitor:
	case <-time.After(10 * time.Second):
		t.Fatal("the monitor never reported the dropped link")
	}

	closeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	require.NoError(t, c.Close(closeCtx))

	if alive := clientLoops(); alive != "" {
		t.Fatalf("Close returned while the client's loops still ran:\n%s", alive)
	}
}

// clientLoops returns the stacks of goroutines running the client's
// reconnect monitor or publish loop, or "".
func clientLoops() string {
	buf := make([]byte, 1<<20)
	buf = buf[:runtime.Stack(buf, true)]
	var out []string
	for _, g := range strings.Split(string(buf), "\n\n") {
		if strings.Contains(g, "opcua.(*Client).monitor(") || strings.Contains(g, "opcua.(*Client).monitorSubscriptions(") {
			out = append(out, g)
		}
	}
	return strings.Join(out, "\n\n")
}

// linkCutter is a TCP proxy whose links the test can cut, as a dropped
// network link would.
type linkCutter struct {
	ln    net.Listener
	mu    sync.Mutex
	conns []net.Conn
}

func newLinkCutter(t *testing.T, target string) *linkCutter {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	p := &linkCutter{ln: ln}
	t.Cleanup(func() { _ = ln.Close(); p.cutAll() })
	go func() {
		for {
			in, err := ln.Accept()
			if err != nil {
				return
			}
			out, err := net.Dial("tcp", target)
			if err != nil {
				_ = in.Close()
				continue
			}
			p.mu.Lock()
			p.conns = append(p.conns, in, out)
			p.mu.Unlock()
			go func() { _, _ = io.Copy(out, in); _ = out.Close() }()
			go func() { _, _ = io.Copy(in, out); _ = in.Close() }()
		}
	}()
	return p
}

func (p *linkCutter) addr() string { return p.ln.Addr().String() }

func (p *linkCutter) cutAll() {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, c := range p.conns {
		_ = c.Close()
	}
	p.conns = nil
}
