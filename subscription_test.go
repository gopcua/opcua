package opcua

import (
	"context"
	"testing"

	"github.com/gopcua/opcua/ua"
	"github.com/stretchr/testify/require"
)

// Running tool: /Users/frank/sdk/go1.17.1/bin/go test -benchmem -run=^$ -bench ^BenchmarkUnmonitorItems$ github.com/gopcua/opcua

// goos: darwin
// goarch: arm64
// pkg: github.com/gopcua/opcua
// BenchmarkUnmonitorItems/slice-8         	51153620	        24.03 ns/op	      20 B/op	       0 allocs/op
// --- BENCH: BenchmarkUnmonitorItems/slice-8
//     subscription_test.go:29: src 1 dst 0
//     subscription_test.go:29: src 100 dst 50
//     subscription_test.go:29: src 10000 dst 5000
//     subscription_test.go:29: src 1000000 dst 500000
//     subscription_test.go:29: src 51153620 dst 25576810
// BenchmarkUnmonitorItems/slice_pre-alloc-8         	91635986	        22.77 ns/op	       8 B/op	       0 allocs/op
// --- BENCH: BenchmarkUnmonitorItems/slice_pre-alloc-8
//     subscription_test.go:51: src 1 dst 0
//     subscription_test.go:51: src 100 dst 50
//     subscription_test.go:51: src 10000 dst 5000
//     subscription_test.go:51: src 1000000 dst 500000
//     subscription_test.go:51: src 91635986 dst 45817993
// BenchmarkUnmonitorItems/map-8                     	39885550	        43.72 ns/op	       0 B/op	       0 allocs/op
// --- BENCH: BenchmarkUnmonitorItems/map-8
//     subscription_test.go:75: src 0
//     subscription_test.go:75: src 50
//     subscription_test.go:75: src 5000
//     subscription_test.go:75: src 500000
//     subscription_test.go:75: src 19942775
// PASS
// ok  	github.com/gopcua/opcua	116.192s

func BenchmarkUnmonitorItems(b *testing.B) {
	b.Run("slice", func(b *testing.B) {
		src := make([]*monitoredItem, b.N)
		for i := 0; i < b.N; i++ {
			src[i] = &monitoredItem{
				res: &ua.MonitoredItemCreateResult{
					MonitoredItemID: uint32(i),
				},
			}
		}

		b.ResetTimer()
		var dst []*monitoredItem
		for _, item := range src {
			if item.res.MonitoredItemID%2 == 0 {
				continue
			}
			dst = append(dst, item)
		}

		b.Log("src", len(src), "dst", len(dst)) // ensure src and dst are not GC'ed
	})

	b.Run("slice pre-alloc", func(b *testing.B) {
		src := make([]*monitoredItem, b.N)
		for i := 0; i < b.N; i++ {
			src[i] = &monitoredItem{
				res: &ua.MonitoredItemCreateResult{
					MonitoredItemID: uint32(i),
				},
			}
		}

		b.ResetTimer()
		dst := make([]*monitoredItem, 0, len(src))
		for _, item := range src {
			if item.res.MonitoredItemID%2 == 0 {
				continue
			}
			dst = append(dst, item)
		}

		b.Log("src", len(src), "dst", len(dst)) // ensure src and dst are not GC'ed
	})

	b.Run("map", func(b *testing.B) {
		idsToDelete := []uint32{}
		src := make(map[uint32]*monitoredItem, b.N)
		for i := 0; i < b.N; i++ {
			id := uint32(i)
			src[id] = &monitoredItem{
				res: &ua.MonitoredItemCreateResult{
					MonitoredItemID: id,
				},
			}

			if id%2 == 0 {
				idsToDelete = append(idsToDelete, id)
			}
		}

		b.ResetTimer()
		for _, id := range idsToDelete {
			delete(src, id)
		}

		b.Log("src", len(src)) // ensure src and dst are not GC'ed
	})
}

// monitoredItemsResponder answers a CreateMonitoredItems request with a status
// per node id, defaulting to StatusOK. Accepted items get sequential ids
// starting at 100 so a dropped item cannot be mistaken for an accepted one.
func monitoredItemsResponder(statusByNode map[string]ua.StatusCode) func(ua.Request, func(ua.Response) error) error {
	return func(req ua.Request, h func(ua.Response) error) error {
		r, ok := req.(*ua.CreateMonitoredItemsRequest)
		if !ok {
			return ua.StatusBadServiceUnsupported
		}

		res := &ua.CreateMonitoredItemsResponse{ResponseHeader: &ua.ResponseHeader{}}
		var nextID uint32 = 100
		for _, item := range r.ItemsToCreate {
			status, ok := statusByNode[item.ItemToMonitor.NodeID.String()]
			if !ok {
				status = ua.StatusOK
			}
			result := &ua.MonitoredItemCreateResult{StatusCode: status}
			if status == ua.StatusOK {
				result.MonitoredItemID = nextID
				nextID++
			}
			res.Results = append(res.Results, result)
		}
		return h(res)
	}
}

// newSubscriptionWithItems builds an unregistered subscription monitoring the
// given nodes, as it would look after a successful Monitor call.
func newSubscriptionWithItems(stub *stubClient, nodes ...*ua.NodeID) *Subscription {
	sub := &Subscription{
		SubscriptionID: 1,
		params:         &SubscriptionParameters{Interval: DefaultSubscriptionInterval},
		items:          map[uint32]*monitoredItem{},
		c:              stub,
	}
	for i, node := range nodes {
		id := uint32(i + 1)
		sub.items[id] = &monitoredItem{
			req: NewMonitoredItemCreateRequestWithDefaults(node, ua.AttributeIDValue, id),
			res: &ua.MonitoredItemCreateResult{MonitoredItemID: id},
			ts:  ua.TimestampsToReturnBoth,
		}
	}
	return sub
}

// TestRecreateMonitoredItems covers #886.
//
// Part 4, 5.13.2 makes the status of a MonitoredItemCreateResult an operation
// level result for that item, so a node the server rejects must not take the
// whole subscription down with it. recreate_monitoredItems used to clear
// s.items up front and return the first bad status, which left the recreated
// subscription registered but monitoring nothing.
func TestRecreateMonitoredItems(t *testing.T) {
	gone := ua.NewStringNodeID(2, "gone")
	alive := ua.NewStringNodeID(2, "alive")

	t.Run("keeps the items the server accepted", func(t *testing.T) {
		stub := &stubClient{send: monitoredItemsResponder(map[string]ua.StatusCode{
			gone.String(): ua.StatusBadNodeIDUnknown,
		})}
		sub := newSubscriptionWithItems(stub, gone, alive)

		require.NoError(t, sub.recreate_monitoredItems(context.Background()))

		require.Len(t, sub.items, 1)
		for _, item := range sub.items {
			require.Equal(t, alive.String(), item.req.ItemToMonitor.NodeID.String())
			require.Equal(t, ua.StatusOK, item.res.StatusCode)
		}
	})

	t.Run("every item rejected", func(t *testing.T) {
		stub := &stubClient{send: monitoredItemsResponder(map[string]ua.StatusCode{
			gone.String():  ua.StatusBadNodeIDUnknown,
			alive.String(): ua.StatusBadNodeIDUnknown,
		})}
		sub := newSubscriptionWithItems(stub, gone, alive)

		// the subscription itself is still valid, so the restore succeeds
		require.NoError(t, sub.recreate_monitoredItems(context.Background()))
		require.Empty(t, sub.items)
	})

	// A failed request tells us nothing about the items, so they have to stay
	// in place for the next reconnect attempt to send.
	t.Run("request error keeps the previous items", func(t *testing.T) {
		stub := &stubClient{send: func(ua.Request, func(ua.Response) error) error {
			return ua.StatusBadServerNotConnected
		}}
		sub := newSubscriptionWithItems(stub, gone, alive)

		require.Equal(t, ua.StatusBadServerNotConnected, sub.recreate_monitoredItems(context.Background()))
		require.Len(t, sub.items, 2)
	})

	// Part 4, 5.13.2.2: the results match the size and order of itemsToCreate.
	// Indexing them against the requested items would panic otherwise.
	t.Run("result count mismatch", func(t *testing.T) {
		stub := &stubClient{send: func(req ua.Request, h func(ua.Response) error) error {
			return h(&ua.CreateMonitoredItemsResponse{
				ResponseHeader: &ua.ResponseHeader{},
				Results:        []*ua.MonitoredItemCreateResult{{MonitoredItemID: 100}},
			})
		}}
		sub := newSubscriptionWithItems(stub, gone, alive)

		require.Error(t, sub.recreate_monitoredItems(context.Background()))
		require.Len(t, sub.items, 2)
	})
}
