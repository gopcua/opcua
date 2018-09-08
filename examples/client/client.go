// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

/*
Command client provides a connection establishment of OPC UA Secure Conversation.

XXX - Currently this command just initiates the connection(UACP) to the specified endpoint.
*/
package main

import (
	"flag"
	"log"
	"time"

	"github.com/wmnsk/gopcua/uacp"
)

func main() {
	var (
		endpoint = flag.String("endpoint", "opc.tcp://example.com/foo/bar", "OPC UA Endpoint URL")
		bufsize  = flag.Int("bufsize", 0xffff, "Receive Buffer Size")
	)
	flag.Parse()

	cpClient := uacp.NewClient(*endpoint, uint32(*bufsize), 5*time.Second, 3)
	conn, err := cpClient.Dial(nil)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	log.Printf("Successfully established the connection with %v", conn.RemoteAddr())
}
