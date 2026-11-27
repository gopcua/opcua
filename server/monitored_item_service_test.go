package server

import (
	"math"
	"testing"
	"time"

	"github.com/gopcua/opcua/id"
	"github.com/gopcua/opcua/ua"
	"github.com/stretchr/testify/require"
)

// Part 4 §5.13.4.2 and §5.13.6.2: an id that names no monitored item gets the
// per-item status Bad_MonitoredItemIdInvalid, and the remaining ids are still
// processed.

// monitoredItemFixture returns a server with a session that owns a
// subscription with a single monitored item, together with the subscription
// and the item ids.
func monitoredItemFixture(t *testing.T) (*Server, *session, uint32, uint32) {
	t.Helper()

	srv := New()
	srv.initHandlers()
	sess := srv.sb.NewSession()

	subRespI, err := srv.SubscriptionService.CreateSubscription(nil, &ua.CreateSubscriptionRequest{
		RequestHeader: &ua.RequestHeader{AuthenticationToken: sess.AuthTokenID},
		// keep the background goroutine quiet while the test runs
		RequestedPublishingInterval: 100,
		RequestedLifetimeCount:      math.MaxUint32,
		RequestedMaxKeepAliveCount:  math.MaxUint32,
	}, 0)
	require.NoError(t, err)
	subResp, ok := subRespI.(*ua.CreateSubscriptionResponse)
	require.True(t, ok, "expected *ua.CreateSubscriptionResponse, got %T", subRespI)

	t.Cleanup(func() {
		srv.SubscriptionService.DeleteSubscription(subResp.SubscriptionID)
	})

	itemRespI, err := srv.MonitoredItemService.CreateMonitoredItems(nil, &ua.CreateMonitoredItemsRequest{
		RequestHeader:  &ua.RequestHeader{AuthenticationToken: sess.AuthTokenID},
		SubscriptionID: subResp.SubscriptionID,
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

	return srv, sess, subResp.SubscriptionID, itemResp.Results[0].MonitoredItemID
}

// An unknown monitored item id gets Bad_MonitoredItemIdInvalid and the
// remaining ids are still processed.
func TestSetMonitoringModeUnknownMonitoredItemID(t *testing.T) {
	srv, sess, _, itemID := monitoredItemFixture(t)

	const unknownID = 42 // names no monitored item on this server

	respI, err := srv.MonitoredItemService.SetMonitoringMode(nil, &ua.SetMonitoringModeRequest{
		RequestHeader:    &ua.RequestHeader{AuthenticationToken: sess.AuthTokenID},
		MonitoredItemIDs: []uint32{unknownID, itemID},
		MonitoringMode:   ua.MonitoringModeDisabled,
	}, 0)
	require.NoError(t, err)
	resp, ok := respI.(*ua.SetMonitoringModeResponse)
	require.True(t, ok, "expected *ua.SetMonitoringModeResponse, got %T", respI)

	// the unknown id gets the per-item status the spec gives...
	require.Len(t, resp.Results, 2)
	require.Equal(t, ua.StatusBadMonitoredItemIDInvalid, resp.Results[0])
	// ...and the loop goes on with the next id
	require.Equal(t, ua.StatusOK, resp.Results[1])

	item, ok := lookupItemOK(srv, itemID)
	require.True(t, ok, "monitored item %d must still exist", itemID)
	require.Equal(t, ua.MonitoringMode(ua.MonitoringModeDisabled), item.Mode)
}

// An unknown monitored item id gets Bad_MonitoredItemIdInvalid and the
// remaining ids are still processed.
func TestDeleteMonitoredItemsUnknownMonitoredItemID(t *testing.T) {
	srv, sess, _, itemID := monitoredItemFixture(t)

	const unknownID = 42 // names no monitored item on this server

	respI, err := srv.MonitoredItemService.DeleteMonitoredItems(nil, &ua.DeleteMonitoredItemsRequest{
		RequestHeader:    &ua.RequestHeader{AuthenticationToken: sess.AuthTokenID},
		MonitoredItemIDs: []uint32{unknownID, itemID},
	}, 0)
	require.NoError(t, err)
	resp, ok := respI.(*ua.DeleteMonitoredItemsResponse)
	require.True(t, ok, "expected *ua.DeleteMonitoredItemsResponse, got %T", respI)

	// the unknown id gets the per-item status the spec gives...
	require.Len(t, resp.Results, 2)
	require.Equal(t, ua.StatusBadMonitoredItemIDInvalid, resp.Results[0])
	// ...and the loop goes on with the next id
	require.Equal(t, ua.StatusOK, resp.Results[1])

	// the real item is deleted in the background, once the handler's lock is
	// released
	require.Eventually(t, func() bool {
		_, ok := lookupItemOK(srv, itemID)
		return !ok
	}, time.Second, 5*time.Millisecond, "monitored item %d must be deleted", itemID)
}

// A session that does not own the item gets Bad_SessionIDInvalid and the item
// is left untouched.
func TestSetMonitoringModeForeignSessionItem(t *testing.T) {
	srv, _, _, itemID := monitoredItemFixture(t)

	// a second, valid session that owns nothing
	other := srv.sb.NewSession()

	// CreateMonitoredItems never initializes Mode ("//TODO: use this"), so
	// capture whatever the item actually has and require it unchanged
	srv.MonitoredItemService.Mu.Lock()
	modeBefore := srv.MonitoredItemService.Items[itemID].Mode
	srv.MonitoredItemService.Mu.Unlock()

	respI, err := srv.MonitoredItemService.SetMonitoringMode(nil, &ua.SetMonitoringModeRequest{
		RequestHeader:    &ua.RequestHeader{AuthenticationToken: other.AuthTokenID},
		MonitoredItemIDs: []uint32{itemID},
		MonitoringMode:   ua.MonitoringModeDisabled,
	}, 0)
	require.NoError(t, err)
	resp, ok := respI.(*ua.SetMonitoringModeResponse)
	require.True(t, ok, "expected *ua.SetMonitoringModeResponse, got %T", respI)

	require.Len(t, resp.Results, 1)
	require.Equal(t, ua.StatusBadSessionIDInvalid, resp.Results[0])

	// the mode of an item the session does not own must not be changed
	// (CreateMonitoredItems never initializes Mode, so compare against the
	// value it actually had)
	item, ok := lookupItemOK(srv, itemID)
	require.True(t, ok, "monitored item %d must still exist", itemID)
	require.Equal(t, modeBefore, item.Mode, "the mode of an item the session does not own must not be changed")
}

// A session that does not own the item gets Bad_SessionIDInvalid and the item
// is left untouched.
func TestDeleteMonitoredItemsForeignSessionItem(t *testing.T) {
	srv, _, _, itemID := monitoredItemFixture(t)

	// a second, valid session that owns nothing
	other := srv.sb.NewSession()

	respI, err := srv.MonitoredItemService.DeleteMonitoredItems(nil, &ua.DeleteMonitoredItemsRequest{
		RequestHeader:    &ua.RequestHeader{AuthenticationToken: other.AuthTokenID},
		MonitoredItemIDs: []uint32{itemID},
	}, 0)
	require.NoError(t, err)
	resp, ok := respI.(*ua.DeleteMonitoredItemsResponse)
	require.True(t, ok, "expected *ua.DeleteMonitoredItemsResponse, got %T", respI)

	require.Len(t, resp.Results, 1)
	require.Equal(t, ua.StatusBadSessionIDInvalid, resp.Results[0])

	// an item the session does not own must not be deleted
	require.Never(t, func() bool {
		_, ok := lookupItemOK(srv, itemID)
		return !ok
	}, 100*time.Millisecond, 5*time.Millisecond, "monitored item %d must not be deleted by a session that does not own it", itemID)
	_, ok = lookupItemOK(srv, itemID)
	require.True(t, ok, "monitored item %d must still exist", itemID)
}

// lookupItemOK reads a monitored item under the service lock, since
// DeleteMonitoredItems removes items from a goroutine of its own.
func lookupItemOK(srv *Server, id uint32) (*MonitoredItem, bool) {
	srv.MonitoredItemService.Mu.Lock()
	defer srv.MonitoredItemService.Mu.Unlock()
	item, ok := srv.MonitoredItemService.Items[id]
	return item, ok
}
