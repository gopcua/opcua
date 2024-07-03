// Copyright 2018-2020 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"crypto/rsa"
	"crypto/tls"
	"encoding/xml"
	"flag"
	"io"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/gopcua/opcua/debug"
	"github.com/gopcua/opcua/schema"
	"github.com/gopcua/opcua/server"
	"github.com/gopcua/opcua/ua"
)

var (
	endpoint = flag.String("endpoint", "0.0.0.0", "OPC UA Endpoint URL")
	port     = flag.Int("port", 4840, "OPC UA Endpoint port")
	certfile = flag.String("cert", "cert.pem", "Path to certificate file")
	keyfile  = flag.String("key", "key.pem", "Path to PEM Private Key file")
	gencert  = flag.Bool("gen-cert", false, "Generate a new certificate")
)

type Logger int

func (l Logger) Debug(msg string, args ...any) {
	if l < 0 {
		log.Printf(msg, args...)
	}
}
func (l Logger) Info(msg string, args ...any) {
	if l < 1 {
		log.Printf(msg, args...)
	}
}
func (l Logger) Warn(msg string, args ...any) {
	if l < 2 {
		log.Printf(msg, args...)
	}
}
func (l Logger) Error(msg string, args ...any) {
	if l < 3 {
		log.Printf(msg, args...)
	}
}

func main() {
	flag.BoolVar(&debug.Enable, "debug", false, "enable debug logging")
	flag.Parse()
	log.SetFlags(0)

	var opts []server.Option

	// Set your security options.
	opts = append(opts,
		server.EnableSecurity("None", ua.MessageSecurityModeNone),
		/*
			These security modes are not implemented yet.
			server.EnableSecurity("Basic128Rsa15", ua.MessageSecurityModeSign),
			server.EnableSecurity("Basic128Rsa15", ua.MessageSecurityModeSignAndEncrypt),
			server.EnableSecurity("Basic256", ua.MessageSecurityModeSign),
			server.EnableSecurity("Basic256", ua.MessageSecurityModeSignAndEncrypt),
			server.EnableSecurity("Basic256Sha256", ua.MessageSecurityModeSignAndEncrypt),
			server.EnableSecurity("Basic256Sha256", ua.MessageSecurityModeSign),
			server.EnableSecurity("Aes128_Sha256_RsaOaep", ua.MessageSecurityModeSign),
			server.EnableSecurity("Aes128_Sha256_RsaOaep", ua.MessageSecurityModeSignAndEncrypt),
			server.EnableSecurity("Aes256_Sha256_RsaPss", ua.MessageSecurityModeSign),
			server.EnableSecurity("Aes256_Sha256_RsaPss", ua.MessageSecurityModeSignAndEncrypt),
		*/
	)

	// Set your user authentication options.
	opts = append(opts,
		server.EnableAuthMode(ua.UserTokenTypeAnonymous),
		/*
			These authentication modes are not implemented yet
			server.EnableAuthMode(ua.UserTokenTypeUserName),
			server.EnableAuthMode(ua.UserTokenTypeCertificate),
		*/
		//		server.EnableAuthWithoutEncryption(), // Dangerous and not recommended, shown for illustration only
	)

	// Here we're automatically adding the hostname and localhost to the endpoint list.
	// Some clients are picky about the endpoint matching the connection url, so be sure to add any addresses/hostnames that
	// clients will use to connect to the server.
	//
	// be sure the hostname(s) also match the certificate the server is going to use.
	hostname, err := os.Hostname()
	if err != nil {
		log.Fatalf("Error getting host name %v", err)
	}

	opts = append(opts,
		server.EndPoint(*endpoint, *port),
		server.EndPoint("localhost", *port),
		server.EndPoint(hostname, *port),
	)

	// the server.SetLogger takes a server.Logger interface.  This interface is met by
	// the slog.Logger{}.  A simple wrapper could be made for other loggers if they don't already
	// meet the interface.
	logger := Logger(2)
	opts = append(opts,
		server.SetLogger(logger),
	)

	// Here is an example of certificate generation.  This is not necessary if you already have a certificate.
	if *gencert {
		// it is important that the certificate is generated with the correct hostname/IP address URIs
		// or the clients may not accept the certificate.
		endpoints := []string{
			"localhost",
			hostname,
			*endpoint,
		}

		c, k, err := GenerateCert(endpoints, 4096, time.Minute*60*24*365*10)
		if err != nil {
			log.Fatalf("problem creating cert: %v", err)
		}
		err = os.WriteFile(*certfile, c, 0)
		if err != nil {
			log.Fatalf("problem writing cert: %v", err)
		}
		err = os.WriteFile(*keyfile, k, 0)
		if err != nil {
			log.Fatalf("problem writing key: %v", err)
		}

	}

	var cert []byte
	if *gencert || (*certfile != "" && *keyfile != "") {
		log.Printf("Loading cert/key from %s/%s", *certfile, *keyfile)
		c, err := tls.LoadX509KeyPair(*certfile, *keyfile)
		if err != nil {
			log.Printf("Failed to load certificate: %s", err)
		} else {
			pk, ok := c.PrivateKey.(*rsa.PrivateKey)
			if !ok {
				log.Fatalf("Invalid private key")
			}
			cert = c.Certificate[0]
			opts = append(opts, server.PrivateKey(pk), server.Certificate(cert))
		}
	}

	// Now that all the options are set, create the server.
	// When the server is created, it will automatically create namespace 0 and populate it with
	// the core opc ua nodes.
	s := server.New(opts...)

	// Now we'll import our NodeSet2.xml file.
	// These files often create additional namespaces and reference them assuming they
	// stat at namespace 1.  So you'll want to import the nodeset first, then add any custom namespaces
	// after that.

	// first, we read the file and unmarshal it into a schema.UANodeSet struct.  Then it can be imported
	file, err := os.Open("Opc.Ua.Di.NodeSet2.xml")
	if err != nil {
		log.Fatalf("error opening nodeset file: %v", err)
	}
	node_data, err := io.ReadAll(file)
	if err != nil {
		log.Fatalf("error reading nodeset file: %v", err)
	}
	var nodes schema.UANodeSet
	xml.Unmarshal(node_data, &nodes)
	s.ImportNodeSet(&nodes)

	// At this point you can lookup any specific node by its nodeid to add references or modify it or whatever
	// your heart desires
	node := s.Node(ua.NewNumericNodeID(1, 15044))
	if node != nil {
		log.Printf("Found node %v", node)
	}

	// Start the server
	if err := s.Start(context.Background()); err != nil {
		log.Fatalf("Error starting server, exiting: %s", err)
	}
	defer s.Close()

	// catch ctrl-c and gracefully shutdown the server.
	sigch := make(chan os.Signal, 1)
	signal.Notify(sigch, os.Interrupt)
	defer signal.Stop(sigch)
	log.Printf("Press CTRL-C to exit")

	<-sigch
	log.Printf("Shutting down the server...")
}
