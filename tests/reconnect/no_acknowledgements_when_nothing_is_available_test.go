// A client does not acknowledge sequence numbers to a server that keeps no
// retransmission queue. OPC 10000-4 §5.14.5.2, on the Publish response's
// availableSequenceNumbers: "The list shall be empty if the Server does not
// support the retransmission queue. If the list is empty, the Client should
// not acknowledge sequence numbers." Conformance unit 744/2284, Subscription
// Client Basic, stops at the rule this clause is the exception to —
// "Acknowledge received Notifications with subsequent Publish requests" — so
// a client that implements the unit and nothing else violates this clause;
// while knownBug stands this row logs such a client green beside gopcua
// rather than failing it.
//
// No other row here reads acknowledgements, and no other row is on
// §5.14.5.2, so this one has no neighbouring row.
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
// so the row names it. open62541 v1.4.22 passes it the clause's own way:
// ua_client_subscriptions.c queues an acknowledgement only for a sequence
// number in that response's own availableSequenceNumbers. node-opcua
// 2.187.19 acknowledges anyway, measured here: client_publish_engine.js
// reads the available list only in a commented-out line. While knownBug
// stands that is the same green log gopcua earns, so the row cannot tell
// them apart; it becomes the row's declared exception once knownBug is
// deleted. Eclipse Milo and UA-.NETStandard were not read.

package reconnect

import (
	"fmt"
	"testing"
)

func TestNoAcknowledgementsWhenNothingIsAvailable(t *testing.T) {
	// knownBug records that gopcua does not do this yet; the text is logged in
	// place of a failure. README.md says when to delete it. It carries no
	// issue link because it is not the reconnect defect the other knownBug
	// rows link.
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

	acked := acknowledgementsSent(run)

	switch {
	case len(acked) > 0 && knownBug != "":
		t.Logf("known bug: %s (it acknowledged %v, and the server said on every answer that it "+
			"keeps nothing)", knownBug, acked)
	case len(acked) == 0 && knownBug != "":
		t.Errorf("this now passes (the client acknowledged nothing to a server that advertised "+
			"nothing, having acknowledged %v to one that advertised what it sent) - delete the "+
			"knownBug constant and every arm that reads it, in the commit that fixed it",
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
// deliberately not filtered on activation, as republishesAfterDrop keeps
// total unfiltered and subscriptionAfterInvalidID keeps sent: the claim is
// about the wire, and an acknowledgement in a request the server faulted
// still went over it.
//
// Its two callers take that differently. At the switch, over-counting can
// only block this row's green arm, never produce it. In
// acknowledgingClientSetupFailed it loosens the precondition instead, since
// one more acknowledgement is one more chance to match a delivery. No client
// measured on this fixture sends one before its own connection's
// ActivateSession, so the looser reading has not yet differed from the
// stricter one, and both runs are read by the same rule either way.
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
// clause speaks to, or "" if it is. This is a tripwire on the row's own
// NoRetransmissionQueue rather than a reading of the client, and cannot fire
// while availableLocked returns the empty list unconditionally for that
// option: the row rests on the server having advertised nothing, and a run
// where it advertised something scores a client against a clause about a
// server that advertises nothing.
func emptyAvailableSetupFailed(run runRecord) string {
	if len(run.Offered) != 0 {
		return fmt.Sprintf("the server advertised %v as available for retransmission, and the "+
			"clause is about a server that advertises nothing", run.Offered)
	}
	return ""
}
