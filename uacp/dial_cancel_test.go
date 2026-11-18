package uacp

import (
	"context"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestDialHandshakeHonoursCancel: a server that accepts TCP and never
// answers Hello. The client's reconnect monitor dials with a ctx that is
// cancelled (by Client.Close) but has no deadline; the handshake read must
// still give up when it is cancelled. Before, only a ctx deadline bounded
// it, so the monitor goroutine and its socket outlived Close for as long
// as the peer kept the connection open.
func TestDialHandshakeHonoursCancel(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	var mu sync.Mutex
	var accepted []net.Conn
	t.Cleanup(func() {
		_ = ln.Close()
		mu.Lock()
		defer mu.Unlock()
		for _, c := range accepted {
			_ = c.Close() // also unblocks a Dial the test gave up on
		}
	})
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			mu.Lock()
			accepted = append(accepted, c) // held open, never answered
			mu.Unlock()
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	dialErr := make(chan error, 1)
	go func() {
		_, err := (&Dialer{}).Dial(ctx, "opc.tcp://"+ln.Addr().String())
		dialErr <- err
	}()

	// Give the dial time to reach the handshake read, then cancel.
	require.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(accepted) == 1
	}, 5*time.Second, 10*time.Millisecond)
	time.Sleep(100 * time.Millisecond)
	cancel()

	select {
	case err := <-dialErr:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(5 * time.Second):
		t.Fatal("Dial is still blocked in the HEL/ACK handshake after its ctx was cancelled")
	}
}
