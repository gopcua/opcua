// Copyright 2018-2020 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

// Package main provides an example to query the available endpoints of a server.
package main

import (
	"context"
	"flag"
	"fmt"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/internal/ualog"
)

func main() {
	var (
		endpoint = flag.String("endpoint", "opc.tcp://localhost:4840", "OPC UA Endpoint URL")
		debug    = flag.Bool("debug", false, "enable debug logging")
	)
	flag.Parse()
	ualog.SetDebugLogger(*debug)

	eps, err := opcua.GetEndpoints(context.Background(), *endpoint)
	if err != nil {
		ualog.Fatal("GetEndpoints failed", "error", err)
	}

	for i, ep := range eps {
		ualog.Info(fmt.Sprintf("%d.", i), "ep_url", ep.EndpointURL, "sec_policy", ep.SecurityPolicyURI, "sec_mode", ep.SecurityMode)
	}
}
