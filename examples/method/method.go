// Copyright 2018-2020 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"flag"
	"log"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/debug"
	"github.com/gopcua/opcua/ua"
)

func main() {
	var (
		endpoint = flag.String("endpoint", "opc.tcp://localhost:4840", "OPC UA Endpoint URL")
	)
	flag.BoolVar(&debug.Enable, "debug", false, "enable debug logging")
	flag.Parse()
	log.SetFlags(0)

	ctx := context.Background()

	c := opcua.NewClient(*endpoint, opcua.SecurityMode(ua.MessageSecurityModeNone))
	if err := c.Connect(ctx); err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	in := int64(12)
	req := &ua.CallMethodRequest{
		ObjectID:       ua.NewStringNodeID(2, "main"),
		MethodID:       ua.NewStringNodeID(2, "even"),
		InputArguments: []*ua.Variant{ua.MustVariant(in)},
	}

	resp, err := c.Call(req)
	if err != nil {
		log.Fatal(err)
	}
	if got, want := resp.StatusCode, ua.StatusOK; got != want {
		log.Fatalf("got status %v want %v", got, want)
	}
	out := resp.OutputArguments[0].Value()
	log.Printf("%d is even: %v", in, out)
}
