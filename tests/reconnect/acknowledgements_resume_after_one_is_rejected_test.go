//go:build conformance

// A client keeps acknowledging notifications after the server has rejected
// one acknowledgement. OPC 10000-4 §5.14.1.1: "Clients are required to
// acknowledge NotificationMessages as they are received if the Publish
// response parameter availableSequenceNumbers is not an empty array."
// Conformance unit 744/2284, Subscription Client Basic, ends at that rule,
// which is what this row scores. The requirement binds per notification
// received, not per acknowledgement served: §5.14.5.2 pairs the Publish
// response's results with the request's acknowledgements one to one, so one
// Bad_SequenceNumberUnknown answers one acknowledgement and says nothing
// about the notifications that follow, and the list stays non-empty for as
// long as the server keeps anything back.
//
// gopcua passes it: on main the client drops the rejected acknowledgement
// and keeps sending the later ones. The row scores
// https://github.com/gopcua/opcua/pull/899, which latches on one
// Bad_SequenceNumberUnknown and sends no acknowledgement at all while the
// latch stands, so no Good result ever comes back to clear the latch, and
// the client never acknowledges again.
//
// Its closest neighbour is
// no_acknowledgements_when_nothing_is_available_test.go, the only other row
// that reads acknowledgements and availableSequenceNumbers: that row scores
// a server whose empty list lifts the requirement altogether, this one a
// server that keeps advertising and rejects an acknowledgement anyway. The
// Republish rows around it score asking for what was lost, not
// acknowledging what arrived afterwards.
//
// The server is the default one, set to name every notification it sends in
// the response that carries it (AdvertisesSentSequence). It answers the
// first acknowledgement it ever answers with Bad_SequenceNumberUnknown and
// every later one with StatusOK (RejectsFirstAck). See harness.go.
//
// The row reads the acknowledgements on the wire, split into the one the
// server rejected and everything acknowledged after it, and requires the
// later half to cover the notifications the application received after the
// drop, all of them but the last few whose acknowledgements can still be in
// flight at the window's end: under AdvertisesSentSequence the server
// advertises every notification it sends, so a delivered one was advertised.
// The record keeps requests and not responses, so the row takes the rejected
// acknowledgement to be the first one sent after the drop: that is the one
// RejectsFirstAck rejects, and no response's Results are recorded to check
// it against. The row does not score how the client chose its
// acknowledgements, whether it retried the rejected one, or what it did
// about the numbers the server advertised but never sent, past requiring
// that something it acknowledged after the rejection was one it was given;
// on this server no Republish gets those back.
//
// open62541 v1.4.22 was read from source for this row, and it keeps
// acknowledging: ua_client_subscriptions.c queues an acknowledgement for a
// notification only when that response's own availableSequenceNumbers names
// it (:1335-1350), sends every queued acknowledgement once with the next
// Publish and frees it as it goes, never to be retried (:1062-1087), and
// never reads the Publish response's per-acknowledgement results at all, so
// one Bad_SequenceNumberUnknown costs that one notification its
// acknowledgement and nothing else. open62541 passes this row and takes no
// exception. node-opcua, Eclipse Milo and UA-.NETStandard were neither run
// here nor read in source.

package reconnect

import (
	"slices"
	"testing"
	"time"
)

func TestAcknowledgementsResumeAfterOneIsRejected(t *testing.T) {
	// minSettle is the window a client needs after the drop to reconnect,
	// re-activate and acknowledge again. Below it, a client that never
	// acknowledges again cannot be told from one that had no time to, so the
	// floor has to exceed the longest a client that does acknowledge spends
	// getting back to the stream. The acknowledgement itself is
	// sub-millisecond on loopback; the reconnect in front of it is not.
	// node-opcua spends about 1.2s of dead time after the drop before it is
	// back in the stream, measured on this fixture by
	// data_resumes_after_reconnect_test.go, and Eclipse Milo schedules its
	// first reactivation a whole second out. Two seconds clears both.
	const minSettle = 2 * time.Second

	// A tripwire, not a check: both sides are constants, so a settleDefault
	// shortened below minSettle fails the build here rather than a run.
	const _ = uint64(settleDefault - minSettle)

	// ackSlack is how many of the notifications delivered after the drop a
	// client may leave unacknowledged at the window's end: the last ones,
	// whose acknowledgements can still be in flight when the run stops. The
	// slack rests on two facts, one about clients and one about the
	// fixture: a conforming client keeps no more than two acknowledgements
	// in flight at the cut, and the one the server rejects is the
	// acknowledgement of a notification delivered before the drop, so the
	// rejection never consumes a settleable one. On main the client
	// acknowledged every one of the 24 notifications delivered after the
	// rejection, measured, so a slack of two passes main with two to
	// spare and forgives nothing a settled client owes.
	const ackSlack = 2

	run := runScenario(runOptions{
		AdvertisesSentSequence: true,
		RejectsFirstAck:        true,
	})

	if why := setupFailed(run); why != "" {
		t.Fatalf("could not set the test up: %s", why)
	}
	if why := rejectedAckSetupFailed(run); why != "" {
		t.Fatalf("could not set the test up: %s", why)
	}
	if why := nothingAdvertisedSetupFailed(run); why != "" {
		t.Fatalf("could not set the test up: %s", why)
	}

	acks := postDropAcks(run)
	rejected, later := acks[0], acks[1:]
	delivered := deliveredAfterDrop(run)

	// The setup check is an arm of the switch, not a Fatalf above it, so the
	// arms that read the client's acknowledgements run first: a client that
	// acknowledged nothing after the rejection is reported on that, whatever
	// this reason says. Only a run with nothing delivered and no
	// acknowledgement after the drop reaches the setup arm, and sitting above
	// the default one it is what stops a run that measured nothing from
	// passing.
	why := noDeliveryAfterRejectionSetupFailed(run)

	// covered records the notifications the server delivered after the drop
	// that the client acknowledged afterwards, one entry per distinct
	// sequence number, so a client that repeats one number does not multiply
	// its score.
	covered := map[uint32]bool{}
	for _, a := range later {
		if delivered[a.Seq] {
			covered[a.Seq] = true
		}
	}
	// deliveredSeqs walks the delivered numbers in order, so the messages
	// below name them in sequence order and the tail check sees the last
	// ones as the last.
	var deliveredSeqs []uint32
	for seq := range delivered {
		deliveredSeqs = append(deliveredSeqs, seq)
	}
	slices.Sort(deliveredSeqs)
	// acked lists the delivered numbers the client acknowledged.
	var acked []uint32
	for _, seq := range deliveredSeqs {
		if covered[seq] {
			acked = append(acked, seq)
		}
	}
	// The row forgives only the tail: all of the delivered numbers but the
	// last ackSlack of them have to be covered, because only those last
	// acknowledgements can still be in flight when the window closes.
	settleable := deliveredSeqs[:max(0, len(deliveredSeqs)-ackSlack)]
	unsettled := 0
	var settleableMissed []uint32
	for _, seq := range settleable {
		if !covered[seq] {
			unsettled++
			settleableMissed = append(settleableMissed, seq)
		}
	}

	// len(acked) > 0 keeps a run that delivered no more than ackSlack
	// notifications from passing a client that acknowledged none of them:
	// with settleable empty, unsettled is zero whatever the client did.
	switch {
	case len(later) > 0 && unsettled == 0 && len(acked) > 0:
		t.Logf("after the server rejected the acknowledgement of notification %d, the client went "+
			"on acknowledging what the server advertised and delivered: %v", rejected.Seq, acked)
	case len(later) > 0:
		t.Errorf("after the server rejected the acknowledgement of notification %d, the client "+
			"acknowledged %v but never %v: it stopped short of the notifications the server "+
			"delivered and advertised after the rejection", rejected.Seq, acked, settleableMissed)
	case why != "":
		t.Fatalf("could not set the test up: %s", why)
	default:
		t.Errorf("the server rejected the client's first acknowledgement with "+
			"Bad_SequenceNumberUnknown, still had %v held back and went on delivering notifications, "+
			"and the client never acknowledged anything again: the only acknowledgement it sent "+
			"after the drop on an activated connection was the one the server rejected", run.Offered)
	}
}

// postDropAcks is every acknowledgement the client put in a Publish on a
// connection it opened after the drop and had already activated there, in the
// order they went over the wire. The activation filter is
// republishSeqsAfterDrop's, for its reason: a Publish that arrives before its
// connection's ActivateSession is faulted Bad_SessionNotActivated, and a
// fault carries no results, so the acknowledgements in it were answered with
// nothing at all and were neither rejected nor accepted.
//
// The first of them is the one the server rejects. Every acknowledgement the
// client sent before the drop rode the held Publish, which the server answers
// with nothing for as long as it holds it, and the run's only answered
// pre-drop Publish is the stream's first, which carries none: the client had
// received nothing to acknowledge before it. So the first acknowledgement
// answered in the run is the first one sent after the drop on an activated
// connection, and that is the one RejectsFirstAck answers
// Bad_SequenceNumberUnknown.
func postDropAcks(run runRecord) []ack {
	activated := map[int]bool{}
	var out []ack
	for _, r := range run.Requests {
		switch {
		case r.Service == "ActivateSessionRequest":
			activated[r.Conn] = true
		case r.Service == "PublishRequest" && r.Conn > run.DropConn && activated[r.Conn]:
			out = append(out, r.Acks...)
		}
	}
	return out
}

// rejectedAckSetupFailed reports why the server never rejected an
// acknowledgement, or "" if it did. A client that sent no acknowledgement
// after the drop on a connection it had activated was never answered one --
// the pre-drop acknowledgements are all withheld with the held Publish, see
// postDropAcks -- so the first acknowledgement it was set to reject never
// existed, and everything the run went on to do or not do about
// acknowledgements is beside the point.
func rejectedAckSetupFailed(run runRecord) string {
	if len(postDropAcks(run)) == 0 {
		return "the client sent no acknowledgement after the drop on a connection it had " +
			"activated, so the server never answered one, and there was no first acknowledgement " +
			"to reject"
	}
	return ""
}

// nothingAdvertisedSetupFailed is a tripwire on the row's own options, not a
// reading of the client: the scored server names every notification it sends
// in the list that carries it (AdvertisesSentSequence) and always holds at
// least one notification back, so what it advertises cannot be empty while
// the fixture builds the list as the options describe. If it ever is, the
// server has told the client it keeps nothing, and a client that
// acknowledges nothing from then on is conforming, so the run scores nothing.
func nothingAdvertisedSetupFailed(run runRecord) string {
	if len(run.Offered) == 0 {
		return "the server advertised nothing as available for retransmission, and the clause " +
			"binds a client only while availableSequenceNumbers is not an empty array"
	}
	return ""
}

// deliveredAfterDrop is the sequence numbers the application's callback
// received after the drop. What arrived before it is beside the point here:
// the rejection rides the first Publish answered after the drop, and that
// response carries the first notification the client is given after it, so a
// notification delivered after the drop is one delivered at the rejection or
// later.
func deliveredAfterDrop(run runRecord) map[uint32]bool {
	out := map[uint32]bool{}
	for _, d := range run.Deliveries {
		if d.At.After(run.DropAt) {
			out[d.Seq] = true
		}
	}
	return out
}

// noDeliveryAfterRejectionSetupFailed reports why a client that stopped
// acknowledging after the rejection would have nothing to be wrong about, or
// "" if there is something. The server has to have delivered a notification
// after the drop: a client given nothing after the rejection acknowledges
// nothing conformantly, whatever its reason. Every notification delivered
// after the drop was advertised in the response that carried it, which is
// AdvertisesSentSequence's doing; the advertisement itself is
// nothingAdvertisedSetupFailed's check, up with the other preconditions.
func noDeliveryAfterRejectionSetupFailed(run runRecord) string {
	if len(deliveredAfterDrop(run)) == 0 {
		return "the server delivered no notification after the drop, so there was nothing later " +
			"for the client to acknowledge"
	}
	return ""
}
