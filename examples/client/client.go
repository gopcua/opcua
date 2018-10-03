// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

/*
Command client provides a connection establishment of OPC UA Secure Conversation.

XXX - Currently this command just initiates the connection(UACP) to the specified endpoint.
*/
package main

import (
	"context"
	"encoding/hex"
	"flag"
	"log"
	"time"

	"github.com/wmnsk/gopcua/uacp"
	"github.com/wmnsk/gopcua/uasc"
	"github.com/wmnsk/gopcua/utils"
)

func main() {
	var (
		endpoint   = flag.String("endpoint", "opc.tcp://example.com/foo/bar", "OPC UA Endpoint URL")
		payloadHex = flag.String("payload", "deadbeef", "Payload to send in hex stream format")
	)
	flag.Parse()

	// Create context for UACP to be used by statemachine working background.
	uacpCtx := context.Background()
	uacpCtx, cancel := context.WithCancel(uacpCtx)
	defer cancel()

	// Establish UACP Connection with the Endpoint specified.
	// No need for conn.Close(), as context handles the cancellation.
	conn, err := uacp.Dial(uacpCtx, *endpoint)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Successfully established connection with %v", conn.RemoteEndpoint())

	// Create context for UASC to be used by statemachine working background.
	uascCtx, cancel := context.WithCancel(uacpCtx)
	defer cancel()

	// Open SecureChannel on top of UACP Connection established above.
	// No need for secChan.Close(), as context handles the cancellation.
	cfg := uasc.NewConfig(
		1, "http://opcfoundation.org/UA/SecurityPolicy#None", nil, nil, 0, 0,
	)
	secChan, err := uasc.OpenSecureChannel(uascCtx, conn, cfg, 1, 1, nil)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Successfully opened secure channel with %v", conn.RemoteEndpoint())

	// Send FindServersRequest to remote Endpoint.
	if err := secChan.FindServersRequest([]string{"ja-JP", "de-DE", "en-US"}, []string{"gopcua-server"}); err != nil {
		log.Fatal(err)
	}
	log.Println("Successfully sent FindServersRequest")
	time.Sleep(1 * time.Second)

	// Send GetEndpointsRequest to remote Endpoint.
	if err := secChan.GetEndpointsRequest([]string{"ja-JP", "de-DE", "en-US"}, []string{"gopcua-server"}); err != nil {
		log.Fatal(err)
	}
	log.Println("Successfully sent GetEndpointsRequest")
	time.Sleep(1 * time.Second)

	// Send arbitrary payload on top of UASC SecureChannel.
	payload, err := hex.DecodeString(*payloadHex)
	if err != nil {
		log.Fatal(err)
	}
	if _, err := secChan.WriteService(payload); err != nil {
		log.Fatal(err)
	}
	log.Printf("Successfully sent message: %x\n%s", payload, utils.Wireshark(0, payload))
	time.Sleep(1 * time.Second)

	// Send CloseSecureChannelRequest to remote Endpoint.
	if err := secChan.CloseSecureChannelRequest(); err != nil {
		log.Fatal(err)
	}
	log.Println("Successfully sent CloseSecureChannelRequest")
	time.Sleep(1 * time.Second)
}
