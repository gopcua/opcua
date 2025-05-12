// Copyright 2018-2020 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"flag"
	"fmt"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/id"
	"github.com/gopcua/opcua/internal/ualog"
	"github.com/gopcua/opcua/ua"
)

func main() {
	var (
		endpoint = flag.String("endpoint", "opc.tcp://localhost:4840", "OPC UA Endpoint URL")
		nodePath = flag.String("path", "device_led.temperature", "path of a node's browse name")
		ns       = flag.Int("namespace", 0, "namespace of the node")
		debug    = flag.Bool("debug", false, "enable debug logging")
	)
	flag.Parse()
	ualog.SetDebugLogger(*debug)

	ctx := context.Background()

	c, err := opcua.NewClient(*endpoint)
	if err != nil {
		ualog.Fatal("NewClient failed", "error", err)
	}
	if err := c.Connect(ctx); err != nil {
		ualog.Fatal("Connect failed", "error", err)
	}
	defer c.Close(ctx)

	root := c.Node(ua.NewTwoByteNodeID(id.ObjectsFolder))
	nodeID, err := root.TranslateBrowsePathInNamespaceToNodeID(ctx, uint16(*ns), *nodePath)
	if err != nil {
		ualog.Fatal("TranslateBrowsePathInNamespaceToNodeID failed", "error", err)
	}
	fmt.Println(nodeID)
}
