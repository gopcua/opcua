// A client that has replaced a disowned subscription receives data on the
// replacement. OPC 10000-4 §6.7 asks for the replacement — "If the Republish
// returns Bad_SubscriptionIdInvalid, then the Client needs to create a new
// Subscription" — and the same clause says what the whole of §6.7 is for:
// "This will result in the Client receiving data and event Notifications
// without losing information provided the queues in the MonitoredItems do not
// overflow."
//
// new_subscription_after_republish_returns_invalid_id_test.go is the first
// half of that and this file is the second. A Subscription on its own
// publishes nothing: the MonitoredItems are what the data comes from, and
// they belong to the Subscription the server has just disowned, so a client
// that creates the replacement and stops there has an empty Subscription and
// an application that never hears anything again. It has done what the first
// sentence says and achieved nothing the second one promises. That client
// also costs nothing to produce by accident — recreating the items is a
// second service call on a path a client reaches only after a reconnect, and
// it is the call that gets dropped.
//
// The scripted server is row 9's, with every Republish answered
// Bad_SubscriptionIdInvalid. One thing it does is this row's whole mechanism:
// a CreateSubscription makes the fixture forget that anything is monitored,
// and it answers every Publish with a keep-alive until a CreateMonitoredItems
// arrives for that subscription. So a client that recreates the Subscription
// and not its items is answered for the rest of the run and given nothing,
// which is what the defect does to a real application. See harness.go.
//
// The row asserts the outcome and not the mechanism, which README.md's sixth
// review question asks for and which here is also the stronger assertion. The
// mechanism — a CreateMonitoredItems follows the CreateSubscription — would
// pass a client that sent the request for a subscription it had already
// abandoned, or that sent it and never polled again. What the row reads
// instead is runRecord.SeqAtRecreate, the sequence number the server had
// reached when the replacement was created, and whether anything above it
// reached the application's own callback. The server creates no notification
// at all while nothing is monitored, so a single delivery above that line is
// proof the items came back; how many followed is
// data_resumes_after_reconnect_test.go's question and not this one's.
//
// Whether there is a replacement at all is read off runRecord.RecreateAfter,
// which the fixture set in the same locked block as SeqAtRecreate. That is
// README.md's first review question answered by construction rather than by
// assertion: the row cannot find a replacement the line was never fixed for,
// because the two are one record. See newSubscriptionAfterDrop.
//
// The row does not require the replacement to follow a Republish, and that is
// deliberate rather than an oversight. Row 9 is the row that establishes the
// replacement was an answer to the status; what this one measures is what the
// replacement achieves, and a client that recreates a subscription without
// being told to is under the same obligation to make it carry data. Requiring
// a Republish first would also make the row blind to exactly the client it
// exists for: a client whose reconnect path recreates subscriptions and skips
// the items never sends a Republish at all, so a row keyed on the loop would
// decline it instead of failing it.
//
// Two kinds of run cannot be scored and they are split, because they are
// different things.
//
// A client that created no new Subscription after the drop is gopcua's known
// bug, logged below with the four other rows that log the same defect: it
// reconnects, re-activates, and leaves the subscription exactly as it was. It
// is not this row's business to object to that, because
// new_subscription_after_republish_returns_invalid_id_test.go already does,
// and that is the one place the knownBug constant costs this row something —
// while it stands, a client that recreates nothing is logged and green here.
// Delete the constant and that run becomes the t.Fatalf below, and the row is
// red for it.
//
// A client that created the replacement and then sent fewer than two Publish
// on it is declined up with the preconditions instead. The fixture answers a
// Publish and nothing else, so a client that did not ask was never in a
// position to be given anything, and "was given no data" cannot be read off
// "never asked for any". Two is what shows the asking was real: the fixture
// paces its answers, so by the time a second Publish went out the first had
// been answered, and that first answer is where the data would have been.
//
// There is no second run, and that is the one place this row departs from
// no_acknowledgements_when_nothing_is_available_test.go's shape. A control
// run exists there to tell the behaviour from a client that is simply silent,
// and the control this row would want — the client receives data after a
// reconnect — is failed by the very defect the row scores, because a client
// that never recreates its monitored items receives nothing after any
// reconnect, on this fixture or any other. Measured: with the m3e patch
// applied, a control run on the default server delivers no data after the
// drop either, so a row that required one would decline the defect instead of
// failing it. The capability check this row has instead is the fixture's own
// precondition. runScenario refuses to go on until a notification has reached
// the application's callback, so every run that reaches the switch below has
// already shown that this client delivers data on this subscription to this
// callback. Silence after the replacement is therefore a change in what the
// client does, not a client that was never able to speak.
//
// On the gopcua branches measured so far this row separates exactly what
// data_resumes_after_reconnect_test.go separates, and nothing more:
// recreateSubscription ends with recreate_monitoredItems, so a branch that
// reaches the recreate path at all brings the items back with the
// subscription. The client this row exists for splits those two, which no
// branch does and the m3e patch constructs. Against the branches the row
// guards a regression rather than telling them apart.
//
// The reference stacks. open62541 v1.4.22 is declined, not failed, and for
// the reason rows 8 and 9 decline it: it has no client-side replacement path
// at all. On a re-activation ua_client_connect.c resumes publishing and
// nothing else, and when the Session does go, cleanupSession calls
// __Client_Subscriptions_clean, which force-removes the client's
// subscriptions locally; the only thing in ua_client_subscriptions.c that
// creates a Subscription is the application's own call. So it never reaches
// the condition this row scores. node-opcua 2.187.19 passes it, measured
// here: data created after the replacement reached the application. Eclipse
// Milo and UA-.NETStandard are not settled, and what settles each is whether
// its own recreate path carries the items. Named and left. No stack read
// here fails this row and no stack run here fails it either, so it takes no
// exception.
package reconnect

import (
	"fmt"
	"testing"
	"time"
)

func TestDataResumesOnRecreatedSubscription(t *testing.T) {
	// knownBug records that gopcua does not do this yet; the text is logged in
	// place of a failure. README.md says when to delete it.
	const knownBug = "the client reconnects and re-activates the session and leaves the subscription exactly as it was, so it never creates the replacement this row reads, let alone monitors anything on it"

	// minSettle is the window a client needs after the drop to reconnect,
	// re-activate, be faulted, create the replacement, monitor the item on it
	// and be answered twice.
	// new_subscription_after_republish_returns_invalid_id_test.go allows two
	// seconds for everything up to the replacement; this row adds one service
	// call and two answers paced at publishPacing on the same loopback. Below
	// it, a client that never gets its data back cannot be told from one that
	// had no time to.
	const minSettle = 3 * time.Second

	if settleDefault < minSettle {
		t.Fatalf("could not set the test up: the settle window of %s is shorter than the %s a client "+
			"needs to reconnect, be faulted, create the replacement, monitor an item on it and be "+
			"answered, so a client that never gets its data back cannot be told from one that was "+
			"not given time. Lengthen settleDefault in harness.go", settleDefault, minSettle)
	}

	run := runScenario(runOptions{RepublishSubInvalid: true})

	if why := setupFailed(run); why != "" {
		t.Fatalf("could not set the test up: %s", why)
	}
	if why := clientSilentSetupFailed(run); why != "" {
		t.Fatalf("could not set the test up: %s", why)
	}
	// A client that never re-activated was never offered the subscription
	// back, so it was never told the subscription id was invalid and had no
	// reason to replace anything.
	if why := reconnectSetupFailed(run); why != "" {
		t.Fatalf("could not set the test up: %s", why)
	}

	// Having created no replacement is the known bug, so it goes through the
	// switch.
	why := newSubscriptionSetupFailed(run)

	// Only once there is a replacement is there something the client could
	// have been given data on, so the question of whether it ever asked is
	// asked here and not before.
	if why == "" {
		if o := publishOpportunitySetupFailed(run); o != "" {
			t.Fatalf("could not set the test up: %s", o)
		}
	}

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
			"on it, and no notification created after the replacement ever reached the application "+
			"(%d reached it in all, none above sequence %d), so the replacement carries nothing the "+
			"application subscribed to", published, len(run.Deliveries), run.SeqAtRecreate)
	}
}

// newSubscriptionAfterDrop reads what the client did about the subscription
// the server disowned: whether it created a replacement, and how many Publish
// requests it sent after that request went over the wire.
//
// Both answers come from runRecord.RecreateAfter, which the fixture set in the
// same locked block as runRecord.SeqAtRecreate, when the request arrived. The
// row does not look for that request itself, and the reason it does not is the
// whole point. It did once, by a rule of its own -- a CreateSubscription on a
// connection above run.DropConn that had already been activated -- and a row
// that decides "there is a replacement" by one rule and "here is the line its
// data has to be above" by another is a row that can be told there is a
// replacement and handed no line. SeqAtRecreate is zero when the fixture never
// fixed one, so every delivery in the run, every pre-drop one included, reads
// as data the replacement carried, and the switch below takes the "this now
// passes" arm for a client that was given nothing.
//
// The two rules part company whenever the client has more than one connection
// open before the drop. run.DropConn names only the connection the held
// Publish was on, and dropWatcher closes only that one, so a second pre-drop
// connection is both above run.DropConn before the drop and alive after it.
// The fixture's rule is the drop having been applied, which is neither of
// those things. No client here opens a second connection before the drop, so
// nothing produced the disagreement; the rule it was waiting on is not one the
// row controls.
//
// What follows is counted on every connection and not only the post-drop one,
// as in new_subscription_after_republish_returns_invalid_id_test.go: the
// clause asks for a Subscription that carries data, not for one on any
// particular channel. The fixture serves one subscription, so a Publish from
// here on is a Publish the replacement could have been answered under.
func newSubscriptionAfterDrop(run runRecord) (created bool, published int) {
	if run.RecreateAfter == 0 {
		return false, 0
	}
	for _, r := range run.Requests[run.RecreateAfter:] {
		if r.Service == "PublishRequest" {
			published++
		}
	}
	return true, published
}

// newSubscriptionSetupFailed reports why there is no replacement subscription
// to read anything off, or "" if there is one. A client that left the
// disowned subscription in place has not failed to put data on a replacement;
// it has no replacement, and what it has instead is
// new_subscription_after_republish_returns_invalid_id_test.go's to object to.
// The activation rule the message names is the fixture's and no longer this
// row's: a CreateSubscription arriving before its own connection's
// ActivateSession is faulted Bad_SessionNotActivated and never reaches the
// arm that sets RecreateAfter.
func newSubscriptionSetupFailed(run runRecord) string {
	if created, _ := newSubscriptionAfterDrop(run); !created {
		return "the client created no new Subscription after the drop on a connection it had " +
			"activated, so there is no replacement for it to have received anything on"
	}
	return ""
}

// publishOpportunitySetupFailed reports why the client was never in a position
// to be given data on the replacement, or "" if it was. The fixture answers a
// Publish and sends nothing uninvited, so a client that did not poll was not
// given data because it did not ask, and that says nothing about its
// monitored items.
//
// Two Publish requests are what make the asking real. The fixture waits
// publishPacing before answering one, so a client that sent a second had its
// first answered first, and that first answer is the one that would have
// carried data if the items were back.
func publishOpportunitySetupFailed(run runRecord) string {
	created, published := newSubscriptionAfterDrop(run)
	if !created {
		return ""
	}
	if published < 2 {
		return fmt.Sprintf("the client created a new Subscription and then sent %d Publish after "+
			"it, so the server was never in a position to hand it data on the replacement", published)
	}
	return ""
}

// dataAfterNewSubscription is every distinct notification the application's
// callback received that the server created after the replacement
// subscription existed, ascending by nothing in particular — the order is the
// order they arrived in.
//
// runRecord.SeqAtRecreate is the line. The server's sequence counter is
// global and never resets, and it creates no notification at all while
// nothing is monitored, so anything above that line was created for the
// replacement and only after something was monitored on it. Under
// RepublishSubInvalid no Republish is answered with a notification either, so
// a data PublishResponse is the only way one of these could have reached the
// client.
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
