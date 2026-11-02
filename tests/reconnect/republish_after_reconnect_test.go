// After reconnecting, a client that missed a notification must ask the
// server to resend it. OPC 10000-4 §6.7, conformance unit 276: "Evaluate the
// sequence number in Notifications to detect lost Notifications. Use
// Republish to request missing Notifications."
//
// The rule only binds once something has been lost, so the test loses
// something: the server holds one notification back and never sends it, then
// drops the connection. See harness.go.
//
// The test checks only that the client asked. How often it asks, and where it
// stops, is a separate behaviour with its own file. One Republish is the whole
// answer, so unlike the tests that count events across the settle window this
// one needs no minimum window length.
package reconnect

import "testing"

func TestRepublishAfterReconnect(t *testing.T) {
	// knownBug records that gopcua does not do this yet. While it is set, this
	// text is logged in place of a failure and the build stays green. README.md
	// says when to delete it. It is declared here because it names this file's
	// behaviour.
	const knownBug = "the client reconnects and re-activates the session, but never asks for the notification it missed"

	run := runScenario(scenarioOptions{})

	if why := setupFailed(run); why != "" {
		t.Fatalf("could not set the test up: %s", why)
	}
	if why := lossNotConstructed(run); why != "" {
		t.Fatalf("could not set the test up: %s", why)
	}
	t.Logf("the server held notification %d back and never sent it, then dropped the connection",
		run.HeldSeq)

	asked, total := republishesAfterDrop(run)

	switch {
	case asked > 0 && knownBug != "":
		t.Errorf("this now passes (%d Republish of %d requests after the reconnect) - "+
			"delete the knownBug constant in this file, in the commit that fixed it", asked, total)
	case asked > 0:
		t.Logf("the client asked for the missed notification (%d Republish of %d requests)", asked, total)
	case knownBug != "":
		t.Logf("known bug: %s (%d requests after the reconnect, none a Republish)", knownBug, total)
	default:
		t.Errorf("the client missed notification %d and never asked for it "+
			"(%d requests after the reconnect, none a Republish)", run.HeldSeq, total)
	}
}

// lossNotConstructed reports why nothing was lost after all, or "" if the
// loss was constructed. Until it is, the test has nothing to ask about.
func lossNotConstructed(run record) string {
	if run.HeldSeq == 0 {
		return "the server never held a notification back, so nothing was lost"
	}
	for _, d := range run.Deliveries {
		if d.Seq == run.HeldSeq && d.At.Before(run.DropAt) {
			return "the held notification reached the client before the drop, so nothing was lost"
		}
	}
	if _, total := republishesAfterDrop(run); total == 0 {
		return "the client sent nothing at all after the drop"
	}
	return ""
}

// republishesAfterDrop counts what the client sent on a connection it opened
// after the drop. The server numbers each connection it serves, so anything
// numbered 2 or higher arrived on a reconnect.
func republishesAfterDrop(run record) (asked, total int) {
	for _, r := range run.Requests {
		if r.Epoch < 2 {
			continue
		}
		total++
		if r.Service == "RepublishRequest" {
			asked++
		}
	}
	return asked, total
}
