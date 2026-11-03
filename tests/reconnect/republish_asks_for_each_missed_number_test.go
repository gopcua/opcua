// After a reconnect, a client that missed notifications asks for them one
// after another until the server says it has no more. OPC 10000-4 §6.7:
// "After re-establishing the connection the Client shall call Republish in a
// loop, starting with the next expected sequence number and incrementing the
// sequence number until the Server returns the status
// Bad_MessageNotAvailable." Conformance unit 276: "Evaluate the sequence
// number in Notifications to detect lost Notifications. Use Republish to
// request missing Notifications."
//
// Whether the client asks at all is republish_after_reconnect_test.go. This
// file is about the asks after the first one, because a client that asks once
// and stops recovers one notification of however many it lost and looks the
// same as a conforming one on a server that had one to give. What the client
// does once the loop has ended — resume Publish with the normal handling — is
// a separate behaviour with its own file.
//
// The scripted server here is the one that keeps the retransmission queue OPC
// 10000-4 §5.14.1.1 describes, which is the transfer-refusing one; the other
// answers every Republish with Bad_MessageNotAvailable, and against it a loop
// of one and a loop of five leave the same trace. Besides the notification it
// holds back, the server queues furtherLost more as the drop is applied:
// responses it sent that the connection took with it, which is the case §6.7
// opens with — "When the connection is lost, Publish responses may have been
// sent but not received by the Client". So the client's next expected
// sequence number is the one the server held, the server answers that one and
// the furtherLost behind it, and faults the next. See harness.go.
//
// A conforming client asks for every one of them, in turn. Whether it then
// asks for the number that gets the fault is left open: the clause says to
// increment until the Server returns Bad_MessageNotAvailable, but a client
// that has been told by availableSequenceNumbers that nothing is left above
// the number it just recovered may stop there, and gopcua's own loop does
// exactly that when it is given the list. Both are accepted. Leaving behind a
// number the server still had is not, and nor is going on asking once the
// fault has come back.
//
// open62541 has no client-side Republish at all, so it never starts the loop
// and this row declines to score it rather than failing it: the row that
// reddens open62541 is republish_after_reconnect_test.go, which takes that
// exception explicitly. Measured, node-opcua walks the whole gap and stops on
// the fault, exactly as the pull request under review does. Eclipse Milo and
// UA-.NETStandard are source reads; neither can be run here.
//
// The test reads the sequence numbers that went over the wire: a client
// cannot produce them by resuming the stream, and one that stops while the
// server still has a number for it does not produce them either.
package reconnect

import (
	"fmt"
	"testing"
	"time"
)

func TestRepublishAsksForEachMissedNumber(t *testing.T) {
	// knownBug records that gopcua does not do this yet; the text is logged in
	// place of a failure. README.md says when to delete it.
	const knownBug = "the client reconnects and re-activates the session, but never asks for anything it missed, so there is no loop to walk"

	// furtherLost is how many notifications the server keeps for
	// retransmission behind the one it held back. Two means it can give back
	// three, so a client that asks once and a client that asks twice both
	// leave something behind and the row names the numbers they left. Nothing
	// in the row depends on the length: what it reads is whether the asks
	// cover what the server had.
	const furtherLost = 2

	// settleLoop replaces settleDefault for this run: the window has to hold
	// the whole loop, and a client paces its Republish — gopcua's own loop
	// sleeps a second between asks. minLoopWindow is what that pacing costs,
	// and the window is twice it, the rest being margin for a loaded machine.
	// A window that closed mid-loop would look like a client that gave up;
	// loopScorableSetupFailed declines those runs rather than scoring them,
	// and the margin is there to make them rare.
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

	// A loop the row cannot score is not a defect of gopcua's, so it is a
	// setup failure here rather than something the knownBug arm excuses.
	if why := loopScorableSetupFailed(run); why != "" {
		t.Fatalf("could not set the test up: %s", why)
	}

	// Having no loop at all is the known bug, so it goes through the switch.
	why := loopRanSetupFailed(run)

	asked := republishSeqsAfterDrop(run)

	// republishLoopWrong reads a loop whose first ask the server answered,
	// which loopScorableSetupFailed has established.
	var wrong string
	if why == "" {
		wrong = republishLoopWrong(asked, run.Answerable)
	}

	switch {
	case why != "" && knownBug != "":
		t.Logf("known bug: %s (%s; the server had %v to give back)", knownBug, why, run.Answerable)
	case why != "":
		t.Fatalf("could not set the test up: %s", why)
	case wrong == "" && knownBug != "":
		t.Errorf("this now passes (the client asked for %v, %d Republish in all, and the server had "+
			"%v to give back) - delete the knownBug constant and the two arms that read it, in the "+
			"commit that fixed it", asked, len(asked), run.Answerable)
	case wrong == "":
		t.Logf("the client walked the gap: it asked for %v, %d Republish in all, and the server had "+
			"%v to give back", asked, len(asked), run.Answerable)
	default:
		t.Errorf("%s", wrong)
	}
}

// loopScorableSetupFailed reports why a loop that ran could not be scored
// against §6.7, or "" if it could. None of these is gopcua's defect, so each
// is a plain setup failure rather than something the knownBug arm excuses; a
// run with no loop at all is loopRanSetupFailed's to report, and that one is
// the defect.
//
// The first check is a tripwire on the row's own FurtherLost rather than a
// reading of the client: the server's queue runs contiguously from the
// notification it held, so a run of this row always has FurtherLost+1
// numbers to give back.
//
// The client's first ask has to be the first of them, because that is the ask
// the server answers with a notification, and until one comes back nothing
// has told the client there might be more; a client that asks for something
// else is faulted and stops for a reason the clause allows, and
// republish_after_reconnect_test.go is the row that scores the number it
// asked for. And where the client has not yet asked for the last number the
// server has, the last thing it sent has to be something other than a
// Republish: a run whose last request is a Republish may have been cut off
// mid-loop when the settle window closed, and the record cannot tell that
// from a client that gave up.
func loopScorableSetupFailed(run runRecord) string {
	if len(run.Answerable) < 2 {
		return fmt.Sprintf("the server had %d notification(s) to give back, and a loop needs more "+
			"than one", len(run.Answerable))
	}
	asked := republishSeqsAfterDrop(run)
	if len(asked) == 0 {
		return ""
	}
	first := run.Answerable[0]
	if asked[0] != first {
		return fmt.Sprintf("the client's first ask was for notification %d, which the server does "+
			"not have, so it was faulted rather than answered and the client was never given a "+
			"reason to ask again; the first one it missed was %d", asked[0], first)
	}
	lastAnswerable := run.Answerable[len(run.Answerable)-1]
	if asked[len(asked)-1] < lastAnswerable && lastAfterDrop(run) == "RepublishRequest" {
		return fmt.Sprintf("the client had asked for %v, the server still had notifications up to "+
			"%d to give back, and the last thing it sent was another Republish, so the settle "+
			"window may have closed while it was still walking the gap rather than after it gave "+
			"up", asked, lastAnswerable)
	}
	return ""
}

// lastAfterDrop is the service name of the last request the client sent on a
// connection it opened after the drop, or "" if it sent none.
func lastAfterDrop(run runRecord) string {
	after := afterDrop(run)
	if len(after) == 0 {
		return ""
	}
	return after[len(after)-1].Service
}

// republishLoopWrong reports how the sequence numbers the client asked for
// fall short of the loop §6.7 describes, or "" if they do not. The loop starts
// at the first number the server can give back, which is the next one the
// client expects, and goes up by one until it has covered every number the
// server has. Asking for the one past them, the number that gets
// Bad_MessageNotAvailable, is allowed and so is stopping without it; asking
// for anything beyond that is not. The caller has established that the client
// asked for the first number and was answered.
func republishLoopWrong(asked, answerable []uint32) string {
	first := answerable[0]
	lastAnswerable := answerable[len(answerable)-1]
	// The first ask is the first answerable number, by the precondition above,
	// so the walk starts at the second.
	for i := 1; i < len(asked); i++ {
		if want := first + uint32(i); asked[i] != want {
			return fmt.Sprintf("the client asked for %v: after %d it asked for %d, where the next "+
				"number up is %d, so it is not incrementing", asked, asked[i-1], asked[i], want)
		}
	}
	switch last := asked[len(asked)-1]; {
	case last < lastAnswerable:
		return fmt.Sprintf("the client asked for %v and stopped at %d, and the server still had %v "+
			"to give back", asked, last, answerable[last-first+1:])
	case last > lastAnswerable+1:
		return fmt.Sprintf("the client asked for %v: the server returned Bad_MessageNotAvailable "+
			"for %d and it went on asking after that", asked, lastAnswerable+1)
	}
	return ""
}
