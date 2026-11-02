// After a reconnect, a client re-activates its Session and creates a new one
// only if that fails. OPC 10000-4 §6.7, conformance unit 669: "The Client
// shall only create a new Session if ActivateSession fails."
//
// The scripted server accepts every ActivateSession, so the test asserts that
// no CreateSession arrives on a connection opened after the drop. A client
// that creates a new Session anyway still recovers data, but it abandons the
// subscriptions the old Session held, so a test that only watched for data
// would pass it. See harness.go.
package reconnect

import "testing"

func TestNewSessionOnlyAfterFailedActivation(t *testing.T) {
	run := runScenario(scenarioOptions{})

	if why := setupFailed(run); why != "" {
		t.Fatalf("could not set the test up: %s", why)
	}

	// Why this run could not have told a client that re-activates from one
	// that never tried, or "" if it could. A client that never reconnects
	// creates no Session either, and that is not evidence.
	whyNot := reactivationMeasurementFailed(run)

	created := createSessionsAfterDrop(run)

	switch {
	case created > 0:
		t.Errorf("the client created a new Session on %d connection(s) opened after the drop, "+
			"although the server accepted its ActivateSession", created)
	case whyNot != "":
		t.Fatalf("could not set the test up: %s", whyNot)
	}
}

// createSessionsAfterDrop counts the CreateSession requests the client sent on
// connections it opened after the drop.
func createSessionsAfterDrop(run record) int {
	n := 0
	for _, r := range run.Requests {
		if r.Epoch > run.DropEpoch && r.Service == "CreateSessionRequest" {
			n++
		}
	}
	return n
}

// reactivationMeasurementFailed reports why this run could not have told a
// client that keeps its Session from one that never reconnected, or "" if it
// could. The client has to open a connection after the drop and try to
// activate a Session on it before a missing CreateSession means anything.
func reactivationMeasurementFailed(run record) string {
	for _, r := range run.Requests {
		if r.Epoch > run.DropEpoch && r.Service == "ActivateSessionRequest" {
			return ""
		}
	}
	return "the client never tried to activate a Session after the drop, so it was never " +
		"in a position to keep the old one or replace it"
}
