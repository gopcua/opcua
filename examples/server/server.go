// Copyright 2018-2020 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"flag"
	"os"
	"time"

	"github.com/gopcua/opcua/uacp"
	"github.com/gopcua/opcua/ualog"
)

func main() {
	var (
		endpoint = flag.String("endpoint", "opc.tcp://example.com/foo/bar", "OPC UA Endpoint URL")
	)
	flag.Parse()

	ctx := context.Background()

	l, err := uacp.Listen(ctx, *endpoint, nil)
	if err != nil {
		fatal(ctx, "failed to listen for connections", err)
	}

	ualog.Info(ctx, "listening for connections", ualog.String("endpoint", *endpoint))

	c, err := l.Accept(ctx)
	if err != nil {
		fatal(ctx, "failed to accept incoming connection", err)
	}
	defer c.Close()

	ualog.Info(ctx, "connection received", ualog.Uint32("conn", c.ID()), ualog.Any("remote", c.RemoteAddr()))

	// listener, err := uacp.Listen(*endpoint, uint32(*bufsize))
	// if err != nil {
	//  ualog.Fatal(ctx, "", ualog.Err(err))
	// }
	// ualog.Info(ctx, "started listening for connections", ualog.Any("endpoint", listener.Endpoint()))

	// cfg := uasc.NewServerConfig(
	// 	"http://opcfoundation.org/UA/SecurityPolicy#None",
	// 	nil, nil, 1111, services.SecModeNone, 2222, 3600000,
	// )
	// for {
	// 	func() {
	// 		ctx := context.Background()
	// 		ctx, cancel := context.WithCancel(ctx)
	// 		defer cancel()

	// 		conn, err := listener.Accept(ctx)
	// 		if err != nil {
	//          ualog.Error(ctx, "listen failed", ualog.Err(err))
	// 			return
	// 		}
	// 		defer func() {
	// 			conn.Close()
	// 			ualog.Info(ctx, "successfully closed connection")
	// 		}()
	// 		ualog.Info(ctx, "successfully established connection", ualog.Any("remote", conn.RemoteAddr()))

	// 		secChan, err := uasc.ListenAndAcceptSecureChannel(ctx, conn, cfg)
	// 		if err != nil {
	// 			ualog.Fatal(ctx, "", ualog.Err(err))
	// 		}
	// 		defer func() {
	// 			secChan.Close()
	// 			ualog.Info(ctx, "successfully closed secure channel", ualog.Any("remote", conn.RemoteAddr()))
	// 		}()
	// 		ualog.Info(ctx, "successfully opened secure channel", ualog.Any("remote", conn.RemoteAddr()))

	// 		sessCfg := uasc.NewServerSessionConfig(secChan)
	// 		session, err := uasc.ListenAndAcceptSession(ctx, secChan, sessCfg)
	// 		if err != nil {
	// 			ualog.Fatal(ctx, "", ualog.Err(err))
	// 		}
	// 		defer func() {
	// 			session.Close()
	// 			ualog.Info(ctx, "successfully closed session", ualog.Any("remote", conn.RemoteAddr()))
	// 		}()
	// 		ualog.Info(ctx, "successfully activated session", ualog.Any("remote", conn.RemoteAddr()))

	// 		buf := make([]byte, 1024)
	// 		for {
	// 			n, err := session.ReadService(buf)
	// 			if err != nil {
	// 				ualog.Error(ctx, "couldn't read uasc", ualog.Err(err))
	// 				continue
	// 			}
	//			ualog.Info(ctx, "successfully received message", ualog.String("bytes", fmt.Sprintf("%x", buf[:n])), ualog.String("wireshark", utils.Wireshark(0, buf[:n])))
	// 			srv, err := services.Decode(buf[:n])
	// 			if err != nil {
	// 				ualog.Error(ctx, "couldn't decode received bytes as Service", ualog.Err(err))
	// 				continue
	// 			}
	// 			ualog.Info(ctx, "successfully decoded as Service", ualog.Any("service", srv))
	// 		}
	// 	}()
	// }
}

func fatal(ctx context.Context, reason string, err error) {
	ualog.Error(ctx, "FATAL: "+reason, ualog.Err(err))
	time.Sleep(time.Second)
	os.Exit(1)
}
