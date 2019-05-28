// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"flag"
	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/debug"
	"github.com/gopcua/opcua/ua"
	"log"
	"time"
)

func main() {
	endpoint := flag.String("endpoint", "opc.tcp://localhost:4840", "OPC UA Endpoint URL")
	namespace := flag.Uint("namespace", 0, "Namespace id of the node to subscribe to")
	stringId := flag.String("id", "", "String id of the node to subscribe to")
	flag.BoolVar(&debug.Enable, "debug", false, "enable debug logging")
	flag.Parse()
	log.SetFlags(0)

	c := opcua.NewClient(*endpoint)
	if err := c.Connect(); err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	sub, err := c.Subscribe(&opcua.SubscriptionParameters{
		Interval: 500 * time.Millisecond,
		Notifs:   make(chan opcua.PublishNotificationData, 37),
	})
	if err != nil {
		log.Fatal(err)
	}
	defer sub.Cancel()
	log.Printf("created Subscription with id %v", sub.SubscriptionID)

	miCreateRequest := opcua.NewMonitoredItemCreateRequestWithDefaults(ua.NewStringNodeID(uint16(*namespace), *stringId), ua.AttributeIDValue, 42)
	res, err := sub.Monitor(ua.TimestampsToReturnBoth, miCreateRequest)
	if err != nil || res.Results[0].StatusCode != ua.StatusOK {
		log.Fatal(err)
	}

	ctx, _ := context.WithTimeout(context.Background(), 30*time.Second)

	go sub.Run(ctx) // start Publish loop

	// read from subscription's notification channel until ctx is cancelled
	for {
		select {
		case <-ctx.Done():
			return
		case res := <-sub.Notifs:
			if res.Error != nil {
				log.Print(res.Error)
				continue
			}

			switch x := res.Value.(type) {
			case *ua.DataChangeNotification:
				for _, item := range x.MonitoredItems {
					data := item.Value.Value.Value
					log.Printf("MonitoredItem with client handle %v = %v", item.ClientHandle, data)
				}

			default:
				log.Printf("what's this publish result? %T", res.Value)
			}
		}
	}
}
