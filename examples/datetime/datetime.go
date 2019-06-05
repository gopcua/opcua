// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package main

import (
	"flag"
	"log"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/debug"
	"github.com/gopcua/opcua/ua"
)

func main() {
	endpoint := flag.String("endpoint", "opc.tcp://localhost:4840", "OPC UA Endpoint URL")
	policy := flag.String("policy", "None", "Security policy")
	mode := flag.String("mode", "None", "Security mode")
	// certFile := flag.String("cert", "", "Path to cert.pem")
	keyFile := flag.String("key", "", "Path to private key.pem")
	flag.BoolVar(&debug.Enable, "debug", false, "enable debug logging")
	flag.Parse()
	log.SetFlags(0)

	opts := []opcua.Option{
		opcua.SecurityPolicy(*policy),
		opcua.SecurityModeString(*mode),
		// opcua.X509KeyPair(*certFile, *keyFile),
		opcua.PrivateKeyFile(*keyFile),
		opcua.AuthAnonymous(),
	}

	c := opcua.NewClient(*endpoint, opts...)
	if err := c.Connect(); err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	v, err := c.Node(ua.NewNumericNodeID(0, 2258)).Value()
	switch {
	case err != nil:
		log.Fatal(err)
	case v == nil:
		log.Print("v == nil")
	default:
		log.Print(v.Value)
	}
}
