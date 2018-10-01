// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

// Command sender provides a "manual establishment" of OPC UA Secure Conversation.
//
// The built-in connection establishment API is to be implemented in the package itself in the future.
package main

import (
	"flag"
	"log"
	"net"
	"time"

	"github.com/wmnsk/gopcua/datatypes"
	"github.com/wmnsk/gopcua/services"
	"github.com/wmnsk/gopcua/uacp"
	"github.com/wmnsk/gopcua/uasc"
)

func main() {
	var (
		ip     = flag.String("ip", "127.0.0.1", "Destination IP Address")
		port   = flag.String("port", "11111", "Destination Port Number")
		sndBuf = flag.Int("sndbuf", 65535, "SendBufferSize")
		rcvBuf = flag.Int("rcvbuf", 65535, "ReceiveBufferSize")
		maxMsg = flag.Int("maxmsg", 0, "MaxMessageSize")
		url    = flag.String("url", "opc.tcp://deadbeef.example/foo/bar", "OPC UA Endpoint URL")
		uri    = flag.String("uri", "http://opcfoundation.org/UA/SecurityPolicy#None", "OPC UA Secure Policy URI")
	)
	flag.Parse()

	// Setup Hello
	hello, err := uacp.NewHello(0, uint32(*sndBuf), uint32(*rcvBuf), uint32(*maxMsg), *url).Serialize()
	if err != nil {
		log.Fatalf("Failed to serialize Hello: %s", err)
	}

	// Setup OpenSecureChannel
	cfg := &uasc.Config{
		SecureChannelID:   1,
		SecurityPolicyURI: *uri,
		RequestID:         1,
		SecurityTokenID:   0,
		SequenceNumber:    0,
	}

	g := services.NewGetEndpointsRequest(
		services.NewRequestHeader(
			datatypes.NewTwoByteNodeID(0), time.Now(), 0, 0,
			0xffff, "", services.NewNullAdditionalHeader(), nil,
		),
		*url, nil, nil,
	)
	g.SetDiagAll()

	o := services.NewOpenSecureChannelRequest(
		services.NewRequestHeader(
			datatypes.NewTwoByteNodeID(0), time.Now(), 0, 0,
			0xffff, "", services.NewNullAdditionalHeader(), nil,
		),
		0, services.ReqTypeIssue, services.SecModeNone, 6000000, nil,
	)
	o.SetDiagAll()

	// Prepare TCP connection
	raddr, err := net.ResolveTCPAddr("tcp", *ip+":"+*port)
	if err != nil {
		log.Fatalf("Failed to resolve TCP Address: %s", err)
	}

	conn, err := net.DialTCP("tcp", nil, raddr)
	if err != nil {
		log.Fatalf("Failed to open TCP connection: %s", err)
	}
	defer conn.Close()

	// Send Hello and wait for Acknowledge to come
	if _, err := conn.Write(hello); err != nil {
		log.Fatalf("Failed to write Hello: %s", err)
	}
	log.Printf("Successfully sent Hello: %x", hello)

	buf := make([]byte, 10000)
	n, err := conn.Read(buf)
	if err != nil {
		log.Fatalf("Failed to read from conn: %s", err)
	}

	cp, err := uacp.Decode(buf[:n])
	if err != nil {
		log.Fatalf("Something went wrong: %s", err)
	}

	switch cp.MessageTypeValue() {
	case uacp.MessageTypeAcknowledge:
		log.Printf("Received Acknowledge: %s", cp)

		// Send OpenSecureChannelRequest and wait for Response to come
		opn, err := uasc.New(o, cfg).Serialize()
		if err != nil {
			log.Fatalf("Failed to serialize OpenSecureChannel: %s", err)
		}
		cfg.SequenceNumber++

		if _, err := conn.Write(opn); err != nil {
			log.Fatalf("Failed to write OpenSecureChannel: %s", err)
		}
		log.Printf("Successfully sent OpenSecureChannel: %x", opn)

		l, err := conn.Read(buf)
		if err != nil {
			log.Fatalf("Failed to read from conn: %s", err)
		}

		// Decode OpenSecureChannelResponse and retrieve the values told from server.
		sc, err := uasc.Decode(buf[:l])
		if err != nil {
			log.Printf("Something went wrong: %s", err)
		}
		log.Printf("Received: %s\nRaw: %x", sc, buf[:l])

		osc, ok := sc.Service.(*services.OpenSecureChannelResponse)
		if !ok {
			log.Fatal("Assertion failed.")
		}
		cfg.SecureChannelID = osc.SecurityToken.ChannelID
		cfg.SecurityTokenID = osc.SecurityToken.TokenID

		// Send GetEndpointsRequest and wait for Response to come
		gep, err := uasc.New(g, cfg).Serialize()
		if err != nil {
			log.Fatalf("Failed to serialize GetEndpointsRequest: %s", err)
		}
		cfg.SequenceNumber++

		if _, err := conn.Write(gep); err != nil {
			log.Fatalf("Failed to write GetEndpoints: %s", err)
		}
		log.Printf("Successfully sent GetEndpoints: %x", gep)

		m, err := conn.Read(buf)
		if err != nil {
			log.Fatalf("Failed to read from conn: %s", err)
		}

		gres, err := uasc.Decode(buf[:m])
		if err != nil {
			log.Printf("Something went wrong: %s", err)
		}
		log.Printf("Received: %s\nRaw: %x", gres, buf[:m])

		c := services.NewCloseSecureChannelRequest(
			services.NewRequestHeader(
				datatypes.NewTwoByteNodeID(0), time.Now(), 0, 0,
				0xffff, "", services.NewNullAdditionalHeader(), nil,
			), cfg.SecureChannelID,
		)
		c.SetDiagAll()
		clo, err := uasc.New(c, cfg).Serialize()
		if err != nil {
			log.Fatalf("Failed to serialize CloseSecureChannelRequest: %s", err)
		}

		if _, err := conn.Write(clo); err != nil {
			log.Fatalf("Failed to write CloseSecureChannel: %s", err)
		}
		log.Printf("Successfully sent CloseSecureChannel: %x", gep)

	case uacp.MessageTypeError:
		log.Fatalf("Received Error, closing: %s", cp)
	default:
		log.Fatalf("Received unexpected message, closing: %s", cp)
	}
}
