// Copyright 2018-2020 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/debug"
	"github.com/gopcua/opcua/id"
	"github.com/gopcua/opcua/ua"
)

func main() {
	var (
		endpoint      = flag.String("endpoint", "opc.tcp://localhost:4840", "OPC UA Endpoint URL")
		policy        = flag.String("policy", "", "Security policy: None, Basic128Rsa15, Basic256, Basic256Sha256. Default: auto")
		mode          = flag.String("mode", "", "Security mode: None, Sign, SignAndEncrypt. Default: auto")
		certFile      = flag.String("cert", "", "Path to cert.pem. Required for security mode/policy != None")
		keyFile       = flag.String("key", "", "Path to private key.pem. Required for security mode/policy != None")
		triggerNodeID = flag.String("trigger", "", "node id to trigger with")
		reportNodeID  = flag.String("report", "", "node id value to report on trigger")
		filter        = flag.String("filter", "timestamp", "DataFilter: status, value, timestamp.")
		interval      = flag.Duration("interval", opcua.DefaultSubscriptionInterval, "subscription interval")
	)
	flag.BoolVar(&debug.Enable, "debug", false, "enable debug logging")
	flag.Parse()
	log.SetFlags(0)

	// add an arbitrary timeout to demonstrate how to stop a subscription
	// with a context.
	d := 60 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), d)
	defer cancel()

	endpoints, err := opcua.GetEndpoints(ctx, *endpoint)
	if err != nil {
		log.Fatal(err)
	}
	ep, err := opcua.SelectEndpoint(endpoints, *policy, ua.MessageSecurityModeFromString(*mode))
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("*", ep.SecurityPolicyURI, ep.SecurityMode)

	opts := []opcua.Option{
		opcua.SecurityPolicy(*policy),
		opcua.SecurityModeString(*mode),
		opcua.CertificateFile(*certFile),
		opcua.PrivateKeyFile(*keyFile),
		opcua.AuthAnonymous(),
		opcua.SecurityFromEndpoint(ep, ua.UserTokenTypeAnonymous),
	}

	c, err := opcua.NewClient(ep.EndpointURL, opts...)
	if err != nil {
		log.Fatal(err)
	}
	if err := c.Connect(ctx); err != nil {
		log.Fatal(err)
	}
	defer c.Close(ctx)

	notifyCh := make(chan *opcua.PublishNotificationData)

	sub, err := c.Subscribe(ctx, &opcua.SubscriptionParameters{
		Interval: *interval,
	}, notifyCh)
	if err != nil {
		log.Fatal(err)
	}
	defer sub.Cancel(ctx)
	log.Printf("Created subscription with id %v", sub.SubscriptionID)

	triggeringNode, err := ua.ParseNodeID(*triggerNodeID)
	if err != nil {
		log.Fatal(err)
	}

	triggeredNode, err := ua.ParseNodeID(*reportNodeID)
	if err != nil {
		log.Fatal(err)
	}

	miCreateRequests := []*ua.MonitoredItemCreateRequest{
		opcua.NewDefaultMonitoredItemCreateRequest(opcua.MonitoredItemCreateRequestArgs{
			NodeID:       triggeringNode,
			AttributeID:  ua.AttributeIDValue,
			ClientHandle: 42,
		}),
		{
			ItemToMonitor: &ua.ReadValueID{
				NodeID:       triggeredNode,
				AttributeID:  ua.AttributeIDValue,
				DataEncoding: &ua.QualifiedName{},
			},
			MonitoringMode: ua.MonitoringModeSampling,
			RequestedParameters: &ua.MonitoringParameters{
				ClientHandle:     43,
				DiscardOldest:    true,
				Filter:           getFilter(*filter),
				QueueSize:        10,
				SamplingInterval: 0.0,
			},
		},
	}

	subRes, err := sub.Monitor(ctx, ua.TimestampsToReturnBoth, miCreateRequests...)
	if err != nil || subRes.Results[0].StatusCode != ua.StatusOK {
		log.Fatal(err)
	}

	triggeringServerID, triggeredServerID := subRes.Results[0].MonitoredItemID, subRes.Results[1].MonitoredItemID
	tRes, err := sub.SetTriggering(ctx, triggeringServerID, []uint32{triggeredServerID}, nil)

	if err != nil {
		log.Fatal(err)
	}

	if tRes.AddResults[0] != ua.StatusOK {
		log.Fatal(tRes.AddResults[0].Error())
	}

	// read from subscription's notification channel until ctx is cancelled
	for {
		select {
		case <-ctx.Done():
			return
		case res := <-notifyCh:
			if res.Error != nil {
				log.Print(res.Error)
				continue
			}

			switch x := res.Value.(type) {
			case *ua.DataChangeNotification:
				for _, item := range x.MonitoredItems {
					data := item.Value.Value.Value()
					log.Printf("MonitoredItem with client handle %v = %v", item.ClientHandle, data)
				}

			default:
				log.Printf("what's this publish result? %T", res.Value)
			}
		}
	}
}

func getFilter(filterStr string) *ua.ExtensionObject {

	var filter ua.DataChangeFilter
	switch filterStr {
	case "status":
		filter = ua.DataChangeFilter{Trigger: ua.DataChangeTriggerStatus}
	case "value":
		filter = ua.DataChangeFilter{Trigger: ua.DataChangeTriggerStatusValue}
	case "timestamp":
		filter = ua.DataChangeFilter{Trigger: ua.DataChangeTriggerStatusValueTimestamp}
	default:
		log.Fatalf("Unable to match to a valid filter type: %v\nShould be status, value, or timestamp", filterStr)
	}

	return &ua.ExtensionObject{
		EncodingMask: ua.ExtensionObjectBinary,
		TypeID: &ua.ExpandedNodeID{
			NodeID: ua.NewNumericNodeID(0, id.DataChangeFilter_Encoding_DefaultBinary),
		},
		Value: filter,
	}
}
