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
// second request to send. Each arm's client is the real
// opcua.Client, driven through Dial and no further: Connect would
// send CreateSession, ActivateSession and a Read of its own, each
// of which would consume answers the script has not chosen for it,
// and no session and no monitor - Dial starts neither - is what
// makes the error return the whole observable. Each arm's peer
// answers the OPN with a response whose SequenceNumber is 1 - its
// first chunk on the channel - and the arm's one request, a raw
// Read the row itself sends and bounds with a context of its own,
// with a response chunk whose SequenceNumber is 1 again in the
// reused arm, a reuse within the same TokenId, and 3 in the skipped
// arm, skipping 2, using the harness's msgSeqs script.
// See harness.go.
//
// The row asserts that the reused arm's client does not hand the
// reused-number response to its caller, and that the skipped arm's
// client does not hand the skipped-number response to its caller
// either - per the README's definition of the phrase, the response
// body never arrives at the caller; on these unfixed trees that shows
// as each arm's Read returning its own timeout error, and a fixed
// client returns the closed channel's error instead, which is why the
// arms key on the body - and a receiver with no sequence check at all
// accepts both, so the second arm separates a partial check from
// none. Each arm scores its own violation alone. It deliberately
// asserts nothing about which error the client returns, and nothing
// about a dialling behaviour: the clause's "force a reconnect" shows
// as the channel the client closes, not as a redial. A fixed client
// rejects the arm's chunk and closes the channel - "If an error
// occurs during Message verification the receiver shall close the
// communication channel." - and the arm's pass is the response body
// never arriving, per the README's definition of the phrase; a close
// in the peer's close records corroborates that pass and is not
// required for it, and a close is never a decline: the client owes
// each arm the one request whose answer is scored, and a channel
// closed on the first rejection never owes a second. It declines, and
// does not score, when the fixture fails its half - the peer's own
// failure, the channel never opening, the arm's request never
// arriving - and it scores, never declines, a client whose own
// behaviour leaves the run unscoreable: a frame the harness flags as
// the client's own bytes is that, the setup check's report of
// gopcua's own behaviour, a failure the row scores with t.Errorf and
// never a decline. The knownBug constant records that gopcua does not
// do this yet, and no pull request fixes it: the receive path decodes
// the sequence header at uasc/secure_channel.go:520 (n, err :=
// m.SequenceHeader.Decode(m.Data)) but never compares the number it
// carries, a single-chunk message is returned without any look at it
// (uasc/secure_channel.go:1231), and the only read of a received
// SequenceNumber silently skips a duplicate within one chunked
// message instead of raising the transport error
// (uasc/secure_channel.go:1238), so a reused number and a skipped
// number are both accepted on every tree measured. The knownBug arm
// is either arm's caller being handed the response whose
// SequenceNumber was reused or skipped - a presence a fixed client
// does not show - and what deletes the constant is a fix that
// verifies the SequenceNumber of every received chunk and raises the
// transport error; there is no pull request to name. When the
// constant stands, the acceptance is logged as the known bug, with
// which arm's caller was handed its response, so exactly one arm
// accepting is still the known bug: a partial fix - a check that
// catches a reused number and lets a skipped one by - stays logged
// and visible, not mistaken for the whole fix; when both arms'
// responses are rejected, that is a hard error telling whoever runs
// it to delete the constant, every arm that reads it, and anything
// only those arms used.
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

// answerBound bounds the arm's one Read, the request whose answer the
// arm scores. A healthy answer crosses the loopback in milliseconds;
// a chunk a tree rejects reaches no handler, and the bound is what a
// rejection costs the arm - the arm keys on the body, never on which
// error the Read returned along the way. It sits under the harness's
// read bound (harness.go), so the peer's connection is never released
// for idling before the Read returns.
const answerBound = 5 * time.Second

// The SequenceNumbers the arms' response chunks carry in place of
// the peer's next one: the reused arm's repeats the OPN response's
// 1, a reuse within the same TokenId, and the skipped arm's is 3,
// skipping 2.
const (
	reusedSeq  = 1
	skippedSeq = 3
)

func TestReceivedSequenceNumberIsVerified(t *testing.T) {
	// knownBug records that gopcua does not do this yet; the text is
	// logged in place of a failure. ../README.md says when to delete
	// it.
	const knownBug = "the receive path decodes the sequence header but " +
		"never compares the number it carries, so a response chunk that " +
		"reuses a SequenceNumber within the same TokenId and one that " +
		"skips a number are both handed to the caller"

	// One arm per violation, a fresh peer and a fresh client each
	// time, so a client that closes the channel on the first
	// rejection, as the clause orders, never has a second request to
	// send.
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
	if !reused.readArrived {
		t.Fatalf("could not set the test up: the reused arm's Read never " +
			"arrived at the peer, so the response whose reuse the arm scores " +
			"was never sent and this run cannot be scored")
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
	if !skipped.readArrived {
		t.Fatalf("could not set the test up: the skipped arm's Read never " +
			"arrived at the peer, so the response whose skip the arm scores " +
			"was never sent and this run cannot be scored")
	}

	// Which arm's caller was handed its response is part of the
	// log, so exactly one arm accepting - a partial fix - stays
	// logged and visible, not mistaken for the whole fix.
	var handed []string
	if reused.responseArrived {
		handed = append(handed, "the reused arm's caller was handed the "+
			"response whose SequenceNumber repeated 1 within the same TokenId")
	}
	if skipped.responseArrived {
		handed = append(handed, "the skipped arm's caller was handed the "+
			"response whose SequenceNumber skipped 2")
	}

	switch {
	case len(handed) > 0 && knownBug != "":
		t.Logf("known bug: %s (%s)", knownBug,
			strings.Join(handed, "; "))
	case len(handed) > 0:
		t.Errorf("%s, and no knownBug constant stands",
			strings.Join(handed, "; "))
	case knownBug != "":
		t.Errorf("this now passes (the reused arm's caller was not handed " +
			"the response whose SequenceNumber repeated 1 within the same " +
			"TokenId, and the skipped arm's caller was not handed the " +
			"response whose SequenceNumber skipped 2, so neither body ever " +
			"arrived at a caller) - delete the knownBug constant, every arm " +
			"that reads it, and anything only those arms used, in the " +
			"commit that fixed it")
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
	readArrived     bool
	setupWhy        string
	setupFromGopcua bool
}

// runSequenceNumberArm runs one arm of the row end to end: a fresh
// peer under SecurityPolicy None, whose OPN response carries the
// SequenceNumber 1 - its first chunk on the channel - and whose one
// answer, to the arm's one request, is stamped with the responseSeq
// the arm is given in place of the peer's next number. The arm
// launches a fresh client - the real opcua.Client, driven through
// Dial and no further - sends one raw Read bounded by answerBound,
// and closes the client within openBound. It returns what the arm's
// run left to score and the peer's teardown for the row to register
// with t.Cleanup; a peer that could not be started, a client that
// could not be built or a channel that never opened is a decline
// whichever side caused it, reported through o.setupWhy and printed
// by the row with "could not set the test up", per ../README.md. The
// runner never aborts the test; it returns plain values.
func runSequenceNumberArm(responseSeq uint32) (o sequenceArmOutcome,
	teardown func()) {
	// The channel id and the token id both 1, an hour of lifetime, so
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
		msgSeqs: []uint32{responseSeq},
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

	dialCtx, cancel := context.WithTimeout(context.Background(), openBound)
	dialErr := client.Dial(dialCtx)
	cancel()
	if dialErr != nil {
		if w, g := peer.setupCheck(); w != "" {
			o.setupWhy, o.setupFromGopcua = w, g
			return o, teardown
		}
		o.setupWhy = fmt.Sprintf("the channel never opened: %v", dialErr)
		return o, teardown
	}

	readCtx, readCancel := context.WithTimeout(context.Background(),
		answerBound)
	res, _ := client.Read(readCtx, &ua.ReadRequest{
		NodesToRead: []*ua.ReadValueID{{
			NodeID:      ua.NewNumericNodeID(0, 2255),
			AttributeID: ua.AttributeIDValue,
		}}},
	)
	readCancel()
	// The arm keys on the body, never on which error the Read
	// returned along the way.
	o.responseArrived = res != nil

	closeCtx, closeCancel := context.WithTimeout(context.Background(),
		openBound)
	_ = client.Close(closeCtx)
	closeCancel()

	for _, r := range peer.record().Requests {
		if _, ok := r.Req.(*ua.ReadRequest); ok {
			o.readArrived = true
		}
	}
	o.setupWhy, o.setupFromGopcua = peer.setupCheck()
	return o, teardown
}
