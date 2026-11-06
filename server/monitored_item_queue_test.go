package server

import (
	"math"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gopcua/opcua/id"
	"github.com/gopcua/opcua/ua"
)

const statusGoodOverflow = statusInfoTypeDataValue | statusInfoBitOverflow // 0x00000480

func int32Value(v int32) *ua.DataValue {
	return &ua.DataValue{EncodingMask: ua.DataValueValue, Value: ua.MustVariant(v)}
}

func pushValues(q *notificationQueue, vals ...int32) {
	for i, v := range vals {
		q.push(queuedValue{seq: uint64(i + 1), value: int32Value(v)})
	}
}

// checkQueue compares the values and the Overflow bits of the queued entries.
func checkQueue(t *testing.T, entries []queuedValue, want []int32, overflow []bool) {
	t.Helper()
	if len(entries) != len(want) {
		t.Fatalf("got %d queued values, want %d", len(entries), len(want))
	}
	for i, e := range entries {
		if got := e.value.Value.Value(); got != want[i] {
			t.Errorf("value %d: got %v, want %d", i, got, want[i])
		}
		wantStatus := ua.StatusOK
		if overflow[i] {
			wantStatus = statusGoodOverflow
		}
		if e.value.Status != wantStatus {
			t.Errorf("value %d: got status 0x%08X, want 0x%08X", i, uint32(e.value.Status), uint32(wantStatus))
		}
	}
}

func TestRevisedQueueSize(t *testing.T) {
	tests := []struct {
		requested, want uint32
	}{
		{0, 1},
		{1, 1},
		{2, 2},
		{10, 10},
		{5000, 5000},
		{5001, 5000},
		{math.MaxUint32, 5000},
	}
	for _, tt := range tests {
		if got := revisedQueueSize(tt.requested, 5000); got != tt.want {
			t.Errorf("revisedQueueSize(%d): got %d, want %d", tt.requested, got, tt.want)
		}
	}
}

// Part 4 §5.13.1.5: queue overflow handling for both discard policies and the
// queue of size one.
func TestNotificationQueuePush(t *testing.T) {
	tests := []struct {
		name          string
		size          int
		discardOldest bool
		want          []int32
		overflow      []bool
	}{
		{"not full", 10, true, []int32{1, 2, 3, 4, 5}, []bool{false, false, false, false, false}},
		{"discard oldest", 3, true, []int32{3, 4, 5}, []bool{true, false, false}},
		{"discard newest", 3, false, []int32{1, 2, 5}, []bool{false, false, true}},
		{"size one, discard oldest", 1, true, []int32{5}, []bool{false}},
		{"size one, discard newest", 1, false, []int32{5}, []bool{false}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := &notificationQueue{size: tt.size, discardOldest: tt.discardOldest}
			pushValues(q, 1, 2, 3, 4, 5)
			checkQueue(t, q.entries, tt.want, tt.overflow)
		})
	}
}

// The Overflow bit leaves the rest of the StatusCode, the value and the
// timestamps unchanged, and does not modify the DataValue it was given.
func TestNotificationQueueOverflowKeepsStatusAndTimestamps(t *testing.T) {
	src := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	bad := &ua.DataValue{
		EncodingMask:    ua.DataValueValue | ua.DataValueStatusCode | ua.DataValueSourceTimestamp | ua.DataValueServerTimestamp,
		Value:           ua.MustVariant(int32(2)),
		Status:          ua.StatusBadTypeMismatch,
		SourceTimestamp: src,
		ServerTimestamp: src.Add(time.Second),
	}

	q := &notificationQueue{size: 2, discardOldest: true}
	q.push(queuedValue{seq: 1, value: int32Value(1)})
	q.push(queuedValue{seq: 2, value: bad})
	q.push(queuedValue{seq: 3, value: int32Value(3)})

	got := q.entries[0].value
	if want := ua.StatusBadTypeMismatch | statusGoodOverflow; got.Status != want {
		t.Fatalf("got status 0x%08X, want 0x%08X", uint32(got.Status), uint32(want))
	}
	if got.Value.Value() != int32(2) || !got.SourceTimestamp.Equal(src) || !got.ServerTimestamp.Equal(src.Add(time.Second)) {
		t.Fatalf("value or timestamps changed: %+v", got)
	}
	if bad.Status != ua.StatusBadTypeMismatch {
		t.Fatalf("source DataValue was modified: status 0x%08X", uint32(bad.Status))
	}

	// A Good value without a StatusCode in the encoding mask gets one.
	if dv := withOverflow(int32Value(1)); dv.Status != statusGoodOverflow || !dv.Has(ua.DataValueStatusCode) {
		t.Fatalf("got status 0x%08X mask 0x%02X", uint32(dv.Status), dv.EncodingMask)
	}
}

// StructureChanged and SemanticsChanged, Part 4 §7.38.1, Table 176.
const (
	structureChanged ua.StatusCode = 0x00008000
	semanticsChanged ua.StatusCode = 0x00004000
)

// flaggedValues returns queued values 1..n; flags[i] is added to the status
// of value i.
func flaggedValues(n int, flags map[int32]ua.StatusCode) []queuedValue {
	var vs []queuedValue
	for i := int32(1); i <= int32(n); i++ {
		dv := int32Value(i)
		if f := flags[i]; f != 0 {
			dv.Status = f
			dv.EncodingMask |= ua.DataValueStatusCode
		}
		vs = append(vs, queuedValue{seq: uint64(i), value: dv})
	}
	return vs
}

// checkStatuses compares the values and statuses of the queued entries.
func checkStatuses(t *testing.T, q *notificationQueue, want []int32, status []ua.StatusCode) {
	t.Helper()
	if len(q.entries) != len(want) {
		t.Fatalf("got %d queued values, want %d", len(q.entries), len(want))
	}
	for i, e := range q.entries {
		if got := e.value.Value.Value(); got != want[i] {
			t.Errorf("value %d: got %v, want %d", i, got, want[i])
		}
		if e.value.Status != status[i] {
			t.Errorf("value %v: got status 0x%08X, want 0x%08X", want[i], uint32(e.value.Status), uint32(status[i]))
		}
	}
}

// Part 4 §7.38.1, Table 176: the Overflow bit is set only where the InfoType
// gives the info bits a meaning. Other DataValue info bits are kept, info bits
// of an InfoType NotUsed are cleared, and a reserved InfoType is left as it is.
func TestWithOverflowInfoType(t *testing.T) {
	const otherInfoBit ua.StatusCode = 0x00000001
	for _, tt := range []struct {
		name       string
		status     ua.StatusCode
		want       ua.StatusCode
		sameAsSent bool
	}{
		{"not used", ua.StatusOK, statusGoodOverflow, false},
		{"not used, stray info bits", otherInfoBit, statusGoodOverflow, false},
		{"data value, other info bit", statusInfoTypeDataValue | otherInfoBit, statusGoodOverflow | otherInfoBit, false},
		{"reserved", 0x00000800 | otherInfoBit, 0x00000800 | otherInfoBit, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			v := &ua.DataValue{EncodingMask: ua.DataValueStatusCode, Status: tt.status}
			got := withOverflow(v)
			if got.Status != tt.want {
				t.Fatalf("got status 0x%08X, want 0x%08X", uint32(got.Status), uint32(tt.want))
			}
			if (got == v) != tt.sameAsSent {
				t.Fatalf("returned the given DataValue: %v, want %v", got == v, tt.sameAsSent)
			}
		})
	}
}

// Part 4 §7.38.1, Table 176: when a value with StructureChanged or
// SemanticsChanged set is discarded, the next value in the queue gets the bit.
func TestNotificationQueueCarriesChangedBits(t *testing.T) {
	t.Run("discard oldest", func(t *testing.T) {
		q := &notificationQueue{size: 3, discardOldest: true}
		for _, v := range flaggedValues(4, map[int32]ua.StatusCode{1: structureChanged}) {
			q.push(v)
		}
		checkStatuses(t, q, []int32{2, 3, 4}, []ua.StatusCode{structureChanged | statusGoodOverflow, 0, 0})
	})
	t.Run("discard newest", func(t *testing.T) {
		q := &notificationQueue{size: 3, discardOldest: false}
		for _, v := range flaggedValues(4, map[int32]ua.StatusCode{3: semanticsChanged}) {
			q.push(v)
		}
		checkStatuses(t, q, []int32{1, 2, 4}, []ua.StatusCode{0, 0, semanticsChanged | statusGoodOverflow})
	})
	t.Run("size one", func(t *testing.T) {
		q := &notificationQueue{size: 1, discardOldest: true}
		vs := flaggedValues(2, map[int32]ua.StatusCode{1: semanticsChanged})
		for _, v := range vs {
			q.push(v)
		}
		checkStatuses(t, q, []int32{2}, []ua.StatusCode{semanticsChanged})
		// The value had no StatusCode in its encoding mask; the inherited
		// bit is only sent if the mask now has one.
		if !q.entries[0].value.Has(ua.DataValueStatusCode) {
			t.Fatal("inherited bit not in the encoding mask")
		}
		if vs[1].value.Status != 0 {
			t.Fatalf("source DataValue was modified: status 0x%08X", uint32(vs[1].value.Status))
		}
	})
}

// queueTest drives the MonitoredItem service against a subscription that is
// registered without starting its publish loop, so that the test is the only
// consumer of the subscription's queues.
type queueTest struct {
	t      *testing.T
	srv    *Server
	ns     *NodeNameSpace
	sess   *session
	sub    *Subscription
	values map[string]int32
}

// queueTestServer is shared by the queue tests that need no options, since
// New parses the predefined NodeSet and is slow, in particular under -race.
// Each test gets its own namespace, session and subscription on it.
var queueTestServer = sync.OnceValue(func() *Server {
	srv := New()
	srv.initHandlers()
	return srv
})

// queueTestSubID numbers the subscriptions of the queue tests.
var queueTestSubID atomic.Uint32

// newQueueTest returns a queue test on the shared server, or on a server of
// its own if options are given.
func newQueueTest(t *testing.T, opts ...Option) *queueTest {
	t.Helper()
	var srv *Server
	if len(opts) == 0 {
		srv = queueTestServer()
	} else {
		srv = New(opts...)
		srv.initHandlers()
	}
	qt := &queueTest{
		t:      t,
		srv:    srv,
		ns:     NewNodeNameSpace(srv, "urn:test:queue"),
		sess:   srv.sb.NewSession(),
		values: map[string]int32{},
	}
	qt.sub = NewSubscription()
	qt.sub.srv = srv.SubscriptionService
	qt.sub.Session = qt.sess
	qt.sub.ID = queueTestSubID.Add(1)
	qt.sub.RevisedPublishingInterval = 100
	srv.SubscriptionService.Mu.Lock()
	srv.SubscriptionService.Subs[qt.sub.ID] = qt.sub
	srv.SubscriptionService.Mu.Unlock()
	t.Cleanup(func() { srv.SubscriptionService.DeleteSubscription(qt.sub.ID) })
	return qt
}

func (qt *queueTest) node(name string) *ua.NodeID {
	nodeID := ua.NewStringNodeID(qt.ns.ID(), name)
	qt.ns.AddNode(NewVariableNode(nodeID, name, func() *ua.DataValue {
		return DataValueFromValue(qt.values[name])
	}))
	return nodeID
}

func (qt *queueTest) header() *ua.RequestHeader {
	return &ua.RequestHeader{AuthenticationToken: qt.sess.AuthTokenID}
}

func (qt *queueTest) create(nodeID *ua.NodeID, clientHandle, queueSize uint32, discardOldest bool) *ua.MonitoredItemCreateResult {
	qt.t.Helper()
	return qt.createWith(nodeID, &ua.MonitoringParameters{
		ClientHandle:  clientHandle,
		QueueSize:     queueSize,
		DiscardOldest: discardOldest,
	})
}

func (qt *queueTest) createWith(nodeID *ua.NodeID, params *ua.MonitoringParameters) *ua.MonitoredItemCreateResult {
	qt.t.Helper()
	resp, err := qt.srv.MonitoredItemService.CreateMonitoredItems(nil, &ua.CreateMonitoredItemsRequest{
		RequestHeader:  qt.header(),
		SubscriptionID: qt.sub.ID,
		ItemsToCreate: []*ua.MonitoredItemCreateRequest{{
			ItemToMonitor:       &ua.ReadValueID{NodeID: nodeID, AttributeID: ua.AttributeIDValue},
			MonitoringMode:      ua.MonitoringModeReporting,
			RequestedParameters: params,
		}},
	}, 0)
	if err != nil {
		qt.t.Fatalf("CreateMonitoredItems: %v", err)
	}
	res := resp.(*ua.CreateMonitoredItemsResponse).Results[0]
	if res.StatusCode != ua.StatusOK {
		qt.t.Fatalf("CreateMonitoredItems: status %v", res.StatusCode)
	}
	return res
}

// set changes the value of the node and reports the change.
func (qt *queueTest) set(nodeID *ua.NodeID, vals ...int32) {
	for _, v := range vals {
		qt.values[nodeID.StringID()] = v
		qt.srv.ChangeNotification(nodeID)
	}
}

type handleValue struct {
	handle uint32
	value  int32
}

// drain empties the subscription's queues as a publish response without a
// notification limit would.
func (qt *queueTest) drain() []handleValue {
	items, more := qt.sub.drainQueues(0)
	if more {
		qt.t.Errorf("more notifications reported without a limit")
	}
	return qt.handleValues(items)
}

// handleValues returns the client handles and values of notifications and
// reports any notification whose status is not Good.
func (qt *queueTest) handleValues(items []*ua.MonitoredItemNotification) []handleValue {
	var got []handleValue
	for _, n := range items {
		v, _ := n.Value.Value.Value().(int32)
		got = append(got, handleValue{n.ClientHandle, v})
		if n.Value.Status != ua.StatusOK {
			qt.t.Errorf("value %d: unexpected status 0x%08X", v, uint32(n.Value.Status))
		}
	}
	return got
}

func (qt *queueTest) expect(got []handleValue, want ...handleValue) {
	qt.t.Helper()
	if len(got) != len(want) {
		qt.t.Fatalf("got %v, want %v", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			qt.t.Fatalf("got %v, want %v", got, want)
		}
	}
}

func TestCreateMonitoredItemsRevisesQueueSize(t *testing.T) {
	qt := newQueueTest(t)
	nodeID := qt.node("value")
	for _, tt := range []struct{ requested, want uint32 }{
		{0, 1}, {1, 1}, {2, 2}, {10, 10}, {5000, 5000}, {5001, 5000}, {math.MaxUint32, 5000},
	} {
		if got := qt.create(nodeID, 1, tt.requested, true).RevisedQueueSize; got != tt.want {
			t.Errorf("queueSize %d: got revisedQueueSize %d, want %d", tt.requested, got, tt.want)
		}
	}
}

// Every sampled value is delivered, oldest first, not only the latest one.
func TestMonitoredItemQueuesEveryValue(t *testing.T) {
	qt := newQueueTest(t)
	nodeID := qt.node("value")
	qt.create(nodeID, 7, 10, true)
	qt.expect(qt.drain(), handleValue{7, 0})

	qt.set(nodeID, 1, 2, 3, 4, 5, 6, 7)
	qt.expect(qt.drain(),
		handleValue{7, 1}, handleValue{7, 2}, handleValue{7, 3}, handleValue{7, 4},
		handleValue{7, 5}, handleValue{7, 6}, handleValue{7, 7})
	qt.expect(qt.drain())
}

// Items that share a ClientHandle keep separate queues.
func TestMonitoredItemQueuesAreIsolated(t *testing.T) {
	for _, tt := range []struct {
		queueSize uint32
		want      []handleValue
	}{
		{5, []handleValue{{7, 101}, {7, 201}, {7, 102}, {7, 202}, {7, 103}, {7, 203}}},
		{1, []handleValue{{7, 103}, {7, 203}}},
	} {
		qt := newQueueTest(t)
		a, b := qt.node("a"), qt.node("b")
		qt.create(a, 7, tt.queueSize, true)
		qt.create(b, 7, tt.queueSize, true)
		qt.drain()

		for i := int32(1); i <= 3; i++ {
			qt.set(a, 100+i)
			qt.set(b, 200+i)
		}
		qt.expect(qt.drain(), tt.want...)
	}
}

// Creating an item samples only that item, not the other items on the node.
func TestCreateMonitoredItemsSamplesOnlyTheNewItem(t *testing.T) {
	qt := newQueueTest(t)
	nodeID := qt.node("value")
	qt.create(nodeID, 1, 10, true)
	qt.drain()

	qt.create(nodeID, 2, 10, true)
	qt.expect(qt.drain(), handleValue{2, 0})
}

// Missing MonitoringParameters in a create request count as defaults.
func TestCreateMonitoredItemsWithoutParameters(t *testing.T) {
	qt := newQueueTest(t)
	nodeID := qt.node("value")
	item := qt.createWith(nodeID, nil)
	if item.RevisedQueueSize != 1 {
		t.Fatalf("got revisedQueueSize %d, want 1", item.RevisedQueueSize)
	}
	qt.expect(qt.drain(), handleValue{0, 0})
}

// DeleteMonitoredItem, which also deletes the items of a deleted
// Subscription, removes the item's queue.
func TestDeleteMonitoredItemRemovesQueue(t *testing.T) {
	qt := newQueueTest(t)
	item := qt.create(qt.node("value"), 1, 10, true)
	qt.srv.MonitoredItemService.DeleteMonitoredItem(item.MonitoredItemID)
	qt.sub.queueMu.Lock()
	_, ok := qt.sub.queues[item.MonitoredItemID]
	qt.sub.queueMu.Unlock()
	if ok {
		t.Fatal("queue of the deleted item still exists")
	}
}

// Deleted items lose their queued values at once.
func TestDeleteMonitoredItemsDiscardsQueuedValues(t *testing.T) {
	qt := newQueueTest(t)
	a, b := qt.node("a"), qt.node("b")
	itemA := qt.create(a, 1, 10, true)
	qt.create(b, 2, 10, true)
	qt.drain()
	qt.set(a, 1, 2, 3)
	qt.set(b, 4)

	_, err := qt.srv.MonitoredItemService.DeleteMonitoredItems(nil, &ua.DeleteMonitoredItemsRequest{
		RequestHeader:    qt.header(),
		SubscriptionID:   qt.sub.ID,
		MonitoredItemIDs: []uint32{itemA.MonitoredItemID},
	}, 0)
	if err != nil {
		t.Fatalf("DeleteMonitoredItems: %v", err)
	}
	qt.set(a, 5)
	qt.expect(qt.drain(), handleValue{2, 4})
}

// The server keeps its own copy of a create request: filling in default
// parameters leaves the caller's request as it was.
func TestCreateMonitoredItemsKeepsCallerRequest(t *testing.T) {
	qt := newQueueTest(t)
	nodeID := qt.node("value")
	withoutParams := &ua.MonitoredItemCreateRequest{
		ItemToMonitor:  &ua.ReadValueID{NodeID: nodeID, AttributeID: ua.AttributeIDValue},
		MonitoringMode: ua.MonitoringModeReporting,
	}
	_, err := qt.srv.MonitoredItemService.CreateMonitoredItems(nil, &ua.CreateMonitoredItemsRequest{
		RequestHeader:  qt.header(),
		SubscriptionID: qt.sub.ID,
		ItemsToCreate:  []*ua.MonitoredItemCreateRequest{withoutParams},
	}, 0)
	if err != nil {
		t.Fatalf("CreateMonitoredItems: %v", err)
	}
	if withoutParams.RequestedParameters != nil {
		t.Fatal("create filled in the caller's request")
	}
}

// The MaxMonitoredItemsQueueSize option limits the revised queue size and is
// reported in the address space; zero is treated as one.
func TestMaxMonitoredItemsQueueSizeOption(t *testing.T) {
	qt := newQueueTest(t, MaxMonitoredItemsQueueSize(20))
	nodeID := qt.node("value")
	for _, tt := range []struct{ requested, want uint32 }{{50, 20}, {10, 10}} {
		if got := qt.create(nodeID, 1, tt.requested, true).RevisedQueueSize; got != tt.want {
			t.Errorf("queueSize %d: got revisedQueueSize %d, want %d", tt.requested, got, tt.want)
		}
	}
	ns, err := qt.srv.Namespace(0)
	if err != nil {
		t.Fatal(err)
	}
	dv := ns.Attribute(ua.NewNumericNodeID(0, id.Server_ServerCapabilities_MaxMonitoredItemsQueueSize), ua.AttributeIDValue)
	if dv.Value.Value() != uint32(20) {
		t.Errorf("capability reports %v, want 20", dv.Value.Value())
	}

	for _, tt := range []struct{ n, want uint32 }{{0, 1}, {math.MaxInt32 + 1, math.MaxInt32}, {math.MaxUint32, math.MaxInt32}} {
		var cfg serverConfig
		MaxMonitoredItemsQueueSize(tt.n)(&cfg)
		if cfg.cap.MaxMonitoredItemsQueueSize != tt.want {
			t.Errorf("MaxMonitoredItemsQueueSize(%d): got %d, want %d", tt.n, cfg.cap.MaxMonitoredItemsQueueSize, tt.want)
		}
	}
}

// Part 5 §6.3.2: ServerCapabilities.MaxMonitoredItemsQueueSize reports the
// limit that CreateMonitoredItems applies.
func TestServerCapabilitiesMaxMonitoredItemsQueueSize(t *testing.T) {
	srv := queueTestServer()
	ns, err := srv.Namespace(0)
	if err != nil {
		t.Fatal(err)
	}
	dv := ns.Attribute(ua.NewNumericNodeID(0, id.Server_ServerCapabilities_MaxMonitoredItemsQueueSize), ua.AttributeIDValue)
	if dv.Status != ua.StatusOK || dv.Value.Value() != uint32(5000) {
		t.Fatalf("got status %v value %v", dv.Status, dv.Value)
	}
}
