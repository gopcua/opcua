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
		debug    = flag.Bool("debug", false, "enable debug logging")
	)
	flag.Parse()
	slog.SetDefault(slog.New(ualog.NewTextHandler(*debug)))

	ctx := context.Background()

	c, err := opcua.NewClient(*endpoint, opcua.SecurityMode(ua.MessageSecurityModeNone))
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

	regResp, err := c.RegisterNodes(ctx, &ua.RegisterNodesRequest{
		NodesToRegister: []*ua.NodeID{id},
	})
	if err != nil {
		ualog.Fatal("RegisterNodes failed", "error", err)
	}

	req := &ua.ReadRequest{
		MaxAge: 2000,
		NodesToRead: []*ua.ReadValueID{
			{NodeID: regResp.RegisteredNodeIDs[0]},
		},
		TimestampsToReturn: ua.TimestampsToReturnBoth,
	}

	resp, err := c.Read(ctx, req)
	if err != nil {
		ualog.Fatal("Read failed", "error", err)
	}
	if resp.Results[0].Status != ua.StatusOK {
		ualog.Fatal("Status not OK", "status_code", resp.Results[0].Status)
	}
	ualog.Info(fmt.Sprintf("Value: %#v", resp.Results[0].Value.Value()))

	_, err = c.UnregisterNodes(ctx, &ua.UnregisterNodesRequest{
		NodesToUnregister: []*ua.NodeID{id},
	})
	if err != nil {
		ualog.Fatal("UnregisterNodes failed", "error", err)
	}
}
