//go:build conformance

// The chunks a client receives carry SequenceNumbers the client
// verifies: a response chunk that reuses a number already used within
// the same TokenId, and one that skips a number, are each rejected
// instead of handed to the caller.
//
// The clauses: OPC 10000-6 §6.7.2.4: "A SequenceNumber shall not be
// reused for any TokenId. The SecurityToken lifetime shall be short
// enough to ensure that this never happens; however, if it does the
// receiver shall treat it as a transport error and force a reconnect.
// The SequenceNumber does not reset when a new TokenId is issued and it
// shall be incremented by exactly one for each MessageChunk sent." OPC
// 10000-6 §6.7.7: "The contents of the MessageChunk shall not be
// interpreted until the Message is decrypted and the signature and
// sequence number verified." "After the security validation is
// complete the receiver shall verify the RequestId and the
// SequenceNumber. If these checks fail a Bad_SecurityChecksFailed error
// is reported. The RequestId is verified by the Client since only the
// Client knows if it is valid or not. If the SequenceNumber is not
// valid, the receiver shall log a Bad_SequenceNumberInvalid error."
// and: "If an error occurs during Message verification the receiver
// shall close the communication channel." Both of the row's arms rest
// on §6.7.7's verification order: it covers a skipped number as well as
// a reused one, so the row scores both on the same clause.
//
// Neighbouring rows: expired_security_token_is_rejected_test.go scores
// the token check §6.7.7 orders beside this one, where this row scores
// the sequence-number check; security_token_lifetime_is_measured_from_
// receipt_test.go scores the clock that decides when a token expires,
// where this row scores what the channel does with the numbers the
// token governs; service_fault_leaves_the_channel_open_test.go scores
// a well-formed answer's effect on the channel, where this row scores
// chunks that fail verification, which do close it. The sender
// direction of §6.7.2.4 - the client's own chunks incrementing by
// exactly one - is spec-clean on every tree measured and is not a row.
//
// The row runs against the scripted peer from harness.go twice - a
// fresh peer and a fresh client each time, under SecurityPolicy
// None - one arm per violation, so that a client which closes the
// channel on the first rejection, as the clause orders, never has a
// request to send after the answer it rejected. Each arm's client
// is the real opcua.Client, driven through Dial and no further:
// Connect would send CreateSession, ActivateSession and a Read of
// its own, each of which would consume answers the script has not
// chosen for it, and no session and no monitor - Dial starts
// neither - is what makes the arm's Reads the whole exchange the
// row scores. Each arm's peer answers the OPN with a response whose
// SequenceNumber is 1 - its first chunk on the channel. The arm's
// two requests, raw Reads the row itself sends and bounds with
// contexts of their own, the peer answers with two answer chunks:
// the first - the well-formed answer, the arm's ground - carries
// the peer's correct next number, 2, and the second the violated
// number the arm is given - 2 again in the reused arm, a reuse of
// the first answer's number within the same TokenId, and 4 in the
// skipped arm, skipping 3 - using the harness's msgSeqs script. The
// first answer's body must reach the caller on both arms before the
// violated one is scored: it grounds the arm's rejection - a client
// that rejects every chunk would otherwise read the same as the
// fix. It comes first because a fixed client closes the channel on
// the violated answer, as §6.7.7 orders, and would leave a later
// ground undelivered.
// See harness.go.
//
// The row asserts that the reused arm's client does not hand the
// reused-number response to its caller, and that the skipped arm's
// client does not hand the skipped-number response to its caller
// either - per the README's definition of the phrase, the response
// body never arrives at the caller; a rejection shows as the arm's
// Read returning its own timeout error, and a fixed client returns
// the closed channel's error instead, which is why the arms key on
// the body - and a receiver with no sequence check at all accepts
// both, so the first arm separates a partial check from none, and the
// second a partial fix from the whole one. Each arm scores its own
// violation alone. It deliberately asserts nothing about which error
// the client returns, and nothing about a dialling behaviour: the
// clause's "force a reconnect" shows as the channel the client
// closes, not as a redial. A fixed client rejects the arm's chunk and
// closes the channel - "If an error occurs during Message
// verification the receiver shall close the communication channel." -
// and the arm's pass is the response body never arriving, per the
// README's definition of the phrase; a close in the peer's close
// records corroborates that pass and is not required for it, and a
// close is never a decline: the client owes each arm its two
// requests, and a channel closed on the first rejection never owes a
// request after the answer it rejected. It declines, and does not
// score, when the fixture fails its half or a stage the story needs
// never ran, whichever side caused it - the peer's own failure, the
// channel never opening, the well-formed answer never arriving at the
// caller though both the arm's Reads reached the peer - and it scores,
// never declines, a client whose own behaviour leaves the run
// unscoreable: a frame the harness flags as the client's own bytes is
// that, the setup check's report of gopcua's own behaviour, and so is
// the arm's Read never arriving at the peer - the setup check holds the
// peer's half clean, so a Read that never arrives can only be the
// client's own doing - a failure the row scores with t.Errorf and never
// a decline. The knownBug constant records that gopcua does not do this
// yet: the receive path decodes the sequence header at
// uasc/secure_channel.go:563 (n, err :=
// m.SequenceHeader.Decode(m.Data)) but never compares the number it
// carries, a single-chunk message is returned without any look at it
// (uasc/secure_channel.go:1295-1296), and the only read of a received
// SequenceNumber silently skips a duplicate within one chunked message
// instead of raising the transport error (uasc/secure_channel.go:1302),
// so a reused number and a skipped number are both accepted on every
// tree measured. The knownBug arm is either arm's caller being handed
// the response whose SequenceNumber was reused or skipped - a presence
// a fixed client does not show - and what deletes the constant is a fix
// that verifies the SequenceNumber of every received chunk and raises
// the transport error; there is no pull request to name. When the
// constant stands, the acceptance is logged as the known bug, with
// which arm's caller was handed its response, so exactly one arm
// accepting is still the known bug: a partial fix - a check that
// catches a reused number and lets a skipped one by - stays logged and
// visible, not mistaken for the whole fix; when both arms' responses
// are rejected and both arms' well-formed answers handed to their
// callers, that is a hard error telling whoever runs it to delete the
// constant, every arm that reads it, and anything only those arms used.
//
// No reference stack has been read for this behaviour; the measurement
// on the trees this suite was authored against is the row's own.
package securechannel

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/ua"
)

// openBound bounds each arm's Dial and the client's close at the
// arm's end: above the milliseconds a healthy HEL/ACK and OPN
// exchange takes, so a loaded machine finishes inside it, and short
// enough that a channel wedged past its own exchange cannot hold
// the arm.
const openBound = 10 * time.Second

// answerBound bounds the arm's Reads, the ground Read and the scored
// one. A healthy answer crosses the loopback in milliseconds; a
// chunk a tree rejects reaches no handler, and the bound is what a
// rejection costs the arm. An answer slower than the bound is
// indistinguishable from a rejection on both arms. It sits under
// the harness's read bound (harness.go), so the peer's connection
// is never released for idling before a Read returns.
const answerBound = 5 * time.Second

// The SequenceNumbers the arms' violated answer chunks carry in
// place of the peer's next one: the reused arm's repeats the
// well-formed answer's 2, a reuse within the same TokenId, and the
// skipped arm's is 4, skipping 3.
const (
	reusedSeq  = 2
	skippedSeq = 4
)

func TestReceivedSequenceNumberIsVerified(t *testing.T) {
	// knownBug records that gopcua does not do this yet; the text is
	// logged in place of a failure. ../README.md says when to delete
	// it.
	const knownBug = "the receive path decodes the sequence header but " +
		"never compares the number it carries, so a response chunk that " +
		"reuses a SequenceNumber within the same TokenId and one that " +
		"skips a number are both handed to the caller"

	reused, teardown := runSequenceNumberArm(reusedSeq)
	t.Cleanup(teardown)
	if reused.setupWhy != "" {
		if reused.setupFromGopcua {
			t.Errorf("gopcua's own behaviour made this run unscoreable: %s",
				reused.setupWhy)
			return
		}
		t.Fatalf("could not set the test up: %s", reused.setupWhy)
	}
	if reused.readArrived < 2 {
		t.Errorf("the reused arm's Reads never both reached the peer, " +
			"so the arm scored nothing: the harness's setup check " +
			"reports no failure on the peer's side, so the client's " +
			"own behaviour stopped it, and that is scored as " +
			"gopcua's failure, never declined")
		return
	}

	skipped, teardown := runSequenceNumberArm(skippedSeq)
	t.Cleanup(teardown)
	if skipped.setupWhy != "" {
		if skipped.setupFromGopcua {
			t.Errorf("gopcua's own behaviour made this run unscoreable: %s",
				skipped.setupWhy)
			return
		}
		t.Fatalf("could not set the test up: %s", skipped.setupWhy)
	}
	if skipped.readArrived < 2 {
		t.Errorf("the skipped arm's Reads never both reached the peer, " +
			"so the arm scored nothing: the harness's setup check " +
			"reports no failure on the peer's side, so the client's " +
			"own behaviour stopped it, and that is scored as " +
			"gopcua's failure, never declined")
		return
	}

	// Which arm's caller was handed its response is part of the log.
	var handed []string
	if reused.responseArrived {
		handed = append(handed, "the reused arm's caller was handed the "+
			"response whose SequenceNumber repeated 2 within the same TokenId")
	}
	if skipped.responseArrived {
		handed = append(handed, "the skipped arm's caller was handed the "+
			"response whose SequenceNumber skipped 3")
	}

	switch {
	case len(handed) > 0 && knownBug != "":
		t.Logf("known bug: %s (%s)", knownBug,
			strings.Join(handed, "; "))
	case len(handed) > 0:
		t.Errorf("%s, and no knownBug constant stands",
			strings.Join(handed, "; "))
	case !reused.groundArrived:
		t.Fatalf("could not set the test up: the reused arm's client did " +
			"not hand the well-formed answer to its caller, so its " +
			"rejection of the reused-number response is not grounded in " +
			"the number, and this run cannot be scored")
	case !skipped.groundArrived:
		t.Fatalf("could not set the test up: the skipped arm's client did " +
			"not hand the well-formed answer to its caller, so its " +
			"rejection of the skipped-number response is not grounded in " +
			"the number, and this run cannot be scored")
	case knownBug != "":
		t.Errorf("this now passes (the reused arm's caller was not handed " +
			"the response whose SequenceNumber repeated 2 within the same " +
			"TokenId, and the skipped arm's caller was not handed the " +
			"response whose SequenceNumber skipped 3, so neither violated " +
			"body ever arrived at a caller, and both arms handed the " +
			"well-formed answer to their callers, so neither channel was " +
			"dead) - delete the knownBug constant, every arm that reads " +
			"it, and anything only those arms used, in the commit that " +
			"fixed it")
	default:
		t.Logf("both arms rejected their responses: neither caller saw " +
			"the body")
	}
}

// sequenceArmOutcome is what one arm's run left within the row's
// reach. It holds no handle on the peer: the runner reads the peer's
// records after the arm's exchange is over, so the row never races a
// live connection for it.
type sequenceArmOutcome struct {
	responseArrived bool
	groundArrived   bool
	readArrived     int
	setupWhy        string
	setupFromGopcua bool
}

// runSequenceNumberArm runs one arm of the row end to end: a fresh
// peer under SecurityPolicy None, whose OPN response carries the
// SequenceNumber 1 - its first chunk on the channel. The peer's
// answers, to the arm's two raw Reads, are stamped with the peer's
// correct next number and then with the responseSeq the arm is
// given in place of the peer's next one, using the harness's msgSeqs
// script. The arm launches a fresh client - the real opcua.Client,
// driven through Dial and no further - sends two raw Reads, each
// bounded by answerBound, and closes the client within openBound.
// It returns what the arm's run left to score and the peer's
// teardown for the row to register with t.Cleanup; a peer that
// could not be started, a client that could not be built or a
// channel that never opened is a decline whichever side caused it,
// reported through o.setupWhy and printed by the row with "could
// not set the test up", per ../README.md. The runner never aborts
// the test; it returns plain values.
func runSequenceNumberArm(responseSeq uint32) (o sequenceArmOutcome,
	teardown func()) {
	// The same token for issue and renew, an hour of lifetime, so
	// no renewal comes near the arm, and every chunk the peer sends
	// carries the same TokenId.
	token := channelToken{
		channelID: 1,
		tokenID:   1,
		createdAt: time.Now(),
		lifetime:  3600000,
	}
	peer, teardown, why := startPeer(peerScript{
		issue: token,
		renew: token,
		answer: func(req ua.Request, reqID uint32) ua.Response {
			if r, ok := req.(*ua.ReadRequest); ok {
				return readResponse(r, reqID)
			}
			return nil
		},
		// The peer's answers in order: the correct next number, 2,
		// then the violated responseSeq - the ground before the
		// violation, since a fixed client closes the channel on
		// the violated answer.
		msgSeqs: []uint32{2, responseSeq},
	})
	if why != "" {
		o.setupWhy = why
		return o, teardown
	}

	client, err := opcua.NewClient("opc.tcp://"+peer.addr(),
		opcua.SecurityPolicy(ua.SecurityPolicyURINone),
		opcua.SecurityMode(ua.MessageSecurityModeNone),
	)
	if err != nil {
		o.setupWhy = fmt.Sprintf("build the client: %v", err)
		return o, teardown
	}

	if why, fromGopcua := openChannel(client, peer, openBound); why != "" {
		o.setupWhy, o.setupFromGopcua = why, fromGopcua
		return o, teardown
	}

	o.groundArrived = readBodyArrived(client, answerBound)
	o.responseArrived = readBodyArrived(client, answerBound)

	closeCtx, closeCancel := context.WithTimeout(context.Background(),
		openBound)
	_ = client.Close(closeCtx)
	closeCancel()

	for _, r := range peer.record().Requests {
		if _, ok := r.Req.(*ua.ReadRequest); ok {
			o.readArrived++
		}
	}
	o.setupWhy, o.setupFromGopcua = peer.setupCheck()
	return o, teardown
}
