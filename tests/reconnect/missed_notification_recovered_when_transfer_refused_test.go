// A client whose session survived the drop recovers the notification it
// missed even when the server refuses to transfer the subscription. OPC
// 10000-4 §6.7: "The ActivateSession assigns the new SecureChannel to the
// existing Session and allows the Client to reuse the Session and
// Subscriptions in the Server." Conformance unit 276: "Evaluate the sequence
// number in Notifications to detect lost Notifications. Use Republish to
// request missing Notifications."
//
// The clause supports the outcome, not a mechanism. It does not say "do not
// transfer a subscription the session still holds"; it says the re-activated
// session may reuse its subscriptions, and the conformance unit says what to
// do about a notification that went missing. The argument this test makes is
// that a client which answers a refused transfer by recreating the
// subscription throws away a notification the server still had.
//
// The scripted server here refuses TransferSubscriptions with
// Bad_ServiceUnsupported, which gopcua's own server does
// (server/subscription_service.go) and real servers do too, and it still
// holds the notification it withheld, so a Republish asking for that sequence
// number gets it back. The server never sends that number in a Publish
// response — its counter only moves forward — so the application receives it
// only if the client asked for it. A client that recreates the subscription
// instead gets a stream of later notifications and never that one. See
// harness.go.
//
// A known-good client fails this: open62541 has no client-side Republish at
// all (no UA_TYPES_REPUBLISHREQUEST in its client library), so it resumes
// Publish, logs the gap and never recovers what it lost. The test is kept
// because the other three stacks recover it — Eclipse Milo and node-opcua
// with the §6.7 Republish loop on re-activation, UA-.NETStandard with its
// gap-walking Republish when the next notification arrives — and because it
// is an outcome: a client that resumes a stream, replays what the
// application already had, or fabricates notifications the server never
// created does not pass it.
package reconnect

import "testing"

func TestMissedNotificationRecoveredWhenTransferRefused(t *testing.T) {
	// knownBug records that gopcua does not do this yet; the text is logged in
	// place of a failure. README.md says when to delete it.
	const knownBug = "the client reconnects and re-activates the session, and the server still has the notification it missed, but the client never gets it back"

	run := runScenario(runOptions{RefusesTransfer: true})

	if why := setupFailed(run); why != "" {
		t.Fatalf("could not set the test up: %s", why)
	}
	if why := lossSetupFailed(run); why != "" {
		t.Fatalf("could not set the test up: %s", why)
	}
	// A client that never re-activated was never offered the subscription
	// back, so it was never in a position to ask for what the server kept.
	if why := reconnectSetupFailed(run); why != "" {
		t.Fatalf("could not set the test up: %s", why)
	}

	// The row cannot require that a refusal was actually put to the client:
	// a client that recovers by republishing on the surviving session never
	// sends a TransferSubscriptions at all, and that is the client expected
	// to pass. So the messages say what the server was configured to do, not
	// what it was asked. A client that sends a transfer and is faulted for
	// some other reason — before activating, say — is therefore scored as if
	// the refusal had been put to it. Telling those apart needs the record to
	// keep responses as well as requests, which it does not. A client that
	// sends the transfer before activating is the case that shows it.
	asked, _, _ := republishesAfterDrop(run)
	recovered, arrived := missedNotificationBack(run)

	switch {
	case recovered && knownBug != "":
		t.Errorf("this now passes (notification %d came back after the drop, with %d other notification(s)) - "+
			"delete the knownBug constant and the two arms that read it, in the commit that fixed it",
			run.HeldSeq, arrived-1)
	case recovered:
		t.Logf("the client recovered notification %d although the server was set to refuse the transfer "+
			"(%d notifications after the drop in all)", run.HeldSeq, arrived)
	case asked > 0 && answerableAsk(run):
		// The server answers that one sequence number with the notification,
		// so a client that asked for it on an activated connection and did
		// not hand it to the application dropped it on the way. That is not
		// the client knownBug describes, so this arm fires whether or not
		// knownBug stands.
		t.Errorf("the client asked the server to resend notification %d after activating the Session, "+
			"and it never reached the application; %d notifications arrived after the drop", run.HeldSeq, arrived)
	case knownBug != "":
		t.Logf("known bug: %s (the server was set to refuse the transfer and kept notification %d; "+
			"%d notifications arrived after the drop, none of them that one)", knownBug, run.HeldSeq, arrived)
	default:
		t.Errorf("the server was set to refuse the transfer and still had notification %d to give back, "+
			"and the client never got it; %d notifications arrived after the drop, none of them that one",
			run.HeldSeq, arrived)
	}
}

// missedNotificationBack reports whether the notification the server held
// back reached the application after the drop, and how many notifications
// reached it after the drop in all. The server sends that sequence number
// only in answer to a Republish for it, so a delivery carrying it is a
// notification the client asked for and was given.
func missedNotificationBack(run runRecord) (recovered bool, arrived int) {
	for _, d := range run.Deliveries {
		if !d.At.After(run.DropAt) {
			continue
		}
		arrived++
		if d.Seq == run.HeldSeq {
			recovered = true
		}
	}
	return recovered, arrived
}

// answerableAsk reports whether a Republish for the held sequence number
// arrived where the server could answer it: after its own connection's
// ActivateSession. One that arrives before is faulted Bad_SessionNotActivated
// and never reaches the retransmission queue, so a run carrying only those
// has not asked the server anything it could answer.
func answerableAsk(run runRecord) bool {
	activated := map[int]bool{}
	for _, r := range afterDrop(run) {
		switch {
		case r.Service == "ActivateSessionRequest":
			activated[r.Conn] = true
		case r.Service == "RepublishRequest" && r.RepublishSeq == run.HeldSeq && activated[r.Conn]:
			return true
		}
	}
	return false
}
