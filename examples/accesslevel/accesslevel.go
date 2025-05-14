// Copyright 2018-2020 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"flag"
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
		ualog.Fatal("ParseNodeID failed", "error", err)
	}

	n := c.Node(id)
	accessLevel, err := n.AccessLevel(ctx)
	if err != nil {
		ualog.Fatal("AccessLevel failed", "error", err)
	}
	ualog.Info("Got access level", "access_level", accessLevel)

	userAccessLevel, err := n.UserAccessLevel(ctx)
	if err != nil {
		ualog.Fatal("AccessLevel failed", "error", err)
	}
	ualog.Info("Got user access level", "user_access_level", userAccessLevel)

	v, err := n.Value(ctx)
	switch {
	case err != nil:
		ualog.Fatal("Value failed", "error", err)
	case v == nil:
		ualog.Info("v == nil")
	default:
		ualog.Info("Got value", "value", v.Value())
	}
}
