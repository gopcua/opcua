//go:build integration
// +build integration

package uatest2

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/id"
	"github.com/gopcua/opcua/server"
	"github.com/gopcua/opcua/ua"
	"github.com/stretchr/testify/require"
)

// TestSubscriptionDeliversQueuedValues checks that a data MonitoredItem with
// a queue size larger than one delivers every value sampled within a
// publishing interval, in queue order per item (Part 4 §5.13.1.5, §7.25.1),
// and that two items with the same ClientHandle do not share a queue. The
// expected interleaving of the two items is this server's sample-time order,
// which the spec does not require.
func TestSubscriptionDeliversQueuedValues(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var mu sync.Mutex
	values := map[string]int32{"a": 0, "b": 0}

	srv := server.New(
		server.EnableSecurity("None", ua.MessageSecurityModeNone),
		server.EnableAuthMode(ua.UserTokenTypeAnonymous),
		server.EndPoint("localhost", 4843),
	)
	ns := server.NewNodeNameSpace(srv, "urn:test:queue")
	root, _ := srv.Namespace(0)
	root.Objects().AddRef(ns.Objects(), id.HasComponent, true)
	nodes := map[string]*ua.NodeID{}
	for name := range values {
		name := name
		nodeID := ua.NewStringNodeID(ns.ID(), name)
		n := server.NewVariableNode(nodeID, name, func() *ua.DataValue {
			mu.Lock()
			defer mu.Unlock()
			return server.DataValueFromValue(values[name])
		})
		ns.AddNode(n)
		ns.Objects().AddRef(n, id.HasComponent, true)
		nodes[name] = nodeID
	}
	require.NoError(t, srv.Start(ctx), "Start failed")
	defer srv.Close()

	c, err := opcua.NewClient("opc.tcp://localhost:4843", opcua.SecurityMode(ua.MessageSecurityModeNone))
	require.NoError(t, err, "NewClient failed")
	require.NoError(t, c.Connect(ctx), "Connect failed")
	defer c.Close(ctx)

	notifyCh := make(chan *opcua.PublishNotificationData, 16)
	sub, err := c.Subscribe(ctx, &opcua.SubscriptionParameters{Interval: time.Second}, notifyCh)
	require.NoError(t, err, "Subscribe failed")
	defer sub.Cancel(ctx)

	const handle = 42
	var reqs []*ua.MonitoredItemCreateRequest
	for _, name := range []string{"a", "b"} {
		req := opcua.NewMonitoredItemCreateRequestWithDefaults(nodes[name], ua.AttributeIDValue, handle)
		req.RequestedParameters.QueueSize = 10
		reqs = append(reqs, req)
	}
	res, err := sub.Monitor(ctx, ua.TimestampsToReturnBoth, reqs...)
	require.NoError(t, err, "Monitor failed")
	for _, r := range res.Results {
		require.Equal(t, ua.StatusOK, r.StatusCode)
		require.Equal(t, uint32(10), r.RevisedQueueSize)
	}

	// collect returns the values of the next n data change notifications.
	collect := func(n int) []int32 {
		var got []int32
		for len(got) < n {
			select {
			case msg := <-notifyCh:
				require.NoError(t, msg.Error)
				dcn, ok := msg.Value.(*ua.DataChangeNotification)
				require.True(t, ok, "unexpected notification %T", msg.Value)
				for _, item := range dcn.MonitoredItems {
					require.Equal(t, uint32(handle), item.ClientHandle)
					require.Equal(t, ua.StatusOK, item.Value.Status)
					got = append(got, item.Value.Value.Value().(int32))
				}
			case <-ctx.Done():
				t.Fatalf("got %v, want %d values", got, n)
			}
		}
		return got
	}
	require.ElementsMatch(t, []int32{0, 0}, collect(2), "initial values")

	// Change both values three times within one publishing interval.
	for i := int32(1); i <= 3; i++ {
		for _, name := range []string{"a", "b"} {
			mu.Lock()
			values[name] = i
			if name == "b" {
				values[name] = 100 + i
			}
			mu.Unlock()
			srv.ChangeNotification(nodes[name])
		}
	}
	require.Equal(t, []int32{1, 101, 2, 102, 3, 103}, collect(6))
}
