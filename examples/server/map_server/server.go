// Copyright 2018-2020 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.
//
// This example program shows how to create a simple OPC UA server with data backed by a map.
// This allows you to easily create a server with a simple data model that can be updated from
// other parts of your application.  This example also shows how to monitor the data for changes
// and how to trigger change notifications to clients when the data changes.

package main

import (
	"context"
	"crypto/rsa"
	"crypto/tls"
	"flag"
	"github.com/gopcua/opcua/tests/utils"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/gopcua/opcua/id"
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

func main() {
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

	// Here is an example of certificate generation.  This is not necessary if you already have a certificate.
	if *gencert {
		// it is important that the certificate is generated with the correct hostname/IP address URIs
		// or the clients may not accept the certificate.
		endpoints := []string{
			"localhost",
			hostname,
			*endpoint,
		}

		c, k, err := utils.GenerateCert(endpoints, 4096, time.Minute*60*24*365*10)
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

	// Create some map namespaces.  These are backed by go map[string]T
	// which may be more convenient for some use cases than the NodeNamespace which requires
	// your application's data structure to match the opcua node model.
	myMapNamespace1 := server.NewMapNamespace[any](s, "MyTestNamespace")
	log.Printf("map namespace 1 added at index %d", myMapNamespace1.ID())
	myMapNamespace2 := server.NewMapNamespace[any](s, "SomeOtherNamespace")
	log.Printf("map namespace 2 added at index %d", myMapNamespace2.ID())

	// fill them with data.
	myMapNamespace1.SetValue("Tag1", 123.4)
	myMapNamespace1.SetValue("Tag2", 42)
	myMapNamespace1.SetValue("Tag3.Tag4", "some string")
	myMapNamespace1.SetValue("Tag5", true)
	myMapNamespace1.SetValue("Tag6", time.Now())

	myMapNamespace2.SetValue("Tag7", 56.78)
	myMapNamespace2.SetValue("Tag8", 92)
	myMapNamespace2.SetValue("Tag9", "different string")
	myMapNamespace2.SetValue("Tag10", false)
	myMapNamespace2.SetValue("Tag11", time.Now().Add(time.Hour))

	// simulate a background process updating the data in the map namespace.
	go func() {
		updates := 0
		num := 42
		tag5 := true
		time.Sleep(time.Second * 10)
		for {
			updates++
			num++
			// to update the map, you can use the SetValue function which updates the value in a thread safe way
			myMapNamespace1.SetValue("Tag2", num)
			if updates == 10 {
				tag5 = !tag5
				myMapNamespace1.SetValue("Tag5", tag5)
				updates = 0
			}

			// You can range over the map using All(), Keys(), or Values().  This is thread safe
			for k, v := range myMapNamespace1.All() {
				log.Printf("key: %s, value: %v", k, v)
			}

			time.Sleep(time.Second)
		}
	}()

	// simulate monitoring one of the namespaces for change events.
	// this is how you would be notified when a write to the map
	// occurs through the opc ua server
	go func() {
		for {
			changed_key := <-myMapNamespace2.ExternalNotification
			log.Printf("%s changed to %v", changed_key, myMapNamespace2.GetValue(changed_key))
		}
	}()

	// add the namespaces to the server. If you want them to show up in a browse, you'll
	// also have to add a reference to them (probably from the object node).
	root_ns, _ := s.Namespace(0)
	root_obj_node := root_ns.Objects()

	// then we add the namespace to the server and add a reference to it from the object node.
	// the object node of the map namespace is a virtual node that contains all the "nodes" for each
	// map key
	root_obj_node.AddRef(myMapNamespace1.Objects(), id.HasComponent, true)
	root_obj_node.AddRef(myMapNamespace2.Objects(), id.HasComponent, true)

	// Start the server
	// Note that you can add namespaces before or after starting the server.
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
