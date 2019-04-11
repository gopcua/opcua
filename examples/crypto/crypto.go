// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package main

import (
	"crypto/rsa"
	"crypto/tls"
	"flag"
	"fmt"
	"log"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/debug"
	"github.com/gopcua/opcua/ua"
)

func main() {
	endpoint := flag.String("endpoint", "opc.tcp://localhost:4840", "OPC UA Endpoint URL")
	certFile := flag.String("certfile", "cert.pem", "Path to certificate file; will be generated if not found")
	keyFile := flag.String("keyfile", "key.pem", "Path to PEM Private Key file; will be generated if not found")
	flag.BoolVar(&debug.Enable, "debug", false, "enable debug logging")
	flag.Parse()
	log.SetFlags(0)

	// Get a list of the endpoints for our target server
	endpoints, err := opcua.GetEndpoints(*endpoint)
	if err != nil {
		log.Fatal(err)
	}

	// Find the endpoint recommended by the server (highest SecurityMode+SecurityLevel)
	var serverEndpoint *ua.EndpointDescription
	for _, e := range endpoints {
		if serverEndpoint == nil || (e.SecurityMode >= serverEndpoint.SecurityMode && e.SecurityLevel >= serverEndpoint.SecurityLevel) {
			serverEndpoint = e
		}
	}

	// Local certificate
	var localKey *rsa.PrivateKey
	var localCert []byte
	if serverEndpoint.SecurityMode != ua.MessageSecurityModeNone {
		var cert tls.Certificate
		cert, err = tls.LoadX509KeyPair(*certFile, *keyFile)
		if err != nil {
			debug.Printf("could not load certificate and key; generating new ones")
			generate_cert("urn:gopcua-example-client", 2048, *certFile, *keyFile)
			cert, err = tls.LoadX509KeyPair(*certFile, *keyFile)
			if err != nil {
				log.Printf("could not load certificate and key again; exiting")
				return
			}
		}

		localCert = cert.Certificate[0]

		var ok bool
		localKey, ok = cert.PrivateKey.(*rsa.PrivateKey)
		if !ok {
			log.Printf("Private key invalid\n")
			return
		}
	}

	// Finally, create our Client object
	c := opcua.NewClient(*endpoint,
		opcua.SecurityFromEndpoint(serverEndpoint),
		opcua.PrivateKey(localKey),
		opcua.Certificate(localCert),
	)
	log.Printf("Connecting to %s, security mode: %s, %s \n", *endpoint, serverEndpoint.SecurityPolicyURI, serverEndpoint.SecurityMode)
	if err := c.Connect(); err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	// Use our connection (read the server's time)
	v, err := c.Node(ua.NewNumericNodeID(0, 2258)).Value()
	if err != nil {
		log.Fatal(err)
	}
	if v != nil {
		fmt.Printf("Server's Time | Conn 1 %s | ", v.Value)
	} else {
		log.Print("v == nil")
	}

	// Detach our session and try re-establish it on a different secure channel
	s, err := c.DetachSession()
	if err != nil {
		log.Fatalf("Error detaching session: %s", err)
	}

	d := opcua.NewClient(*endpoint,
		opcua.SecurityFromEndpoint(serverEndpoint),
		opcua.PrivateKey(localKey),
		opcua.Certificate(localCert),
	)

	// Create a channel only and do not activate it automatically
	d.Dial()
	defer d.Close()

	// Activate the previous session on the new channel
	err = d.ActivateSession(s)
	if err != nil {
		log.Fatalf("Error reactivating session: %s", err)
	}

	// Read the time again to prove our session is still OK
	v, err = d.Node(ua.NewNumericNodeID(0, 2258)).Value()
	if err != nil {
		log.Fatal(err)
	}
	if v != nil {
		fmt.Printf("Conn 2: %s\n", v.Value)
	} else {
		log.Print("v == nil")
	}

}
