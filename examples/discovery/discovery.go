// Copyright 2018-2020 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/debug"
	"log"
)

func main() {
	endpoint := flag.String("endpoint", "opc.tcp://localhost:4840", "OPC UA Endpoint URL")
	flag.BoolVar(&debug.Enable, "debug", false, "enable debug logging")
	flag.Parse()
	log.SetFlags(0)

	ctx := context.Background()

	{
		log.Println("Finding servers on network")
		servers, err := opcua.FindServersOnNetwork(ctx, *endpoint)
		if err != nil {
			log.Printf("Error calling find servers on network: %v", err)
		} else {
			for i, server := range servers {
				fmt.Printf("%d Server on network:\n", i)
				fmt.Printf("  -- RecordID: %v\n", server.RecordID)
				fmt.Printf("  -- ServerName: %v\n", server.ServerName)
				fmt.Printf("  -- DiscoveryURL: %v\n", server.DiscoveryURL)
				fmt.Printf("  -- ServerCapabilities: %v\n", server.ServerCapabilities)
			}
		}
	}

	{
		log.Println("Finding servers")
		servers, err := opcua.FindServers(ctx, *endpoint)
		if err != nil {
			log.Fatal(err)
		}
		for i, server := range servers {
			fmt.Printf("%dth Server:\n", i+1)
			fmt.Printf("  -- ApplicationURI: %v\n", server.ApplicationURI)
			fmt.Printf("  -- ProductURI: %v\n", server.ProductURI)
			fmt.Printf("  -- ApplicationName: %v\n", server.ApplicationName)
			fmt.Printf("  -- ApplicationType: %v\n", server.ApplicationType)
			fmt.Printf("  -- GatewayServerURI: %v\n", server.GatewayServerURI)
			fmt.Printf("  -- DiscoveryProfileURI: %v\n", server.DiscoveryProfileURI)
			fmt.Printf("  -- DiscoveryURLs: %v\n", server.DiscoveryURLs)
		}
	}
}
