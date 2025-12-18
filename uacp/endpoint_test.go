// Copyright 2018-2020 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uacp

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
	_ "github.com/stretchr/testify/require"
)

func TestResolveEndpoint(t *testing.T) {
	cases := []struct {
		input       string
		network     string
		u           *url.URL
		errStr      string
		preResolver PreResolver
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
			nil,
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
			nil,
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
			nil,
		},
		{ // Valid, hostname resolved by pre-resolver
			"opc.tcp://preresolver-known:4840/foo/bar",
			"tcp",
			&url.URL{
				Scheme: "opc.tcp",
				Host:   "1.2.3.4:4840",
				Path:   "/foo/bar",
			},
			"",
			&mockPreResolver{},
		},
		{ // Invalid, schema is not "opc.tcp://"
			"tcp://10.0.0.1:4840/foo/bar",
			"",
			nil,
			"opcua: unsupported scheme tcp",
			nil,
		},
		{ // Invalid, bad formatted schema
			"opc.tcp:/10.0.0.1:4840/foo1337bar/baz",
			"",
			nil,
			"lookup : no such host",
			nil,
		},
		{ // Invalid, pre-resolver fails
			"opc.tcp://preresolver-fail:4840/foo/bar",
			"",
			nil,
			"pre-resolver failed: pre-resolver error",
			&mockPreResolver{},
		},
	}

	for _, c := range cases {
		network, u, err := ResolveEndpoint(context.Background(), c.input, c.preResolver)
		if c.errStr != "" {
			require.EqualError(t, err, c.errStr)
		} else {
			require.Equal(t, c.network, network)
			require.Equal(t, c.u, u)
		}
	}
}

type mockPreResolver struct{}

func (m *mockPreResolver) LookupIPAddr(ctx context.Context, host string) ([]net.IPAddr, error) {
	switch host {
	case "preresolver-known":
		return []net.IPAddr{{IP: net.ParseIP("1.2.3.4")}}, nil
	case "preresolver-fail":
		return nil, fmt.Errorf("pre-resolver error")
	}

	// Not a name we know about
	return nil, nil
}
