// A Session is not used until it is activated, on every connection. OPC
// 10000-4 §5.7.2.1: "The Session created with this Service shall not be used
// until the Client calls the ActivateSession Service and proves possession of
// its ApplicationInstanceCertificate and any user identity token that it
// provided."
//
// After a reconnect the client sends its first request for the Session on a
// new connection, and that connection's ActivateSession has to come before
// it. A client that sends a Publish first is faulted with
// Bad_SessionNotActivated and may recover and receive data, so a test that
// only watched for data would pass it. See harness.go.
//
// Which services use the Session is usesSession in harness.go, the same rule
// the scripted server faults on: CreateSession creates it, ActivateSession is
// what is being waited for, GetEndpoints and FindServers need no Session, and
// CloseSession is how a client abandons one it could not activate.

package reconnect

import "testing"

func TestSessionUnusedBeforeActivation(t *testing.T) {
	run := runScenario(runOptions{})

	if why := setupFailed(run); why != "" {
		t.Fatalf("could not set the test up: %s", why)
	}

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
