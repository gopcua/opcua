// Copyright 2018-2020 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"crypto/rsa"
	"crypto/tls"
	"errors"
	"flag"
	"log/slog"
	"os"
	"time"

	"github.com/gopcua/opcua/debug"
	"github.com/gopcua/opcua/server"
	"github.com/gopcua/opcua/ua"
	"github.com/gopcua/opcua/ualog"
)

var (
	endpoint = flag.String("endpoint", "0.0.0.0", "OPC UA Endpoint URL")
	port     = flag.Int("port", 4840, "OPC UA Endpoint port")
	certfile = flag.String("cert", "cert.pem", "Path to certificate file")
	keyfile  = flag.String("key", "key.pem", "Path to PEM Private Key file")
	gencert  = flag.Bool("gen-cert", true, "Generate a new certificate")
)

func main() {
	flag.BoolVar(&debug.Enable, "debug", false, "enable debug logging")
	flag.Parse()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, func() *slog.HandlerOptions {
		if debug.Enable {
			return &slog.HandlerOptions{Level: slog.LevelDebug}
		}
		return nil
	}()))

	logger = logger.With("foo", "bar")
	ctx := ualog.Logger(context.Background(), ualog.WithLogger(logger))

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
		fatal(ctx, "unable to get host name", err)
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
		c, k, err := GenerateCert(endpoints, 4096, time.Minute*60*24*365*10)
		if err != nil {
			fatal(ctx, "problem creating certificate", err)
		}
		err = os.WriteFile(*certfile, c, 0)
		if err != nil {
			fatal(ctx, "problem writing certificate", err)
		}
		err = os.WriteFile(*keyfile, k, 0)
		if err != nil {
			fatal(ctx, "problem writing key", err)
		}
	}

	var cert []byte
	if *gencert || (*certfile != "" && *keyfile != "") {
		ualog.Info(ctx, "loading certificate and key from files", ualog.String("cert", *certfile), ualog.String("key", *keyfile))
		c, err := tls.LoadX509KeyPair(*certfile, *keyfile)
		if err != nil {
			ualog.Error(ctx, "failed to load certificate", ualog.Err(err))
		} else {
			pk, ok := c.PrivateKey.(*rsa.PrivateKey)
			if !ok {
				fatal(ctx, "invalid private key", errors.New("incorrect type"))
			}
			cert = c.Certificate[0]
			opts = append(opts, server.PrivateKey(pk), server.Certificate(cert))
		}
	}

	s := server.New(ctx, opts...)

	// Create a new node namespace.  You can add namespaces before or after starting the server.
	// Start the server
	if err := s.Start(ctx); err != nil {
		fatal(ctx, "unable to start server", err)
	}
	defer s.Close(ctx)

	select {}
}

func fatal(ctx context.Context, reason string, err error) {
	ualog.Error(ctx, "FATAL: "+reason, ualog.Err(err))
	time.Sleep(time.Second)
	os.Exit(1)
}
