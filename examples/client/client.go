// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

/*
Command client provides a connection establishment of OPC UA Secure Conversation.
*/
package main

func main() {
	// 	var (
	// 		endpoint = flag.String("endpoint", "opc.tcp://example.com/foo/bar", "OPC UA Endpoint URL")
	// 		// payloadHex = flag.String("payload", "deadbeef", "Payload to send in hex stream format")
	// 	)
	// 	flag.Parse()

	// 	// Create context for UACP to be used by statemachine working background.
	// 	ctx := context.Background()
	// 	ctx, cancel := context.WithCancel(ctx)
	// 	defer cancel()

	// 	// Establish UACP Connection with the Endpoint specified.
	// 	conn, err := uacp.Dial(ctx, *endpoint)
	// 	if err != nil {
	// 		log.Fatal(err)
	// 	}
	// 	defer func() {
	// 		if err := conn.Close(); err != nil {
	// 			log.Fatalf("Failed to shutdown connection: %s", err)
	// 		}
	// 		log.Println("Successfully shutdown connection")
	// 	}()
	// 	log.Printf("Successfully established connection with %v", conn.RemoteEndpoint())

	// 	// Open SecureChannel on top of UACP Connection established above.
	// 	cfg := uasc.NewClientConfigSecurityNone(3333, 3600000)
	// 	secChan, err := uasc.OpenSecureChannel(ctx, conn, cfg, 5*time.Second, 3)
	// 	if err != nil {
	// 		log.Fatal(err)
	// 	}
	// 	defer func() {
	// 		if err := secChan.Close(); err != nil {
	// 			log.Fatalf("Failed to close secure channel: %s", err)
	// 		}
	// 		log.Printf("Successfully closed secure channel with %v", conn.RemoteEndpoint())
	// 	}()
	// 	log.Printf("Successfully opened secure channel with %v", secChan.RemoteEndpoint())

	// 	// Send FindServersRequest to remote Endpoint.
	// 	if err := secChan.FindServersRequest([]string{"ja-JP", "de-DE", "en-US"}, "gopcua-server"); err != nil {
	// 		log.Fatal(err)
	// 	}
	// 	log.Println("Successfully sent FindServersRequest")
	// 	time.Sleep(500 * time.Millisecond)

	// 	// Send GetEndpointsRequest to remote Endpoint.
	// 	if err := secChan.GetEndpointsRequest([]string{"ja-JP", "de-DE", "en-US"}, []string{"gopcua-server"}); err != nil {
	// 		log.Fatal(err)
	// 	}
	// 	log.Println("Successfully sent GetEndpointsRequest")
	// 	time.Sleep(500 * time.Millisecond)

	// 	sessCfg := uasc.NewClientSessionConfig(
	// 		[]string{"ja-JP"},
	// 		datatypes.NewAnonymousIdentityToken("anonymous"),
	// 	)
	// 	session, err := uasc.CreateSession(ctx, secChan, sessCfg, 3, 5*time.Second)
	// 	if err != nil {
	// 		log.Fatal(err)
	// 	}
	// 	defer func() {
	// 		session.Close()
	// 		log.Printf("Successfully closed secure channel with %v", secChan.RemoteEndpoint())
	// 	}()
	// 	log.Printf("Successfully created secure channel with %v", secChan.RemoteEndpoint())

	// 	if err := session.Activate(); err != nil {
	// 		log.Fatal(err)
	// 	}
	// 	log.Printf("Successfully activated secure channel with %v", secChan.RemoteEndpoint())

	// 	if err := session.ReadRequest(
	// 		2000, services.TimestampsToReturnBoth, datatypes.NewReadValueID(
	// 			datatypes.NewNumericNodeID(0, 11111), datatypes.IntegerIDValue, "", 0, "",
	// 		),
	// 	); err != nil {
	// 		log.Fatal(err)
	// 	}
	// 	log.Println("Successfully sent ReadRequest")

	// 	// Send arbitrary payload on top of UASC SecureChannel.
	// 	// payload, err := hex.DecodeString(*payloadHex)
	// 	// if err != nil {
	// 	// 	log.Fatal(err)
	// 	// }
	// 	// if err := secChan.WriteService(payload); err != nil {
	// 	// 	log.Fatal(err)
	// 	// }
	// 	// log.Printf("Successfully sent message: %x\n%s", payload, utils.Wireshark(0, payload))
	// 	time.Sleep(500 * time.Millisecond)
}
