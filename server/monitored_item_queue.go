package server

import (
	"container/heap"
	"slices"

	"github.com/gopcua/opcua/ua"
)

// DataValue StatusCode info bits, OPC UA Part 4 §7.38.1: InfoType and
// InfoBits in Table 176, the DataValue InfoBits in Table 177.
const (
	statusInfoTypeMask      ua.StatusCode = 0x00000C00 // InfoType, bits 10:11
	statusInfoTypeDataValue ua.StatusCode = 0x00000400 // InfoType DataValue (01)
	statusInfoBitsMask      ua.StatusCode = 0x000003FF // InfoBits, bits 0:9
	statusInfoBitOverflow   ua.StatusCode = 0x00000080 // Overflow, bit 7

	// StructureChanged (bit 15) and SemanticsChanged (bit 14). When a data
	// change notification with one of them set is discarded, the bit is set
	// on the next notification in the queue (Table 176).
	statusChangedBits ua.StatusCode = 0x0000C000
)

// defaultMaxMonitoredItemsQueueSize is the default upper bound for the queue
// size of a data MonitoredItem, the default of the
// ServerCapabilities.MaxMonitoredItemsQueueSize field.
const defaultMaxMonitoredItemsQueueSize = 5000

// defaultMaxNotificationsPerPublish is the default upper bound for the number
// of notifications in one Publish response, the default of the
// ServerCapabilities.MaxNotificationsPerPublish field. It bounds the number of
// notifications in a response when the client sets no limit or a large one;
// it does not bound the size in bytes.
const defaultMaxNotificationsPerPublish = 1000

// revisedQueueSize returns the queue size the server uses for a data
// MonitoredItem.
//
// Part 4 §7.21, Table 140: for a data MonitoredItem a requested size of 0 or
// 1 gives the default size 1; larger sizes may be limited by the server. The
// server does not implement event MonitoredItems, whose sizes Table 140
// revises differently, so every item is revised by this rule.
func revisedQueueSize(requested, limit uint32) uint32 {
	return min(max(requested, 1), max(limit, 1))
}

// queuedValue is one sampled value waiting for the next Publish response.
// seq numbers the values of all queues of a Subscription in the order they
// were sampled.
type queuedValue struct {
	seq   uint64
	value *ua.DataValue
}

// notificationQueue is the queue of a single data MonitoredItem.
//
// https://reference.opcfoundation.org/Core/Part4/v105/docs/5.13.1.5
type notificationQueue struct {
	clientHandle  uint32
	size          int
	discardOldest bool
	entries       []queuedValue
}

// push queues v according to the size and discard policy of the queue
// (Part 4 §5.13.1.5). The value that takes the place of a discarded one also
// inherits its StructureChanged and SemanticsChanged bits.
func (q *notificationQueue) push(v queuedValue) {
	switch {
	case q.size <= 1:
		// A queue of size one always holds the newest value. The discard
		// policy is ignored and the Overflow bit is not set.
		v.value = withChangedBits(v.value, q.entries...)
		q.entries = append(q.entries[:0], v)
	case len(q.entries) < q.size:
		q.entries = append(q.entries, v)
	case q.discardOldest:
		// Drop the oldest value and flag the value that is now oldest.
		dropped := q.entries[0]
		q.entries[0] = queuedValue{}
		q.entries = append(q.entries[1:], v)
		q.entries[0].value = withOverflow(withChangedBits(q.entries[0].value, dropped))
	default:
		// Replace the newest value and flag the new value.
		v.value = withOverflow(withChangedBits(v.value, q.entries[len(q.entries)-1]))
		q.entries[len(q.entries)-1] = v
	}
}

// withOverflow returns a copy of v with the Overflow info bit set. The rest
// of the StatusCode, the value and the timestamps are unchanged. v itself is
// not modified since namespaces may hand out shared DataValues.
func withOverflow(v *ua.DataValue) *ua.DataValue {
	var dv ua.DataValue
	if v != nil {
		dv = *v
	}
	switch dv.Status & statusInfoTypeMask {
	case 0:
		dv.Status = dv.Status&^statusInfoBitsMask | statusInfoTypeDataValue | statusInfoBitOverflow
	case statusInfoTypeDataValue:
		dv.Status |= statusInfoBitOverflow
	default:
		// Reserved InfoType: the info bits have no defined meaning.
		return v
	}
	dv.EncodingMask |= ua.DataValueStatusCode
	return &dv
}

// withChangedBits returns v with the StructureChanged and SemanticsChanged
// bits of the discarded values added. v is copied when a bit is added, and
// returned as it is otherwise.
func withChangedBits(v *ua.DataValue, discarded ...queuedValue) *ua.DataValue {
	var bits ua.StatusCode
	for _, d := range discarded {
		if d.value != nil {
			bits |= d.value.Status & statusChangedBits
		}
	}
	if bits == 0 || (v != nil && v.Status&bits == bits) {
		return v
	}
	var dv ua.DataValue
	if v != nil {
		dv = *v
	}
	dv.Status |= bits
	dv.EncodingMask |= ua.DataValueStatusCode
	return &dv
}

// createQueue creates the queue of a new MonitoredItem with the given id.
func (s *Subscription) createQueue(id, clientHandle, size uint32, discardOldest bool) {
	s.queueMu.Lock()
	defer s.queueMu.Unlock()
	if s.queues == nil {
		s.queues = map[uint32]*notificationQueue{}
	}
	s.queues[id] = &notificationQueue{
		clientHandle:  clientHandle,
		size:          int(size),
		discardOldest: discardOldest,
	}
}

// removeQueue discards the queue of the MonitoredItem with the given id and
// all values in it.
func (s *Subscription) removeQueue(id uint32) {
	s.queueMu.Lock()
	defer s.queueMu.Unlock()
	delete(s.queues, id)
}

// enqueue adds a sampled value to the queue of the MonitoredItem with the
// given id. Values for an item without a queue are dropped.
func (s *Subscription) enqueue(id uint32, v *ua.DataValue) {
	s.queueMu.Lock()
	defer s.queueMu.Unlock()
	q, ok := s.queues[id]
	if !ok {
		return
	}
	s.queueSeq++
	q.push(queuedValue{seq: s.queueSeq, value: v})
}

// hasQueued reports whether any MonitoredItem of the subscription has a
// value waiting to be published.
func (s *Subscription) hasQueued() bool {
	s.queueMu.Lock()
	defer s.queueMu.Unlock()
	for _, q := range s.queues {
		if len(q.entries) > 0 {
			return true
		}
	}
	return false
}

// drainQueues takes the queued values of the subscription and returns one
// MonitoredItemNotification per value. The values of each item keep their
// queue order (Part 4 §5.13.1.5, §7.25.1); across items they are in the order
// they were sampled, which the spec does not require. At most limit values
// are taken; a limit of zero takes all of them (§5.14.2.2, Table 82,
// maxNotificationsPerPublish). The values that do not fit stay queued, and
// more reports whether any are left (§5.14.1.2; moreNotifications in
// §5.14.5.2, Table 89).
func (s *Subscription) drainQueues(limit uint32) (items []*ua.MonitoredItemNotification, more bool) {
	s.queueMu.Lock()
	defer s.queueMu.Unlock()

	// Merge the queues by the sequence number of their oldest value: every
	// queue is looked at once, and of the queued values only those taken.
	var heads queueHeads
	var total int
	for _, q := range s.queues {
		if len(q.entries) > 0 {
			heads = append(heads, &queueHead{q: q})
			total += len(q.entries)
		}
	}
	if total == 0 {
		return nil, false
	}
	n := total
	if limit > 0 && uint64(limit) < uint64(total) {
		n = int(limit)
	}
	touched := slices.Clone(heads)
	heap.Init(&heads)

	items = make([]*ua.MonitoredItemNotification, 0, n)
	for len(items) < n {
		h := heads[0]
		items = append(items, &ua.MonitoredItemNotification{
			ClientHandle: h.q.clientHandle,
			Value:        h.q.entries[h.next].value,
		})
		h.next++
		if h.next == len(h.q.entries) {
			heap.Pop(&heads)
		} else {
			heap.Fix(&heads, 0)
		}
	}

	// The values of each queue are in sample order, so the values taken
	// from a queue are always at its front. Clear them so that the queue
	// does not keep them alive.
	for _, h := range touched {
		switch {
		case h.next == len(h.q.entries):
			h.q.entries = nil
		case h.next > 0:
			clear(h.q.entries[:h.next])
			h.q.entries = h.q.entries[h.next:]
		}
	}
	return items, len(heads) > 0
}

// queueHead points at the oldest value of a queue that drainQueues has not
// taken yet.
type queueHead struct {
	q    *notificationQueue
	next int
}

// queueHeads is a min-heap of queues ordered by the sequence number of the
// value each head points at.
type queueHeads []*queueHead

func (h queueHeads) Len() int { return len(h) }
func (h queueHeads) Less(i, j int) bool {
	return h[i].q.entries[h[i].next].seq < h[j].q.entries[h[j].next].seq
}
func (h queueHeads) Swap(i, j int) { h[i], h[j] = h[j], h[i] }
func (h *queueHeads) Push(x any)   { *h = append(*h, x.(*queueHead)) }
func (h *queueHeads) Pop() any {
	old := *h
	x := old[len(old)-1]
	*h = old[:len(old)-1]
	return x
}
