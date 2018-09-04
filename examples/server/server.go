// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

/*
Command server provides a connection establishement of OPC UA Secure Conversation as a server.

XXX - Currently this command just handles the UACP connection from any client.
*/
package main

import (
	"flag"
	"log"

	"github.com/wmnsk/gopcua/uacp"
)

func main() {
	var (
		endpoint = flag.String("endpoint", "opc.tcp://example.com/foo/bar", "OPC UA Endpoint URL")
		bufsize  = flag.Int("bufsize", 0xffff, "Receive Buffer Size")
	)
	flag.Parse()

	srv := uacp.NewServer(*endpoint, uint32(*bufsize))
	listener, err := srv.Listen(nil)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Started listening on %s.", srv.Endpoint)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Print(err)
			continue
		}
		defer conn.Close()

		log.Printf("Successfully established connection with %v", conn.RemoteAddr())
	}
}
