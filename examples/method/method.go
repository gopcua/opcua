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

	in := int64(12)
	req := &ua.CallMethodRequest{
		ObjectID:       ua.NewStringNodeID(2, "main"),
		MethodID:       ua.NewStringNodeID(2, "even"),
		InputArguments: []*ua.Variant{ua.MustVariant(in)},
	}

	resp, err := c.Call(ctx, req)
	if err != nil {
		ualog.Fatal("Call failed", "error", err)
	}
	if got, want := resp.StatusCode, ua.StatusOK; got != want {
		ualog.Fatal("status not OK", "got", got, "want", want)
	}
	out := resp.OutputArguments[0].Value()
	slog.Info("result", "in", in, "out", out)
}
