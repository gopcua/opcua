// Copyright 2018-2020 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package server

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/ua"
)

// freePort asks the kernel for a free TCP port on the loopback interface.
func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

// startTestServer starts a server with one variable node in a node namespace
// and returns it together with its endpoint URL.
func startTestServer(t *testing.T) (*Server, string, *Node) {
	t.Helper()
	port := freePort(t)
	srv := New(
		EndPoint("127.0.0.1", port),
		EnableSecurity("None", ua.MessageSecurityModeNone),
		EnableAuthMode(ua.UserTokenTypeAnonymous),
	)
	ns := NewNodeNameSpace(srv, "test")
	node := ns.AddNewVariableNode("v", int32(1))
	ns.Objects().AddRef(node, RefTypeIDHasComponent, true)

	ctx, cancel := context.WithCancel(context.Background())
	require.NoError(t, srv.Start(ctx))
	t.Cleanup(func() {
		cancel()
		srv.Close()
	})
	return srv, fmt.Sprintf("opc.tcp://127.0.0.1:%d", port), node
}

func (s *SubscriptionService) subscription(id uint32) *Subscription {
	s.Mu.Lock()
	defer s.Mu.Unlock()
	return s.Subs[id]
}

// TestPublishWithoutSubscriptionsReturnsBadNoSubscription: a Publish request on a
// session that has no subscriptions must be answered immediately with
// Bad_NoSubscription (Part 4, 5.13.5) instead of being queued forever, which
// leaves the client waiting for notifications that never come.
func TestPublishWithoutSubscriptionsReturnsBadNoSubscription(t *testing.T) {
	srv := New()
	srv.SubscriptionService = &SubscriptionService{srv: srv, Subs: make(map[uint32]*Subscription)}
	sess := srv.sb.NewSession()

	req := &ua.PublishRequest{
		RequestHeader: &ua.RequestHeader{AuthenticationToken: sess.AuthTokenID, RequestHandle: 7},
	}
	resp, err := srv.SubscriptionService.Publish(nil, req, 1)
	require.NoError(t, err)
	require.NotNil(t, resp, "Publish without subscriptions must be answered right away")

	pr, ok := resp.(*ua.PublishResponse)
	require.True(t, ok, "expected *ua.PublishResponse, got %T", resp)
	require.Equal(t, ua.StatusBadNoSubscription, pr.ResponseHeader.ServiceResult)
	require.Equal(t, uint32(7), pr.ResponseHeader.RequestHandle)
	require.Empty(t, sess.PublishRequests, "the request must not be queued")
}

// TestSubscriptionSurvivesSecureChannelReconnect reproduces what UaExpert does
// after a connection loss: it opens a new secure channel and re-activates the
// same session. The subscription belongs to the session, so it must keep
// publishing on the new channel. Before the fix the server replied on the
// channel the subscription was created on, hit "broken pipe" on the dead
// socket and deleted the subscription while the session stayed alive.
func TestSubscriptionSurvivesSecureChannelReconnect(t *testing.T) {
	srv, endpoint, node := startTestServer(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	opts := func() []opcua.Option {
		return []opcua.Option{
			opcua.SecurityMode(ua.MessageSecurityModeNone),
			opcua.AutoReconnect(false),
		}
	}
	params := &opcua.SubscriptionParameters{
		Interval:          50 * time.Millisecond,
		MaxKeepAliveCount: 2,
		LifetimeCount:     1000,
	}
	notify := make(chan *opcua.PublishNotificationData, 100)

	// client 1: session A with a subscription on secure channel 1
	c1, err := opcua.NewClient(endpoint, opts()...)
	require.NoError(t, err)
	require.NoError(t, c1.Connect(ctx))
	sub1, err := c1.Subscribe(ctx, params, notify)
	require.NoError(t, err)
	_, err = sub1.Monitor(ctx, ua.TimestampsToReturnBoth,
		opcua.NewMonitoredItemCreateRequestWithDefaults(node.ID(), ua.AttributeIDValue, 1))
	require.NoError(t, err)
	require.NotNil(t, srv.SubscriptionService.subscription(sub1.SubscriptionID))

	// let a few publish cycles run on channel 1
	time.Sleep(300 * time.Millisecond)

	// drop secure channel 1 but keep session A
	sessA, err := c1.DetachSession(ctx)
	require.NoError(t, err)
	require.NoError(t, c1.Close(ctx))

	// client 2: new secure channel 2, same session A
	c2, err := opcua.NewClient(endpoint, opts()...)
	require.NoError(t, err)
	require.NoError(t, c2.Dial(ctx))
	defer c2.Close(ctx)
	require.NoError(t, c2.ActivateSession(ctx, sessA))

	// A Publish request sent on channel 2 must be answered on channel 2. With
	// no data change pending the answer is the keep-alive of subscription 1.
	// Before the fix the answer went to channel 1 and this timed out.
	for i := 0; i < 3; i++ {
		var pr *ua.PublishResponse
		err = c2.Send(ctx, &ua.PublishRequest{SubscriptionAcknowledgements: []*ua.SubscriptionAcknowledgement{}}, func(v ua.Response) error {
			r, ok := v.(*ua.PublishResponse)
			if !ok {
				return fmt.Errorf("expected *ua.PublishResponse, got %T", v)
			}
			pr = r
			return nil
		})
		require.NoError(t, err, "publish %d on the new secure channel", i)
		require.Equal(t, ua.StatusOK, pr.ResponseHeader.ServiceResult)
		require.Equal(t, sub1.SubscriptionID, pr.SubscriptionID)
	}

	require.NotNil(t, srv.SubscriptionService.subscription(sub1.SubscriptionID),
		"subscription %d was deleted after the client reconnected on a new secure channel", sub1.SubscriptionID)

	// the session itself is still usable on channel 2
	rr, err := c2.Read(ctx, &ua.ReadRequest{NodesToRead: []*ua.ReadValueID{{NodeID: node.ID(), AttributeID: ua.AttributeIDValue}}})
	require.NoError(t, err)
	require.Equal(t, ua.StatusOK, rr.Results[0].Status)
}
