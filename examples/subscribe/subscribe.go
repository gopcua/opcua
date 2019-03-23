// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package main

import (
	"flag"
	"log"
	"time"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/debug"
)

func main() {
	endpoint := flag.String("endpoint", "opc.tcp://localhost:4840", "OPC UA Endpoint URL")
	flag.BoolVar(&debug.Enable, "debug", false, "enable debug logging")
	flag.Parse()
	log.SetFlags(0)

	c := opcua.NewClient(*endpoint, nil)
	if err := c.Open(); err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	sub, err := c.Subscribe(time.Second)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("sub: %v", sub)
}
