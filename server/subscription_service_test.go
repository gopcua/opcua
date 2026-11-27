package server

import (
	"math"
	"testing"
	"time"

	"github.com/gopcua/opcua/id"
	"github.com/gopcua/opcua/ua"
	"github.com/stretchr/testify/require"
)

// Part 4 §5.14.2.2: an illegal requested publishing interval is revised to a
// value the server supports instead of generating an error, and the revised
// value is what the server stores and reports.

func TestRevisePublishingInterval(t *testing.T) {
	tests := []struct {
		name     string
		request  float64
		expected float64
	}{
		{"zero is revised to the fastest supported interval", 0, minPublishingInterval},
		{"a negative value is revised to the fastest supported interval", -1, minPublishingInterval},
		{"NaN is revised to the fastest supported interval", math.NaN(), minPublishingInterval},
		{"a sub-millisecond interval is revised up to the fastest supported interval", 0.5, minPublishingInterval},
		{"the fastest supported interval itself is used as-is", minPublishingInterval, minPublishingInterval},
		{"+Inf is revised to the slowest supported interval", math.Inf(1), maxPublishingInterval},
		{"an overflowing value is revised to the slowest supported interval", 1e30, maxPublishingInterval},
		{"the slowest supported interval itself is used as-is", maxPublishingInterval, maxPublishingInterval},
		{"a legal interval is used as-is", 500, 500},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.expected, revisePublishingInterval(tt.request))
		})
	}
}

// TestCreateSubscriptionReportsRevisedPublishingInterval checks the wiring: the
// revised interval in the response is the one the server actually uses.
func TestCreateSubscriptionReportsRevisedPublishingInterval(t *testing.T) {
	tests := []struct {
		name     string
		request  float64
		expected float64
	}{
		{"an illegal interval is revised", 0, minPublishingInterval},
		{"a legal interval is used as-is", 500, 500},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := New()
			srv.initHandlers()
			sess := srv.sb.NewSession()

			respI, err := srv.SubscriptionService.CreateSubscription(nil, &ua.CreateSubscriptionRequest{
				RequestHeader: &ua.RequestHeader{AuthenticationToken: sess.AuthTokenID},
				// keep the background goroutine from expiring the subscription
				// or answering a keep-alive while the test is running
				RequestedPublishingInterval: tt.request,
				RequestedLifetimeCount:      math.MaxUint32,
				RequestedMaxKeepAliveCount:  math.MaxUint32,
			}, 0)
			require.NoError(t, err)
			resp, ok := respI.(*ua.CreateSubscriptionResponse)
			require.True(t, ok, "expected *ua.CreateSubscriptionResponse, got %T", respI)

			t.Cleanup(func() {
				srv.SubscriptionService.DeleteSubscription(resp.SubscriptionID)
			})

			require.Equal(t, tt.expected, resp.RevisedPublishingInterval)

			sub, ok := srv.SubscriptionService.Subs[resp.SubscriptionID]
			require.True(t, ok, "subscription %d not registered", resp.SubscriptionID)
			require.Equal(t, tt.expected, sub.RevisedPublishingInterval)
		})
	}
}

// TestSubscriptionUpdateReportsRevisedPublishingInterval checks that Update,
// which ModifySubscription applies to a stored subscription, revises the
// interval the same way CreateSubscription does.
func TestSubscriptionUpdateReportsRevisedPublishingInterval(t *testing.T) {
	tests := []struct {
		name     string
		request  float64
		expected float64
	}{
		{"an illegal interval is revised", 0, minPublishingInterval},
		{"a legal interval is used as-is", 250, 250},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sub := NewSubscription()
			sub.RevisedPublishingInterval = 100

			sub.Update(&ua.ModifySubscriptionRequest{
				RequestedPublishingInterval: tt.request,
			})

			require.Equal(t, tt.expected, sub.RevisedPublishingInterval)
		})
	}
}

// OPC 10000-4 §5.14.2: the SubscriptionId is unique on the server, so the
// id of a deleted subscription is never handed to a subscription created
// after it - not even to another session's.

// createSubscription has sess create a subscription and returns its id.
func createSubscription(t *testing.T, srv *Server, sess *session) uint32 {
	t.Helper()

	respI, err := srv.SubscriptionService.CreateSubscription(nil, &ua.CreateSubscriptionRequest{
		RequestHeader: &ua.RequestHeader{AuthenticationToken: sess.AuthTokenID},
		// keep the background goroutine quiet while the test runs
		RequestedPublishingInterval: 100,
		RequestedLifetimeCount:      math.MaxUint32,
		RequestedMaxKeepAliveCount:  math.MaxUint32,
	}, 0)
	require.NoError(t, err)
	resp, ok := respI.(*ua.CreateSubscriptionResponse)
	require.True(t, ok, "expected *ua.CreateSubscriptionResponse, got %T", respI)

	t.Cleanup(func() {
		srv.SubscriptionService.DeleteSubscription(resp.SubscriptionID)
	})

	return resp.SubscriptionID
}

// deleteSubscriptions has sess delete the given ids and requires that each
// one was accepted.
func deleteSubscriptions(t *testing.T, srv *Server, sess *session, ids ...uint32) {
	t.Helper()

	respI, err := srv.SubscriptionService.DeleteSubscriptions(nil, &ua.DeleteSubscriptionsRequest{
		RequestHeader:   &ua.RequestHeader{AuthenticationToken: sess.AuthTokenID},
		SubscriptionIDs: ids,
	}, 0)
	require.NoError(t, err)
	resp, ok := respI.(*ua.DeleteSubscriptionsResponse)
	require.True(t, ok, "expected *ua.DeleteSubscriptionsResponse, got %T", respI)
	require.Len(t, resp.Results, len(ids))
	for i, r := range resp.Results {
		require.Equal(t, ua.StatusOK, r, "deleting subscription %d must be accepted", ids[i])
	}
}

// lookupSubOK reads a subscription under the service lock, since
// DeleteSubscriptions removes subscriptions from a goroutine of its own.
func lookupSubOK(srv *Server, id uint32) (*Subscription, bool) {
	srv.SubscriptionService.Mu.Lock()
	defer srv.SubscriptionService.Mu.Unlock()
	sub, ok := srv.SubscriptionService.Subs[id]
	return sub, ok
}

// requireSubscriptionGone waits for the background deletion of a
// subscription to finish, since DeleteSubscriptions deletes from a
// goroutine of its own.
func requireSubscriptionGone(t *testing.T, srv *Server, id uint32) {
	t.Helper()
	require.Eventually(t, func() bool {
		_, ok := lookupSubOK(srv, id)
		return !ok
	}, time.Second, 5*time.Millisecond, "subscription %d must be deleted", id)
}

// TestCreateSubscriptionDoesNotReuseALiveSubscriptionsID: session A creates
// 1, session B creates 2, A deletes 1 and creates again. A's new subscription
// must not get B's id; A deleting its own ids must leave B's subscription and
// monitored items untouched; and A deleting B's id must be refused.
func TestCreateSubscriptionDoesNotReuseALiveSubscriptionsID(t *testing.T) {
	srv := New()
	srv.initHandlers()
	sessA := srv.sb.NewSession()
	sessB := srv.sb.NewSession()

	idA1 := createSubscription(t, srv, sessA)
	idB := createSubscription(t, srv, sessB)
	deleteSubscriptions(t, srv, sessA, idA1)
	requireSubscriptionGone(t, srv, idA1)
	idA2 := createSubscription(t, srv, sessA)

	// OPC 10000-4 §5.14.2: the SubscriptionId is unique on the server,
	// so A's new subscription must not have B's id
	require.NotEqual(t, idB, idA2, "A's new subscription must not reuse B's live id")

	// B's monitored item must survive A deleting A's own subscription
	itemRespI, err := srv.MonitoredItemService.CreateMonitoredItems(nil, &ua.CreateMonitoredItemsRequest{
		RequestHeader:  &ua.RequestHeader{AuthenticationToken: sessB.AuthTokenID},
		SubscriptionID: idB,
		ItemsToCreate: []*ua.MonitoredItemCreateRequest{{
			ItemToMonitor:  &ua.ReadValueID{NodeID: ua.NewNumericNodeID(0, id.Server_ServerStatus_CurrentTime)},
			MonitoringMode: ua.MonitoringModeReporting,
			RequestedParameters: &ua.MonitoringParameters{
				ClientHandle:     42,
				SamplingInterval: 100,
				QueueSize:        1,
				DiscardOldest:    true,
			},
		}},
	}, 0)
	require.NoError(t, err)
	itemResp, ok := itemRespI.(*ua.CreateMonitoredItemsResponse)
	require.True(t, ok, "expected *ua.CreateMonitoredItemsResponse, got %T", itemRespI)
	require.Len(t, itemResp.Results, 1)
	require.Equal(t, ua.StatusOK, itemResp.Results[0].StatusCode)
	itemB := itemResp.Results[0].MonitoredItemID

	deleteSubscriptions(t, srv, sessA, idA2)
	requireSubscriptionGone(t, srv, idA2)

	_, ok = lookupSubOK(srv, idB)
	require.True(t, ok, "B's subscription %d must survive A deleting its own ids", idB)
	_, ok = lookupItemOK(srv, itemB)
	require.True(t, ok, "B's monitored item %d must survive A deleting its own ids", itemB)

	// A's DeleteSubscriptions of B's id is refused: B owns it
	respI, err := srv.SubscriptionService.DeleteSubscriptions(nil, &ua.DeleteSubscriptionsRequest{
		RequestHeader:   &ua.RequestHeader{AuthenticationToken: sessA.AuthTokenID},
		SubscriptionIDs: []uint32{idB},
	}, 0)
	require.NoError(t, err)
	resp, ok := respI.(*ua.DeleteSubscriptionsResponse)
	require.True(t, ok, "expected *ua.DeleteSubscriptionsResponse, got %T", respI)
	require.Len(t, resp.Results, 1)
	require.Equal(t, ua.StatusBadSessionIDInvalid, resp.Results[0], "A deleting B's subscription must be refused")

	// the refused delete must have left both alone
	require.Never(t, func() bool {
		_, ok := lookupSubOK(srv, idB)
		return !ok
	}, 100*time.Millisecond, 5*time.Millisecond, "B's subscription %d must not be deleted by a session that does not own it", idB)
	_, ok = lookupSubOK(srv, idB)
	require.True(t, ok, "B's subscription %d must still exist", idB)
	_, ok = lookupItemOK(srv, itemB)
	require.True(t, ok, "B's monitored item %d must still exist", itemB)
}

// TestCreateSubscriptionDoesNotReuseADeletedSubscriptionsID: the id of a
// deleted subscription is not handed out to the next one created.
func TestCreateSubscriptionDoesNotReuseADeletedSubscriptionsID(t *testing.T) {
	srv := New()
	srv.initHandlers()
	sess := srv.sb.NewSession()

	id1 := createSubscription(t, srv, sess)
	deleteSubscriptions(t, srv, sess, id1)
	requireSubscriptionGone(t, srv, id1)

	id2 := createSubscription(t, srv, sess)

	// OPC 10000-4 §5.14.2: the SubscriptionId is unique on the server,
	// and that includes the ids of subscriptions already deleted
	require.NotEqual(t, id1, id2, "the id of a deleted subscription must not be reused")
}

// Part 4 §7.32: the authenticationToken must name an existing session. The
// Subscription and MonitoredItem service handlers answer Bad_SessionIdInvalid
// for a token that does not.

// unknownSessionToken is the AuthenticationToken of a request that names no
// session: the same TwoByteNodeID(0) uasc puts on the wire when a client has
// no session.
var unknownSessionToken = ua.NewTwoByteNodeID(0)

// requireBadSessionIDInvalid asserts that the handler answered with
// Bad_SessionIdInvalid and nothing else.
func requireBadSessionIDInvalid(t *testing.T, respI ua.Response, err error) {
	t.Helper()
	require.Nil(t, respI, "no response expected for a request without a session")
	require.Error(t, err)
	sc, ok := err.(ua.StatusCode)
	require.True(t, ok, "expected a ua.StatusCode error, got %T: %v", err, err)
	require.Equal(t, ua.StatusBadSessionIDInvalid, sc)
}

func TestCreateSubscriptionRejectsUnknownSessionToken(t *testing.T) {
	srv := New()
	srv.initHandlers()

	req := &ua.CreateSubscriptionRequest{
		RequestHeader:               &ua.RequestHeader{AuthenticationToken: unknownSessionToken},
		RequestedPublishingInterval: 10,
		RequestedLifetimeCount:      100,
		RequestedMaxKeepAliveCount:  1,
	}
	respI, err := srv.SubscriptionService.CreateSubscription(nil, req, 0)
	requireBadSessionIDInvalid(t, respI, err)

	// nothing may have been created for a token nobody owns
	require.Empty(t, srv.SubscriptionService.Subs)
}

func TestDeleteSubscriptionsRejectsUnknownSessionToken(t *testing.T) {
	// a subscription owned by a real session, so the request below gets past
	// the subscription lookup and reaches the session check
	srv, _, subID, _ := monitoredItemFixture(t)

	req := &ua.DeleteSubscriptionsRequest{
		RequestHeader:   &ua.RequestHeader{AuthenticationToken: unknownSessionToken},
		SubscriptionIDs: []uint32{subID},
	}
	respI, err := srv.SubscriptionService.DeleteSubscriptions(nil, req, 0)
	requireBadSessionIDInvalid(t, respI, err)

	// the subscription must be untouched
	_, ok := srv.SubscriptionService.Subs[subID]
	require.True(t, ok, "subscription %d must not be deleted by a request without a session", subID)
}

func TestCreateMonitoredItemsRejectsUnknownSessionToken(t *testing.T) {
	srv, _, subID, _ := monitoredItemFixture(t)

	respI, err := srv.MonitoredItemService.CreateMonitoredItems(nil, &ua.CreateMonitoredItemsRequest{
		RequestHeader:  &ua.RequestHeader{AuthenticationToken: unknownSessionToken},
		SubscriptionID: subID,
		ItemsToCreate: []*ua.MonitoredItemCreateRequest{{
			ItemToMonitor:  &ua.ReadValueID{NodeID: ua.NewNumericNodeID(0, id.Server_ServerStatus_CurrentTime)},
			MonitoringMode: ua.MonitoringModeReporting,
		}},
	}, 0)
	requireBadSessionIDInvalid(t, respI, err)
}

func TestSetMonitoringModeRejectsUnknownSessionToken(t *testing.T) {
	srv, _, _, itemID := monitoredItemFixture(t)

	respI, err := srv.MonitoredItemService.SetMonitoringMode(nil, &ua.SetMonitoringModeRequest{
		RequestHeader:    &ua.RequestHeader{AuthenticationToken: unknownSessionToken},
		MonitoredItemIDs: []uint32{itemID},
		MonitoringMode:   ua.MonitoringModeDisabled,
	}, 0)
	requireBadSessionIDInvalid(t, respI, err)
}

func TestDeleteMonitoredItemsRejectsUnknownSessionToken(t *testing.T) {
	srv, _, _, itemID := monitoredItemFixture(t)

	respI, err := srv.MonitoredItemService.DeleteMonitoredItems(nil, &ua.DeleteMonitoredItemsRequest{
		RequestHeader:    &ua.RequestHeader{AuthenticationToken: unknownSessionToken},
		SubscriptionID:   0,
		MonitoredItemIDs: []uint32{itemID},
	}, 0)
	requireBadSessionIDInvalid(t, respI, err)

	// the item must still be there
	_, ok := lookupItemOK(srv, itemID)
	require.True(t, ok, "monitored item %d must not be deleted by a request without a session", itemID)
}
