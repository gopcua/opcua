// Copyright 2018-2020 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uacp

import (
	"context"
	"net"
	"net/url"

	"github.com/gopcua/opcua/errors"
)

const defaultPort = "4840"

// ResolveEndpoint returns network type, address, and error split from EndpointURL.
//
// Expected format of input is "opc.tcp://<addr[:port]/path/to/somewhere"
func ResolveEndpoint(ctx context.Context, endpoint string) (network string, u *url.URL, err error) {
	network, u, err = parseEndpoint(endpoint)
	if err != nil {
		return
	}

	_, port, err := net.SplitHostPort(u.Host)
	if err != nil {
		return
	}

	var resolver net.Resolver

	addrs, err := resolver.LookupIPAddr(ctx, u.Hostname())
	if err != nil {
		return
	}

	if len(addrs) == 0 {
		err = errors.Errorf("could not resolve address %s", u.Hostname())
		return
	}

	u.Host = net.JoinHostPort(addrs[0].String(), port)

	return
}

// parseEndpoint checks the scheme of an EndpointURL and returns its host with
// the port filled in, without resolving the host.
func parseEndpoint(endpoint string) (network string, u *url.URL, err error) {
	u, err = url.Parse(endpoint)
	if err != nil {
		return
	}

	if u.Scheme != "opc.tcp" {
		err = errors.Errorf("unsupported scheme %s", u.Scheme)
		return
	}

	network = "tcp"

	port := u.Port()
	if port == "" {
		port = defaultPort
	}

	u.Host = net.JoinHostPort(u.Hostname(), port)

	return
}
