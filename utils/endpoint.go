// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package utils

import (
	"fmt"
	"net"
	"strings"

	"github.com/gopcua/opcua/errors"
)

// ResolveEndpoint returns network type, address, and error splitted from EndpointURL.
//
// Expected format of input is "opc.tcp://<addr[:port]/path/to/somewhere"
func ResolveEndpoint(endpoint string) (network string, addr *net.TCPAddr, err error) {
	elems := strings.Split(endpoint, "/")
	if elems[0] != "opc.tcp:" {
		return "", nil, errors.NewErrUnsupported(elems[0], "should be in \"opc.tcp://<addr[:port]>/path/to/somewhere\" format.")
	}

	addrString := elems[2]
	if !strings.Contains(addrString, ":") {
		addrString += ":4840"
	}

	network = "tcp"
	addr, err = net.ResolveTCPAddr(network, addrString)
	switch err.(type) {
	case *net.DNSError:
		return "", nil, errors.New("could not resolve address")
	}
	return
}

// GetPath returns the path that follows after address[:port] in EndpointURL.
//
// Expected format of input is "opc.tcp://<addr[:port]/path/to/somewhere"
func GetPath(endpoint string) (path string, err error) {
	elems := strings.Split(endpoint, "/")
	if len(elems) < 3 {
		return "", fmt.Errorf("invalid input: %s", endpoint)
	}

	return "/" + strings.Join(elems[3:], "/"), nil
}
