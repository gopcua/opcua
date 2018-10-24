// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"flag"
	"log"
	"time"

	"github.com/wmnsk/gopcua/datatypes"
	"github.com/wmnsk/gopcua/services"
	"github.com/wmnsk/gopcua/uacp"
	"github.com/wmnsk/gopcua/uasc"
)

func main() {
	endpoint := flag.String("endpoint", "opc.tcp://example.com/foo/bar", "OPC UA Endpoint URL")
	flag.Parse()

	// Create context for UACP to be used by statemachine working background.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Establish UACP Connection with the Endpoint specified.
	conn, err := uacp.Dial(ctx, *endpoint)
	if err != nil {
		log.Fatal("uacp.Dial: ", err)
	}
	defer conn.Close()
	log.Printf("Successfully established connection with %v", conn.RemoteEndpoint())

	// Open SecureChannel on top of UACP Connection established above.
	cfg := uasc.NewClientConfigSecurityNone(3333, 3600000)
	secChan, err := uasc.OpenSecureChannel(ctx, conn, cfg, 5*time.Second, 3)
	if err != nil {
		log.Fatal(err)
	}
	defer secChan.Close()
	log.Printf("Successfully created secure channel with %v", secChan.RemoteEndpoint())

	sessCfg := uasc.NewClientSessionConfig(nil, datatypes.NewAnonymousIdentityToken(""))
	session, err := uasc.CreateSession(ctx, secChan, sessCfg, 3, 5*time.Second)
	if err != nil {
		log.Fatal(err)
	}
	defer session.Close()
	log.Printf("Successfully created session with %v", secChan.RemoteEndpoint())

	if err := session.Activate(); err != nil {
		log.Fatal(err)
	}
	log.Printf("Successfully activated session with %v", secChan.RemoteEndpoint())

	if err := session.ReadRequest(
		2000, services.TimestampsToReturnBoth, datatypes.NewReadValueID(
			datatypes.NewFourByteNodeID(0, 2258), datatypes.IntegerIDValue, "", 0, "",
		),
	); err != nil {
		log.Fatal(err)
	}
	log.Println("Successfully sent ReadRequest")

	// read response
	svc, err := session.ReadService()
	if err != nil {
		log.Fatal("ReadService: ", err)
	}
	log.Printf("svc: %#v", svc)
}
