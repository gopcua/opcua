// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

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

		log.Printf("Successfully established and closed connection with %v", conn.RemoteAddr())
	}
}
