// A Session is not used until it is activated, on every connection. OPC
// 10000-4 §5.7.2.1: "The Session created with this Service shall not be used
// until the Client calls the ActivateSession Service and proves possession of
// its ApplicationInstanceCertificate and any user identity token that it
// provided."
//
// Whether the client re-activates the Session it has rather than creating a
// new one is new_session_only_after_failed_activation_test.go: a different
// clause, §6.7, and a different reading of the same requests, whether a
// CreateSession is among them at all rather than the order they arrive in.
// Whether the data comes back is data_resumes_after_reconnect_test.go, scored
// on the application's own callback rather than on the wire; a client that
// sends a Publish before its ActivateSession is faulted with
// Bad_SessionNotActivated and may recover and receive data anyway, so that
// row would pass it.
//
// The scripted server is the default one. It grants every ActivateSession.
// A session service that arrives on a connection before that connection's
// ActivateSession is faulted rather than answered. See harness.go.
//
// After a reconnect the client sends its first request for the Session on a
// new connection, and that connection's ActivateSession has to come before
// it. Which services use the Session is usesSession in harness.go, the same
// rule the scripted server faults on: CreateSession creates it,
// ActivateSession is what is being waited for, GetEndpoints and FindServers
// need no Session, and CloseSession is how a client abandons one it could
// not activate.
//
// The reference stacks. node-opcua 2.187.19 opens the connection after the
// drop with ActivateSession, measured. open62541 v1.4.22 was measured by
// request counts per connection, which do not settle an ordering; Eclipse
// Milo and UA-.NETStandard are unsettled. No stack here fails it: no
// exception.

package reconnect

import "testing"

func TestSessionUnusedBeforeActivation(t *testing.T) {
	run := runScenario(runOptions{})

	if why := setupFailed(run); why != "" {
		t.Fatalf("could not set the test up: %s", why)
	}

	// firstEarlyUse reads every connection and the decline below reads only
	// those opened after the drop, so a declining run can still hold a
	// violation on the connection before it: the decline is the last arm
	// rather than a precondition. setupFailed in harness.go has the rule.
	why := sessionUseSetupFailed(run)

	early, found := firstEarlyUse(run)

	switch {
	case found:
		t.Errorf("the client sent %s on connection %d before it activated the Session there",
			early.Service, early.Conn)
	case why != "":
		t.Fatalf("could not set the test up: %s", why)
	}
}

// firstEarlyUse returns the first request that used the Session on a
// connection before that connection's ActivateSession.
func firstEarlyUse(run runRecord) (requestRecord, bool) {
	activated := map[int]bool{}
	for _, r := range run.Requests {
		switch {
		case r.Service == "ActivateSessionRequest":
			activated[r.Conn] = true
		case usesSession(r.Service) && !activated[r.Conn]:
			return r, true
		}
	}
	return requestRecord{}, false
}

// sessionUseSetupFailed reports why this run could not have told a client
// that activates first from one that never used its Session again, or "" if
// it could. The client has to use its Session on a connection opened after
// the drop before an absence of early use means anything.
func sessionUseSetupFailed(run runRecord) string {
	for _, r := range afterDrop(run) {
		if usesSession(r.Service) {
			return ""
		}
	}
	return "the client sent nothing for its Session after the drop, so it was never " +
		"in a position to use it too early"
}
