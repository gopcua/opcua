// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/debug"
	"github.com/gopcua/opcua/ua"
)

func main() {
	var (
		endpoint = flag.String("endpoint", "opc.tcp://localhost:4840", "OPC UA Endpoint URL")
		policy   = flag.String("policy", "", "Security policy: None, Basic128Rsa15, Basic256, Basic256Sha256. Default: auto")
		mode     = flag.String("mode", "", "Security mode: None, Sign, SignAndEncrypt. Default: auto")
		certFile = flag.String("cert", "", "Path to cert.pem. Required for security mode/policy != None")
		keyFile  = flag.String("key", "", "Path to private key.pem. Required for security mode/policy != None")
		nodeID   = flag.String("node", "", "NodeID to read")
	)
	flag.BoolVar(&debug.Enable, "debug", false, "enable debug logging")
	flag.Parse()
	log.SetFlags(0)

	id, err := ua.ParseNodeID(*nodeID)
	if err != nil {
		log.Fatalf("invalid node id: %v", err)
	}

	ctx := context.Background()

	endpoints, err := opcua.GetEndpoints(*endpoint)
	if err != nil {
		log.Fatal(err)
	}
	ep := opcua.SelectEndpoint(endpoints, *policy, ua.MessageSecurityModeFromString(*mode))
	if ep == nil {
		log.Fatal("Failed to find suitable endpoint")
	}

	fmt.Println("*", ep.SecurityPolicyURI, ep.SecurityMode)

	opts := []opcua.Option{
		opcua.SecurityPolicy(*policy),
		opcua.SecurityModeString(*mode),
		opcua.CertificateFile(*certFile),
		opcua.PrivateKeyFile(*keyFile),
		opcua.AuthAnonymous(),
		opcua.SecurityFromEndpoint(ep, ua.UserTokenTypeAnonymous),
	}

	c := opcua.NewClient(ep.EndpointURL, opts...)
	if err := c.Connect(ctx); err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	v, err := c.Node(id).Value()
	switch {
	case err != nil:
		log.Fatal(err)
	case v == nil:
		log.Print("v == nil")
	default:
		log.Printf("val: %#v", v.Value.(*ua.ExtensionObject).Value)
	}
}

// MyUDT is a user-defined type (UDT) which is encoded as an extension object.
// It must be registered with ua.RegisterExtensionObject before using it.
//
// Encoding and decoding is handled with reflection. Therefore, defining and
// registering the custom type is sufficient for most cases.
//
// If encoding and decoding requires custom logic like handling flags then you
// can add custom Encode and Decode methods. See node_id.go for good examples.
type MyUDT struct {
	AlarmAck   int32
	AlarmBlock int32
	E1         bool
	Man        bool
	ManOut     float32
}

func init() {
	ua.RegisterExtensionObject(ua.NewStringNodeID(3, `TE_"udtAnaIn_Cmd"`), new(MyUDT))
	ua.RegisterExtensionObject(ua.NewNumericNodeID(4, 2038), new(MyUDT))
}
