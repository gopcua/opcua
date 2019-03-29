// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

// Package main provides an example to query the available endpoints of a server.
package main

import (
	"flag"
	"log"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/debug"
)

func main() {
	var endpoint = flag.String("endpoint", "opc.tcp://localhost:4840", "OPC UA Endpoint URL")
	flag.BoolVar(&debug.Enable, "debug", false, "enable debug logging")
	flag.Parse()
	log.SetFlags(0)

	eps, err := opcua.GetEndpoints(*endpoint)
	if err != nil {
		log.Fatal(err)
	}

	for _, ep := range eps {
		log.Println(ep.EndpointURL, ep.SecurityPolicyURI, ep.SecurityMode)
	}
}
