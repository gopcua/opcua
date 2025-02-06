package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/debug"
	"github.com/gopcua/opcua/id"
	"github.com/gopcua/opcua/monitor"
	"github.com/gopcua/opcua/ua"
)

func main() {
	var (
		endpoint = flag.String("endpoint", "opc.tcp://localhost:4840", "OPC UA Endpoint URL")
		policy   = flag.String("policy", "", "Security policy: None, Basic128Rsa15, Basic256, Basic256Sha256. Default: auto")
		mode     = flag.String("mode", "", "Security mode: None, Sign, SignAndEncrypt. Default: auto")
		certFile = flag.String("cert", "", "Path to cert.pem. Required for security mode/policy != None")
		keyFile  = flag.String("key", "", "Path to private key.pem. Required for security mode/policy != None")
		nodeID   = flag.String("node", "", "node id to subscribe to")
		interval = flag.Duration("interval", opcua.DefaultSubscriptionInterval, "subscription interval")
		event    = flag.Bool("event", false, "are you subscribing to events")
	)
	flag.BoolVar(&debug.Enable, "debug", false, "enable debug logging")
	flag.Parse()

	// log.SetFlags(0)

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		<-signalCh
		println()
		cancel()
	}()

	endpoints, err := opcua.GetEndpoints(ctx, *endpoint)
	if err != nil {
		log.Fatal(err)
	}

	ep, err := opcua.SelectEndpoint(endpoints, *policy, ua.MessageSecurityModeFromString(*mode))
	if err != nil {
		log.Fatal(err)
	}

	log.Print("*", ep.SecurityPolicyURI, ep.SecurityMode)

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

	m, err := monitor.NewNodeMonitor(c)
	if err != nil {
		log.Fatal(err)
	}

	m.SetErrorHandler(func(_ *opcua.Client, sub *monitor.Subscription, err error) {
		log.Printf("error: sub=%d err=%s", sub.SubscriptionID(), err.Error())
	})
	wg := &sync.WaitGroup{}

	fieldNames := []string{"EventId", "EventType", "Severity", "Time", "Message"}
	selects := make([]*ua.SimpleAttributeOperand, len(fieldNames))
	for i, name := range fieldNames {
		selects[i] = &ua.SimpleAttributeOperand{
			TypeDefinitionID: ua.NewNumericNodeID(0, id.BaseEventType),
			BrowsePath:       []*ua.QualifiedName{{NamespaceIndex: 0, Name: name}},
			AttributeID:      ua.AttributeIDValue,
		}
	}

	wheres := &ua.ContentFilter{
		Elements: []*ua.ContentFilterElement{
			{
				FilterOperator: ua.FilterOperatorGreaterThanOrEqual,
				FilterOperands: []*ua.ExtensionObject{
					{
						EncodingMask: 1,
						TypeID: &ua.ExpandedNodeID{
							NodeID: ua.NewNumericNodeID(0, id.SimpleAttributeOperand_Encoding_DefaultBinary),
						},
						Value: ua.SimpleAttributeOperand{
							TypeDefinitionID: ua.NewNumericNodeID(0, id.BaseEventType),
							BrowsePath:       []*ua.QualifiedName{{NamespaceIndex: 0, Name: "Severity"}},
							AttributeID:      ua.AttributeIDValue,
						},
					},
					{
						EncodingMask: 1,
						TypeID: &ua.ExpandedNodeID{
							NodeID: ua.NewNumericNodeID(0, id.LiteralOperand_Encoding_DefaultBinary),
						},
						Value: ua.LiteralOperand{
							Value: ua.MustVariant(uint16(0)),
						},
					},
				},
			},
		},
	}

	filter := ua.EventFilter{
		SelectClauses: selects,
		WhereClause:   wheres,
	}

	filterExtObj := ua.ExtensionObject{
		EncodingMask: ua.ExtensionObjectBinary,
		TypeID: &ua.ExpandedNodeID{
			NodeID: ua.NewNumericNodeID(0, id.EventFilter_Encoding_DefaultBinary),
		},
		Value: filter,
	}

	if *event {
		// start callback-based subscription
		wg.Add(1)
		go startCallbackSub(ctx, m, *interval, 0, wg, *event, &filterExtObj, *nodeID)

		// start channel-based subscription
		wg.Add(1)
		go startChanSub(ctx, m, *interval, 0, wg, *event, &filterExtObj, *nodeID)
	} else {
		// start callback-based subscription
		wg.Add(1)
		go startCallbackSub(ctx, m, *interval, 0, wg, *event, nil, *nodeID)

		// start channel-based subscription
		wg.Add(1)
		go startChanSub(ctx, m, *interval, 0, wg, *event, nil, *nodeID)
	}
	<-ctx.Done()
	wg.Wait()
}

func startCallbackSub(ctx context.Context, m *monitor.NodeMonitor, interval, lag time.Duration, wg *sync.WaitGroup, isEvent bool, filter *ua.ExtensionObject, nodes ...string) {
	fieldNames := []string{"EventId", "EventType", "Severity", "Time", "Message"}
	sub, err := m.Subscribe(
		ctx,
		&opcua.SubscriptionParameters{
			Interval: interval,
		},
		func(s *monitor.Subscription, msg monitor.Message) {
			switch v := msg.(type) {
			case *monitor.DataChangeMessage:
				if v.Error != nil {
					log.Printf("[callback] sub=%d error=%s", s.SubscriptionID(), v.Error)
				} else {
					log.Printf("[callback] sub=%d ts=%s node=%s value=%v",
						s.SubscriptionID(),
						v.SourceTimestamp.UTC().Format(time.RFC3339),
						v.NodeID,
						v.Value.Value())
				}
			case *monitor.EventMessage:
				if v.Error != nil {
					log.Printf("[callback] sub=%d error=%s", s.SubscriptionID(), v.Error)
				} else {
					log.Printf("[callback] sub=%d event details:", s.SubscriptionID())
					for i, field := range v.EventFields {
						if i < len(fieldNames) {
							fieldName := fieldNames[i]
							log.Printf("  %s: %v", fieldName, field.Value.Value())
						}
					}
				}
			default:
				log.Printf("[callback] sub=%d unknown message type=%T", s.SubscriptionID(), msg)
			}
			time.Sleep(lag)
		},
		isEvent, filter, nodes...)
	if err != nil {
		log.Fatal(err)
	}

	defer cleanup(ctx, sub, wg)

	<-ctx.Done()
}

func startChanSub(ctx context.Context, m *monitor.NodeMonitor, interval, lag time.Duration, wg *sync.WaitGroup, isEvent bool, filter *ua.ExtensionObject, nodes ...string) {
	ch := make(chan monitor.Message, 16)
	sub, err := m.ChanSubscribe(ctx, &opcua.SubscriptionParameters{Interval: interval}, ch, isEvent, filter, nodes...)

	if err != nil {
		log.Fatal(err)
	}

	defer cleanup(ctx, sub, wg)

	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-ch:
			switch v := msg.(type) {
			case *monitor.DataChangeMessage:
				if v.Error != nil {
					log.Printf("[channel] sub=%d error=%s", sub.SubscriptionID(), v.Error)
				} else {
					log.Printf("[channel] sub=%d ts=%s node=%s value=%v",
						sub.SubscriptionID(),
						v.SourceTimestamp.UTC().Format(time.RFC3339),
						v.NodeID,
						v.Value.Value())
				}
			case *monitor.EventMessage:
				if v.Error != nil {
					log.Printf("[channel] sub=%d error=%s", sub.SubscriptionID(), v.Error)
				} else {
					out := v.EventFields[0].Value.Value()
					log.Printf("[channel] sub=%d event fields=%d",
						sub.SubscriptionID(), out)
				}
			default:
				log.Printf("[channel] sub=%d unknown message type: %T", sub.SubscriptionID(), msg)
			}
			time.Sleep(lag)
		}
	}
}

func cleanup(ctx context.Context, sub *monitor.Subscription, wg *sync.WaitGroup) {
	log.Printf("stats: sub=%d delivered=%d dropped=%d", sub.SubscriptionID(), sub.Delivered(), sub.Dropped())
	sub.Unsubscribe(ctx)
	wg.Done()
}
