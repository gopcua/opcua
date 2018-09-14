// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uasc

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/wmnsk/gopcua/services"
	"github.com/wmnsk/gopcua/uacp"
)

func TestSecureChannel(t *testing.T) {
	ep := "opc.tcp://127.0.0.1:4840/foo/bar"
	policyURI := "http://opcfoundation.org/UA/SecurityPolicy#None"
	ln, err := uacp.Listen(ep, 0xffff)
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	cfg := NewConfig(
		1, policyURI, nil, nil, 0, 1,
	)

	go func() {
		defer ln.Close()
		srvConn, err := ln.Accept(ctx)
		if err != nil {
			t.Fatal(err)
		}

		if _, err := ListenAndAcceptSecureChannel(ctx, srvConn, cfg); err != nil {
			t.Error(err)
		}
	}()

	cliConn, err := uacp.Dial(ctx, ep)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := OpenSecureChannel(ctx, cliConn, cfg, services.SecModeNone, 0xffff, nil); err != nil {
		t.Error(err)
	}
}

func TestClientWrite(t *testing.T) {
	ep := "opc.tcp://127.0.0.1:4840/foo/bar"
	policyURI := "http://opcfoundation.org/UA/SecurityPolicy#None"
	ln, err := uacp.Listen(ep, 0xffff)
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var (
		cliConn, srvConn *uacp.Conn
		cliChan, srvChan *SecureChannel
	)

	done := make(chan int)
	cfg := NewConfig(
		1, policyURI, nil, nil, 0, 1,
	)
	go func() {
		defer ln.Close()
		srvConn, err = ln.Accept(ctx)
		if err != nil {
			t.Fatal(err)
		}

		srvChan, err = ListenAndAcceptSecureChannel(ctx, srvConn, cfg)
		if err != nil {
			t.Fatal(err)
		}
		done <- 0
	}()

	cliConn, err = uacp.Dial(ctx, ep)
	if err != nil {
		t.Fatal(err)
	}

	cliChan, err = OpenSecureChannel(ctx, cliConn, cfg, services.SecModeNone, 0xffff, nil)
	if err != nil {
		t.Fatal(err)
	}

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

	msg := []byte{0xde, 0xad, 0xbe, 0xef}
	if _, err := cliChan.Write(msg); err != nil {
		t.Fatal(err)
	}

	buf := make([]byte, 1024)
	n, err := srvChan.Read(buf)
	if err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(buf[:n], msg); diff != "" {
		t.Error(diff)
	}
}

func TestServerWrite(t *testing.T) {
	ep := "opc.tcp://127.0.0.1:4840/foo/bar"
	policyURI := "http://opcfoundation.org/UA/SecurityPolicy#None"
	ln, err := uacp.Listen(ep, 0xffff)
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var (
		cliConn, srvConn *uacp.Conn
		cliChan, srvChan *SecureChannel
	)

	done := make(chan int)
	cfg := NewConfig(
		1, policyURI, nil, nil, 0, 1,
	)
	go func() {
		defer ln.Close()
		srvConn, err = ln.Accept(ctx)
		if err != nil {
			t.Fatal(err)
		}

		srvChan, err = ListenAndAcceptSecureChannel(ctx, srvConn, cfg)
		if err != nil {
			t.Fatal(err)
		}
		done <- 0
	}()

	cliConn, err = uacp.Dial(ctx, ep)
	if err != nil {
		t.Fatal(err)
	}

	cliChan, err = OpenSecureChannel(ctx, cliConn, cfg, services.SecModeNone, 0xffff, nil)
	if err != nil {
		t.Fatal(err)
	}

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

	msg := []byte{0xde, 0xad, 0xbe, 0xef}
	if _, err := srvChan.Write(msg); err != nil {
		t.Fatal(err)
	}

	buf := make([]byte, 1024)
	n, err := cliChan.Read(buf)
	if err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(buf[:n], msg); diff != "" {
		t.Error(diff)
	}
}
