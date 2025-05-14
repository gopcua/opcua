// Copyright 2018-2020 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"time"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/id"
	"github.com/gopcua/opcua/internal/ualog"
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
		debug         = flag.Bool("debug", false, "enable debug logging")
	)
	flag.Parse()
	slog.SetDefault(slog.New(ualog.NewTextHandler(*debug)))

	// add an arbitrary timeout to demonstrate how to stop a subscription
	// with a context.
	d := 60 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), d)
	defer cancel()

	endpoints, err := opcua.GetEndpoints(ctx, *endpoint)
	if err != nil {
		ualog.Fatal("GetEndpoints failed", "error", err)
	}
	ep, err := opcua.SelectEndpoint(endpoints, *policy, ua.MessageSecurityModeFromString(*mode))
	if err != nil {
		ualog.Fatal("SelectEndpoint failed", "error", err)
	}

	ualog.Info("*", "sec_policy", ep.SecurityPolicyURI, "sec_mode", ep.SecurityMode)

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
		ualog.Fatal("NewClient failed", "error", err)
	}
	if err := c.Connect(ctx); err != nil {
		ualog.Fatal("Connect failed", "error", err)
	}
	defer c.Close(ctx)

	notifyCh := make(chan *opcua.PublishNotificationData)

	sub, err := c.Subscribe(ctx, &opcua.SubscriptionParameters{
		Interval: *interval,
	}, notifyCh)
	if err != nil {
		ualog.Fatal("Subscribe failed", "error", err)
	}
	defer sub.Cancel(ctx)
	ualog.Info("Created subscription", "sub_id", sub.SubscriptionID)

	triggeringNode, err := ua.ParseNodeID(*triggerNodeID)
	if err != nil {
		ualog.Fatal("parse trigger node id failed", "node_id", *triggerNodeID, "error", err)
	}

	triggeredNode, err := ua.ParseNodeID(*reportNodeID)
	if err != nil {
		ualog.Fatal("parse report node id failed", "node_id", *reportNodeID, "error", err)
	}

	miCreateRequests := []*ua.MonitoredItemCreateRequest{
		opcua.NewMonitoredItemCreateRequestWithDefaults(triggeringNode, ua.AttributeIDValue, 42),
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
		ualog.Fatal("Monitor failed", "error", err)
	}

	triggeringServerID, triggeredServerID := subRes.Results[0].MonitoredItemID, subRes.Results[1].MonitoredItemID
	tRes, err := sub.SetTriggering(ctx, triggeringServerID, []uint32{triggeredServerID}, nil)
	if err != nil {
		ualog.Fatal("SetTriggering failed", "error", err)
	}

	if tRes.AddResults[0] != ua.StatusOK {
		ualog.Fatal("Status code is not OK", "status_code", tRes.AddResults[0].Error())
	}

	// read from subscription's notification channel until ctx is cancelled
	for {
		select {
		case <-ctx.Done():
			return
		case res := <-notifyCh:
			if res.Error != nil {
				ualog.Error("notifyCh has an error", "error", res.Error)
				continue
			}

			switch x := res.Value.(type) {
			case *ua.DataChangeNotification:
				for _, item := range x.MonitoredItems {
					data := item.Value.Value.Value()
					ualog.Info("MonitoredItem with client handle", "client_handle", item.ClientHandle, "data", data)
				}

			default:
				ualog.Warn("what's this publish result?", "type", fmt.Sprintf("%T", res.Value))
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
		ualog.Fatal("Unable to match to a valid filter\nShould be status, value, or timestamp", "type", filterStr)
	}

	return &ua.ExtensionObject{
		EncodingMask: ua.ExtensionObjectBinary,
		TypeID: &ua.ExpandedNodeID{
			NodeID: ua.NewNumericNodeID(0, id.DataChangeFilter_Encoding_DefaultBinary),
		},
		Value: filter,
	}
}
