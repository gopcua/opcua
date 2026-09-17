package opcua

import (
	"context"
	"expvar"
	"sync"
	"testing"
	"time"

	"github.com/gopcua/opcua/stats"
	"github.com/gopcua/opcua/ua"
	"github.com/stretchr/testify/require"
)

// stubClient is a ClientInterface which answers the requests a subscription
// sends while it is being recreated. It lets the reconnect logic be tested
// without a server.
type stubClient struct {
	mu       sync.Mutex
	send     func(req ua.Request, h func(ua.Response) error) error
	requests []ua.Request
}

var _ ClientInterface = (*stubClient)(nil)

func (c *stubClient) Send(ctx context.Context, req ua.Request, h func(ua.Response) error) error {
	c.mu.Lock()
	c.requests = append(c.requests, req)
	c.mu.Unlock()
	return c.send(req, h)
}

func (c *stubClient) sentRequests() []ua.Request {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]ua.Request{}, c.requests...)
}

func (c *stubClient) Browse(context.Context, *ua.BrowseRequest) (*ua.BrowseResponse, error) {
	return nil, ua.StatusBadNotImplemented
}

func (c *stubClient) BrowseNext(context.Context, *ua.BrowseNextRequest) (*ua.BrowseNextResponse, error) {
	return nil, ua.StatusBadNotImplemented
}

func (c *stubClient) Read(context.Context, *ua.ReadRequest) (*ua.ReadResponse, error) {
	return nil, ua.StatusBadNotImplemented
}

func (c *stubClient) Node(id *ua.NodeID) *Node { return NewNode(id, c) }
func (c *stubClient) NodeFromExpandedNodeID(id *ua.ExpandedNodeID) *Node {
	return NewNode(&ua.NodeID{}, c)
}
func (c *stubClient) ForgetSubscription(context.Context, uint32) {}
func (c *stubClient) RequestTimeout() time.Duration              { return time.Second }

// recreateResponder answers the DeleteSubscriptions, CreateSubscription and
// CreateMonitoredItems requests which recreateSubscription sends. newSubID is
// the subscription id the server assigns to the recreated subscription and
// itemStatus is the operation level status code returned for every monitored
// item.
func recreateResponder(newSubID uint32, itemStatus ua.StatusCode) func(ua.Request, func(ua.Response) error) error {
	return func(req ua.Request, h func(ua.Response) error) error {
		switch req := req.(type) {
		case *ua.DeleteSubscriptionsRequest:
			return h(&ua.DeleteSubscriptionsResponse{
				ResponseHeader: &ua.ResponseHeader{},
				Results:        []ua.StatusCode{ua.StatusOK},
			})

		case *ua.CreateSubscriptionRequest:
			return h(&ua.CreateSubscriptionResponse{
				ResponseHeader: &ua.ResponseHeader{},
				SubscriptionID: newSubID,
			})

		case *ua.CreateMonitoredItemsRequest:
			res := &ua.CreateMonitoredItemsResponse{ResponseHeader: &ua.ResponseHeader{}}
			for i := range req.ItemsToCreate {
				res.Results = append(res.Results, &ua.MonitoredItemCreateResult{
					StatusCode:      itemStatus,
					MonitoredItemID: uint32(i + 1),
				})
			}
			return h(res)

		default:
			return ua.StatusBadServiceUnsupported
		}
	}
}

// rejectingRecreateResponder answers DeleteSubscriptions and then rejects
// CreateSubscription with a service level status, so the recreate fails before
// the replacement subscription exists. A monitored item the server rejects is
// not enough to fail a recreate, see recreate_monitoredItems.
func rejectingRecreateResponder(status ua.StatusCode) func(ua.Request, func(ua.Response) error) error {
	return func(req ua.Request, h func(ua.Response) error) error {
		switch req.(type) {
		case *ua.DeleteSubscriptionsRequest:
			return h(&ua.DeleteSubscriptionsResponse{
				ResponseHeader: &ua.ResponseHeader{},
				Results:        []ua.StatusCode{ua.StatusOK},
			})

		case *ua.CreateSubscriptionRequest:
			return h(&ua.CreateSubscriptionResponse{
				ResponseHeader: &ua.ResponseHeader{ServiceResult: status},
			})

		default:
			return ua.StatusBadServiceUnsupported
		}
	}
}

// newTestSubscription registers a subscription with one monitored item on c.
func newTestSubscription(t *testing.T, c *Client, stub *stubClient, id uint32) *Subscription {
	t.Helper()

	sub := &Subscription{
		SubscriptionID: id,
		params: &SubscriptionParameters{
			Interval:          DefaultSubscriptionInterval,
			LifetimeCount:     DefaultSubscriptionLifetimeCount,
			MaxKeepAliveCount: DefaultSubscriptionMaxKeepAliveCount,
		},
		items: map[uint32]*monitoredItem{
			1: {
				req: NewMonitoredItemCreateRequestWithDefaults(ua.NewNumericNodeID(0, 2258), ua.AttributeIDValue, 1),
				res: &ua.MonitoredItemCreateResult{MonitoredItemID: 1},
				ts:  ua.TimestampsToReturnBoth,
			},
		},
		c: stub,
	}

	c.subMux.Lock()
	defer c.subMux.Unlock()
	require.NoError(t, c.registerSubscription_NeedsSubMuxLock(sub))
	return sub
}

// TestRepublishOrRecreateSubscriptions covers the restoreSubscriptions step of
// the reconnect state machine.
//
// The regression it guards against is #876: when a subscription could not be
// recreated the step used to return control to the reconnect loop with the
// action cleared, which marked the client Connected while the subscription was
// gone and the publish loop was dead.
func TestRepublishOrRecreateSubscriptions(t *testing.T) {
	// republish always fails here since the test client has neither a session
	// nor a secure channel, so a subscription handed to it is recreated
	// instead. See sendRepublishRequests.
	t.Run("nothing to restore", func(t *testing.T) {
		c, err := NewClient("opc.tcp://example.com:4840")
		require.NoError(t, err)

		action, activeSubs := c.republishOrRecreateSubscriptions(context.Background(), nil, nil, nil)
		require.Equal(t, none, action)
		require.Equal(t, 0, activeSubs)
	})

	t.Run("recreate succeeds", func(t *testing.T) {
		c, err := NewClient("opc.tcp://example.com:4840")
		require.NoError(t, err)

		stub := &stubClient{send: recreateResponder(4711, ua.StatusOK)}
		newTestSubscription(t, c, stub, 1)

		action, activeSubs := c.republishOrRecreateSubscriptions(context.Background(), nil, []uint32{1}, nil)
		require.Equal(t, none, action)
		require.Equal(t, 1, activeSubs)
		require.Equal(t, []uint32{4711}, c.SubscriptionIDs())
	})

	// This is the #876 case: the subscription cannot be brought back, here
	// because the server refuses to create it. The step has to hand
	// recreateSession back to the reconnect loop instead of none, otherwise
	// the loop marks the client Connected with no live subscription and never
	// tries again.
	t.Run("recreate fails", func(t *testing.T) {
		c, err := NewClient("opc.tcp://example.com:4840")
		require.NoError(t, err)

		stub := &stubClient{send: rejectingRecreateResponder(ua.StatusBadTooManySubscriptions)}
		newTestSubscription(t, c, stub, 6647)

		action, activeSubs := c.republishOrRecreateSubscriptions(context.Background(), nil, []uint32{6647}, nil)
		require.Equal(t, recreateSession, action)
		require.Equal(t, 0, activeSubs)
	})

	// One subscription restores fine and one does not. The failure has to win,
	// otherwise the healthy subscription hides the broken one and the client
	// comes up Connected with a subscription that no longer exists.
	t.Run("one of two recreates fails", func(t *testing.T) {
		c, err := NewClient("opc.tcp://example.com:4840")
		require.NoError(t, err)

		good := &stubClient{send: recreateResponder(4711, ua.StatusOK)}
		bad := &stubClient{send: rejectingRecreateResponder(ua.StatusBadTooManySubscriptions)}
		newTestSubscription(t, c, good, 1)
		newTestSubscription(t, c, bad, 2)

		action, activeSubs := c.republishOrRecreateSubscriptions(context.Background(), nil, []uint32{1, 2}, nil)
		require.Equal(t, recreateSession, action)
		require.Equal(t, 1, activeSubs)
	})

	// A subscription which the server transferred but could not republish is
	// recreated. If that recreate fails as well the step must still ask for a
	// new session.
	t.Run("republish falls back to recreate", func(t *testing.T) {
		c, err := NewClient("opc.tcp://example.com:4840")
		require.NoError(t, err)

		stub := &stubClient{send: rejectingRecreateResponder(ua.StatusBadTooManySubscriptions)}
		newTestSubscription(t, c, stub, 3)

		action, _ := c.republishOrRecreateSubscriptions(context.Background(), []uint32{3}, nil, map[uint32][]uint32{3: {1}})
		require.Equal(t, recreateSession, action)

		// the subscription was recreated after the republish failed
		var sawCreate bool
		for _, req := range stub.sentRequests() {
			if _, ok := req.(*ua.CreateSubscriptionRequest); ok {
				sawCreate = true
			}
		}
		require.True(t, sawCreate, "subscription was not recreated after republish failed")
	})

	// The retry the failure above asks for is only useful if the subscription
	// is still there to retry. recreateSubscription used to forget it before
	// creating the replacement, so a rejected create dropped it for good and
	// the next round had nothing to transfer.
	t.Run("failed create keeps the subscription registered", func(t *testing.T) {
		c, err := NewClient("opc.tcp://example.com:4840")
		require.NoError(t, err)

		stub := &stubClient{send: rejectingRecreateResponder(ua.StatusBadTooManySubscriptions)}
		newTestSubscription(t, c, stub, 6647)

		action, _ := c.republishOrRecreateSubscriptions(context.Background(), nil, []uint32{6647}, nil)
		require.Equal(t, recreateSession, action)
		require.Equal(t, []uint32{6647}, c.SubscriptionIDs())
	})
}

// TestReconnectRecoversRegisteredSubscriptions guards against the reconnect
// state machine (client.go) silently abandoning subscriptions.
//
// Both reconnect paths funnel their registered subscription ids through
// transferSubscriptions into republishOrRecreateSubscriptions:
//   - recreateSession builds a new session and always did so.
//   - restoreSession reactivates the same session and now does too, because a
//     Republish alone does not make servers resume delivery after the secure
//     channel was rebuilt.
//
// If that step is handed nothing (as restoreSession used to do by jumping
// straight to restoreSubscriptions) it returns action==none and activeSubs==0,
// which marks the client Connected while the subscription is left registered
// but untouched and the publish loop stays paused forever: the production hang
// reported for short network blips.
func TestReconnectRecoversRegisteredSubscriptions(t *testing.T) {
	t.Run("empty input strands a registered subscription", func(t *testing.T) {
		c, err := NewClient("opc.tcp://example.com:4840")
		require.NoError(t, err)

		stub := &stubClient{send: recreateResponder(4711, ua.StatusOK)}
		newTestSubscription(t, c, stub, 1)

		// The regression: recovering with nothing to do leaves subscription 1
		// registered under its old id, never republished or recreated, while
		// the caller reads this as "everything recovered".
		action, activeSubs := c.republishOrRecreateSubscriptions(context.Background(), nil, nil, nil)

		require.Equal(t, none, action)
		require.Equal(t, 0, activeSubs)
		require.Equal(t, []uint32{1}, c.SubscriptionIDs(), "subscription must not be silently abandoned")
	})

	t.Run("registered subscription ids are recovered", func(t *testing.T) {
		c, err := NewClient("opc.tcp://example.com:4840")
		require.NoError(t, err)

		stub := &stubClient{send: recreateResponder(4711, ua.StatusOK)}
		newTestSubscription(t, c, stub, 1)

		// Fed the ids the way transferSubscriptions feeds them, the
		// subscription is processed: republish fails (no session/channel) and
		// falls back to recreate, so activeSubs > 0 and the reconnect loop
		// resumes the publish loop.
		action, activeSubs := c.republishOrRecreateSubscriptions(context.Background(), c.SubscriptionIDs(), nil, map[uint32][]uint32{})

		require.Equal(t, none, action)
		require.Greater(t, activeSubs, 0, "activeSubs must be > 0 so the publish loop is resumed")
		require.Equal(t, []uint32{4711}, c.SubscriptionIDs(), "subscription must have been recreated, not abandoned")
	})
}

// errorCount reads the current value of the named counter in the global
// stats.Error() expvar map, treating a missing counter as zero.
func errorCount(key string) int64 {
	v := stats.Error().Get(key)
	if v == nil {
		return 0
	}
	iv, ok := v.(*expvar.Int)
	if !ok {
		return 0
	}
	return iv.Value()
}

// TestMonitorSubscriptionsSurvivesPauseResumeRace guards against
// https://github.com/gopcua/opcua/issues/895: pausech/resumech used to be two
// buffered channels read by the same select in monitorSubscriptions. When a
// pause and a resume were both pending -- e.g. Subscription.Cancel() followed
// by Client.Subscribe() -- Go's select picked one at random; if the resume was
// picked first it was discarded and the loop then parked on the pause forever.
//
// The level-based publishPaused plus a best-effort wake-up removes the race.
// This test hammers pauseSubscriptions/resumeSubscriptions concurrently and
// checks that a final resume always gets the loop running again.
func TestMonitorSubscriptionsSurvivesPauseResumeRace(t *testing.T) {
	c, err := NewClient("opc.tcp://example.com:4840")
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		c.monitorSubscriptions(ctx)
	}()
	defer func() {
		cancel()
		<-done
	}()

	// c has no real connection, so a running loop reacts to being resumed by
	// attempting exactly one PublishRequest, which fails immediately with
	// StatusBadServerNotConnected (see sendWithTimeout) and re-pauses on its
	// own. Counting this error proves the loop actually woke up and executed
	// the publish path, rather than remaining parked forever.
	const key = "ua.StatusBadServerNotConnected"
	baseline := errorCount(key)

	for i := 0; i < 200; i++ {
		var wg sync.WaitGroup
		wg.Add(2)
		go func() { defer wg.Done(); c.pauseSubscriptions(ctx) }()
		go func() { defer wg.Done(); c.resumeSubscriptions(ctx) }()
		wg.Wait()

		// Whatever order the two racing calls above landed in, make the
		// desired end state explicit and unambiguous: the loop must run.
		c.resumeSubscriptions(ctx)

		require.Eventually(t, func() bool {
			return errorCount(key) > baseline
		}, time.Second, time.Millisecond, "iteration %d: publish loop appears stuck after a pause/resume race", i)
		baseline = errorCount(key)
	}
}
