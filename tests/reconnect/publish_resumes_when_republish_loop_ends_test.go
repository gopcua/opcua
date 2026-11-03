// Once the Republish loop has been told the server has no more, the client
// goes back to polling. OPC 10000-4 §6.7: "After re-establishing the
// connection the Client shall call Republish in a loop, starting with the
// next expected sequence number and incrementing the sequence number until
// the Server returns the status Bad_MessageNotAvailable. After receiving this
// status, the Client shall start sending Publish requests with the normal
// Publish handling." Conformance unit 276 names Republish and not the
// terminator, so the second sentence is the clause's alone.
//
// Whether the client asks at all is republish_after_reconnect_test.go, and
// whether it walks the whole gap is republish_asks_for_each_missed_number_test.go.
// This file is about what happens after the walk, because a client that
// recovers every notification the server kept and then never polls again has
// satisfied both of those rows and delivers nothing from that point on.
//
// The row asserts the outcome and not the reasoning: after the last Republish
// the client sent on a post-drop connection it had activated, at least one
// Publish follows on that same connection. Nothing is said about how the
// client decides to resume, or how soon.
//
// The record keeps requests and not responses, so whether the loop was told
// to stop is read off the rule the server answers by, which is the workaround
// missed_notification_recovered_when_transfer_refused_test.go uses for the
// same reason: runRecord.Answerable is every number the server would have
// handed back, and a Republish for anything else was faulted with
// Bad_MessageNotAvailable. Two kinds of run are therefore not evidence, and
// the row declines them in two different ways, because they are two different
// things.
//
// A run where no Republish was sent after the drop at all is gopcua's known
// bug: the client reconnects, re-activates, and never asks for anything it
// missed, so there is no loop to end and nothing to resume from. That is the
// same defect the other rows log, and it goes through the knownBug arm below
// with them.
//
// A run whose last ask was for a number the server had is not a bug. The loop
// stopped for a reason of the client's own — the gap filled, or
// availableSequenceNumbers ran out — and the terminator was never put to it.
// That is UA-.NETStandard's reactive shape, which walks from the last number
// it received to the newly received one and stops without ever meeting the
// fault, and which republish_asks_for_each_missed_number_test.go deliberately
// passes. Logging it as a known bug would say gopcua has a defect when what
// happened is that a conforming client took a route this row cannot score, so
// it is a plain setup failure, up with the preconditions.
//
// The scripted server is republish_asks_for_each_missed_number_test.go's: the
// transfer-refusing one, which keeps the retransmission queue OPC 10000-4
// §5.14.1.1 describes, with furtherLost more notifications behind the one it
// held back. A client that walks that queue runs off the end of it, and the
// end of it is the stimulus this row needs. See harness.go.
//
// The reference stacks, as measured and as read:
//
//   - node-opcua 2.187.19 passes it, measured against this fixture under
//     RefusesTransfer in GATE-MEASURE-report.md before furtherLost existed:
//     its reconnected connection reads ActivateSession, Republish(2),
//     Republish(3), Publish x35, where the server had only 2 to give back. It
//     ran off the end of the queue and went back to polling.
//   - open62541 v1.4.22 is declined, not failed. Measured on the same
//     fixture, its reconnected connection reads ActivateSession, Publish x60:
//     it has no client-side Republish at all, so no loop of its own ever ends.
//     The row that reddens it is republish_after_reconnect_test.go, which
//     takes that exception explicitly.
//   - Eclipse Milo and UA-.NETStandard are source reads only; neither can be
//     run here. Milo blocks its Publish pipeline until the §6.7 loop drains
//     and resumes after it, which is this row's shape; UA-.NETStandard's gap
//     walk is the declined shape above.
//
// A client cannot reach the passing branch by resuming the stream without
// asking, because then there is no loop for the row to find the end of, and
// the run is declined rather than passed.
package reconnect

import (
	"fmt"
	"testing"
	"time"
)

func TestPublishResumesWhenRepublishLoopEnds(t *testing.T) {
	// knownBug records that gopcua does not do this yet; the text is logged in
	// place of a failure. README.md says when to delete it.
	const knownBug = "the client reconnects and re-activates the session, but never asks for anything it missed, so there is no loop of its own to end and nothing to resume from"

	// furtherLost is how many notifications the server keeps for
	// retransmission behind the one it held back. Any number above zero gives
	// the loop a range to walk; what the row needs is that the walk ends on a
	// number the server does not have, and the queue is finite whatever its
	// length. Two matches the loop row so that both run the same server.
	const furtherLost = 2

	// settleLoop replaces settleDefault for this run. The window has to hold
	// the whole loop and leave room after it: a client paces its Republish —
	// gopcua's own loop sleeps a second between asks — and a window that
	// closed on the last ask would look like a client that stopped polling.
	// minLoopWindow is what the pacing costs, and the window is twice it, so
	// what is left after the loop is at least as long as the loop itself.
	const minLoopWindow = time.Duration(furtherLost+2) * time.Second
	const settleLoop = 10 * time.Second

	// A tripwire, not a check: both sides are constants, so a settleLoop
	// shortened below twice minLoopWindow fails the build, not a run.
	const _ = uint64(settleLoop - 2*minLoopWindow)

	run := runScenario(runOptions{RefusesTransfer: true, FurtherLost: furtherLost, Settle: settleLoop})

	if why := setupFailed(run); why != "" {
		t.Fatalf("could not set the test up: %s", why)
	}
	if why := clientSilentSetupFailed(run); why != "" {
		t.Fatalf("could not set the test up: %s", why)
	}
	// A client that never re-activated was never offered the subscription
	// back, so it was never in a position to ask for anything on it.
	if why := reconnectSetupFailed(run); why != "" {
		t.Fatalf("could not set the test up: %s", why)
	}
	// A client whose loop ended on its own terms was never told to stop, and
	// that is a route a conforming client takes rather than a defect, so it is
	// a setup failure here and not something the knownBug arm excuses.
	if why := loopEndSetupFailed(run); why != "" {
		t.Fatalf("could not set the test up: %s", why)
	}

	// Having no loop at all is the known bug, so it goes through the switch.
	why := loopRanSetupFailed(run)

	lastAsk, published, sent, _ := afterLastRepublish(run)

	switch {
	case why != "" && knownBug != "":
		t.Logf("known bug: %s (%s; the server had %v to give back)", knownBug, why, run.Answerable)
	case why != "":
		t.Fatalf("could not set the test up: %s", why)
	case published > 0 && knownBug != "":
		t.Errorf("this now passes (the client's last ask was for notification %d, which the server "+
			"answers with Bad_MessageNotAvailable, and %d Publish followed it on the same "+
			"connection) - delete the knownBug constant and the two arms that read it, in the "+
			"commit that fixed it", lastAsk, published)
	case published > 0:
		t.Logf("the client's last ask was for notification %d, which the server answers with "+
			"Bad_MessageNotAvailable, and %d Publish followed it on the same connection "+
			"(the server had %v to give back)", lastAsk, published, run.Answerable)
	default:
		t.Errorf("the client's last ask was for notification %d, which the server answers with "+
			"Bad_MessageNotAvailable, so its Republish loop was over, and it sent no Publish after "+
			"it on that connection (%d requests after it, none of them a Publish; the server had "+
			"%v to give back)", lastAsk, sent, run.Answerable)
	}
}

// afterLastRepublish reads what the client did once it stopped asking: the
// sequence number of the last Republish it sent on a post-drop connection it
// had already activated, how many Publish requests it sent on that same
// connection afterwards, and how many requests of any kind it sent there
// afterwards. asked is false when it sent no such Republish at all, and the
// other three are meaningless then.
//
// The activation rule is republishSeqsAfterDrop's: a Republish that arrives
// before its connection's ActivateSession is faulted with
// Bad_SessionNotActivated rather than answered by the retransmission rule, so
// it is not part of the loop. Counting only the connection the loop ended on
// is the strict reading: a client that dropped that connection itself and
// resumed on a later one reads as having published nothing. The fixture drops
// once and refuses no reconnect in this scenario, so a further connection
// would be the client's own doing.
func afterLastRepublish(run runRecord) (lastAsk uint32, published, sent int, asked bool) {
	activated := map[int]bool{}
	last, conn := -1, 0
	for i, r := range run.Requests {
		switch {
		case r.Service == "ActivateSessionRequest":
			activated[r.Conn] = true
		case r.Service == "RepublishRequest" && r.Conn > run.DropConn && activated[r.Conn]:
			last, conn, lastAsk = i, r.Conn, r.RepublishSeq
		}
	}
	if last < 0 {
		return 0, 0, 0, false
	}
	for _, r := range run.Requests[last+1:] {
		if r.Conn != conn {
			continue
		}
		sent++
		if r.Service == "PublishRequest" {
			published++
		}
	}
	return lastAsk, published, sent, true
}

// loopEndSetupFailed reports why the loop that ran ended for a reason other
// than the one this row is about, or "" if it ended for that reason. A client
// that stopped while the server still had the number it asked for stopped on
// its own terms and was never returned Bad_MessageNotAvailable; the file's
// opening comment says which clients do that and why it is not a defect. The
// sequence numbers here are the server's own, so the reason names what the
// server had rather than what the client was handed.
//
// A run with no Republish at all is loopRanSetupFailed's to report, so this
// one says nothing about it.
func loopEndSetupFailed(run runRecord) string {
	lastAsk, _, _, asked := afterLastRepublish(run)
	if !asked {
		return ""
	}
	for _, seq := range run.Answerable {
		if seq == lastAsk {
			return fmt.Sprintf("the client's last ask was for notification %d, which the server "+
				"answers with the notification itself, so the loop stopped for a reason of the "+
				"client's own and the server never returned Bad_MessageNotAvailable to end it; "+
				"the server had %v to give back", lastAsk, run.Answerable)
		}
	}
	return ""
}
