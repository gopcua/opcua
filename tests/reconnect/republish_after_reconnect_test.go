// After reconnecting, a client that missed a notification must ask the
// server to resend it. OPC 10000-4 §6.7, conformance unit 276: "Evaluate the
// sequence number in Notifications to detect lost Notifications. Use
// Republish to request missing Notifications."
//
// The rule only binds once something has been lost, so the test loses
// something: the server holds one notification back and never sends it, then
// drops the connection. See harness.go. This server transfers the
// subscription when asked, which is what separates this row from the one
// that withholds the transfer: here a client has every route back to the
// notification, and the row asks only whether it takes one.
//
// The test checks only that the client asked, and for which sequence number.
// How often it asks, and where it stops, is a separate behaviour with its own
// file.
//
// A known-good client fails this: open62541 has no client-side Republish at
// all, so it resumes Publish and never asks for what it lost. The test is
// kept because node-opcua asks, measured against this fixture, and Eclipse
// Milo and UA-.NETStandard both ask too, read from source and not run here;
// and because it asserts an outcome a client cannot fake by resuming the
// stream.
package reconnect

import (
	"testing"
	"time"
)

func TestRepublishAfterReconnect(t *testing.T) {
	// knownBug records that gopcua does not do this yet; the text is logged in
	// place of a failure. README.md says when to delete it.
	const knownBug = "the client reconnects and re-activates the session, but never asks for the notification it missed"

	// minSettle is the window a client needs after the drop to reconnect,
	// re-activate and ask. The whole sequence measured under a millisecond on
	// loopback; the rest is margin for a loaded machine. Below it, a client
	// that never asks cannot be told from one that had no time to.
	const minSettle = 1 * time.Second

	if settleDefault < minSettle {
		t.Fatalf("could not set the test up: the settle window of %s is shorter than the %s a client "+
			"needs to reconnect and ask, so a client that never asks cannot be told from one that was "+
			"not given time. Lengthen settleDefault in harness.go", settleDefault, minSettle)
	}

	run := runScenario(runOptions{})

	if why := setupFailed(run); why != "" {
		t.Fatalf("could not set the test up: %s", why)
	}
	if why := clientSilentSetupFailed(run); why != "" {
		t.Fatalf("could not set the test up: %s", why)
	}

	// While knownBug stands the client is expected not to get far enough to be
	// asked, and the shortfall is logged with it; once someone deletes knownBug
	// to claim the fix, a run that measured nothing must not pass. That is why
	// the setup arm sits below the knownBug arm here and nowhere else.
	why := republishSetupFailed(run)

	asked, wrongSeqs, total := republishesAfterDrop(run)

	switch {
	case asked > 0 && knownBug != "":
		t.Errorf("this now passes (%d Republish for notification %d, of %d requests after the reconnect) - "+
			"delete the knownBug constant and the two arms that read it, in the commit that fixed it",
			asked, run.HeldSeq, total)
	case asked > 0:
		t.Logf("the client asked for the missed notification (%d Republish for notification %d, of %d requests)",
			asked, run.HeldSeq, total)
	case len(wrongSeqs) > 0:
		t.Errorf("the client asked the server to resend %v, and the notification it missed was %d: "+
			"it republished without evaluating the sequence number it lost", wrongSeqs, run.HeldSeq)
	case knownBug != "":
		t.Logf("known bug: %s (notification %d was held back; %d requests after the reconnect, "+
			"none a Republish)", knownBug, run.HeldSeq, total)
		if why != "" {
			t.Logf("this run could not have told asking from not asking anyway: %s", why)
		}
	case why != "":
		t.Fatalf("could not set the test up: %s", why)
	default:
		t.Errorf("the client missed notification %d and never asked for it "+
			"(%d requests after the reconnect, none a Republish)", run.HeldSeq, total)
	}
}

// republishSetupFailed reports why this run could not have told a client that
// declined to ask from one that was never asked, or "" if it could. A client
// has to poll again after the drop, and the server has to have created
// something after the notification it held, before a missing Republish means
// anything.
func republishSetupFailed(run runRecord) string {
	polled := false
	for _, r := range afterDrop(run) {
		if r.Service == "PublishRequest" {
			polled = true
		}
	}
	if !polled {
		return "the client never polled again after the drop, so it was never in a " +
			"position to notice the loss"
	}
	if run.LastSeq <= run.HeldSeq {
		return "the server created nothing after the notification it held, so there " +
			"was no gap for the client to notice"
	}
	return ""
}
