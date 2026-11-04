// Once the server has returned Bad_MessageNotAvailable the client goes back
// to polling. OPC 10000-4 §6.7: "After receiving this status, the Client
// shall start sending Publish requests with the normal Publish handling."
// Conformance unit 276 names Republish and not the terminator, so this
// sentence is the clause's alone.
//
// Whether the client asks at all is republish_after_reconnect_test.go, and
// whether it walks the whole gap — with the loop sentence this one follows
// quoted in full — is republish_asks_for_each_missed_number_test.go.
//
// The scripted server is the transfer-refusing one of
// missed_notification_recovered_when_transfer_refused_test.go, here holding
// furtherLost more notifications behind the one it withheld, so a client
// that walks the queue runs off the end of it. See harness.go.
//
// After the last Republish the client sent on a post-drop connection it had
// activated, at least one Publish follows on that same connection. Whether
// the loop was told to stop is read off run.Answerable, because the record
// keeps requests and not responses — the limit
// missed_notification_recovered_when_transfer_refused_test.go also takes.
//
// open62541 has no client-side Republish, so no loop of its own ever ends.
// While knownBug stands that absence is logged green here beside gopcua and
// the row cannot tell the two apart; republish_after_reconnect_test.go is
// the row that will fail it once knownBug is deleted, and declares that
// exception in its own header. node-opcua runs off the end of the queue and
// resumes, measured on this fixture. Eclipse Milo, read in source and not
// run here, resumes after the loop drains; UA-.NETStandard, read the same
// way, walks only to the number it just received and is declined for
// stopping before the terminator; gopcua's own loop takes that route too
// once it republishes at all, because it stops at the top of
// availableSequenceNumbers when the server hands it a non-empty list. No stack here fails it -- open62541 is
// logged green rather than failed while knownBug stands, and UA-.NETStandard
// is declined rather than failed -- so the row takes no exception. A client
// cannot reach the passing branch by resuming the stream without asking. It
// can reach it without any post-terminator logic, though. The count is
// positional rather than temporal: afterLastRepublish counts the Publishes
// recorded after the last Republish's index, so a client whose publish loop
// runs concurrently with its Republish loop passes on that overlap alone.
// The row assumes a client that pauses publishing while it reconnects, which
// gopcua does: its publish loop stays paused until setState(ctx, Connected),
// after restoreSubscriptions.

package reconnect

import (
	"fmt"
	"testing"
	"time"
)

func TestPublishResumesWhenRepublishLoopEnds(t *testing.T) {
	// knownBug records that gopcua does not do this yet; the text is logged in
	// place of a failure. README.md says when to delete it.
	// https://github.com/gopcua/opcua/issues/879
	const knownBug = "the client reconnects and re-activates the session, but never asks for anything it missed, so there is no loop of its own to end and nothing to resume from"

	// furtherLost is how many notifications the server keeps for
	// retransmission behind the one it held back. Any positive length ends the
	// walk on a number the server does not have, which is all this row needs;
	// two is what republish_asks_for_each_missed_number_test.go needs, so both
	// rows run the same server.
	const furtherLost = 2

	// settleLoop replaces settleDefault for this run. The window has to hold
	// the whole loop and leave room after it: gopcua's own loop sleeps a
	// second between asks, and a window that closed on the last ask would
	// look like a client that stopped polling. minLoopWindow is what the
	// pacing costs, and the window is twice it, the rest being margin for a
	// loaded machine.
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

// afterLastRepublish reads what the client did once it stopped asking, on
// the post-drop connection its last activated Republish went out on: the
// number it asked for, how many Publish followed there, and how many
// requests of any kind did. asked is false when there was no such Republish,
// and the other three are meaningless then. The activation rule is
// republishSeqsAfterDrop's: a Republish arriving before its connection's
// ActivateSession is faulted with Bad_SessionNotActivated, so it is not part
// of the loop.
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
// opening comment names UA-.NETStandard, and gopcua once it republishes at
// all, as the stacks that take that route.
// The sequence numbers here are the server's own, so the reason names what
// the server had rather than what the client was handed.
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
