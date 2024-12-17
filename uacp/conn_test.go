// Copyright 2018-2020 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uacp

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/gopcua/opcua/errors"
	"github.com/stretchr/testify/require"
)

func TestConn(t *testing.T) {
	t.Run("server exists ", func(t *testing.T) {
		ep := "opc.tcp://127.0.0.1:4840/foo/bar"
		ln, err := Listen(ep, nil)
		require.NoError(t, err)
		defer ln.Close()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		done := make(chan struct{})
		acceptErr := make(chan error, 1)
		go func() {
			c, err := ln.Accept(ctx)
			if err != nil {
				acceptErr <- err
				return
			}
			defer c.Close()
			close(done)
		}()

		if _, err = Dial(ctx, ep); err != nil {
			t.Error(err)
		}

		select {
		case <-done:
		case err := <-acceptErr:
			require.Fail(t, "accept fail: %v", err)
		case <-time.After(time.Second):
			require.Fail(t, "timed out")
		}
	})

	t.Run("Address resolves, but does not implement a opcua-server", func(t *testing.T) {
		ep := "opc.tcp://example.com:56789"

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_, err := Dial(ctx, ep)
		var operr *net.OpError
		if errors.As(err, &operr) && !operr.Timeout() {
			t.Error(err)
		}
	})
}

func TestClientWrite(t *testing.T) {
	ep := "opc.tcp://127.0.0.1:4840/foo/bar"
	ln, err := Listen(ep, nil)
	require.NoError(t, err, "Listen failed")
	defer ln.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var srvConn *Conn
	done := make(chan int)
	acceptErr := make(chan error, 1)
	go func() {
		defer ln.Close()
		var err error
		srvConn, err = ln.Accept(ctx)
		if err != nil {
			acceptErr <- err
			return
		}
		done <- 0
	}()

	cliConn, err := Dial(ctx, ep)
	require.NoError(t, err, "Dial failed")

	for {
		select {
		case _, ok := <-done:
			require.True(t, ok, "failed to setup secure channel")
			goto NEXT
		case err := <-acceptErr:
			require.Fail(t, "accept fail: %v", err)
		case <-time.After(time.Second):
			require.Fail(t, "timed out")
		}
	}

NEXT:
	msg := &Message{Data: []byte{0xde, 0xad, 0xbe, 0xef}}
	err = cliConn.Send("MSGF", msg)
	require.NoError(t, err, "Send failed")

	got, err := srvConn.Receive()
	require.NoError(t, err, "Receive failed")

	got = got[hdrlen:]

	want, err := msg.Encode()
	require.NoError(t, err, "Encode failed")

	require.Equal(t, want, got)
}

func TestServerWrite(t *testing.T) {
	ep := "opc.tcp://127.0.0.1:4840/foo/bar"
	ln, err := Listen(ep, nil)
	require.NoError(t, err, "Listen failed")
	defer ln.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var srvConn *Conn
	done := make(chan int)
	acceptErr := make(chan error, 1)
	go func() {
		defer ln.Close()
		var err error
		srvConn, err = ln.Accept(ctx)
		if err != nil {
			acceptErr <- err
			return
		}
		done <- 0
	}()

	cliConn, err := Dial(ctx, ep)
	require.NoError(t, err, "Dial failed")

	for {
		select {
		case _, ok := <-done:
			require.True(t, ok, "failed to setup secure channel")
			goto NEXT
		case err := <-acceptErr:
			require.Fail(t, "accept fail: %v", err)
		case <-time.After(time.Second):
			require.Fail(t, "timed out")
		}
	}

NEXT:
	want := []byte{0xde, 0xad, 0xbe, 0xef}
	_, err = srvConn.Write(want)
	require.NoError(t, err, "Write failed")

	got := make([]byte, cliConn.ReceiveBufSize())
	n, err := cliConn.Read(got)
	require.NoError(t, err, "Read failed")

	got = got[:n]
	require.Equal(t, want, got)
}
