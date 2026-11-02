// After the transport drops and the session is re-activated, a client whose
// subscription survived on the server goes on receiving notifications on it.
// OPC 10000-4 §6.7: "The Client shall only create a new Session if
// ActivateSession fails." A client that re-activates the session it already
// has still owns the subscriptions that session holds, so the server keeps
// publishing to it.
//
// The scripted server is that server: the subscription outlives the drop and
// the re-activation. The server holds one notification back and drops the
// connection underneath it, then answers every later Publish with fresh data.
// See harness.go.
//
// The test checks only that data arrives after the drop. Whether the client
// republishes, transfers the subscription or recreates it is not the test's
// business. The server answers Republish with Bad_MessageNotAvailable, the
// answer of a server with nothing to retransmit, so a client that does ask
// is not faulted for it.
package reconnect

import (
	"testing"

	"github.com/gopcua/opcua/ua"
)

func TestDataResumesAfterReconnect(t *testing.T) {
	// knownBug records that gopcua does not do this yet. While it is set, the
	// failure is logged and the build stays green. README.md says when to delete
	// it. It is local because every test file in the package has its own.
	const knownBug = "the client reconnects and re-activates the session, but never receives another notification"

	run := runScenario(scenarioOptions{Override: republishNothingToSend})

	// A client cannot cause either of these.
	switch {
	case run.HarnessErr != "":
		t.Fatalf("could not set the test up: %s", run.HarnessErr)
	case !run.DropApplied:
		t.Fatalf("could not set the test up: the connection was never dropped")
	}

	// The server produces notifications numbered above the held one only
	// after the drop, so each of these is data the client could not have had.
	fresh := 0
	for _, d := range run.Deliveries {
		if d.At.After(run.DropAt) && d.Seq > run.HeldSeq {
			fresh++
		}
	}

	switch {
	case fresh > 0 && knownBug != "":
		t.Errorf("this now passes (%d notifications arrived after the drop) - "+
			"delete the knownBug constant in this file, in the commit that fixed it", fresh)
	case fresh > 0:
		t.Logf("data kept flowing: %d notifications arrived after the drop", fresh)
	case knownBug != "":
		t.Logf("known bug: %s (%d received in all, none after the drop)", knownBug, len(run.Deliveries))
	default:
		t.Errorf("the connection dropped under a held notification and the session was re-activated, "+
			"but no notification arrived after the drop (%d received in all)", len(run.Deliveries))
	}
}

// republishNothingToSend answers Republish the way a server with nothing to
// retransmit does, and leaves every other service to the honest answers.
func republishNothingToSend(service string, req ua.Request) (ua.Response, bool) {
	if service != "RepublishRequest" {
		return nil, false
	}
	return newServiceFault(req.Header().RequestHandle, ua.StatusBadMessageNotAvailable), true
}
