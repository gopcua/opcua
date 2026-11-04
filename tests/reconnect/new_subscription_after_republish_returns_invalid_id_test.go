// A client told that the subscription it is republishing on no longer exists
// creates a new one. OPC 10000-4 §6.7 asks for the replacement: "If the
// Republish returns Bad_SubscriptionIdInvalid, then the Client needs to
// create a new Subscription." Conformance unit 276 names Republish and not
// this status, so the sentence is the clause's alone.
//
// Whether the client republishes at all is republish_after_reconnect_test.go,
// what it does when the loop is told the server has no more is
// publish_resumes_when_republish_loop_ends_test.go, and whether the
// replacement carries data is data_resumes_on_recreated_subscription_test.go.
//
// The scripted server answers every Republish with Bad_SubscriptionIdInvalid
// and is otherwise the default one, which transfers the subscription when
// asked. See harness.go.
//
// The row asserts the outcome and not the reasoning: after the last Republish
// the client sent on a post-drop connection it had activated, at least one
// CreateSubscription follows on any connection it had activated. Nothing is
// said about how soon, about DeleteSubscriptions, or about monitored items.
//
// The reference stacks, and the exception this row takes. Eclipse Milo fails
// it: PublishingManager logs any status but Bad_MessageNotAvailable and
// returns without creating anything. UA-.NETStandard fails it in both
// engines, neither recreating unless the application asks. open62541 v1.4.22
// has no client-side Republish, so the status is never put to it. While
// knownBug stands that absence is logged green beside gopcua rather than
// declined, here and in republish_after_reconnect_test.go. node-opcua
// 2.187.19 passes it, measured here. The clause only asks, so the row is kept
// for what failing it costs: silent data loss on a live subscription.

package reconnect

import (
	"testing"
	"time"
)

func TestNewSubscriptionAfterRepublishReturnsInvalidID(t *testing.T) {
	// knownBug records that gopcua does not do this yet; the text is logged in
	// place of a failure. README.md says when to delete it.
	// https://github.com/gopcua/opcua/issues/879
	const knownBug = "the client reconnects and re-activates the session, but never asks for anything it missed, so it is never told the subscription id is invalid and never replaces the subscription"

	// minSettle is the window a client needs after the drop to reconnect,
	// re-activate, ask once, be faulted and create the replacement.
	// republish_after_reconnect_test.go allows a second for everything up to
	// the ask; this row adds three more round trips on the same loopback.
	const minSettle = 2 * time.Second

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
	// back, so it was never in a position to republish on it. It runs before
	// loopRanSetupFailed, which subsumes it; harness.go says why.
	if why := reconnectSetupFailed(run); why != "" {
		t.Fatalf("could not set the test up: %s", why)
	}

	// Having asked for nothing is the known bug, so it goes through the switch.
	why := loopRanSetupFailed(run)

	lastAsk, created, sent := subscriptionAfterInvalidID(run)

	switch {
	case why != "" && knownBug != "":
		t.Logf("known bug: %s (%s)", knownBug, why)
	case why != "":
		t.Fatalf("could not set the test up: %s", why)
	case created > 0 && knownBug != "":
		t.Errorf("this now passes (the client's last ask was for notification %d, which the server "+
			"answers with Bad_SubscriptionIdInvalid, and %d CreateSubscription followed it) - delete "+
			"the knownBug constant and the two arms that read it, in the commit that fixed it",
			lastAsk, created)
	case created > 0:
		t.Logf("the client's last ask was for notification %d, which the server answers with "+
			"Bad_SubscriptionIdInvalid, and %d CreateSubscription followed it", lastAsk, created)
	default:
		t.Errorf("the client's last ask was for notification %d, which the server answers with "+
			"Bad_SubscriptionIdInvalid, so the subscription it was republishing on is gone, and it "+
			"created no new one (%d requests after that ask, none of them a CreateSubscription the "+
			"server acted on)", lastAsk, sent)
	}
}

// subscriptionAfterInvalidID reads what the client did once the server had
// told it the subscription id was invalid: the sequence number of the last
// Republish it sent on a post-drop connection it had already activated, how
// many CreateSubscription requests followed it on an activated connection,
// and how many requests of any kind followed it on a post-drop connection.
//
// Activation is the rule on the first two, which are readings of the client:
// answer() faults a session service arriving before its own connection's
// ActivateSession with Bad_SessionNotActivated, so a Republish counted
// without it was never put the status and a CreateSubscription counted
// without it created nothing.
//
// sent is the third and is deliberately not filtered, as republishesAfterDrop
// keeps total unfiltered and for the same reason: it is the denominator the
// failure prints, and a human deciding whether gopcua is broken wants to be
// told everything the client sent, not the part of it the fixture answered.
// That is why the failure says "none of them a CreateSubscription the server
// acted on" and not "none of them a CreateSubscription" -- a client that
// pipelines CreateSubscription ahead of ActivateSession on a redial has one
// inside sent and none inside created, and the shorter sentence would be
// false about the very batch it was counting.
//
// A run with no such Republish returns zeros; loopRanSetupFailed is what
// reports it, as it is for publish_resumes_when_republish_loop_ends_test.go.
func subscriptionAfterInvalidID(run runRecord) (lastAsk uint32, created, sent int) {
	activated := map[int]bool{}
	last := -1
	var creates []int
	for i, r := range run.Requests {
		switch {
		case r.Service == "ActivateSessionRequest":
			activated[r.Conn] = true
		case r.Conn <= run.DropConn || !activated[r.Conn]:
		case r.Service == "RepublishRequest":
			last, lastAsk = i, r.RepublishSeq
		case r.Service == "CreateSubscriptionRequest":
			creates = append(creates, i)
		}
	}
	if last < 0 {
		return 0, 0, 0
	}
	for _, i := range creates {
		if i > last {
			created++
		}
	}
	for _, r := range run.Requests[last+1:] {
		if r.Conn > run.DropConn {
			sent++
		}
	}
	return lastAsk, created, sent
}
