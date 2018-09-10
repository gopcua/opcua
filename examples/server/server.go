// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

/*
Command server provides a connection establishement of OPC UA Secure Conversation as a server.

XXX - Currently this command just handles the UACP connection from any client.
*/
package main

import (
	"context"
	"flag"
	"log"

	"github.com/wmnsk/gopcua/uacp"
	"github.com/wmnsk/gopcua/utils"
)

func main() {
	var (
		endpoint = flag.String("endpoint", "opc.tcp://example.com/foo/bar", "OPC UA Endpoint URL")
		bufsize  = flag.Int("bufsize", 0xffff, "Receive Buffer Size")
	)
	flag.Parse()

	listener, err := uacp.Listen(*endpoint, uint32(*bufsize))
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Started listening on %s.", listener.Endpoint())

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	for {
		conn, err := listener.Accept(ctx)
		if err != nil {
			log.Print(err)
			continue
		}
		defer conn.Close()
		log.Printf("Successfully established connection with %v", conn.LocalEndpoint())

		buf := make([]byte, 1024)
		n, err := conn.Read(buf)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("Successfully received message: %x\n%s", buf[:n], utils.Wireshark(0, buf[:n]))
	}
}
