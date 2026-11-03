// After the transport drops and the session is re-activated, a client whose
// subscription survived on the server goes on receiving notifications on it.
// OPC 10000-4 §6.7: "The Client shall only create a new Session if
// ActivateSession fails." Re-activation assigns the new channel to the
// existing session, so the client keeps the subscriptions that session holds
// and the server keeps publishing to it.
//
// The scripted server is that server: the subscription outlives the drop and
// the re-activation. The server holds one notification back and drops the
// connection underneath it, then answers every later Publish with fresh data.
// See harness.go.
//
// The test checks only that data arrives after the drop. Whether the client
// republishes, transfers the subscription or recreates it is not the test's
// business. The server answers Republish with Bad_MessageNotAvailable, which
// ends a Republish loop, so a client that does ask is not faulted for it.

package reconnect

import "testing"

func TestDataResumesAfterReconnect(t *testing.T) {
	// knownBug records that gopcua does not do this yet; the text is logged in
	// place of a failure. README.md says when to delete it.
	const knownBug = "the client reconnects and re-activates the session, but never receives another notification"

	// minFresh is how many distinct notifications after the drop show a
	// resumed stream: one that arrives and then stalls is not a stream. Half
	// of what the settle window holds at publishPacing.
	const minFresh = int(settleDefault/publishPacing) / 2

	// A tripwire, not a check: both sides are constants, so a settleDefault
	// or publishPacing leaving minFresh below 2 fails the build, not a run.
	const _ = uint64(minFresh - 2)

	run := runScenario(runOptions{})

	if why := setupFailed(run); why != "" {
		t.Fatalf("could not set the test up: %s", why)
	}
	if why := reconnectSetupFailed(run); why != "" {
		t.Fatalf("could not set the test up: %s", why)
	}

	// The server numbers notifications above the held one only after the drop,
	// and LastSeq is the highest it created. A notification inside that range
	// is data the server sent after the drop; the value carries the number, so
	// distinct numbers are distinct values.
	fresh := map[uint32]bool{}
	for _, d := range run.Deliveries {
		if d.At.After(run.DropAt) && d.Seq > run.HeldSeq && d.Seq <= run.LastSeq {
			fresh[d.Seq] = true
		}
	}

	switch {
	case len(fresh) >= minFresh && knownBug != "":
		t.Errorf("this now passes (%d distinct notifications arrived after the drop) - delete the "+
			"knownBug constant and the two arms that read it, in the commit that fixed it", len(fresh))
	case len(fresh) >= minFresh:
		t.Logf("data kept flowing: %d distinct notifications arrived after the drop", len(fresh))
	case knownBug != "":
		t.Logf("known bug: %s (%d distinct notifications after the drop, at least %d expected; %d received in all)",
			knownBug, len(fresh), minFresh, len(run.Deliveries))
	default:
		t.Errorf("a session was activated after the drop, but only %d distinct notifications "+
			"arrived after it, where at least %d are expected (%d received in all)", len(fresh), minFresh, len(run.Deliveries))
	}
}
