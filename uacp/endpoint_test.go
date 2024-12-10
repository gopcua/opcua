// Copyright 2018-2020 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uacp

import (
	"net"
	"testing"

	"github.com/stretchr/testify/require"
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
			"opcua: invalid endpoint tcp://10.0.0.1:4840/foo/bar",
		},
		{ // Invalid, bad formatted schema
			"opc.tcp:/10.0.0.1:4840/foo1337bar/baz",
			"",
			nil,
			"opcua: could not resolve address foo1337bar:4840",
		},
	}

	for _, c := range cases {
		network, addr, err := ResolveEndpoint(c.input)
		require.Equal(t, c.network, network, "network not equal")
		require.Equal(t, c.addr.String(), addr.String(), "addr not equal")
		if err != nil {
			require.ErrorContains(t, err, c.errStr)
		}
	}
}
