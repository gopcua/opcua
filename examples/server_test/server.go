// Copyright 2018-2020 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"crypto/rsa"
	"crypto/tls"
	"flag"
	"log/slog"
	"os"
	"time"

	"github.com/gopcua/opcua/internal/ualog"
	"github.com/gopcua/opcua/server"
	"github.com/gopcua/opcua/tests/utils"
	"github.com/gopcua/opcua/ua"
)

func main() {
	var (
		endpoint = flag.String("endpoint", "0.0.0.0", "OPC UA Endpoint URL")
		port     = flag.Int("port", 4840, "OPC UA Endpoint port")
		certfile = flag.String("cert", "cert.pem", "Path to certificate file")
		keyfile  = flag.String("key", "key.pem", "Path to PEM Private Key file")
		gencert  = flag.Bool("gen-cert", true, "Generate a new certificate")
		debug    = flag.Bool("debug", false, "enable debug logging")
	)
	flag.Parse()
	slog.SetDefault(slog.New(ualog.NewTextHandler(*debug)))

	var opts []server.Option

	opts = append(opts,
		server.EnableSecurity("None", ua.MessageSecurityModeNone),
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
	)

	opts = append(opts,
		server.EnableAuthMode(ua.UserTokenTypeAnonymous),
		server.EnableAuthMode(ua.UserTokenTypeUserName),
		server.EnableAuthMode(ua.UserTokenTypeCertificate),
		//		server.EnableAuthWithoutEncryption(), // Dangerous and not recommended, shown for illustration only
	)
	hostname, err := os.Hostname()
	if err != nil {
		ualog.Fatal("os.Hostname failed", "error", err)
	}

	// not sure if a list of hostnames is better or adding endpoints to the options
	endpoints := []string{
		"localhost",
		hostname,
		*endpoint,
	}

	opts = append(opts,
		server.EndPoint(*endpoint, *port),
		server.EndPoint("localhost", *port),
		server.EndPoint(hostname, *port),
	)

	if *gencert {
		c, k, err := utils.GenerateCert(endpoints, 4096, time.Minute*60*24*365*10)
		if err != nil {
			ualog.Fatal("GenerateCert failed", "error", err)
		}
		err = os.WriteFile(*certfile, c, 0)
		if err != nil {
			ualog.Fatal("Writing certfile failed", "error", err)
		}
		err = os.WriteFile(*keyfile, k, 0)
		if err != nil {
			ualog.Fatal("Writing keyfile failed", "error", err)
		}
	}

	var cert []byte
	if *gencert || (*certfile != "" && *keyfile != "") {
		slog.Info("Loading cert and key", "certfile", *certfile, "keyfile", *keyfile)
		c, err := tls.LoadX509KeyPair(*certfile, *keyfile)
		if err != nil {
			slog.Error("LoadX509KeyPair failed", "error", err)
		} else {
			pk, ok := c.PrivateKey.(*rsa.PrivateKey)
			if !ok {
				ualog.Fatal("Invalid private key", "type", ualog.TypeOf(c.PrivateKey))
			}
			cert = c.Certificate[0]
			opts = append(opts, server.PrivateKey(pk), server.Certificate(cert))
		}
	}

	s := server.New(opts...)

	// Create a new node namespace.  You can add namespaces before or after starting the server.
	// Start the server
	if err := s.Start(context.Background()); err != nil {
		ualog.Fatal("Starting server failed", "error", err)
	}
	defer s.Close()

	select {}
}
