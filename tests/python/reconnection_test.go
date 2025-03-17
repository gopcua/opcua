//go:build integration

package uatest

import (
	"context"
	"testing"
	"time"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/monitor"
	"github.com/gopcua/opcua/ua"
	"github.com/stretchr/testify/require"
)

const (
	currentTimeNodeID   = "ns=0;i=2258"
	disconnectTimeout   = 5 * time.Second
	reconnectionTimeout = 10 * time.Second
)

// TestAutoReconnection performs an integration test the auto reconnection
// from an OPC/UA server.
func TestAutoReconnection(t *testing.T) {
	ctx := context.Background()

	srv := NewPythonServer("reconnection_server.py")
	defer srv.Close()

	c, err := opcua.NewClient(srv.Endpoint, srv.Opts...)
	require.NoError(t, err, "NewClient failed")

	err = c.Connect(ctx)
	require.NoError(t, err, "Connect failed")
	defer c.Close(ctx)

	m, err := monitor.NewNodeMonitor(c)
	require.NoError(t, err, "NewNodeMonitor failed")

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	tests := []struct {
		name string
		req  *ua.CallMethodRequest
	}{
		{
			name: "connection_failure",
			req: &ua.CallMethodRequest{
				ObjectID:       ua.NewStringNodeID(2, "simulations"),
				MethodID:       ua.NewStringNodeID(2, "simulate_connection_failure"),
				InputArguments: []*ua.Variant{},
			},
		},
		{
			name: "securechannel_failure",
			req: &ua.CallMethodRequest{
				ObjectID:       ua.NewStringNodeID(2, "simulations"),
				MethodID:       ua.NewStringNodeID(2, "simulate_securechannel_failure"),
				InputArguments: []*ua.Variant{},
			},
		},
		{
			name: "session_failure",
			req: &ua.CallMethodRequest{
				ObjectID:       ua.NewStringNodeID(2, "simulations"),
				MethodID:       ua.NewStringNodeID(2, "simulate_session_failure"),
				InputArguments: []*ua.Variant{},
			},
		},
		{
			name: "subscription_failure",
			req: &ua.CallMethodRequest{
				ObjectID:       ua.NewStringNodeID(2, "simulations"),
				MethodID:       ua.NewStringNodeID(2, "simulate_subscription_failure"),
				InputArguments: []*ua.Variant{},
			},
		},
	}

	ch := make(chan monitor.Message, 5)
	sctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sub, err := m.ChanSubscribe(
		sctx,
		&opcua.SubscriptionParameters{Interval: opcua.DefaultSubscriptionInterval},
		ch,
		false,
		nil,
		currentTimeNodeID,
	)
	require.NoError(t, err, "ChanSubscribe failed")
	defer sub.Unsubscribe(ctx)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Wait for the first message
			select {
			case msg := <-ch:
				switch v := msg.(type) {
				case *monitor.DataChangeMessage:
					require.NoError(t, v.Error, "No error expected for first value")
				default:
					require.Fail(t, "Unexpected message type: %T", msg)
				}
			case <-time.After(5 * time.Second):
				require.Fail(t, "Timeout waiting for first message")
			}

			downC := make(chan struct{}, 1)
			dTimeout := time.NewTimer(disconnectTimeout)
			go c.Call(ctx, tt.req)

			ctx, cancel := context.WithCancel(context.Background())
			go func() {
				// make sure the connection is down
				for {
					select {
					case <-ctx.Done():
						return
					default:
						if c.State() != opcua.Connected {
							downC <- struct{}{}
							return
						}
						// HACK: scanning the state of client to determine if the connection has failed
						// is not good pratice, as with powerful machine the reconnection could be faster
						// then 1 ms and it will not detect the change, a solution could be a state event
						// or a reconnection counter
						time.Sleep(1 * time.Millisecond)
					}
				}
			}()

			select {
			case <-dTimeout.C:
				cancel()
				require.Fail(t, "Timeout reached, the connection did not go down as expected")
			case <-downC:
			}

			// empty out the channel
			for len(ch) > 0 {
				<-ch
			}

			rTimeout := time.NewTimer(reconnectionTimeout)
			select {
			case <-rTimeout.C:
				require.Fail(t, "Timeout reached, reconnection failed")
			case msg := <-ch:
				switch v := msg.(type) {
				case *monitor.DataChangeMessage:
					require.NoError(t, v.Error)
				default:
					require.Fail(t, "Unexpected message type: %T", msg)
				}
			}
		})
	}
}
