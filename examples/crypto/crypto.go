// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package main

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"flag"
	"io/ioutil"
	"log"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/debug"
	"github.com/gopcua/opcua/ua"
	"github.com/gopcua/opcua/uasc"
)

func main() {
	endpoint := flag.String("endpoint", "opc.tcp://localhost:4840", "OPC UA Endpoint URL")
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
	localCert, localKey, err := getCert()
	if err != nil {
		log.Fatalf("unable to get cert: %s", err)
	}

	log.Printf("Creating client with security: %s , %s\n", serverEndpoint.SecurityPolicyURI, serverEndpoint.SecurityMode)
	config := uasc.NewClientConfig(
		serverEndpoint,
		localKey,
		localCert.Raw,
		0, //lifetime,
	)

	c := opcua.NewClient(*endpoint, config)
	if err := c.Open(); err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	// Request the server time
	v, err := c.Node(ua.NewNumericNodeID(0, 2258)).Value()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("time: %v", v.Value)
}

func getCert() (*x509.Certificate, *rsa.PrivateKey, error) {

	certPEM, err := ioutil.ReadFile("cert.pem")
	if err != nil {
		log.Printf("could not load cert, generating new one.  Error: %s\n", err)
		generate_cert("urn:gopcua-client", 2048)
		certPEM, _ = ioutil.ReadFile("cert.pem")
	}

	block, _ := pem.Decode([]byte(certPEM))
	if block == nil {
		panic("failed to parse certificate PEM")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		panic("failed to parse certificate: " + err.Error())
	}

	keyPem, err := ioutil.ReadFile("key.pem")
	if err != nil {
		log.Printf("could not load private key: %s\n", err)
	}

	block, _ = pem.Decode([]byte(keyPem))
	if block == nil {
		panic("failed to parse certificate PEM")
	}

	pkey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		panic("failed to parse private key: " + err.Error())
	}

	return cert, pkey, nil
}
