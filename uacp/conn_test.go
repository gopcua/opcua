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

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go func() {
		defer ln.Close()
		if _, err := ln.Accept(ctx); err != nil {
			t.Fatal(err)
		}
	}()

	if _, err = Dial(ctx, ep, nil); err != nil {
		t.Error(err)
	}
}

func TestClientWrite(t *testing.T) {
	ep := "opc.tcp://127.0.0.1:4840/foo/bar"
	ln, err := Listen(ep, 0xffff)
	if err != nil {
		t.Fatal(err)
	}

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

	cliConn, err = Dial(ctx, ep, nil)
	if err != nil {
		t.Error(err)
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

	cliConn, err = Dial(ctx, ep, nil)
	if err != nil {
		t.Error(err)
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
