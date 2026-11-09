package opcua

import (
	"context"
	"testing"

	"github.com/gopcua/opcua/ua"
)

func testAck(sub, seq uint32) *ua.SubscriptionAcknowledgement {
	return &ua.SubscriptionAcknowledgement{SubscriptionID: sub, SequenceNumber: seq}
}

func TestRecordAvailSeqs(t *testing.T) {
	c := &Client{availSeqs: map[uint32]map[uint32]struct{}{}}

	c.recordAvailSeqs_NeedsSubMuxLock(&ua.PublishResponse{
		SubscriptionID: 7, AvailableSequenceNumbers: []uint32{3, 4},
	})
	if _, ok := c.availSeqs[7][4]; !ok {
		t.Fatal("sequence number advertised by the server was not recorded")
	}

	// A later response replaces the set: it describes the server's current
	// retransmission queue, it is not cumulative.
	c.recordAvailSeqs_NeedsSubMuxLock(&ua.PublishResponse{
		SubscriptionID: 7, AvailableSequenceNumbers: []uint32{5},
	})
	if _, ok := c.availSeqs[7][3]; ok {
		t.Fatal("stale sequence number kept: we would ack something the server dropped")
	}
	if _, ok := c.availSeqs[7][5]; !ok {
		t.Fatal("new sequence number was not recorded")
	}

	// Servers that retain nothing report an empty list.
	c.recordAvailSeqs_NeedsSubMuxLock(&ua.PublishResponse{
		SubscriptionID: 7, AvailableSequenceNumbers: nil,
	})
	if len(c.availSeqs[7]) != 0 {
		t.Fatalf("empty list should record an empty set, got %v", c.availSeqs[7])
	}
}

func TestFilterAcks(t *testing.T) {
	t.Run("does not infer from absence", func(t *testing.T) {
		// The number is not listed, but the list is not empty either. A server may
		// list it in a later response, so dropping here would leave it
		// unacknowledged forever.
		c := &Client{
			availSeqs:   map[uint32]map[uint32]struct{}{7: {4: {}}},
			pendingAcks: []*ua.SubscriptionAcknowledgement{testAck(7, 3), testAck(7, 4)},
		}
		if got := c.filterAcks_NeedsSubMuxLock(); len(got) != 2 {
			t.Fatalf("nothing may be dropped while the list is non-empty, got %+v", got)
		}
	})

	t.Run("drops when the server retains nothing", func(t *testing.T) {
		c := &Client{
			availSeqs:   map[uint32]map[uint32]struct{}{7: {}},
			pendingAcks: []*ua.SubscriptionAcknowledgement{testAck(7, 3)},
		}
		if got := c.filterAcks_NeedsSubMuxLock(); len(got) != 0 {
			t.Fatalf("server explicitly retains nothing, ack must be dropped, got %+v", got)
		}
	})

	t.Run("passes through before any response was seen", func(t *testing.T) {
		// No entry for the subscription means we never saw a PublishResponse for it,
		// so we cannot conclude anything about what the server retains.
		c := &Client{
			availSeqs:   map[uint32]map[uint32]struct{}{},
			pendingAcks: []*ua.SubscriptionAcknowledgement{testAck(9, 1)},
		}
		if got := c.filterAcks_NeedsSubMuxLock(); len(got) != 1 {
			t.Fatalf("must not drop before any response was seen, got %+v", got)
		}
	})
}

func TestAckRejectedLatch(t *testing.T) {
	// The latch exists because filtering on AvailableSequenceNumbers alone is not
	// enough: a server may advertise a sequence number in the very response that
	// carries it and drop it before the acknowledgement arrives.
	t.Run("latches on BadSequenceNumberUnknown", func(t *testing.T) {
		c := &Client{
			availSeqs:   map[uint32]map[uint32]struct{}{},
			pendingAcks: []*ua.SubscriptionAcknowledgement{testAck(7, 3)},
		}
		c.handleAcks_NeedsSubMuxLock([]ua.StatusCode{ua.StatusBadSequenceNumberUnknown})
		if !c.ackRejected {
			t.Fatal("latch must be set after the server rejected an acknowledgement")
		}
	})

	t.Run("suspends all acks while latched", func(t *testing.T) {
		c := &Client{
			availSeqs:   map[uint32]map[uint32]struct{}{},
			ackRejected: true,
			pendingAcks: []*ua.SubscriptionAcknowledgement{testAck(7, 3), testAck(8, 1)},
		}
		if got := c.filterAcks_NeedsSubMuxLock(); len(got) != 0 {
			t.Fatalf("no acknowledgement may be sent while latched, got %+v", got)
		}
	})

	t.Run("self-corrects on the next accepted ack", func(t *testing.T) {
		c := &Client{
			availSeqs:   map[uint32]map[uint32]struct{}{},
			ackRejected: true,
			pendingAcks: []*ua.SubscriptionAcknowledgement{testAck(7, 3)},
		}
		c.handleAcks_NeedsSubMuxLock([]ua.StatusCode{ua.StatusOK})
		if c.ackRejected {
			t.Fatal("latch must clear once the server accepts an acknowledgement again")
		}
	})
}

func TestForgetSubscriptionClearsAvailSeqs(t *testing.T) {
	c := &Client{
		subs:      map[uint32]*Subscription{7: {SubscriptionID: 7}},
		availSeqs: map[uint32]map[uint32]struct{}{7: {3: {}}},
		pausech:   make(chan struct{}, 2),
		resumech:  make(chan struct{}, 2),
	}
	c.forgetSubscription_NeedsSubMuxLock(context.Background(), 7)
	if _, ok := c.availSeqs[7]; ok {
		t.Fatal("availSeqs entry must be dropped with the subscription, otherwise the map grows without bound")
	}
}
