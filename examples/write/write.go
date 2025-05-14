// Copyright 2018-2020 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/internal/ualog"
	"github.com/gopcua/opcua/ua"
)

func main() {
	var (
		endpoint = flag.String("endpoint", "opc.tcp://localhost:4840", "OPC UA Endpoint URL")
		nodeID   = flag.String("node", "", "NodeID to read")
		value    = flag.String("value", "", "value")
		debug    = flag.Bool("debug", false, "enable debug logging")
	)
	flag.Parse()
	slog.SetDefault(slog.New(ualog.NewTextHandler(*debug)))

	ctx := context.Background()

	c, err := opcua.NewClient(*endpoint)
	if err != nil {
		ualog.Fatal("NewClient failed", "error", err)
	}
	if err := c.Connect(ctx); err != nil {
		ualog.Fatal("Connect failed", "error", err)
	}
	defer c.Close(ctx)

	id, err := ua.ParseNodeID(*nodeID)
	if err != nil {
		ualog.Fatal("invalid node id", "node_id", *nodeID, "error", err)
	}

	v, err := ua.NewVariant(*value)
	if err != nil {
		ualog.Fatal("invalid value", "error", err)
	}

	req := &ua.WriteRequest{
		NodesToWrite: []*ua.WriteValue{
			{
				NodeID:      id,
				AttributeID: ua.AttributeIDValue,
				Value: &ua.DataValue{
					EncodingMask: ua.DataValueValue,
					Value:        v,
				},
			},
		},
	}

	resp, err := c.Write(ctx, req)
	if err != nil {
		ualog.Fatal("Write failed", "error", err)
	}
	ualog.Info(fmt.Sprintf("Value: %v", resp.Results[0]))
}
