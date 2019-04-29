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

	c := opcua.NewClient(*endpoint)
	if err := c.Connect(); err != nil {
		log.Fatal(err)
	}
	c.Close()

	ch := make(chan opcua.PublishNotificationData)
	go c.Publish(ch)

	for {
		var response = <-ch

		var t = time.Now().Format(time.RFC3339)
		if response.Error != nil {
			log.Printf("%s - %s \n", t, response.Error.Error())
			continue
		}

		if response.DataChangeNotification != nil {
			for _, item := range response.DataChangeNotification.MonitoredItems {
				var data, ok = item.Value.Value.Value.(float64)
				if ok {
					log.Printf("%s - %g \n", t, data)
				}
			}
		}
	}
}
