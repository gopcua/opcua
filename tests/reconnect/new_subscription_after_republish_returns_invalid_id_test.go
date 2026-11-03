// A client told that the subscription it is republishing on no longer exists
// creates a new one. OPC 10000-4 §6.7: "If the Republish returns
// Bad_SubscriptionIdInvalid, then the Client needs to create a new
// Subscription." Conformance unit 276 names Republish and not this status, so
// the sentence is the clause's alone.
//
// Whether the client republishes at all is republish_after_reconnect_test.go,
// and what it does when the loop is told the server has no more is
// publish_resumes_when_republish_loop_ends_test.go. This file is about the
// other way a Republish can end: the server rejects the subscription id
// rather than the sequence number, and nothing the client does on that
// subscription afterwards can work.
//
// The scripted server answers every Republish with Bad_SubscriptionIdInvalid
// and is otherwise the default one, which transfers the subscription when
// asked. See harness.go. It says nothing about subscriptions in that answer
// and nothing anywhere else: creating one is entirely the client's decision,
// which is why this is a client behaviour and not an echo of the fixture.
//
// The row asserts the outcome and not the reasoning: after the last Republish
// the client sent on a post-drop connection it had activated, at least one
// CreateSubscription follows. It is deliberately not restricted to that same
// connection, because the clause asks for a new Subscription and not for one
// on any particular channel: a client that answers the status by recreating
// its Session first, which is node-opcua's shape, has still created the
// Subscription the clause asks for. Nothing is said about how soon, about
// DeleteSubscriptions, or about monitored items.
//
// The failing shape is a client that treats Bad_SubscriptionIdInvalid as the
// loop terminator and goes back to polling on a subscription id the server
// has disowned. Its trace is the Bad_MessageNotAvailable control's exactly —
// Republish once, then Publish for the rest of the run — and it is a silent
// data-loss bug: the Publishes are answered, and nothing in them is the data
// the application subscribed to.
//
// One limit, so the row is not read for more than it says. gopcua reaches
// recreation on any Republish error that is not Bad_MessageNotAvailable:
// sendRepublishRequests returns nil for that one status and the error for
// every other, and republishSubscription's own default arm does the same. So
// the row asserts the outcome §6.7 names for the status §6.7 names; it does
// not establish that the client keys on that status rather than on "not the
// terminator". The clause demands only the behaviour, so this is within what
// the row may claim, and a second status would be needed to tell the two
// apart.
//
// A second limit, this one on the row's own instrument. open62541 is where
// knownBug costs this row something: it has no client-side Republish, so it
// sends none after the drop, which is the trace gopcua's defect leaves too,
// and both land in the knownBug arm below, logged and green. While the
// constant stands the row reports gopcua's bug over a client that has none.
// Deleting the constant does not separate them; it moves both to the
// t.Fatalf decline below, which is red and says plainly that nothing was
// measured. Keying the arm on a presence, as
// no_acknowledgements_when_nothing_is_available_test.go does, would escape
// this and is not open here: the defect this row scores is an absence and no
// fixture makes gopcua republish, so this is a limit to state rather than to
// engineer away.
//
// DeleteSubscriptions is not scripted. It falls to the catch-all in
// harness.go and is answered Bad_ServiceUnsupported, which is what a client
// that sends one before creating the replacement gets. The gopcua clients
// measured against this fixture send it and go on to CreateSubscription, so
// the row works as written and the fixture is not missing an arm.
//
// This row has one decline and not two. A run where the client sent no
// Republish after the drop on a connection it had activated is gopcua's known
// bug — it reconnects, re-activates, and asks for nothing it missed — so it
// goes through the knownBug arm below, with the other rows that log the same
// defect. The second decline
// publish_resumes_when_republish_loop_ends_test.go needs, a loop that ended
// on the client's own terms, cannot arise here: this server answers no
// Republish with a notification, so runRecord.Answerable is empty and any
// Republish that reached it was faulted with the status this row is about.
//
// The reference stacks, and the exception this row takes. Two of the four
// fail it, one passes it and one is declined, all read from source and none
// of them run here; node-opcua was also measured by hand against this
// fixture. The row is kept deliberately and the stacks it fails are named:
//
//   - Eclipse Milo fails it. PublishingManager treats Bad_MessageNotAvailable
//     as the normal end of the loop and anything else, Bad_SubscriptionIdInvalid
//     included, as a warning followed by return; the data-lost listener it
//     then calls is empty by default. Nothing in the library creates a
//     subscription.
//   - UA-.NETStandard fails it in both engines. The classic one logs the
//     status and raises PublishError; the V2 one records the number as not
//     dispatched. Neither recreates unless the application asks or the server
//     sends a Bad_Timeout StatusChangeNotification.
//   - open62541 v1.4.22 is declined, not failed, as in
//     republish_after_reconnect_test.go and
//     publish_resumes_when_republish_loop_ends_test.go: it has no client-side
//     Republish at all, so it never gets the status put to it.
//   - node-opcua 2.187.19 passes it, measured here. Its source escalates a
//     Republish failure other than Bad_MessageNotAvailable to recreating the
//     Session; against this fixture it takes the shorter route instead,
//     straight to CreateSubscription on the Session it still has, with no
//     CreateSession, TransferSubscriptions or DeleteSubscriptions after the
//     drop.
//
// So this is a behaviour most libraries leave to the application, and the row
// says so rather than claiming every client should pass. It is kept because
// the clause is explicit and names the status, because the alternative is
// silent data loss on a live subscription, and because the outcome cannot be
// reached by accident: the CreateSubscription this row counts has to follow a
// Republish the server faulted, so a client that recreates its subscription
// on every re-activation without ever asking does not pass.
package reconnect

import (
	"testing"
	"time"
)

func TestNewSubscriptionAfterRepublishReturnsInvalidID(t *testing.T) {
	// knownBug records that gopcua does not do this yet; the text is logged in
	// place of a failure. README.md says when to delete it.
	const knownBug = "the client reconnects and re-activates the session, but never asks for anything it missed, so it is never told the subscription id is invalid and never replaces the subscription"

	// minSettle is the window a client needs after the drop to reconnect,
	// re-activate, ask once, be faulted and create the replacement.
	// republish_after_reconnect_test.go measured reconnect, re-activation and
	// the ask at under a millisecond on loopback and allows a second of
	// margin; this row adds three more round trips on the same loopback, so it
	// takes twice that. Below it, a client that never replaces the
	// subscription cannot be told from one that had no time to.
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
	// loopRanSetupFailed, which subsumes it: that order is what makes the
	// knownBug text's "reconnects and re-activates" true of every run that
	// reaches it.
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
			"created no new one (%d requests after that ask, none of them a CreateSubscription)",
			lastAsk, sent)
	}
}

// subscriptionAfterInvalidID reads what the client did once the server had
// told it the subscription id was invalid: the sequence number of the last
// Republish it sent on a post-drop connection it had already activated, how
// many CreateSubscription requests it sent after that one went over the wire,
// and how many requests of any kind it sent after it.
//
// Which Republish counts is republishSeqsAfterDrop's rule: one that arrives
// before its own connection's ActivateSession is faulted
// Bad_SessionNotActivated rather than reaching the Republish arm at all, so
// the server never put the subscription-id status to it. What comes after is
// counted on every post-drop connection and not only that one, for the reason
// the file's opening comment gives.
//
// A run with no such Republish returns zeros; loopRanSetupFailed is what
// reports it, as it is for publish_resumes_when_republish_loop_ends_test.go.
func subscriptionAfterInvalidID(run runRecord) (lastAsk uint32, created, sent int) {
	activated := map[int]bool{}
	last := -1
	for i, r := range run.Requests {
		switch {
		case r.Service == "ActivateSessionRequest":
			activated[r.Conn] = true
		case r.Service == "RepublishRequest" && r.Conn > run.DropConn && activated[r.Conn]:
			last, lastAsk = i, r.RepublishSeq
		}
	}
	if last < 0 {
		return 0, 0, 0
	}
	for _, r := range run.Requests[last+1:] {
		if r.Conn <= run.DropConn {
			continue
		}
		sent++
		if r.Service == "CreateSubscriptionRequest" {
			created++
		}
	}
	return lastAsk, created, sent
}
