// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

// Command client provides a connection establishment of OPC UA Secure Conversation.
package main

import (
	"flag"
	"log"

	"github.com/wmnsk/gopcua/uacp"
)

func main() {
	var (
		endpoint = flag.String("endpoint", "opc.tcp://example.com/foo/bar", "OPC UA Endpoint URL")
	)
	flag.Parse()

	cfg := uacp.NewClientConfig(*endpoint, 0xffff)

	conn, err := uacp.Dial(cfg, nil)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("%T, %v", conn, conn.RemoteAddr())
}
