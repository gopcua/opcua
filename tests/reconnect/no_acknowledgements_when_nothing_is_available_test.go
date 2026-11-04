// A client does not acknowledge sequence numbers to a server that keeps no
// retransmission queue. OPC 10000-4 §5.14.5.2, on the Publish response's
// availableSequenceNumbers: "The list shall be empty if the Server does not
// support the retransmission queue. If the list is empty, the Client should
// not acknowledge sequence numbers." Conformance unit 744/2284, Subscription
// Client Basic, stops at the rule this clause is the exception to —
// "Acknowledge received Notifications with subsequent Publish requests" — so
// a client that implements the unit and nothing else fails this row.
//
// The server runs twice: a control under AdvertisesSentSequence, which names
// every notification it sends in that response's own list, and the scored
// run under NoRetransmissionQueue, which advertises nothing. See harness.go.
//
// The row reads the acknowledgements on the wire and not how the client
// chose them, and the control is a precondition rather than a score: on the
// scored run alone a client that reads the list and one that acknowledges
// nothing anywhere are the same client. §5.14.7.1 gives the list a second
// meaning after a transfer and is not measured here — the scripted server
// keeps the Session across the drop, so nothing transfers, and reaching that
// clause needs a server that refuses to re-activate the surviving Session.
//
// The clause is a "should" and one of the two stacks read here ignores it,
// so the row names the stack it fails. open62541 v1.4.22 passes it the
// clause's own way: ua_client_subscriptions.c queues an acknowledgement only
// for a sequence number in that response's own availableSequenceNumbers.
// node-opcua 2.187.19 fails it, measured here: client_publish_engine.js
// reads the available list only in a commented-out line. Eclipse Milo and
// UA-.NETStandard were not read.

package reconnect

import (
	"fmt"
	"testing"
)

func TestNoAcknowledgementsWhenNothingIsAvailable(t *testing.T) {
	// knownBug records that gopcua does not do this yet; the text is logged in
	// place of a failure. README.md says when to delete it.
	const knownBug = "the client acknowledges every notification it is given and never reads the server's availableSequenceNumbers, so it acknowledges to a server that has just said it keeps nothing"

	control := runScenario(runOptions{AdvertisesSentSequence: true})

	if why := setupFailed(control); why != "" {
		t.Fatalf("could not set the test up: %s", why)
	}
	if why := acknowledgingClientSetupFailed(control); why != "" {
		t.Fatalf("could not set the test up: %s", why)
	}

	run := runScenario(runOptions{NoRetransmissionQueue: true})

	if why := setupFailed(run); why != "" {
		t.Fatalf("could not set the test up: %s", why)
	}
	if why := emptyAvailableSetupFailed(run); why != "" {
		t.Fatalf("could not set the test up: %s", why)
	}
	if why := ackOpportunitySetupFailed(run); why != "" {
		t.Fatalf("could not set the test up: %s", why)
	}

	acked := acknowledgementsSent(run)

	switch {
	case len(acked) > 0 && knownBug != "":
		t.Logf("known bug: %s (it acknowledged %v, and the server said on every answer that it "+
			"keeps nothing)", knownBug, acked)
	case len(acked) == 0 && knownBug != "":
		t.Errorf("this now passes (the client acknowledged nothing to a server that advertised "+
			"nothing, having acknowledged %v to one that advertised what it sent) - delete the "+
			"knownBug constant and the two arms that read it, in the commit that fixed it",
			acknowledgementsSent(control))
	case len(acked) == 0:
		t.Logf("the client acknowledged nothing to a server that advertised nothing, having "+
			"acknowledged %v to one that advertised what it sent",
			acknowledgementsSent(control))
	default:
		t.Errorf("the server said on every answer that it keeps no retransmission queue, and the "+
			"client acknowledged %v anyway", acked)
	}
}

// acknowledgementsSent is every acknowledgement the client put in a Publish,
// in the order they went over the wire. A request that is not a Publish
// carries none, so the service name does not need checking here. It is
// deliberately not filtered on activation, unlike the counts the other rows
// read: the claim is about the wire, and an acknowledgement in a request the
// server faulted still went over it. Over-counting can only block this row's
// green arm, never produce it.
func acknowledgementsSent(run runRecord) []ack {
	var out []ack
	for _, r := range run.Requests {
		out = append(out, r.Acks...)
	}
	return out
}

// acknowledgingClientSetupFailed reports why the control run says nothing
// about the clause, or "" if it says what the scored run needs. The server
// there names every notification it sends in the list it sends it with, so a
// client that acknowledges what the server still has acknowledges there. One
// that does not has been shown to be silent rather than to read the list, and
// silence in the scored run would then be worth nothing.
//
// Acknowledging a number it was never given would be a different defect and
// is not this row's, so what it checks is that something the client was given
// came back.
func acknowledgingClientSetupFailed(run runRecord) string {
	if len(run.Deliveries) == 0 {
		return "the client's callback was given no notification in the control run, so there was " +
			"nothing for it to acknowledge there"
	}
	for _, a := range acknowledgementsSent(run) {
		for _, d := range run.Deliveries {
			if a.Seq == d.Seq {
				return ""
			}
		}
	}
	return "the client acknowledged none of the notifications it was given by a server that " +
		"advertised every one of them as still available, so this run cannot tell a client that " +
		"acknowledges what the server has from one that never acknowledges at all"
}

// emptyAvailableSetupFailed reports why the scored run is not the one the
// clause speaks to, or "" if it is. This reads the fixture rather than the
// client: the row rests on the server having advertised nothing, and a run
// where it advertised something scores a client against a clause about a
// server that advertises nothing.
func emptyAvailableSetupFailed(run runRecord) string {
	if len(run.Offered) != 0 {
		return fmt.Sprintf("the server advertised %v as available for retransmission, and the "+
			"clause is about a server that advertises nothing", run.Offered)
	}
	return ""
}

// ackOpportunitySetupFailed reports why the client was never in a position to
// acknowledge anything in the scored run, or "" if it was. Acknowledging
// nothing is only evidence where there was something to acknowledge and a
// request to put it in.
//
// Two Publish requests establish the second. The fixture answers the first of
// the stream and holds the next, and runScenario refuses any run in which
// more than one notification was withheld, so the client's second Publish
// went out after the first had been answered and is where the acknowledgement
// would have been.
func ackOpportunitySetupFailed(run runRecord) string {
	if len(run.Deliveries) == 0 {
		return "the client's callback was given no notification, so it had nothing it could have " +
			"acknowledged"
	}
	published := 0
	for _, r := range run.Requests {
		if r.Service == "PublishRequest" {
			published++
		}
	}
	if published < 2 {
		return "the client was given a notification and then sent no further Publish, so it was " +
			"never in a position to acknowledge it"
	}
	return ""
}
