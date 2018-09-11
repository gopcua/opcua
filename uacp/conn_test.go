// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uacp

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestConn(t *testing.T) {
	ep := "opc.tcp://127.0.0.1:4840/foo/bar"
	ln, err := Listen(ep, 0xffff)
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go func() {
		defer ln.Close()
		if _, err := ln.Accept(ctx); err != nil {
			t.Fatal(err)
		}
	}()

	if _, err = Dial(ctx, ep); err != nil {
		t.Error(err)
	}
}

func TestClientWrite(t *testing.T) {
	ep := "opc.tcp://127.0.0.1:4840/foo/bar"
	ln, err := Listen(ep, 0xffff)
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var cliConn, srvConn *Conn
	go func() {
		defer ln.Close()
		srvConn, err = ln.Accept(ctx)
		if err != nil {
			t.Fatal(err)
		}
	}()

	cliConn, err = Dial(ctx, ep)
	if err != nil {
		t.Fatal(err)
	}

	buf := make([]byte, 1024)
	expected := []byte{0xde, 0xad, 0xbe, 0xef}
	for {
		if srvConn != nil {
			if _, err := cliConn.Write(expected); err != nil {
				t.Fatal(err)
			}
			n, err := srvConn.Read(buf)
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(buf[:n], expected); diff != "" {
				t.Error(diff)
			}
			break
		}
	}
}

func TestServerWrite(t *testing.T) {
	ep := "opc.tcp://127.0.0.1:4840/foo/bar"
	ln, err := Listen(ep, 0xffff)
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var cliConn, srvConn *Conn
	go func() {
		defer ln.Close()
		srvConn, err = ln.Accept(ctx)
		if err != nil {
			t.Fatal(err)
		}
	}()

	cliConn, err = Dial(ctx, ep)
	if err != nil {
		t.Fatal(err)
	}

	buf := make([]byte, 1024)
	expected := []byte{0xde, 0xad, 0xbe, 0xef}
	for {
		if srvConn != nil {
			if _, err := srvConn.Write(expected); err != nil {
				t.Fatal(err)
			}
			n, err := cliConn.Read(buf)
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(buf[:n], expected); diff != "" {
				t.Error(diff)
			}
			break
		}
	}
}
func TestConnGetState(t *testing.T) {
	var cases = []struct {
		conn     *Conn
		expected string
	}{
		{
			nil,
			"",
		},
		{
			&Conn{},
			"unknown",
		},
		{
			&Conn{state: cliStateClosed},
			"client closed",
		},
		{
			&Conn{state: cliStateHelloSent},
			"client hello sent",
		},
		{
			&Conn{state: cliStateEstablished},
			"client established",
		},
		{
			&Conn{state: srvStateClosed},
			"server closed",
		},
		{
			&Conn{state: srvStateEstablished},
			"server established",
		},
	}

	for i, c := range cases {
		if diff := cmp.Diff(c.conn.GetState(), c.expected); diff != "" {
			t.Errorf("case #%d failed\n%s", i, diff)
		}
	}
}
