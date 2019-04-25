// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uacp

import (
	"net"
	"testing"
)

func TestResolveEndpoint(t *testing.T) {
	cases := []struct {
		input   string
		network string
		addr    *net.TCPAddr
		errStr  string
	}{
		{ // Valid, full EndpointURL
			"opc.tcp://10.0.0.1:4840/foo/bar",
			"tcp",
			&net.TCPAddr{
				IP:   net.IP([]byte{0x0a, 0x00, 0x00, 0x01}),
				Port: 4840,
			},
			"",
		},
		{ // Valid, port number omitted
			"opc.tcp://10.0.0.1/foo/bar",
			"tcp",
			&net.TCPAddr{
				IP:   net.IP([]byte{0x0a, 0x00, 0x00, 0x01}),
				Port: 4840,
			},
			"",
		},
		{ // Valid, hostname resolved
			"opc.tcp://localhost:4840/foo/bar",
			"tcp",
			&net.TCPAddr{
				IP:   net.IP([]byte{0x7f, 0x00, 0x00, 0x01}),
				Port: 4840,
			},
			"",
		},
		{ // Invalid, schema is not "opc.tcp://"
			"tcp://10.0.0.1:4840/foo/bar",
			"",
			nil,
			"invalid endpoint tcp://10.0.0.1:4840/foo/bar",
		},
		{ // Invalid, bad formatted schema
			"opc.tcp:/10.0.0.1:4840/foo/bar",
			"",
			nil,
			"could not resolve address foo:4840",
		},
	}

	for _, c := range cases {
		var errStr string
		network, addr, err := ResolveEndpoint(c.input)
		if err != nil {
			errStr = err.Error()
		}
		if got, want := network, c.network; got != want {
			t.Fatalf("got network %q want %q", got, want)
		}
		if got, want := addr.String(), c.addr.String(); got != want {
			t.Fatalf("got addr %q want %q", got, want)
		}
		if got, want := errStr, c.errStr; got != want {
			t.Fatalf("got error %q want %q", got, want)
		}
	}
}
