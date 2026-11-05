// After the transport drops and the session is re-activated, a client whose
// subscription survived on the server goes on receiving notifications on it.
// OPC 10000-4 §6.7, conformance unit 669: "The Client shall only create a
// new Session if ActivateSession fails." Re-activation assigns the new
// channel to the existing session, so the client keeps the subscriptions
// that session holds and the server keeps publishing to it.
//
// Data on one the client built to replace a subscription the server
// disowned, rather than on one that survived, is
// data_resumes_on_recreated_subscription_test.go. The same clause sentence
// read on the wire, as whether a CreateSession arrives at all, is
// new_session_only_after_failed_activation_test.go: a client that creates
// one anyway has abandoned the subscriptions the old Session held and still
// recovers data, so this row passes it. Whether the client waits for its own
// ActivateSession before using the Session on a new connection is
// session_unused_before_activation_test.go, a different clause, §5.7.2.1;
// a client that does not wait is faulted with Bad_SessionNotActivated and
// may recover and receive data anyway, so this row passes that too.
//
// The scripted server is the default one: the subscription outlives the drop
// and the re-activation. The server holds one notification back and drops
// the connection underneath it, then answers every later Publish with fresh
// data. See harness.go.
//
// The test checks only that data arrives after the drop. Whether the client
// republishes, transfers the subscription or recreates it is not the test's
// business. The server answers Republish with Bad_MessageNotAvailable, which
// ends a Republish loop, so a client that does ask is not faulted for it.
//
// The reference stacks. The three read here re-issue a Publish as soon as one
// is answered rather than on a timer, so each draws at this fixture's ceiling
// of five answers a second and what separates them is the dead time after the
// drop. open62541 v1.4.22 schedules no wait - ua_client_connect.c calls
// __Client_Subscriptions_backgroundPublish from the ActivateSession response
// handler - and spends about 0.2s of the window on the reconnect, measured
// here with its publish depth lowered by hand to two: at its default of ten,
// this fixture's serialised answers push its CreateMonitoredItems past the
// client's request timeout and void the run. node-opcua 2.187.19 asks
// once, is faulted and refills a five-deep pipeline, about 1.2s, measured
// here. Eclipse Milo, read and not run, leaves Active for ReactivatingWait on
// a lost connection, which schedules its first reactivation a whole second
// out and doubles that on each failure: a second of this window gone by
// design, and the reason minFresh is three rather than a number that measures
// throughput. UA-.NETStandard was not read. No stack here fails it: no
// exception.

package reconnect

import (
	"testing"
	"time"
)

func TestDataResumesAfterReconnect(t *testing.T) {
	// gopcua did not do this until the fix for
	// https://github.com/gopcua/opcua/issues/879; README.md says which pull
	// request closed it.

	// minFresh is how many distinct notifications after the drop show a
	// resumed stream: one that arrives and then stalls is not a stream, and a
	// third rules out a pair that happened to land together. It is not a rate:
	// the fixture sleeps publishPacing before answering each Publish, so a
	// larger number would spend the settle window on the server's own sleeps
	// and charge the client for them.
	const minFresh = 3

	// minSettle is the window a client needs to notice the reset, redial and
	// re-activate before minFresh answers can arrive at publishPacing.
	const minSettle = 2 * time.Second

	// A tripwire, not a check: both sides are constants, so a settleDefault
	// shortened below minSettle fails the build here rather than a run.
	const _ = uint64(settleDefault - minSettle)

	run := runScenario(runOptions{})

	if why := setupFailed(run); why != "" {
		t.Fatalf("could not set the test up: %s", why)
	}
	if why := reconnectSetupFailed(run); why != "" {
		t.Fatalf("could not set the test up: %s", why)
	}

	// Until the drop the server withheld every notification above the one it
	// held, and LastSeq is the highest number it ever created, so a delivery
	// inside that range is data this server sent after the drop and not a
	// number the client invented. The value carries the number, so distinct
	// numbers are distinct values.
	fresh := map[uint32]bool{}
	for _, d := range run.Deliveries {
		if d.At.After(run.DropAt) && d.Seq > run.HeldSeq && d.Seq <= run.LastSeq {
			fresh[d.Seq] = true
		}
	}

	switch {
	case len(fresh) >= minFresh:
		t.Logf("data kept flowing: %d distinct notifications arrived after the drop", len(fresh))
	default:
		t.Errorf("a session was activated after the drop, but only %d distinct notifications "+
			"arrived after it, where at least %d are expected (%d received in all)", len(fresh), minFresh, len(run.Deliveries))
	}
}
