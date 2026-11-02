// After the transport drops, a client whose subscription the server discarded
// still goes on receiving notifications. No clause requires this: the server
// is not wrong to forget a subscription, and the specification says nothing
// about a client that finds its subscription gone beyond the recovery steps it
// gives for other cases. It is an interoperability requirement. gopcua's own
// server discards a session's subscriptions when the session is re-activated,
// and a client that cannot cope with that loses its data on every reconnect.
//
// The scripted server behaves that way. It answers as the harness does until
// the session is activated a second time, and from then on the subscription
// is gone: Publish fails with Bad_NoSubscription, and Republish, Transfer and
// Delete name the subscription as invalid. It accepts a new subscription
// under a new id and publishes on that one. See harness.go.
//
// The test checks only that data arrives after the drop. Whether the client
// notices through Publish, Republish or Transfer, and whether it recreates the
// subscription or does something else, is not the test's business.
package reconnect

import (
	"sync"
	"testing"
	"time"

	"github.com/gopcua/opcua/ua"
)

// recreatedSeq is the base of the numbers a recreated subscription uses.
// It is far above the notifications the drop interrupted, so a notification
// replayed from before the drop cannot pass for data the server sent after it.
const recreatedSeq = 1000

func TestDataResumesWhenServerDiscardsSubscriptions(t *testing.T) {
	// knownBug records that gopcua does not do this yet. While it is set, the
	// failure is logged and the build stays green. README.md says when to delete
	// it. It is local because every test file in the package has its own.
	const knownBug = "the client re-activates the session and never notices that its subscription is gone"

	srv := &discardingServer{}
	run := runScenario(scenarioOptions{Override: srv.answer})

	// A client cannot cause either of these.
	switch {
	case run.HarnessErr != "":
		t.Fatalf("could not set the test up: %s", run.HarnessErr)
	case !run.DropApplied:
		t.Fatalf("could not set the test up: the connection was never dropped")
	}

	fresh := 0
	for _, d := range run.Deliveries {
		if d.At.After(run.DropAt) && d.Seq > run.HeldSeq {
			fresh++
		}
	}

	switch {
	case !srv.discarded():
		t.Errorf("the client never re-activated its session after the drop, " +
			"so the server never discarded the subscription and the test did not run")
	case fresh > 0 && knownBug != "":
		t.Errorf("this now passes (%d notifications arrived after the drop) - "+
			"delete the knownBug constant in this file, in the commit that fixed it", fresh)
	case fresh > 0:
		t.Logf("data kept flowing: %d notifications arrived after the drop", fresh)
	case knownBug != "":
		t.Logf("known bug: %s (%d received in all, none after the drop)", knownBug, len(run.Deliveries))
	default:
		t.Errorf("the server discarded the subscription when the session was re-activated, "+
			"and no notification arrived after the drop (%d received in all)", len(run.Deliveries))
	}
}

// discardingServer answers like a server that drops a session's subscriptions
// when the session is re-activated. Until then it leaves every service to the
// honest answers. Safe for concurrent use: each connection is served on its
// own goroutine.
type discardingServer struct {
	mu          sync.Mutex
	activations int
	live        uint32 // the one subscription the server now holds; 0 = none
	lastID      uint32
	lastSeq     uint32
}

func (d *discardingServer) discarded() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.activations >= 2
}

func (d *discardingServer) answer(service string, req ua.Request) (ua.Response, bool) {
	resp, ok, paced := d.step(req)
	if paced {
		// The honest Publish is paced the same way.
		time.Sleep(200 * time.Millisecond)
	}
	return resp, ok
}

// step decides the answer under the lock. paced is set for a Publish, whose
// answer the caller delays outside the lock.
func (d *discardingServer) step(req ua.Request) (resp ua.Response, ok, paced bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	h := req.Header().RequestHandle

	if _, isActivate := req.(*ua.ActivateSessionRequest); isActivate {
		d.activations++
	}
	if d.activations < 2 {
		return nil, false, false
	}

	switch r := req.(type) {
	case *ua.PublishRequest:
		if d.live == 0 {
			return newServiceFault(h, ua.StatusBadNoSubscription), true, false
		}
		d.lastSeq++
		pub := newPublishResponse(recreatedSeq + d.lastSeq)
		pub.SubscriptionID = d.live
		return pub, true, true

	case *ua.CreateSubscriptionRequest:
		d.lastID++
		d.live = SubID + d.lastID
		return &ua.CreateSubscriptionResponse{
			ResponseHeader:            newResponseHeader(h, ua.StatusOK),
			SubscriptionID:            d.live,
			RevisedPublishingInterval: r.RequestedPublishingInterval,
			RevisedLifetimeCount:      1000,
			RevisedMaxKeepAliveCount:  10,
		}, true, false

	case *ua.RepublishRequest:
		return newServiceFault(h, ua.StatusBadSubscriptionIDInvalid), true, false

	case *ua.TransferSubscriptionsRequest:
		results := make([]*ua.TransferResult, len(r.SubscriptionIDs))
		for i := range results {
			results[i] = &ua.TransferResult{StatusCode: ua.StatusBadSubscriptionIDInvalid}
		}
		return &ua.TransferSubscriptionsResponse{
			ResponseHeader:  newResponseHeader(h, ua.StatusOK),
			Results:         results,
			DiagnosticInfos: []*ua.DiagnosticInfo{},
		}, true, false

	case *ua.DeleteSubscriptionsRequest:
		results := make([]ua.StatusCode, len(r.SubscriptionIDs))
		for i := range results {
			results[i] = ua.StatusBadSubscriptionIDInvalid
		}
		return &ua.DeleteSubscriptionsResponse{
			ResponseHeader:  newResponseHeader(h, ua.StatusOK),
			Results:         results,
			DiagnosticInfos: []*ua.DiagnosticInfo{},
		}, true, false
	}
	return nil, false, false
}
