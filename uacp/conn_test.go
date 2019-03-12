// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uacp

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func TestConn(t *testing.T) {
	ep := "opc.tcp://127.0.0.1:4840/foo/bar"
	ln, err := Listen(ep, 0xffff)
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		c, err := ln.Accept(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer c.Close()
		close(done)
	}()

	if _, err = Dial(ctx, ep); err != nil {
		t.Error(err)
	}

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatalf("timed out")
	}
}

func TestClientWrite(t *testing.T) {
	ep := "opc.tcp://127.0.0.1:4840/foo/bar"
	ln, err := Listen(ep, 0xffff)
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var cliConn, srvConn *Conn
	done := make(chan int)
	go func() {
		defer ln.Close()
		srvConn, err = ln.Accept(ctx)
		if err != nil {
			t.Fatal(err)
		}
		done <- 0
	}()

	cliConn, err = Dial(ctx, ep)
	if err != nil {
		t.Fatal(err)
	}

	buf := make([]byte, 1024)
	expected := []byte{0xde, 0xad, 0xbe, 0xef}
	for {
		select {
		case _, ok := <-done:
			if !ok {
				t.Fatal("failed to setup secure channel")
			}
			goto NEXT
		case <-time.After(10 * time.Second):
			t.Fatalf("timed out")
		}
	}

NEXT:
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
}

func TestServerWrite(t *testing.T) {
	ep := "opc.tcp://127.0.0.1:4840/foo/bar"
	ln, err := Listen(ep, 0xffff)
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var cliConn, srvConn *Conn
	done := make(chan int)
	go func() {
		defer ln.Close()
		srvConn, err = ln.Accept(ctx)
		if err != nil {
			t.Fatal(err)
		}
		done <- 0
	}()

	cliConn, err = Dial(ctx, ep)
	if err != nil {
		t.Fatal(err)
	}

	buf := make([]byte, 1024)
	expected := []byte{0xde, 0xad, 0xbe, 0xef}
	for {
		select {
		case _, ok := <-done:
			if !ok {
				t.Fatal("failed to setup secure channel")
			}
			goto NEXT
		case <-time.After(10 * time.Second):
			t.Fatalf("timed out")
		}
	}

NEXT:
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
}
