// Copyright 2018-2020 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uacp

import (
	"context"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveEndpoint(t *testing.T) {
	cases := []struct {
		input   string
		network string
		u       *url.URL
		errStr  string
	}{
		{ // Valid, full EndpointURL
			"opc.tcp://10.0.0.1:4840/foo/bar",
			"tcp",
			&url.URL{
				Scheme: "opc.tcp",
				Host:   "10.0.0.1:4840",
				Path:   "/foo/bar",
			},
			"",
		},
		{ // Valid, port number omitted
			"opc.tcp://10.0.0.1/foo/bar",
			"tcp",
			&url.URL{
				Scheme: "opc.tcp",
				Host:   "10.0.0.1:4840",
				Path:   "/foo/bar",
			},
			"",
		},
		{ // Valid, hostname resolved
			// note: see https://github.com/cunnie/sslip.io
			"opc.tcp://www.1.1.1.1.sslip.io:4840/foo/bar",
			"tcp",
			&url.URL{
				Scheme: "opc.tcp",
				Host:   "1.1.1.1:4840",
				Path:   "/foo/bar",
			},
			"",
		},
		{ // Invalid, schema is not "opc.tcp://"
			"tcp://10.0.0.1:4840/foo/bar",
			"",
			nil,
			"opcua: unsupported scheme tcp",
		},
		{ // Invalid, bad formatted schema
			"opc.tcp:/10.0.0.1:4840/foo1337bar/baz",
			"",
			nil,
			"lookup : no such host",
		},
	}

	for _, c := range cases {
		network, u, err := ResolveEndpoint(context.Background(), c.input)
		if c.errStr != "" {
			require.EqualError(t, err, c.errStr)
		} else {
			require.Equal(t, c.network, network)
			require.Equal(t, c.u, u)
		}
	}
}
