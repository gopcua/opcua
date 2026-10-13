// Copyright 2018-2020 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uacp

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestListener_HandshakeTimesOutOnASilentClient(t *testing.T) {
	ctx := context.Background()
	l, err := Listen(ctx, "opc.tcp://127.0.0.1:0", nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = l.Close() })

	c, err := net.Dial("tcp", l.Addr().String())
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.Close() })

	tc, err := l.AcceptTCP()
	require.NoError(t, err)

	start := time.Now()
	_, err = l.Handshake(tc, 200*time.Millisecond)
	require.Error(t, err, "a client that never sends Hello must not complete the handshake")
	require.Less(t, time.Since(start), 5*time.Second, "the timeout must end the handshake")
}

func TestListener_HandshakeClearsTheDeadline(t *testing.T) {
	ctx := context.Background()
	l, err := Listen(ctx, "opc.tcp://127.0.0.1:0", nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = l.Close() })

	dialed := make(chan *Conn, 1)
	go func() {
		c, err := Dial(ctx, "opc.tcp://"+l.Addr().String())
		if err != nil {
			t.Errorf("dial: %v", err)
		}
		dialed <- c
	}()

	tc, err := l.AcceptTCP()
	require.NoError(t, err)
	srv, err := l.Handshake(tc, 200*time.Millisecond)
	require.NoError(t, err)
	t.Cleanup(func() { _ = srv.Close() })
	cli := <-dialed
	require.NotNil(t, cli)
	t.Cleanup(func() { _ = cli.Close() })

	// Past the handshake timeout, the established connection must still read.
	time.Sleep(400 * time.Millisecond)
	go func() { _, _ = cli.Write([]byte("ping")) }()
	b := make([]byte, 4)
	_, err = srv.Read(b)
	require.NoError(t, err, "the handshake deadline must not outlive the handshake")
	require.Equal(t, "ping", string(b))
}
