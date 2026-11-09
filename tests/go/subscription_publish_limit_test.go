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

// TestSubscriptionHonorsMaxNotificationsPerPublish checks that a Publish
// response carries at most maxNotificationsPerPublish values (Part 4
// §5.14.2.2), and that the values that did not fit are sent with the next
// Publish responses without waiting for the publishing interval (§5.14.1.2).
func TestSubscriptionHonorsMaxNotificationsPerPublish(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var mu sync.Mutex
	var value int32

	srv := server.New(
		server.EnableSecurity("None", ua.MessageSecurityModeNone),
		server.EnableAuthMode(ua.UserTokenTypeAnonymous),
		server.EndPoint("localhost", 4844),
	)
	ns := server.NewNodeNameSpace(srv, "urn:test:queue")
	root, _ := srv.Namespace(0)
	root.Objects().AddRef(ns.Objects(), id.HasComponent, true)
	nodeID := ua.NewStringNodeID(ns.ID(), "value")
	n := server.NewVariableNode(nodeID, "value", func() *ua.DataValue {
		mu.Lock()
		defer mu.Unlock()
		return server.DataValueFromValue(value)
	})
	ns.AddNode(n)
	ns.Objects().AddRef(n, id.HasComponent, true)
	require.NoError(t, srv.Start(ctx), "Start failed")
	defer srv.Close()

	c, err := opcua.NewClient("opc.tcp://localhost:4844", opcua.SecurityMode(ua.MessageSecurityModeNone))
	require.NoError(t, err, "NewClient failed")
	require.NoError(t, c.Connect(ctx), "Connect failed")
	defer c.Close(ctx)

	const (
		interval = 2 * time.Second
		limit    = 2
	)
	notifyCh := make(chan *opcua.PublishNotificationData, 16)
	sub, err := c.Subscribe(ctx, &opcua.SubscriptionParameters{
		Interval:                   interval,
		MaxNotificationsPerPublish: limit,
	}, notifyCh)
	require.NoError(t, err, "Subscribe failed")
	defer sub.Cancel(ctx)

	req := opcua.NewMonitoredItemCreateRequestWithDefaults(nodeID, ua.AttributeIDValue, 1)
	req.RequestedParameters.QueueSize = 10
	_, err = sub.Monitor(ctx, ua.TimestampsToReturnBoth, req)
	require.NoError(t, err, "Monitor failed")

	// next returns the values of the next data change notification.
	next := func() []int32 {
		select {
		case msg := <-notifyCh:
			require.NoError(t, msg.Error)
			dcn, ok := msg.Value.(*ua.DataChangeNotification)
			require.True(t, ok, "unexpected notification %T", msg.Value)
			var got []int32
			for _, item := range dcn.MonitoredItems {
				got = append(got, item.Value.Value.Value().(int32))
			}
			return got
		case <-ctx.Done():
			t.Fatal("no notification")
			return nil
		}
	}
	require.Equal(t, []int32{0}, next(), "initial value")

	// Queue five values within one publishing interval.
	for i := int32(1); i <= 5; i++ {
		mu.Lock()
		value = i
		mu.Unlock()
		srv.ChangeNotification(nodeID)
	}

	var got []int32
	var first time.Time
	for len(got) < 5 {
		vals := next()
		if first.IsZero() {
			first = time.Now()
		}
		require.LessOrEqual(t, len(vals), limit, "values in one publish response")
		got = append(got, vals...)
	}
	require.Equal(t, []int32{1, 2, 3, 4, 5}, got)
	require.Less(t, time.Since(first), interval, "remaining values waited for the publishing interval")
}
