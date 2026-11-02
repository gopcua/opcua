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
	run := runScenario(runOptions{})

	if why := setupFailed(run); why != "" {
		t.Fatalf("could not set the test up: %s", why)
	}

	why := reconnectSetupFailed(run)

	created := createSessionsAfterDrop(run)

	switch {
	case created > 0:
		t.Errorf("the client created a new Session on %d connection(s) opened after the drop, "+
			"although the server accepted its ActivateSession", created)
	case why != "":
		t.Fatalf("could not set the test up: %s", why)
	}
}

// createSessionsAfterDrop counts the CreateSession requests the client sent on
// connections it opened after the drop.
func createSessionsAfterDrop(run runRecord) int {
	n := 0
	for _, r := range afterDrop(run) {
		if r.Service == "CreateSessionRequest" {
			n++
		}
	}
	return n
}
