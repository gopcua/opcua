// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package main

import (
	"flag"
	"log"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/debug"
	"github.com/gopcua/opcua/ua"
)

func main() {
	endpoint := flag.String("endpoint", "opc.tcp://localhost:4840", "OPC UA Endpoint URL")
	flag.BoolVar(&debug.Enable, "debug", false, "enable debug logging")
	flag.Parse()
	log.SetFlags(0)

	c := opcua.NewClient(*endpoint)
	if err := c.Connect(); err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	ch := make(chan opcua.PublishNotificationData)
	go c.Publish(ch)

	for {
		res := <-ch
		if res.Error != nil {
			log.Print(res.Error)
			continue
		}

		switch x := res.Value.(type) {
		case *ua.DataChangeNotification:
			for _, item := range x.MonitoredItems {
				data, ok := item.Value.Value.Value.(float64)
				if ok {
					log.Printf("%g", data)
				}
			}
		}
	}
}
