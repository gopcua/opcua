package server

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/id"
	"github.com/gopcua/opcua/ua"
)

// Cancelling a subscription while the initial value notifications of its monitored items are
// still queued must not stop the server. ChangeNotification used to block on the subscription's
// full NotifyChannel after its task had returned, holding the monitored item service lock that the
// task's cleanup needed, and the server's message loop then waited forever on the subscription
// service lock in the next CreateSubscription.
func TestServerAnswersAfterSubscriptionCancelledWithNotificationsQueued(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	port := ln.Addr().(*net.TCPAddr).Port
	require.NoError(t, ln.Close())

	srv := New(EndPoint("127.0.0.1", port), EnableSecurity("None", ua.MessageSecurityModeNone))
	require.NoError(t, srv.Start(ctx))
	defer srv.Close()
	endpoint := fmt.Sprintf("opc.tcp://127.0.0.1:%d", port)

	c, err := opcua.NewClient(endpoint, opcua.SecurityMode(ua.MessageSecurityModeNone))
	require.NoError(t, err)
	require.NoError(t, c.Connect(ctx))
	defer c.Close(context.Background())

	// More items than NotifyChannel holds (100). Each one queues its initial notification.
	node := ua.NewNumericNodeID(0, id.Server_ServerStatus_CurrentTime)
	items := make([]*ua.MonitoredItemCreateRequest, 150)
	for i := range items {
		items[i] = opcua.NewMonitoredItemCreateRequestWithDefaults(node, ua.AttributeIDValue, uint32(i+1))
	}

	for round := range 10 {
		rctx, rcancel := context.WithTimeout(ctx, 5*time.Second)
		sub, err := c.Subscribe(rctx, &opcua.SubscriptionParameters{Interval: 100 * time.Millisecond}, make(chan *opcua.PublishNotificationData, 1000))
		require.NoError(t, err, "round %d: subscribe", round)
		_, err = sub.Monitor(rctx, ua.TimestampsToReturnBoth, items...)
		require.NoError(t, err, "round %d: monitor", round)
		// Cancel right away, while the notifications are queued.
		_ = sub.Cancel(rctx)
		rcancel()

		pctx, pcancel := context.WithTimeout(ctx, 5*time.Second)
		_, err = opcua.GetEndpoints(pctx, endpoint)
		pcancel()
		require.NoError(t, err, "round %d: the server stopped answering after a subscription was cancelled", round)
	}

	// The deadlock only shows once the message loop reaches the next CreateSubscription, which
	// each round's Subscribe provides for the round before it. Give the last round one too.
	fctx, fcancel := context.WithTimeout(ctx, 5*time.Second)
	defer fcancel()
	_, err = c.Subscribe(fctx, &opcua.SubscriptionParameters{Interval: time.Second}, make(chan *opcua.PublishNotificationData, 1))
	require.NoError(t, err, "subscribe after the last cancel")
}
