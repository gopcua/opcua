package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/internal/ualog"
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
		debug    = flag.Bool("debug", false, "enable debug logging")
	)
	flag.Parse()
	slog.SetDefault(slog.New(ualog.NewTextHandler(*debug)))

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

	m, err := monitor.NewNodeMonitor(c)
	if err != nil {
		ualog.Fatal("NewNodeMonitor failed", "error", err)
	}

	m.SetErrorHandler(func(_ *opcua.Client, sub *monitor.Subscription, err error) {
		ualog.Error("error", "sub_id", sub.SubscriptionID(), "error", err.Error())
	})
	wg := &sync.WaitGroup{}

	// start callback-based subscription
	wg.Add(1)
	go startCallbackSub(ctx, m, *interval, 0, wg, *nodeID)

	// start channel-based subscription
	wg.Add(1)
	go startChanSub(ctx, m, *interval, 0, wg, *nodeID)

	<-ctx.Done()
	wg.Wait()
}

func startCallbackSub(ctx context.Context, m *monitor.NodeMonitor, interval, lag time.Duration, wg *sync.WaitGroup, nodes ...string) {
	sub, err := m.Subscribe(
		ctx,
		&opcua.SubscriptionParameters{
			Interval: interval,
		},
		func(s *monitor.Subscription, msg *monitor.DataChangeMessage) {
			if msg.Error != nil {
				ualog.Error("[callback]", "sub_id", s.SubscriptionID(), "error", msg.Error)
			} else {
				ualog.Info("[callback]", "sub_id", s.SubscriptionID(), "timestamp", msg.SourceTimestamp.UTC().Format(time.RFC3339), "node_id", msg.NodeID, "value", msg.Value.Value())
			}
			time.Sleep(lag)
		},
		nodes...)

	if err != nil {
		ualog.Fatal("Subscribe failed", "error", err)
	}

	defer cleanup(ctx, sub, wg)

	<-ctx.Done()
}

func startChanSub(ctx context.Context, m *monitor.NodeMonitor, interval, lag time.Duration, wg *sync.WaitGroup, nodes ...string) {
	ch := make(chan *monitor.DataChangeMessage, 16)
	sub, err := m.ChanSubscribe(ctx, &opcua.SubscriptionParameters{Interval: interval}, ch, nodes...)

	if err != nil {
		ualog.Fatal("ChanSubscribe failed", "error", err)
	}

	defer cleanup(ctx, sub, wg)

	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-ch:
			if msg.Error != nil {
				ualog.Error("[channel]", "sub_id", sub.SubscriptionID(), "error", msg.Error)
			} else {
				ualog.Info("[channel]", "sub_id", sub.SubscriptionID(), "timestamp", msg.SourceTimestamp.UTC().Format(time.RFC3339), "node_id", msg.NodeID, "value", msg.Value.Value())
			}
			time.Sleep(lag)
		}
	}
}

func cleanup(ctx context.Context, sub *monitor.Subscription, wg *sync.WaitGroup) {
	ualog.Info("stats", "sub_id", sub.SubscriptionID(), "delivered", sub.Delivered(), "dropped", sub.Dropped())
	sub.Unsubscribe(ctx)
	wg.Done()
}
