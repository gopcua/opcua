// Copyright 2018-2020 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"errors"
	"flag"
	"io"
	"log"
	"time"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/debug"
	"github.com/gopcua/opcua/ua"
)

func main() {
	var (
		endpoint = flag.String("endpoint", "opc.tcp://localhost:4840", "OPC UA Endpoint URL")
		nodeID   = flag.String("node", "", "NodeID to read")
		attrID   = flag.Int("attr", int(ua.AttributeIDValue), "attribute to read (default to value)")
	)
	flag.BoolVar(&debug.Enable, "debug", false, "enable debug logging")
	flag.Parse()
	log.SetFlags(0)

	ctx := context.Background()

	c, err := opcua.NewClient(*endpoint, opcua.SecurityMode(ua.MessageSecurityModeNone))
	if err != nil {
		log.Fatal(err)
	}
	if err := c.Connect(ctx); err != nil {
		log.Fatal(err)
	}
	defer c.Close(ctx)

	id, err := ua.ParseNodeID(*nodeID)
	if err != nil {
		log.Fatalf("invalid node id: %v", err)
	}

	req := &ua.ReadRequest{
		MaxAge: 2000,
		NodesToRead: []*ua.ReadValueID{
			{NodeID: id, AttributeID: ua.AttributeID(*attrID)},
		},
		TimestampsToReturn: ua.TimestampsToReturnBoth,
	}

	var resp *ua.ReadResponse
	for {
		resp, err = c.Read(ctx, req)
		if err == nil {
			break
		}

		// Following switch contains known errors that can be retried by the user.
		// Best practice is to do it on read operations.
		switch {
		case err == io.EOF && c.State() != opcua.Closed:
			// has to be retried unless user closed the connection
			time.After(1 * time.Second)
			continue

		case errors.Is(err, ua.StatusBadSessionIDInvalid):
			// Session is not activated has to be retried. Session will be recreated internally.
			time.After(1 * time.Second)
			continue

		case errors.Is(err, ua.StatusBadSessionNotActivated):
			// Session is invalid has to be retried. Session will be recreated internally.
			time.After(1 * time.Second)
			continue

		case errors.Is(err, ua.StatusBadSecureChannelIDInvalid):
			// secure channel will be recreated internally.
			time.After(1 * time.Second)
			continue

		default:
			log.Fatalf("Read failed: %s", err)
		}
	}

	if resp != nil && resp.Results[0].Status != ua.StatusOK {
		log.Fatalf("Status not OK: %v", resp.Results[0].Status)
	}

	switch v := resp.Results[0].Value.Value().(type) {
	case *ua.NodeID:
		log.Print(v.String())
		if err := c.RegisterExtensionObjectFromServer(ctx, v); err != nil {
			log.Fatalf("%v", err)
		}
		log.Print("registered ", v.String())
	default:
		log.Printf("%#v", v)
	}

}
