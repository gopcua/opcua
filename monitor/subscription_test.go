package monitor

import (
	"context"
	"fmt"
	"net"
	"slices"
	"testing"
	"time"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/server"
	"github.com/gopcua/opcua/ua"
	"github.com/stretchr/testify/require"
)

const notificationTimeout = 5 * time.Second

// testNodes holds distinct Int32 values so a notification's value identifies
// which node produced it.
var testNodes = []struct {
	name  string
	value int32
}{
	{"batch_a", 10},
	{"batch_b", 20},
	{"batch_c", 30},
}

// startTestServer retries on a fresh port because server.New fixes the
// listen address from its first endpoint, and Start has no setter to change it.
func startTestServer(t *testing.T) (string, []*ua.NodeID) {
	t.Helper()

	const attempts = 5
	for range attempts {
		port, err := freePort()
		require.NoError(t, err, "reserve a free port")

		s := server.New(
			server.EndPoint("127.0.0.1", port),
			server.EnableSecurity("None", ua.MessageSecurityModeNone),
			server.EnableAuthMode(ua.UserTokenTypeAnonymous),
		)

		ns := server.NewNodeNameSpace(s, "TestNamespace")
		s.AddNamespace(ns)

		nodeIDs := make([]*ua.NodeID, 0, len(testNodes))
		for _, n := range testNodes {
			nodeIDs = append(nodeIDs, ns.AddNewVariableStringNode(n.name, n.value).ID())
		}

		if err := s.Start(context.Background()); err != nil {
			t.Logf("server did not start on port %d, retrying on a fresh port: %v", port, err)
			_ = s.Close()
			continue
		}
		t.Cleanup(func() { _ = s.Close() })
		return fmt.Sprintf("opc.tcp://127.0.0.1:%d", port), nodeIDs
	}

	t.Fatalf("no free port survived long enough to start the server in %d attempts", attempts)
	return "", nil
}

func freePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	port := l.Addr().(*net.TCPAddr).Port
	// A port still held by an unclosed listener isn't free.
	if err := l.Close(); err != nil {
		return 0, err
	}
	return port, nil
}

func newTestSubscription(t *testing.T, ctx context.Context, endpoint string) (*Subscription, <-chan *DataChangeMessage) {
	t.Helper()

	c, err := opcua.NewClient(endpoint, opcua.SecurityMode(ua.MessageSecurityModeNone))
	require.NoError(t, err, "NewClient failed")
	require.NoError(t, c.Connect(ctx), "Connect failed")
	t.Cleanup(func() { _ = c.Close(ctx) })

	m, err := NewNodeMonitor(c)
	require.NoError(t, err, "NewNodeMonitor failed")
	m.SetErrorHandler(func(_ *opcua.Client, _ *Subscription, err error) {
		t.Logf("subscription reported an out-of-band error: %v", err)
	})

	ch := make(chan *DataChangeMessage, 16)
	sub, err := m.ChanSubscribe(ctx, &opcua.SubscriptionParameters{Interval: 50 * time.Millisecond}, ch)
	require.NoError(t, err, "ChanSubscribe failed")
	t.Cleanup(func() { _ = sub.Unsubscribe(ctx) })

	return sub, ch
}

func valuesByNodeID(t *testing.T, ch <-chan *DataChangeMessage, want int) map[string][]int32 {
	t.Helper()

	got := make(map[string][]int32, want)
	deadline := time.After(notificationTimeout)
	for len(got) < want {
		select {
		case msg := <-ch:
			require.NoError(t, msg.Error, "notification carried an error")
			require.NotNil(t, msg.NodeID, "notification carried no NodeID")
			require.NotNil(t, msg.DataValue, "notification for %s carried no DataValue", msg.NodeID)
			require.NotNil(t, msg.Value, "notification for %s carried no Value", msg.NodeID)

			v, ok := msg.Value.Value().(int32)
			require.True(t, ok, "notification for %s carried %T, want int32", msg.NodeID, msg.Value.Value())

			id := msg.NodeID.String()
			if !slices.Contains(got[id], v) {
				got[id] = append(got[id], v)
			}
		case <-deadline:
			return got
		}
	}
	return got
}

// wantValues pairs the first n nodeIDs with testNodes' values, in order.
func wantValues(nodeIDs []*ua.NodeID, n int) map[string][]int32 {
	want := make(map[string][]int32, n)
	for i := range n {
		want[nodeIDs[i].String()] = []int32{testNodes[i].value}
	}
	return want
}

// TestBatchedAddMonitorItemsNotifiesEachNode verifies that each node's value
// arrives under its own NodeID. A MonitoredItemNotification carries only a
// handle and a value, so duplicate ClientHandles would make values
// indistinguishable.
func TestBatchedAddMonitorItemsNotifiesEachNode(t *testing.T) {
	ctx := context.Background()

	endpoint, nodeIDs := startTestServer(t)
	sub, ch := newTestSubscription(t, ctx, endpoint)

	shared := &ua.MonitoringParameters{
		SamplingInterval: 250,
		QueueSize:        5,
		DiscardOldest:    true,
	}
	reqs := make([]Request, 0, len(nodeIDs))
	for _, nid := range nodeIDs {
		reqs = append(reqs, Request{
			NodeID:               nid,
			MonitoringMode:       ua.MonitoringModeReporting,
			MonitoringParameters: shared,
		})
	}

	items, err := sub.AddMonitorItems(ctx, reqs...)
	require.NoError(t, err, "AddMonitorItems failed")
	require.Len(t, items, len(nodeIDs), "one item per request")

	got := valuesByNodeID(t, ch, len(nodeIDs))
	require.Equal(t, wantValues(nodeIDs, len(nodeIDs)), got,
		"each node's value must arrive under its own NodeID; got %v", got)
}

// TestAddNodeIDsWithNilParametersNotifies verifies that a node monitored with
// nil MonitoringParameters still arrives under its own NodeID.
func TestAddNodeIDsWithNilParametersNotifies(t *testing.T) {
	ctx := context.Background()

	endpoint, nodeIDs := startTestServer(t)
	sub, ch := newTestSubscription(t, ctx, endpoint)

	require.NoError(t, sub.AddNodeIDs(ctx, nodeIDs[0]), "AddNodeIDs failed")

	got := valuesByNodeID(t, ch, 1)
	require.Equal(t, wantValues(nodeIDs, 1), got,
		"the nil-parameters node's value must arrive under its own NodeID; got %v", got)
}
