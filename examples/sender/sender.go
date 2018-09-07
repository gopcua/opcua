// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

// Command sender provides a "manual establishment" of OPC UA Secure Conversation.
//
// The built-in connection establishment API is to be implemented in the package itself in the future.
package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/wmnsk/gopcua/datatypes"
	"github.com/wmnsk/gopcua/services"
	"github.com/wmnsk/gopcua/uacp"
	"github.com/wmnsk/gopcua/uasc"
	"github.com/wmnsk/gopcua/utils"
)

func main() {
	var (
		ip = flag.String("ip", "172.19.191.12", "Destination IP Address")
		// ip     = flag.String("ip", "172.19.1.76", "Destination IP Address")
		port = flag.String("port", "4840", "Destination Port Number")
		// port   = flag.String("port", "26543", "Destination Port Number")
		sndBuf = flag.Int("sndbuf", 65535, "SendBufferSize")
		rcvBuf = flag.Int("rcvbuf", 65535, "ReceiveBufferSize")
		maxMsg = flag.Int("maxmsg", 0, "MaxMessageSize")
		url    = flag.String("url", "opc.tcp://172.19.191.12:4840/hbk/clipx", "OPC UA Endpoint URL")
		// url = flag.String("url", "opc.tcp://zeisig.devel.hbm.com:26543", "OPC UA Endpoint URL")
		uri = flag.String("uri", "http://opcfoundation.org/UA/SecurityPolicy#None", "OPC UA Secure Policy URI")
	)
	flag.Parse()

	// Setup Hello
	hello, err := uacp.NewHello(0, uint32(*sndBuf), uint32(*rcvBuf), uint32(*maxMsg), *url).Serialize()
	if err != nil {
		log.Fatalf("Failed to serialize Hello: %s", err)
	}

	// Setup OpenSecureChannel
	cfg := &uasc.Config{
		SecureChannelID:   0,
		SecurityPolicyURI: *uri,
		RequestID:         1,
		SecurityTokenID:   0,
		SequenceNumber:    0,
	}

	g := services.NewGetEndpointsRequest(
		time.Now(), 1, 0, 0,
		"", *url, nil, nil,
	)
	g.SetDiagAll()

	o := services.NewOpenSecureChannelRequest(
		time.Now(), 0, 1, 0, 0, "", 0,
		services.ReqTypeIssue, services.SecModeNone,
		6000000, nil,
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

		// Send OpenSecureChannelRequest and wait for Resposne to come
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

		// create session request
		createSessionRequest := &services.CreateSessionRequest{
			TypeID: &datatypes.ExpandedNodeID{
				NodeID: datatypes.NewFourByteNodeID(0, services.ServiceTypeCreateSessionRequest),
			},
			RequestHeader: services.NewRequestHeader(
				datatypes.NewTwoByteNodeID(0),
				time.Now(), 1, 0, 0, "",
				services.NewNullAdditionalHeader(),
				nil,
			),
			ClientDescription: services.NewApplicationDescription(
				"urn:zeisig.devel.hbm.com:ProsysOpcUaClient",
				"urn:prosysopc.com:ProsysOpcUaClient",
				"ProsysOpcUaClient",
				services.AppTypeClient,
				"",
				"",
				nil,
			),
			ServerURI:               datatypes.NewString("urn:zeisig.devel.hbm.com:NodeOPCUA-Server"),
			EndpointURL:             datatypes.NewString("opc.tcp://zeisig.devel.hbm.com:26543"),
			SessionName:             datatypes.NewString("session name"),
			ClientNonce:             datatypes.NewByteString(nil),
			ClientCertificate:       datatypes.NewByteString(nil),
			RequestedSessionTimeout: 0x41124f8000000000,
			MaxResponseMessageSize:  0,
		}
		createSessionRequestService, err := uasc.New(createSessionRequest, cfg).Serialize()
		if err != nil {
			log.Fatalf("Failed to serialize CreateSessionRequest: %s", err)
		}
		cfg.SequenceNumber++

		if _, err := conn.Write(createSessionRequestService); err != nil {
			log.Fatalf("Failed to write CreateSessionRequestService: %s", err)
		}
		log.Printf("Successfully sent CreateSessionRequestService: %x", createSessionRequestService)

		// get create session request response
		createSessionResponseLength, err := conn.Read(buf)
		if err != nil {
			log.Fatalf("Failed to read from conn: %s", err)
		}

		createSessionResponse, err := uasc.Decode(buf[:createSessionResponseLength])
		if err != nil {
			log.Printf("Something went wrong: %s", err)
		}
		log.Printf("Received: %s\nRaw: %x", createSessionResponse, buf[:createSessionResponseLength])
		csr, ok := createSessionResponse.Service.(*services.CreateSessionResponse)
		if !ok {
			log.Fatal("create session response assertion failed")
		}

		// activate session request
		activateSessionRequest := &services.ActivateSessionRequest{
			TypeID: &datatypes.ExpandedNodeID{
				NodeID: datatypes.NewFourByteNodeID(
					0,
					services.ServiceTypeActivateSessionRequest,
				),
			},
			RequestHeader: services.NewRequestHeader(
				// datatypes.NewOpaqueNodeID(0, authenticationToken.Identifier),
				// datatypes.NewOpaqueNodeID(0, tokenBytes),
				// datatypes.NewNumericNodeID(0, authenticationToken.Identifier),
				csr.AuthenticationToken,
				time.Now(), 1, 0, 0, "",
				services.NewNullAdditionalHeader(),
				nil,
			),
			ClientSignature:            services.NewSignatureData("", nil),
			ClientSoftwareCertificates: services.NewSignedSoftwareCertificateArray(nil),
			LocaleIDs:                  datatypes.NewStringArray(nil),
			UserIdentityToken: &datatypes.ExtensionObject{
				TypeID: &datatypes.ExpandedNodeID{
					NodeID: datatypes.NewFourByteNodeID(0, 321),
				},
				Length:       5,
				EncodingMask: 0x01,
				Body:         datatypes.NewByteString([]byte("0")),
			},
			UserTokenSignature: services.NewSignatureData("", nil),
		}
		activateSessionRequestService, err := uasc.New(activateSessionRequest, cfg).Serialize()
		if err != nil {
			log.Fatalf("Failed to serialize ActivateSessionRequest: %s", err)
		}
		cfg.SequenceNumber++
		fmt.Println(utils.Wireshark(2, activateSessionRequestService))

		if _, err := conn.Write(activateSessionRequestService); err != nil {
			log.Fatalf("Failed to write ActivateSessionRequestService: %s", err)
		}
		log.Printf("Successfully sent ActivateSessionRequestService: %x", activateSessionRequestService)
		// get activate session response
		activateSessionResponseLength, err := conn.Read(buf)
		if err != nil {
			log.Fatalf("Failed to read from conn: %s", err)
		}
		log.Printf("Received: Raw: %x", buf[:activateSessionResponseLength])

		// Send GetEndpointsRequest and wait for Resposne to come
		// gep, err := uasc.New(g, cfg).Serialize()
		// if err != nil {
		// 	log.Fatalf("Failed to serialize GetEndpointsRequest: %s", err)
		// }
		// cfg.SequenceNumber++

		// if _, err := conn.Write(gep); err != nil {
		// 	log.Fatalf("Failed to write GetEndpoints: %s", err)
		// }
		// log.Printf("Successfully sent GetEndpoints: %x", gep)

		// m, err := conn.Read(buf)
		// if err != nil {
		// 	log.Fatalf("Failed to read from conn: %s", err)
		// }

		// gres, err := uasc.Decode(buf[:m])
		// if err != nil {
		// 	log.Printf("Something went wrong: %s", err)
		// }
		// log.Printf("Received: %s\nRaw: %x", gres, buf[:m])

		// send read request
		// variant type: double
		readRequest := &services.ReadRequest{
			TypeID: &datatypes.ExpandedNodeID{
				NodeID: datatypes.NewFourByteNodeID(0, services.ServiceTypeReadRequest),
			},
			// RequestHeader:      o.RequestHeader,
			RequestHeader: services.NewRequestHeader(
				csr.AuthenticationToken,
				time.Now(), 1, 0, 0, "",
				services.NewNullAdditionalHeader(),
				nil,
			),
			MaxAge:             0,
			TimestampsToReturn: services.TimestampsToReturnNeither,
			NodesToRead: &datatypes.ReadValueIDArray{
				ArraySize: 1,
				ReadValueIDs: []*datatypes.ReadValueID{
					{
						NodeID:       datatypes.NewNumericNodeID(1, 103),
						AttributeID:  datatypes.IntegerIDValue,
						IndexRange:   datatypes.NewString(""),
						DataEncoding: datatypes.NewQualifiedName(0, ""),
					},
				},
			},
		}
		readRequestBytes, err := uasc.New(readRequest, cfg).Serialize()
		fmt.Println(readRequestBytes)
		fmt.Printf("% x", readRequestBytes)
		if err != nil {
			log.Fatalf("Failed to serialize ReadRequest: %s", err)
		}
		cfg.SequenceNumber++
		if _, err := conn.Write(readRequestBytes); err != nil {
			log.Fatalf("Failed to write ReadRequest: %s", err)
		}
		log.Printf("Successfully sent ReadRequest: %x", readRequestBytes)

		readResponseLength, err := conn.Read(buf)
		if err != nil {
			log.Fatalf("Failed to read from conn: %s", err)
		}
		log.Printf("Received: Raw: %x", buf[:readResponseLength])

		// gep, err := uasc.New(g, cfg).Serialize()
		// cfg.SequenceNumber++

		// gres, err := uasc.Decode(buf[:m])
		// if err != nil {
		// 	log.Printf("Something went wrong: %s", err)
		// }
		// log.Printf("Received: %s\nRaw: %x", gres, buf[:m])

		c := services.NewCloseSecureChannelRequest(
			time.Now(), 0, 1, 0, 0, "", cfg.SecureChannelID,
		)
		c.SetDiagAll()
		clo, err := uasc.New(c, cfg).Serialize()
		if err != nil {
			log.Fatalf("Failed to serialize CloseSecureChannelRequest: %s", err)
		}

		if _, err := conn.Write(clo); err != nil {
			log.Fatalf("Failed to write CloseSecureChannel: %s", err)
		}
		// log.Printf("Successfully sent CloseSecureChannel: %x", gep)

	case uacp.MessageTypeError:
		log.Fatalf("Received Error, closing: %s", cp)
	default:
		log.Fatalf("Received unexpected message, closing: %s", cp)
	}
}
