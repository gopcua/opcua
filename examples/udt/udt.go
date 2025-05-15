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
		policy   = flag.String("policy", "", "Security policy: None, Basic128Rsa15, Basic256, Basic256Sha256. Default: auto")
		mode     = flag.String("mode", "", "Security mode: None, Sign, SignAndEncrypt. Default: auto")
		certFile = flag.String("cert", "", "Path to cert.pem. Required for security mode/policy != None")
		keyFile  = flag.String("key", "", "Path to private key.pem. Required for security mode/policy != None")
		nodeID   = flag.String("node", "", "NodeID to read")
		debug    = flag.Bool("debug", false, "enable debug logging")
	)
	flag.Parse()
	slog.SetDefault(slog.New(ualog.NewTextHandler(*debug)))

	id, err := ua.ParseNodeID(*nodeID)
	if err != nil {
		ualog.Fatal("invalid node id", "node_id", *nodeID, "error", err)
	}

	ctx := context.Background()

	endpoints, err := opcua.GetEndpoints(ctx, *endpoint)
	if err != nil {
		ualog.Fatal("GetEndpoints failed", "error", err)
	}
	ep, err := opcua.SelectEndpoint(endpoints, *policy, ua.MessageSecurityModeFromString(*mode))
	if err != nil {
		ualog.Fatal("SelectEndpoint failed", "error", err)
	}

	slog.Info("*", "sec_policy", ep.SecurityPolicyURI, "sec_mode", ep.SecurityMode)

	opts := []opcua.Option{
		opcua.SecurityPolicy(*policy),
		opcua.SecurityModeString(*mode),
		opcua.CertificateFile(*certFile),
		opcua.PrivateKeyFile(*keyFile),
		opcua.AuthAnonymous(),
		opcua.SecurityFromEndpoint(ep, ua.UserTokenTypeAnonymous),
	}

	c, err := opcua.NewClient(ep.EndpointURL, opts...)
	if err != nil {
		ualog.Fatal("NewClient failed", "error", err)
	}
	if err := c.Connect(ctx); err != nil {
		ualog.Fatal("Connect failed", "error", err)
	}
	defer c.Close(ctx)

	v, err := c.Node(id).Value(ctx)
	switch {
	case err != nil:
		ualog.Fatal("Value failed", "error", err)
	case v == nil:
		slog.Info("v == nil")
	default:
		slog.Info("extobj", "value", v.Value().(*ua.ExtensionObject).Value)
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
