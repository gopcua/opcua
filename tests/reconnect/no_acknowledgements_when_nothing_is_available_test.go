// A client does not acknowledge sequence numbers to a server that keeps no
// retransmission queue. OPC 10000-4 §5.14.5.2, on the Publish response's
// availableSequenceNumbers: "The list shall be empty if the Server does not
// support the retransmission queue. If the list is empty, the Client should
// not acknowledge sequence numbers."
//
// Conformance unit 744 / 2284, Subscription Client Basic, reaches the rule
// this one is the exception to, and stops there: "Acknowledge received
// Notifications with subsequent Publish requests." The unit says nothing
// about the available list, so a client that implements the unit and nothing
// else acknowledges every notification it is given and fails this row. The
// exception is the clause's alone, and the two halves pull in opposite
// directions, which is why this row measures both of them.
//
// §5.14.7.1 states the rule for the other place a server hands over an
// available list, and it is not measured here. After a transfer, "The Client
// should acknowledge all Messages in this list for which it will not request
// retransmission". The fixture cannot construct that: the scripted server
// keeps the Session alive across the drop and accepts the re-activation, so a
// conforming client never creates a new Session and never transfers anything.
// No run of this package, under any option, has a TransferSubscriptions in
// its request log; GATE-MEASURE-report.md records the same of node-opcua on
// this fixture. Reaching §5.14.7.1 needs a server that refuses to re-activate
// the surviving Session, and that is the stimulus
// new_session_only_after_failed_activation_test.go exists to keep out of the
// default fixture. Named and left.
//
// The row runs the scripted server twice, because one run cannot tell the
// behaviour from its cheap imitation. A client that never acknowledges
// anything passes the clause for the wrong reason, and on a fixture that
// advertises nothing the two are the same client.
//
//   - The control run is a server that keeps the queue and says so, with
//     AdvertisesSentSequence: every notification it sends is in that
//     response's own availableSequenceNumbers, as §5.14.5.2 requires of such a
//     server. A client has to acknowledge there — that is the conformance unit
//     above — and one that does not has not been measured against the clause,
//     it has only been found silent. That run is a precondition, not a score.
//   - The scored run is a server that keeps no queue, with
//     NoRetransmissionQueue: every PublishResponse and every TransferResult
//     carries an empty list. Nothing the client is given there may come back
//     as an acknowledgement.
//
// So the row asserts an outcome on the wire and not a mechanism: what the
// client acknowledged, under a server that told it there is nothing to
// acknowledge. A client passes it by reading the list, and only by reading
// the list, because the control run has already shown it acknowledges when
// the list names something.
//
// The two declines are different things and are split. A client that
// acknowledged nothing in the control run is declined with a t.Fatalf up with
// the preconditions: the run says nothing about the clause. A client that was
// never given a notification, or that was given one and then sent no Publish
// at all, is declined the same way, because it was never in a position to
// acknowledge and "did not" cannot be read off "could not". Neither of those
// is gopcua's defect, so neither goes through the knownBug arm; the defect
// this row logs is the opposite one, a client that acknowledges when it has
// been told there is nothing to acknowledge.
//
// The reference stacks, and the exception this row takes. One of the two that
// can be read here passes it and one fails it, so the row names the stack it
// fails rather than claiming every client should pass:
//
//   - open62541 v1.4.22 passes it, and passes it the clause's own way.
//     ua_client_subscriptions.c queues an acknowledgement only for a sequence
//     number it finds in that response's availableSequenceNumbers, and skips
//     it otherwise, so an empty list produces no acknowledgement at all. It is
//     also why the control run has to advertise: against a fixture that
//     advertises nothing, open62541 acknowledges nothing in either run and the
//     row would be declining a client that earned the pass.
//   - node-opcua 2.187.19 fails it, measured here: it acknowledged the same
//     31 sequence numbers in the same order in both runs, so what the server
//     said it keeps made no difference to it. client_publish_engine.js reads
//     the available list only in a commented-out line.
//   - Eclipse Milo and UA-.NETStandard cannot be run here and were not read
//     for this row.
//
// The clause is a "should" and one of the two stacks read ignores it, so this
// is not a row every client passes. It is kept because the clause is explicit,
// because open62541 shows the behaviour is ordinary rather than exotic, and
// because the acknowledgement is not free: it tells a server that does keep a
// queue that it may drop the message, and a client that sends it blindly is
// one configuration change away from throwing away the only copy of a
// notification it has not processed.
package reconnect

import "testing"

func TestNoAcknowledgementsWhenNothingIsAvailable(t *testing.T) {
	// knownBug records that gopcua does not do this yet; the text is logged in
	// place of a failure. README.md says when to delete it.
	const knownBug = "the client acknowledges every notification it is given and never reads the server's availableSequenceNumbers, so it acknowledges to a server that has just said it keeps nothing"

	// The control run: the server keeps the queue and names every notification
	// it sends in that response's own list, so a client that reads the list
	// has something to acknowledge. See the file's opening comment for why the
	// scored run alone would not be evidence.
	control := runScenario(runOptions{AdvertisesSentSequence: true})

	if why := setupFailed(control); why != "" {
		t.Fatalf("could not set the test up: %s", why)
	}
	if why := acknowledgingClientSetupFailed(control); why != "" {
		t.Fatalf("could not set the test up: %s", why)
	}

	// The scored run: the server keeps no retransmission queue and says so on
	// every answer.
	run := runScenario(runOptions{NoRetransmissionQueue: true})

	if why := setupFailed(run); why != "" {
		t.Fatalf("could not set the test up: %s", why)
	}
	// A tripwire on the option rather than a reading of the client: the whole
	// row rests on the server having advertised nothing, and a fixture that
	// advertised something would leave the row scoring a client the clause
	// does not speak to.
	if len(run.Offered) != 0 {
		t.Fatalf("could not set the test up: the server advertised %v as available for "+
			"retransmission, and the clause is about a server that advertises nothing",
			run.Offered)
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
	case len(acked) > 0:
		t.Errorf("the server said on every answer that it keeps no retransmission queue, and the "+
			"client acknowledged %v anyway", acked)
	default:
		t.Logf("the client acknowledged nothing to a server that advertised nothing, having "+
			"acknowledged %v to one that advertised what it sent",
			acknowledgementsSent(control))
	}
}

// acknowledgementsSent is every acknowledgement the client put in a Publish,
// in the order they went over the wire. A request that is not a Publish
// carries none, so the service name does not need checking here.
func acknowledgementsSent(run runRecord) []ack {
	var out []ack
	for _, r := range run.Requests {
		out = append(out, r.Acks...)
	}
	return out
}

// acknowledgingClientSetupFailed reports why the control run says nothing
// about the clause, or "" if it says what the scored run needs. The server in
// that run names every notification it sends in the list it sends it with, so
// a client that acknowledges what the server still has acknowledges there. One
// that does not acknowledge there has not been shown to read the list; it has
// been shown to be silent, and silence in the scored run would then be worth
// nothing.
//
// What it checks is that something the client was given came back as an
// acknowledgement. Acknowledging a number it was never given would be a
// different defect and is not this row's.
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

// ackOpportunitySetupFailed reports why the client was never in a position to
// acknowledge anything in the scored run, or "" if it was. Acknowledging
// nothing is only evidence where there was something to acknowledge and a
// request to put it in.
//
// Two Publish requests are enough to establish the second. The fixture answers
// the first one of the stream and holds the next, and runScenario refuses any
// run in which more than one notification was withheld — which is to say, any
// run in which the client had more than one Publish outstanding. So in a run
// that got this far the client sent its second Publish after it had been
// answered and given the first notification, and that Publish is where the
// acknowledgement would have been.
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
