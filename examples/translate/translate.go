// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/debug"
	"github.com/gopcua/opcua/id"
	"github.com/gopcua/opcua/ua"
)

func main() {
	endpoint := flag.String("endpoint", "opc.tcp://localhost:4840", "OPC UA Endpoint URL")
	flag.BoolVar(&debug.Enable, "debug", false, "enable debug logging")
	ns := flag.Int("namespace", 0, "namespace of node")

	// example: "device_led.temperature"
	nodePath := flag.String("path", "", "path of a node's browse name")
	flag.Parse()
	log.SetFlags(0)

	ctx := context.Background()

	c := opcua.NewClient(*endpoint)
	if err := c.Connect(ctx); err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	root := c.Node(ua.NewTwoByteNodeID(id.ObjectsFolder))
	nodeId, err := root.TranslateBrowsePathInSameNamespaceToNodeId(uint8(*ns), *nodePath)
	if err != nil {
		log.Fatal(err)
		return
	}
	fmt.Println(nodeId)
}
