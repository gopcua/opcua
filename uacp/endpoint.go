// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uacp

import (
	"context"
	"fmt"
	"net"
	"net/url"
)

const defaultPort = "4840"

// ResolveEndpoint returns network type, address, and error splitted from EndpointURL.
//
// Expected format of input is "opc.tcp://<addr[:port]/path/to/somewhere"
func ResolveEndpoint(ctx context.Context, endpoint string) (network string, url *url.URL, err error) {
	url, err = url.Parse(endpoint)
	if err != nil {
		return
	}

	if url.Scheme != "opc.tcp" {
		err = fmt.Errorf("unsupported scheme %s", url.Scheme)
		return
	}

	network = "tcp"

	port := url.Port()
	if port == "" {
		port = defaultPort
	}

	var resolver net.Resolver

	addrs, err := resolver.LookupIPAddr(ctx, url.Hostname())
	if err != nil {
		return
	}

	if len(addrs) == 0 {
		err = fmt.Errorf("could not resolve address %s", url.Hostname())
		return
	}

	url.Host = net.JoinHostPort(addrs[0].String(), port)

	return
}
