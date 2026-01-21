//go:build integration
// +build integration

package uatest2

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/server"
	"github.com/gopcua/opcua/ua"
	"github.com/stretchr/testify/require"
)

type subscriptionTest struct {
	name                        string
	queueSize                   uint32
	noChangeNotifications       uint32
	discardOldest               bool
	publishingInterval          time.Duration
	changeNotifications         []int
	expectedNotificationIndices []int
	expectedQueueSize           uint32
}

// TestSubscription performs an integration test to
// create some subscriptions and verify their queueing behaviour
func TestSubscription(t *testing.T) {

	//TODO:
	// - QueueSize 0, this should result in revised queueSize 1
	// - QueueSize 1
	// - QueueSize 10
	// - QueueSize 10, 15 ChangeNotifications with DiscardOldest = true
	// - QueueSize 10, 15 ChangNotifications with DiscardOldest = false
	// - QueueSize 10, 5 Notifications published

	// TO VERIFY:
	// - correct number of ChangeNotifications
	// - correct value in RevisedQueueSize

	tests := []subscriptionTest{
		{
			"test1",
			0,
			1,
			true,
			time.Duration(time.Millisecond * 500),
			[]int{
				1, 2, 3, 4, 5,
			},
			[]int{
				4,
			},
			1,
		},
	}

	ctx := context.Background()

	srv := startServer()
	defer srv.Close()

	time.Sleep(2 * time.Second)

	c, err := opcua.NewClient("opc.tcp://localhost:4840", opcua.SecurityMode(ua.MessageSecurityModeNone))
	require.NoError(t, err, "NewClient failed")

	err = c.Connect(ctx)
	require.NoError(t, err, "Connect failed")
	defer c.Close(ctx)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// testSubscription(t, ctx, srv, c, tt)
		})
	}
}

func testSubscription(t *testing.T, ctx context.Context, srv *server.Server, c *opcua.Client, st subscriptionTest) {
	t.Helper()

	results := make([]*ua.MonitoredItemNotification, 0)
	notifyCh := make(chan *opcua.PublishNotificationData)

	wg := new(sync.WaitGroup)
	wg.Add(2)

	notifications := convertToChangeNotifications(st.changeNotifications)
	expectedNotifications := make([]*ua.MonitoredItemNotification, 0)
	for _, index := range st.expectedNotificationIndices {
		expectedNotifications = append(expectedNotifications, notifications[index])
	}

	sub, err := c.Subscribe(ctx, &opcua.SubscriptionParameters{
		Interval: st.publishingInterval,
	}, notifyCh)
	require.NoError(t, err, "Subscribe failed")
	defer sub.Cancel(ctx)

	nodeID := ua.NewStringNodeID(1, "rw_int32")
	require.NoError(t, err, "not a valid nodeId")

	miCreateRequest := opcua.NewMonitoredItemCreateRequestWithDefaults(nodeID, ua.AttributeIDValue, 11)
	miCreateRequest.RequestedParameters.QueueSize = st.queueSize

	res, err := sub.Monitor(ctx, ua.TimestampsToReturnBoth, miCreateRequest)
	require.NoError(t, err, "Monitor failed")
	require.Equal(t, res.Results[0].StatusCode, ua.StatusOK)

	go func() {
		for {
			select {
			case <-ctx.Done():
				wg.Done()
				return
			case res := <-notifyCh:
				require.NoError(t, res.Error, "received notification with error")

				switch x := res.Value.(type) {
				case *ua.DataChangeNotification:
					results = append(results, x.MonitoredItems...)

					if len(results) == len(expectedNotifications) {
						wg.Done()
						return
					}

				default:
					return
				}
			}
		}
	}()

	go func() {
		time.Sleep(time.Duration(time.Second) * 5)
		for _, value := range notifications {
			node := srv.Node(nodeID)
			err := node.SetAttribute(ua.AttributeIDValue, value.Value)
			if err != nil {
				return
			}
			srv.ChangeNotification(nodeID)
			time.Sleep(time.Duration(time.Second) * 5)
		}

		wg.Done()
	}()

	wg.Wait()

	require.Equal(t, res.Results[0].RevisedQueueSize, st.expectedQueueSize)
	require.ElementsMatch(t, results, expectedNotifications)
}

func convertToChangeNotifications(values []int) []*ua.MonitoredItemNotification {
	startTimestamp, _ := time.Parse(time.RFC3339, "2025-11-03T10:24:10Z")
	results := make([]*ua.MonitoredItemNotification, len(values))

	for i, value := range values {
		results[i] = &ua.MonitoredItemNotification{
			ClientHandle: 11,
			Value: &ua.DataValue{
				Value:           ua.MustVariant(int32(value)),
				SourceTimestamp: startTimestamp.Add(time.Duration(i * int(time.Millisecond))),
				ServerTimestamp: startTimestamp.Add(time.Duration(i * int(time.Millisecond))),
				EncodingMask:    ua.DataValueValue | ua.DataValueSourceTimestamp | ua.DataValueServerTimestamp,
				Status:          ua.StatusOK,
			},
		}
	}
	return results
}
