// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uacp

import (
	"context"
	"net/url"
	"testing"
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
			"opc.tcp://localhost:4840/foo/bar",
			"tcp",
			&url.URL{
				Scheme: "opc.tcp",
				Host:   "127.0.0.1:4840",
				Path:   "/foo/bar",
			},
			"",
		},
		{ // Invalid, scheme is not "opc.tcp://"
			"tcp://10.0.0.1:4840/foo/bar",
			"",
			nil,
			"unsupported scheme tcp",
		},
		{ // Invalid, bad formatted endpoint
			"opc.tcp://10.0.0.1:x4840/foo/bar",
			"",
			nil,
			"parse opc.tcp://10.0.0.1:x4840/foo/bar: invalid port \":x4840\" after host",
		},
	}

	for _, c := range cases {
		var errStr string
		network, u, err := ResolveEndpoint(context.Background(), c.input)
		if err != nil {
			errStr = err.Error()
			if got, want := errStr, c.errStr; got != want {
				t.Fatalf("got error %q want %q", got, want)
			}
		} else {
			if got, want := network, c.network; got != want {
				t.Fatalf("got network %q want %q", got, want)
			}
			if got, want := u.String(), c.u.String(); got != want {
				t.Fatalf("got addr %q want %q", got, want)
			}
		}
	}
}
