// A client that has replaced a disowned subscription receives data on the
// replacement. OPC 10000-4 §6.7 asks for the replacement and says what the
// asking is for: "This will result in the Client receiving data and event
// Notifications without losing information provided the queues in the
// MonitoredItems do not overflow." Conformance unit 276 names Republish and
// not this outcome, so the sentence is the clause's alone.
//
// Data on the subscription that survived the drop, rather than on one built
// to replace it, is data_resumes_after_reconnect_test.go. Whether the client
// replaces the subscription at all is
// new_subscription_after_republish_returns_invalid_id_test.go.
//
// The scripted server is the one in
// new_subscription_after_republish_returns_invalid_id_test.go, with every
// Republish answered Bad_SubscriptionIdInvalid. A CreateSubscription makes
// the fixture forget that anything is monitored, and it answers every Publish
// with a keep-alive until a CreateMonitoredItems arrives for that
// subscription. See harness.go.
//
// The row asserts the outcome and not the mechanism: a notification the
// server created after the replacement existed reached the application's own
// callback. It does not ask that the replacement followed a Republish, which
// would decline the client it exists for — one whose reconnect path recreates
// subscriptions and never asks for anything. While knownBug stands the arm
// keys on an absence: a client that created no replacement is logged green.
//
// The reference stacks. open62541 v1.4.22 has no client-side replacement
// path, so it never reaches the condition this row scores; while knownBug
// stands that is the green log above and not a decline, and the row cannot
// tell it from gopcua. node-opcua 2.187.19 passes it, measured here.
// Eclipse Milo and UA-.NETStandard are unsettled; what settles each is
// whether its own recreate path carries the items. No stack here fails it:
// no exception.

package reconnect

import (
	"fmt"
	"testing"
	"time"
)

func TestDataResumesOnRecreatedSubscription(t *testing.T) {
	// knownBug records that gopcua does not do this yet; the text is logged in
	// place of a failure. README.md says when to delete it.
	// https://github.com/gopcua/opcua/issues/879
	const knownBug = "the client reconnects and re-activates the session and leaves the subscription exactly as it was, so it never creates the replacement this row reads, let alone monitors anything on it"

	// minSettle is the window a client needs after the drop to reconnect,
	// re-activate, be faulted, create the replacement, monitor the item on it
	// and be answered twice: one more service call and two paced answers than
	// new_subscription_after_republish_returns_invalid_id_test.go's two seconds.
	const minSettle = 3 * time.Second

	// A tripwire, not a check: both sides are constants, so a settleDefault
	// shortened below minSettle fails the build here rather than a run.
	const _ = uint64(settleDefault - minSettle)

	run := runScenario(runOptions{RepublishSubInvalid: true})

	if why := setupFailed(run); why != "" {
		t.Fatalf("could not set the test up: %s", why)
	}
	if why := clientSilentSetupFailed(run); why != "" {
		t.Fatalf("could not set the test up: %s", why)
	}
	// A client that never re-activated was never offered the subscription
	// back, so it was never told the id was invalid and had nothing to replace.
	if why := reconnectSetupFailed(run); why != "" {
		t.Fatalf("could not set the test up: %s", why)
	}
	if why := publishOpportunitySetupFailed(run); why != "" {
		t.Fatalf("could not set the test up: %s", why)
	}

	// Having no replacement is the known bug, so it goes through the switch.
	why := newSubscriptionSetupFailed(run)

	fresh := dataAfterNewSubscription(run)
	_, published := newSubscriptionAfterDrop(run)

	switch {
	case why != "" && knownBug != "":
		t.Logf("known bug: %s (%s)", knownBug, why)
	case why != "":
		t.Fatalf("could not set the test up: %s", why)
	case len(fresh) > 0 && knownBug != "":
		t.Errorf("this now passes (the client replaced the subscription the server had disowned and "+
			"notifications %v, all created after the replacement, reached the application) - delete "+
			"the knownBug constant and the two arms that read it, in the commit that fixed it", fresh)
	case len(fresh) > 0:
		t.Logf("the client replaced the subscription the server had disowned and notifications %v, "+
			"all created after the replacement, reached the application", fresh)
	default:
		t.Errorf("the client replaced the subscription the server had disowned and sent %d Publish "+
			"the server acted on, and no notification created after the replacement ever reached "+
			"the application (%d reached it in all, none above sequence %d), so the replacement "+
			"carries nothing the application subscribed to", published, len(run.Deliveries), run.SeqAtRecreate)
	}
}

// newSubscriptionAfterDrop reads what the client did about the subscription
// the server disowned: whether it created a replacement, and how many Publish
// the server could have answered under it afterwards. published is a lower
// bound on what the client sent, and is meaningless when created is false.
//
// Both answers rest on runRecord.RecreateAfter, so the replacement and the
// line its data has to clear are one record rather than two readings that can
// disagree. The count spans every connection, because the fixture serves one
// subscription and the clause asks for data and not for a channel, but not a
// connection that had yet to activate: the server faults a Publish arriving
// before its own ActivateSession and does nothing with it.
func newSubscriptionAfterDrop(run runRecord) (created bool, published int) {
	if run.RecreateAfter == 0 {
		return false, 0
	}
	activated := map[int]bool{}
	for i, r := range run.Requests {
		switch {
		case r.Service == "ActivateSessionRequest":
			activated[r.Conn] = true
		case i >= run.RecreateAfter && r.Service == "PublishRequest" && activated[r.Conn]:
			published++
		}
	}
	return true, published
}

// newSubscriptionSetupFailed reports why there is no replacement subscription
// to read anything off, or "" if there is one. A client that left the
// disowned subscription in place has no replacement to carry data, and that
// is new_subscription_after_republish_returns_invalid_id_test.go's to object
// to. The activation rule the message names is the fixture's own.
func newSubscriptionSetupFailed(run runRecord) string {
	if created, _ := newSubscriptionAfterDrop(run); !created {
		return "the client created no new Subscription after the drop on a connection it had " +
			"activated, so there is no replacement for it to have received anything on"
	}
	return ""
}

// publishOpportunitySetupFailed reports why the client was never in a
// position to be given data on the replacement, or "" if it was, and says
// nothing about a client that created no replacement.
//
// The fixture sends nothing uninvited and waits publishPacing before
// answering a Publish, so a client that sent a second had its first answered,
// and that first answer is where the data would have been.
func publishOpportunitySetupFailed(run runRecord) string {
	created, published := newSubscriptionAfterDrop(run)
	if !created {
		return ""
	}
	if published < 2 {
		return fmt.Sprintf("the client created a new Subscription and then sent %d Publish the "+
			"server acted on after it, so the server was never in a position to hand it data on "+
			"the replacement", published)
	}
	return ""
}

// dataAfterNewSubscription is every distinct notification the application's
// callback received that the server created after the replacement existed,
// in the order they arrived. runRecord.SeqAtRecreate is the line: the
// server's sequence counter never resets and it creates no notification at
// all while nothing is monitored, and under RepublishSubInvalid no Republish
// is answered with one either, so a data PublishResponse is the only route.
func dataAfterNewSubscription(run runRecord) []uint32 {
	var out []uint32
	seen := map[uint32]bool{}
	for _, d := range run.Deliveries {
		if d.Seq > run.SeqAtRecreate && !seen[d.Seq] {
			seen[d.Seq] = true
			out = append(out, d.Seq)
		}
	}
	return out
}
