// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uacp

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func setupConn(endpoint string) (cliConn, srvConn *Conn, err error) {
	server := NewServer(endpoint, 0xffff)
	ln, err := server.Listen()
	if err != nil {
		return nil, nil, err
	}

	go func() {
		defer ln.Close()
		srvConn, err = ln.Accept()
		if err != nil {
			return
		}
	}()

	client := NewClient(ln.Endpoint(), 0xffff, 5*time.Second, 3)
	cliConn, err = client.Dial(nil)
	if err != nil {
		return nil, nil, err
	}

	return
}

func TestClientConn(t *testing.T) {
	server := NewServer("opc.tcp://127.0.0.1:4840/foo/bar", 0xffff)
	ln, err := server.Listen()
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		defer ln.Close()
		if _, err := ln.Accept(); err != nil {
			t.Fatal(err)
		}
	}()

	client := NewClient("opc.tcp://127.0.0.1:4840/foo/bar", 0xffff, 2*time.Second, 3)
	if _, err = client.Dial(nil); err != nil {
		t.Error(err)
	}
}

func TestClientWrite(t *testing.T) {
	srvConn, cliConn, err := setupConn("opc.tcp://127.0.0.1:4840/foo/bar")
	if err != nil {
		t.Fatal(err)
	}

	buf := make([]byte, 1024)
	msg := []byte{0xde, 0xad, 0xbe, 0xef}
	if _, err := cliConn.Write(msg); err != nil {
		t.Fatal(err)
	}
	n, err := srvConn.Read(buf)
	if err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(buf[:n], msg); diff != "" {
		t.Error(diff)
	}
}

func TestServerWrite(t *testing.T) {
	srvConn, cliConn, err := setupConn("opc.tcp://127.0.0.1:4840/foo/bar")
	if err != nil {
		t.Fatal(err)
	}

	buf := make([]byte, 1024)
	msg := []byte{0xde, 0xad, 0xbe, 0xef}
	if _, err := srvConn.Write(msg); err != nil {
		t.Fatal(err)
	}
	n, err := cliConn.Read(buf)
	if err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(buf[:n], msg); diff != "" {
		t.Error(diff)
	}
}
