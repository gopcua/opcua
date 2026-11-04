// A client whose session survived the drop recovers the notification it
// missed even when the server refuses to transfer the subscription. OPC
// 10000-4 §6.7: "The ActivateSession assigns the new SecureChannel to the
// existing Session and allows the Client to reuse the Session and
// Subscriptions in the Server." Conformance unit 276: "Evaluate the sequence
// number in Notifications to detect lost Notifications. Use Republish to
// request missing Notifications." §6.7 permits the other route too, "If
// TransferSubscriptions fails, the Client needs to create a new
// Subscription", so the row scores the notification and not the service call.
// Whether a client asks at all, where the transfer is granted, is
// republish_after_reconnect_test.go.
//
// The scripted server refuses TransferSubscriptions with
// Bad_ServiceUnsupported, as gopcua's own server does, and still holds the
// notification it withheld; its counter only moves forward, so that number
// reaches the application only if the client asked for it. See harness.go.
//
// The verdict comes off the application's own callback. It turns on the
// option the server was set to rather than a refusal the row can prove was
// put to the client, because the record keeps requests and not responses.
//
// While knownBug stands the arm keys on an absence: open62541 has no
// client-side Republish, so it is logged green here rather than failed, and
// the row cannot tell it from gopcua. node-opcua recovers the notification,
// measured on this fixture; Eclipse Milo and UA-.NETStandard recover it in
// source, read and not run here. The row is kept because it scores an
// outcome a client cannot reach by resuming the stream or by replaying what
// the application already had.

package reconnect

import (
	"testing"
	"time"
)

func TestMissedNotificationRecoveredWhenTransferRefused(t *testing.T) {
	// knownBug records that gopcua does not do this yet; the text is logged in
	// place of a failure. README.md says when to delete it.
	// https://github.com/gopcua/opcua/issues/879
	const knownBug = "the client reconnects and re-activates the session, and the server still has the notification it missed, but the client never gets it back"

	// minSettle is the window after the drop this row needs: reconnect,
	// re-activate, be refused the transfer, ask, and have the answer reach the
	// application. That measured under 2ms on loopback; the rest is margin,
	// twice republish_after_reconnect_test.go's for two more round trips.
	const minSettle = 2 * time.Second

	// A tripwire, not a check: both sides are constants, so a settleDefault
	// shortened below minSettle fails the build here rather than a run.
	const _ = uint64(settleDefault - minSettle)

	run := runScenario(runOptions{RefusesTransfer: true})

	if why := setupFailed(run); why != "" {
		t.Fatalf("could not set the test up: %s", why)
	}
	if why := clientSilentSetupFailed(run); why != "" {
		t.Fatalf("could not set the test up: %s", why)
	}
	// A client that never re-activated was never offered the subscription
	// back, so it was never in a position to ask for what the server kept.
	if why := reconnectSetupFailed(run); why != "" {
		t.Fatalf("could not set the test up: %s", why)
	}

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
	case asked > 0:
		// asked is already only the asks the server could answer:
		// republishesAfterDrop counts a Republish for the held number on a
		// post-drop connection the client had activated first, and one that
		// arrives before that connection's ActivateSession is faulted
		// Bad_SessionNotActivated and never reaches the retransmission queue.
		//
		// A client that was given the notification and lost it is not the
		// client knownBug describes, so this arm fires whether or not it stands.
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
// back reached the application after the drop, and how many arrived after the
// drop in all. The server sends that sequence number only in answer to a
// Republish for it, so a delivery carrying it is one the client asked for.
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
