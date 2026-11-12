//go:build conformance

// This is a test of the harness and not a row. It pins publishStep's
// withholding rule: however many Publishes a client keeps outstanding, the
// fixture creates and withholds exactly one notification. OPC 10000-4
// §5.14.5.1 defines pipelining as "sending more than one Publish request for
// each Subscription before receiving a response": a client is allowed to
// keep several Publishes outstanding. If the fixture consumed a sequence
// number for every Publish it held, such a client would leave more than one
// notification created before the drop, and the run's one-lost check would
// fail it for something the client is allowed to do. So publishStep answers
// the first Publish, holds the first one after it with the
// withheld notification, and holds every further one with no response and no
// sequence number; the drop then answers each Publish with a fresh number.
//
// The recorder is driven directly, with no network and nothing monitored:
// publishStep does not read the monitored-items record, which is the
// keep-alive arm's gate in answer(), so the sequence behaviour is visible
// with nothing set up at all.

package reconnect

import "testing"

func TestFixtureWithholdsOneNotificationHoweverManyPublishesWait(t *testing.T) {
	rec := &recorder{}

	// The stream's first Publish is answered with the first sequence number.
	if seq, hold := rec.publishStep(1); hold || seq != 1 {
		t.Fatalf("first Publish: got seq %d, hold %v; want seq 1, hold false", seq, hold)
	}

	// The first Publish after the first answer is held with the withheld
	// notification; every further one is held with none.
	for i, want := range []uint32{2, 0, 0} {
		if seq, hold := rec.publishStep(1); !hold || seq != want {
			t.Fatalf("outstanding Publish %d: got seq %d, hold %v; want seq %d, hold true",
				i+1, seq, hold, want)
		}
	}

	rec.recordDropApplied(0)

	// The drop freezes the counter at the one withheld notification: the
	// fixture's own one-lost check has to pass on this record.
	if held, created := rec.heldPublishSeq(), rec.seqCreatedAtDrop(); held != 2 || created != 2 {
		t.Fatalf("after the drop: heldPublishSeq %d, seqCreatedAtDrop %d; want 2 and 2", held, created)
	}

	// After the drop, each Publish is answered with a fresh number.
	if seq, hold := rec.publishStep(2); hold || seq != 3 {
		t.Fatalf("Publish after the drop: got seq %d, hold %v; want seq 3, hold false", seq, hold)
	}
}
