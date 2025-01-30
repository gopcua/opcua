// Copyright 2018-2020 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"crypto/rsa"
	"crypto/tls"
	"flag"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/gopcua/opcua/debug"
	"github.com/gopcua/opcua/id"
	"github.com/gopcua/opcua/server"
	"github.com/gopcua/opcua/server/attrs"
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
	logger := Logger(1)
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

	// add the namespaces to the server, and add a reference to them if desired.
	// here we are choosing to add the namespaces to the root/object folder
	// to do this we first need to get the root namespace object folder so we
	// get the object node
	root_ns, _ := s.Namespace(0)
	root_obj_node := root_ns.Objects()

	// Start the server
	// Note that you can add namespaces before or after starting the server.
	if err := s.Start(context.Background()); err != nil {
		log.Fatalf("Error starting server, exiting: %s", err)
	}
	defer s.Close()

	// Now we'll add a node namespace.  This is a more traditional way to add nodes to the server
	// and is more in line with the opc ua node model, but may be more cumbersome for some use cases.
	nodeNS := server.NewNodeNameSpace(s, "NodeNamespace")
	log.Printf("Node Namespace added at index %d", nodeNS.ID())

	// add the reference for this namespace's root object folder to the server's root object folder
	// but you can add a reference to whatever node(s) you need
	nns_obj := nodeNS.Objects()
	root_obj_node.AddRef(nns_obj, id.HasComponent, true)

	// Create some nodes for it.  Here we are usin gthe AddNewVariableNode utility function to create a new variable node
	// with an integer node ID that is automatically assigned. (ns=<namespace id>,s=<auto assigned>)
	// be sure to add the reference to the node somewhere if desired, or clients won't be able to browse it.
	var1 := nodeNS.AddNewVariableNode("TestVar1", float32(123.45))
	nns_obj.AddRef(var1, id.HasComponent, true)

	// This node will have a string node id (ns=<namespace id>,s=TestVar2)
	// your variable node's value can also return a ua.Variant from a function if you want to update the value dynamically
	// here we are just incrementing a counter every time the value is read.
	var2Value := int32(0)
	var2 := nodeNS.AddNewVariableStringNode("TestVar2", func() *ua.Variant { var2Value++; return ua.MustVariant(var2Value) })
	nns_obj.AddRef(var2, id.HasComponent, true)

	// Now we'll add a node from scratch.  This is a more manual way to add nodes to the server and gives you full
	// control, but you'll have to build the node up with the correct attributes and references and then reference it from
	// the parent node in the namespace if applicable.
	var3 := server.NewNode(
		ua.NewNumericNodeID(nodeNS.ID(), 12345), // you can use whatever node id you want here, whether it's numeric, string, guid, etc...
		map[ua.AttributeID]*ua.DataValue{
			ua.AttributeIDBrowseName: server.DataValueFromValue(attrs.BrowseName("MyBrowseName")),
			ua.AttributeIDNodeClass:  server.DataValueFromValue(uint32(ua.NodeClassVariable)),
		},
		nil,
		func() *ua.DataValue { return server.DataValueFromValue(12.34) },
	)
	nodeNS.AddNode(var3)
	nns_obj.AddRef(var3, id.HasComponent, true)

	// simulate a background process updating the data in the namespace.
	go func() {
		updates := 0
		num := 42
		time.Sleep(time.Second * 10)
		for {
			updates++
			num++

			// get the current value of the variable
			last_value := var1.Value().Value.Value().(float32)
			// and change it
			last_value += 1

			// wrap the new value in a DataValue and use that to update the Value attribute of the node
			val := ua.DataValue{
				Value:           ua.MustVariant(last_value),
				SourceTimestamp: time.Now(),
				EncodingMask:    ua.DataValueValue | ua.DataValueSourceTimestamp,
			}
			var1.SetAttribute(ua.AttributeIDValue, &val)

			// we also need to let the node namespace know that the value has changed so it can trigger the change notification
			// and send the updated value to any subscribed clients.
			nodeNS.ChangeNotification(var1.ID())

			time.Sleep(time.Second)
		}
	}()

	// simulate monitoring one of the namespaces for change events.
	// this is how you would be notified when a write to a node
	// occurs through the opc ua server
	go func() {
		for {
			changed_id := <-nodeNS.ExternalNotification
			node := nodeNS.Node(changed_id)
			value := node.Value().Value.Value()
			log.Printf("%s changed to %v", changed_id.String(), value)
		}
	}()

	// catch ctrl-c and gracefully shutdown the server.
	sigch := make(chan os.Signal, 1)
	signal.Notify(sigch, os.Interrupt)
	defer signal.Stop(sigch)
	log.Printf("Press CTRL-C to exit")

	<-sigch
	log.Printf("Shutting down the server...")
}
