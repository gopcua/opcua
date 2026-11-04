// After a reconnect, a client re-activates its Session and creates a new one
// only if that fails. OPC 10000-4 §6.7, conformance unit 669: "The Client
// shall only create a new Session if ActivateSession fails."
//
// Whether the data those subscriptions carry comes back is
// data_resumes_after_reconnect_test.go, the same clause sentence scored on
// the application's own callback rather than on the wire. This row reads the
// wire because the callback cannot separate these clients: one that creates
// a new Session anyway still recovers data, having abandoned the
// subscriptions the old Session held, and that row would pass it. Whether
// the client waits for its own ActivateSession before using the Session on a
// new connection is session_unused_before_activation_test.go: a different
// clause, §5.7.2.1, and a different reading of the same requests, their
// order rather than whether a CreateSession is among them at all.
//
// The scripted server is the default one and grants every ActivateSession, so
// nothing in the fixture pushes a client into creating a Session. The test
// asserts that no CreateSession arrives on a connection opened after the
// drop. See harness.go.
//
// The clause's other half is not measured here. No runOptions field answers
// an ActivateSession with anything but a grant, and a run in which the
// fixture did not grant one is declined by setupFailed rather than scored, so
// the case the clause permits — a new Session after a refused activation —
// never arises.

package reconnect

import "testing"

func TestNewSessionOnlyAfterFailedActivation(t *testing.T) {
	run := runScenario(runOptions{})

	if why := setupFailed(run); why != "" {
		t.Fatalf("could not set the test up: %s", why)
	}

	// A client that never re-activated was never offered the old Session's
	// subscriptions back, so an absence of CreateSession says nothing about
	// it. The decline is the last arm rather than a precondition because a
	// CreateSession after the drop is a violation whether or not the client
	// re-activated; setupFailed in harness.go has the rule.
	why := reconnectSetupFailed(run)

	created := createSessionsAfterDrop(run)

	switch {
	case created > 0:
		t.Errorf("the client sent %d CreateSession request(s) on connection(s) opened after the drop, "+
			"although the server accepted its ActivateSession", created)
	case why != "":
		t.Fatalf("could not set the test up: %s", why)
	}
}

// createSessionsAfterDrop counts the CreateSession requests the client sent on
// connections it opened after the drop. It does not filter on activation as
// the helpers in harness.go do: usesSession reads CreateSession as needing no
// Session, so the fixture answers one that arrives before its connection's
// ActivateSession instead of faulting it, and a client is not excused by
// having sent it early.
func createSessionsAfterDrop(run runRecord) int {
	n := 0
	for _, r := range afterDrop(run) {
		if r.Service == "CreateSessionRequest" {
			n++
		}
	}
	return n
}
