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

// minFresh is how many distinct notifications after the drop show a resumed
// stream: half of what the settle window holds at publishPacing. One that
// arrives and then stalls is not a stream.
const minFresh = int(settleDefault/publishPacing) / 2

func TestDataResumesAfterReconnect(t *testing.T) {
	// knownBug records that gopcua does not do this yet. While it is set, this
	// text is logged in place of a failure and the build stays green. README.md
	// says when to delete it. It is declared here because it names this file's
	// behaviour.
	const knownBug = "the client reconnects and re-activates the session, but never receives another notification"

	if minFresh < 2 {
		t.Fatalf("could not set the test up: the settle window of %s holds only %d publish intervals "+
			"of %s, too few to tell a stream from a single notification. Lengthen settleDefault "+
			"or shorten publishPacing in harness.go", settleDefault, int(settleDefault/publishPacing), publishPacing)
	}

	run := runScenario(scenarioOptions{})

	if why := setupFailed(run); why != "" {
		t.Fatalf("could not set the test up: %s", why)
	}

	reactivated := false
	for _, r := range run.Requests {
		if r.Epoch >= 2 && r.Service == "ActivateSessionRequest" {
			reactivated = true
		}
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
	case !reactivated:
		// This arm fails even while knownBug is set, because knownBug names a
		// client that does re-activate. One that never does has a different
		// defect, which this file makes no claim about.
		t.Errorf("the client never re-activated its session after the drop, " +
			"so the server's subscription was never offered back to it")
	case len(fresh) >= minFresh && knownBug != "":
		t.Errorf("this now passes (%d distinct notifications arrived after the drop) - "+
			"delete the knownBug constant in this file, in the commit that fixed it", len(fresh))
	case len(fresh) >= minFresh:
		t.Logf("data kept flowing: %d distinct notifications arrived after the drop", len(fresh))
	case knownBug != "":
		t.Logf("known bug: %s (%d distinct notifications after the drop, at least %d expected; %d received in all)",
			knownBug, len(fresh), minFresh, len(run.Deliveries))
	default:
		t.Errorf("the session was re-activated after the drop, but only %d distinct notifications "+
			"arrived after it, where at least %d are expected (%d received in all)", len(fresh), minFresh, len(run.Deliveries))
	}
}
