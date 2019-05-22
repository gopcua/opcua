// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package main

import (
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

	subParams := opcua.NewDefaultSubscriptionParameters()
	sub, err := c.Subscribe(*subParams)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("created Subscription with id %v", sub.SubscriptionID)

	nodeId := ua.NewStringNodeID(uint16(*namespace), *stringId)

	miCreateRequest := opcua.NewMonitoredItemCreateRequestWithDefaults(nodeId, ua.AttributeIDValue, 42)
	res, err := c.CreateMonitoredItems(sub.SubscriptionID, ua.TimestampsToReturnBoth, miCreateRequest)
	if err != nil {
		log.Fatal(err)
	}
	for _, result := range res.Results {
		if result.StatusCode != ua.StatusOK {
			log.Fatal(result.StatusCode)
		}
	}
	miId := res.Results[0].MonitoredItemID
	log.Printf("created MonitoredItem with id %v", miId)

	quit := time.After(30 * time.Second)
LOOP:
	for {
		select {
		case <-quit:
			break LOOP
		default:
			select {
			case res := <-sub.Channel:
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
			default:
				//log.Println("no notifications")
			}
		}
	}

	// this is just an example call
	// it isn't necessary as the monitored items will be deleted by deleting the Subscription
	_, err = c.DeleteMonitoredItems(sub.SubscriptionID, miId)
	if err != nil {
		log.Fatalf("error while deleting monitored item: %v", err)
	}

	err = c.Unsubscribe(sub)
	if err != nil {
		log.Fatalf("error while unsubscribing: %v", err)
	}
}
