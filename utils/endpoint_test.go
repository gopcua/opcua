// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package utils

import (
	"net"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/wmnsk/gopcua/errors"
)

func TestResolveEndpoint(t *testing.T) {
	cases := []struct {
		input   string
		network string
		addr    *net.TCPAddr
		err     error
	}{
		{ // Valid, full EndpointURL
			"opc.tcp://10.0.0.1:4840/foo/bar",
			"tcp",
			&net.TCPAddr{
				IP:   net.IP([]byte{0x0a, 0x00, 0x00, 0x01}),
				Port: 4840,
			},
			nil,
		},
		{ // Valid, port number omitted
			"opc.tcp://10.0.0.1/foo/bar",
			"tcp",
			&net.TCPAddr{
				IP:   net.IP([]byte{0x0a, 0x00, 0x00, 0x01}),
				Port: 4840,
			},
			nil,
		},
		{ // Valid, hostname resolved
			"opc.tcp://localhost:4840/foo/bar",
			"tcp",
			&net.TCPAddr{
				IP:   net.IP([]byte{0x7f, 0x00, 0x00, 0x01}),
				Port: 4840,
			},
			nil,
		},
		{ // Invalid, schema is not "opc.tcp://"
			"tcp://10.0.0.1:4840/foo/bar",
			"",
			nil,
			errors.NewErrUnsupported("tcp:", "should be in \"opc.tcp://<addr[:port]>/path/to/somewhere\" format."),
		},
		{ // Invalid, bad formatted schema
			"opc.tcp:/10.0.0.1:4840/foo/bar",
			"tcp",
			nil,
			&net.DNSError{Err: "no such host", Name: "foo"},
		},
	}

	for _, c := range cases {
		network, addr, err := ResolveEndpoint(c.input)
		if diff := cmp.Diff(network, c.network); diff != "" {
			t.Error(diff)
		}
		if diff := cmp.Diff(addr, c.addr); diff != "" {
			t.Error(diff)
		}
		if diff := cmp.Diff(err, c.err); diff != "" {
			t.Error(diff)
		}
	}
}

func TestGetPath(t *testing.T) {
	cases := []struct {
		input  string
		path   string
		errStr string
	}{
		{ // Valid, full EndpointURL
			"opc.tcp://10.0.0.1:4840/foo/bar",
			"/foo/bar",
			"",
		},
		{ // Valid, schema is not checked in GetPath()
			"tcp://10.0.0.1:4840/foo/bar",
			"/foo/bar",
			"",
		},
		{ // Valid, no path following the address
			"tcp://10.0.0.1:4840",
			"/",
			"",
		},
		{ // Invalid, empty string
			"",
			"",
			"invalid input: ",
		},
	}

	for _, c := range cases {
		var errStr string
		path, err := GetPath(c.input)
		if err != nil {
			errStr = err.Error()
		}

		if diff := cmp.Diff(path, c.path); diff != "" {
			t.Error(diff)
		}
		if diff := cmp.Diff(errStr, c.errStr); diff != "" {
			t.Error(diff)
		}
	}
}
