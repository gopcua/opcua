//go:build conformance

// A response chunk secured with an expired SecurityToken is not
// accepted: after the token's lifetime plus its 25 % grace window has
// passed on the client's clock, a chunk that still carries the old
// token is rejected and the channel is closed.
//
// The clauses: OPC 10000-6 §6.7.4: "The Client shall continue to accept
// the old SecurityToken until it expires or it receives a Message from
// the Server secured with the new SecurityToken." OPC 10000-4 §5.6.2.1,
// the grace window: "Clients should accept Messages secured by an
// expired SecurityToken for up to 25 % of the token lifetime." OPC
// 10000-6 §6.7.7: "If the Message is secured with symmetric algorithms,
// then a Bad_SecureChannel TokenUnknown error shall be reported if the
// TokenId refers to a SecurityToken that has expired or is not
// recognized." (sic: the source has a space in "SecureChannel
// TokenUnknown") and: "If an error occurs during Message verification
// the receiver shall close the communication channel." Under
// SecurityMode None, §6.7.4 says: "The SecureChannelId and the TokenId
// are still assigned but no security is applied to Messages exchanged
// via the channel. The SecurityToken shall still be renewed before the
// RevisedLifetime expires. Receivers shall still ignore invalid or
// expired TokenIds." - but "ignore" there can be read as discarding
// such Messages or as disregarding the field, and gopcua's receive
// path under None never compares the TokenId on the wire at all, so a
// None arm cannot separate a fixed tree from this one and the row
// scores only its Basic256Sha256/Sign arm, where §6.7.7 applies
// plainly; the None behaviour is noted here and not scored.
//
// Neighbouring rows: security_token_lifetime_is_measured_from_receipt_
// test.go scores the clock a token's life is measured on, where this
// row scores what the channel owes a token whose life is over;
// received_sequence_number_is_verified_test.go scores the other check
// §6.7.7 orders on a received chunk, where this row scores the token
// check; service_fault_leaves_the_channel_open_test.go scores a
// well-formed answer's effect on the channel, where this row scores a
// chunk that fails verification, which does close it.
//
// The row runs against the scripted peer from harness.go, under
// Basic256Sha256 in mode Sign - the policy's asymmetric and symmetric
// signing lives with this row, in the codec its body supplies,
// because no other row calls it - and the codec derives each token's
// keys from the request it is answering: under Basic256Sha256 a
// token's symmetric keys come from the ClientNonce of the OPN that
// drew them and the ServerNonce of the response that carried them, so
// the harness hands the codec the decoded OPN, and the row's codec
// mints each token's keys off that pair. The row runs in two arms
// that differ only in the ids the peer hands out, each arm on a peer
// of its own with a client of its own. Each arm's client is the real
// opcua.Client, driven through Dial and no further: Connect would
// send CreateSession, ActivateSession and a Read of its own, each of
// which would consume answers the script has not chosen for it, and
// the row's scored requests are the raw Reads it sends and bounds.
// The peer hands out a short token and answers the client's own
// renewal with a new token id on the same channel - a number the
// issue token never carried, so the channel holds two tokens to tell
// apart. Once the renewal is in the peer's records, and before the
// old token's lifetime plus 25 % has passed on the client's clock,
// the arm sends a Read that the peer answers with a chunk secured
// with the OLD token, and that body must reach the caller on both
// arms: it is the instrument check's positive half, a client that
// verifies and accepts a chunk under a live token, so the rejection
// the row scores next is grounded in expiry and not in a token the
// channel never held - OPC 10000-6 §6.7.4: "The Client shall
// continue to accept the old SecurityToken until it expires or it
// receives a Message from the Server secured with the new
// SecurityToken" - and a client that hands no answer to its caller
// cannot be measured, on either arm, and the run is declined. The
// row's scored Read goes out only after the old token's lifetime
// plus 25 % has passed, so the stimulus is expired on any tree that
// measures the window correctly, and the peer answers it with a
// response chunk whose symmetric security header carries the old,
// expired token id and whose signature is made with the old token's
// keys, chosen with the harness's answerToken script: the scored
// arm's server assigned a channel id that differs from the token id,
// the control arm's the same number, and the ids are all that
// separates them. Nothing is ever answered with the renewed token,
// before the scored Read or after it: §6.7.4 lets the client drop
// the old token on the first Message the new one secures, and a
// rejection grounded in that instead of in expiry is not the
// behaviour the row scores. A client that closes the channel on
// the failed check does what §6.7.7 orders - "If an error occurs
// during Message verification the receiver shall close the
// communication channel" - and the row accepts the close: it
// asserts only that the expired-token body never reaches the
// caller, and a fixed client that shuts the channel shows the same
// shape a conforming one does.
// See harness.go.
//
// The row asserts that the scored arm's client does not hand the
// expired-token response to its caller - per the README's definition of
// the phrase, the response body never arrives at the caller; on
// a tree without the fix that shows as the arm's scored Read returning
// its own timeout error, and a fixed client returns the closed channel's
// error instead, which is why the arm keys on the body - and that
// the control arm's client does reject it: the control arm is the
// instrument check - even before the fix the channel id = token id
// arm's expiration removed the right instance, so it shows
// that the instrument can see a rejection at all, and a run in which
// it does not see one proves nothing about the scored arm. It
// declines, and does not score, when the fixture fails its half or
// a stage the story needs never ran, whichever side caused it -
// the peer's own failure, the channel never opening, the peer's OPN
// records holding no renewal, the live old-token answer never
// arriving at the caller - and it scores, never declines, a
// client whose own behaviour leaves the scored arm unscoreable: a
// frame the harness flags as the client's own bytes is that case's
// clearest shape, the setup check's report of gopcua's own
// behaviour, a failure the row scores with t.Errorf and never a
// decline. The records hold one
// entry per OpenSecureChannelRequest the peer answered, with the
// connection generation it arrived on, whether it asked for a new
// token or renewed the current one, and the moment it arrived, and
// without a renewal there is no new token id on the same channel
// for the scored answer to be old against. The row's scored Read
// never reaching the peer is scored by arm: on the scored arm it is scored
// as gopcua's failure - unhandled
// it would read as a rejection, the body never arriving, a false
// pass, and the harness's setupCheck already holds the peer's half
// clean, so a Read that never arrives can only be the client's own
// doing - and on the control arm it is a decline, because the
// instrument check never ran: a control run whose scored Read never
// reached the peer has no rejection to see, and proves nothing about the
// scored arm. gopcua did not do this until the fix for
// https://github.com/gopcua/opcua/pull/951 (issue #941) merged:
// before it, scheduleExpiration emptied and rebuilt the instance map
// under the TOKEN id (uasc/secure_channel.go:952,
// uasc/secure_channel.go:954, uasc/secure_channel.go:964) while every
// arriving chunk was looked up under the CHANNEL id
// (uasc/secure_channel.go:579, uasc/secure_channel.go:603), so when
// the server assigned the two different numbers the expired instance
// was never removed and the old-token signature still verified; the
// fix keyed scheduleExpiration by the channel id instead of the token
// id, after which both arms hand the live old-token answer to their
// callers and reject the expired-token response. The red arm's
// presence is the scored arm accepting the old-token response - a
// presence a fixed client does not show, and the control arm passing
// shows the row is not scoring an instrument that rejects nothing -
// and a client that hands no answer to its caller is a decline,
// never a fix; measured on main before and after #911 and #904
// merged, the scored arm still accepted, because neither touches
// the expiration's keying. When both arms reject, the row passes.
//
// No reference stack has been read for this behaviour; the measurement
// on main before and after PRs #911 and #904 merged is the row's own.
package securechannel

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/binary"
	"fmt"
	"math/big"
	"sync"
	"testing"
	"time"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/ua"
	"github.com/gopcua/opcua/uapolicy"
	"github.com/gopcua/opcua/uasc"
)

// tokenLifetime is the short lifetime the peer's issue token carries.
// Measured on this tree, the client renews at three quarters of it
// and wipes the expired instance at its lifetime plus the 25 %
// grace window - for a whole number of seconds the 75 % and 125 %
// factors in uasc truncate to exactly those points - so the arm's
// scored Read goes out graceWait past both the wipe and the window
// the specification names, and the stimulus is expired on any tree
// that measures the window correctly either way. Four seconds
// leave the arm a second of slack for the renewal, due at three
// quarters of the lifetime, before the pre-expiry Read it sends
// once the renewal is in the peer's records, and that Read still
// has the second left before the window it must land inside closes,
// so a renewal that runs late on a loaded machine still leaves the
// arm its live-token answer.
const tokenLifetime = 4 * time.Second

// renewalSlack is how far past the 75 % of the issue token's
// lifetime at which uasc schedules the client's own renewal the arm
// waits for that renewal in the peer's records before it gives up:
// the renewal is a timer on a goroutine, and a loaded machine
// fires it late. The deadline it sets still leaves the pre-expiry
// Read the rest of the token's lifetime plus its grace window to
// land in.
const renewalSlack = time.Second

// preReadMargin is how far inside the issue token's lifetime plus
// its 25 % grace window the pre-expiry Read's answer must land:
// the bound on that Read ends this far before the window closes, so
// a slow answer is never verified against a token the client has
// already wiped on its own truncated schedule.
const preReadMargin = 100 * time.Millisecond

// renewalLifetime is the lifetime the renewal token carries: a full
// minute, so neither the renewed token's own renewal at 75 % nor its
// expiry at 125 % falls inside the arm.
const renewalLifetime = time.Minute

// graceWait is how far past the issue token's lifetime plus its 25 %
// grace window the arm waits before it sends the scored Read, and
// the window is waited out from the stamp the peer put on the
// token: gopcua schedules the expiration from that stamp,
// and a tree that measures from receipt ends its window a handshake
// later, which the same wait covers. The wipe is a timer on a
// goroutine, and the wait gives the goroutine time to fire and its
// wipe to land on a loaded machine; a delay that outruns the wait
// on the scored arm alone fails a fixed tree, and
// the row cannot tell it from a real one.
const graceWait = 400 * time.Millisecond

// dialBound bounds each arm's Dial and the client's close at the
// arm's end: above the milliseconds a healthy HEL/ACK and OPN
// exchange takes, so a loaded machine finishes inside it, and short
// enough that a channel wedged past its own exchange cannot hold the
// arm.
const dialBound = 10 * time.Second

// readBound is the bound on the arm's Reads. On a tree without the
// fix a rejection shows as the Read returning this bound's own timeout
// error - a rejected chunk reaches no handler - so the bound is
// what a rejected answer costs the arm; a healthy answer crosses
// the loopback in milliseconds, and one slower than the bound is
// indistinguishable from a rejection on both arms. readBound sits
// under the harness's read bound (harness.go), so the peer's
// connection is never released for idling before a Read returns.
const readBound = 3 * time.Second

// The token ids the peer hands out: the issue token's id, and the
// new id the renewal carries - a number the issue token never had,
// so the channel holds two tokens to tell apart.
const (
	issueTokenID = 15
	renewTokenID = 16
)

// The channel ids the arms differ in: the control arm's server
// assigns the channel id and the token id the same number, so even
// before the fix the expiration removed the right
// instance, and the scored arm's assigns the two different numbers.
// The ids are all that separates the arms.
const (
	controlChannelID = 15
	scoredChannelID  = 20
)

func TestExpiredSecurityTokenIsRejected(t *testing.T) {
	// gopcua did not do this until
	// https://github.com/gopcua/opcua/pull/951 merged; it passes since.

	// The control arm first: the server assigns the channel id and
	// the token id the same number, and even before the fix the
	// expiration removed the right instance, so the arm
	// is the instrument check - a run in which it sees no
	// rejection proves nothing about the scored arm. Its four
	// declines below - no renewal in the peer's records, the live
	// old-token answer never reaching the caller, the scored Read
	// never reaching the peer, the expired-token response handed
	// to the caller - each leaves the check unrun or its rejection
	// ungrounded, whichever side caused the decline.
	control, teardown := runRejectionArm(controlChannelID)
	t.Cleanup(teardown)
	if control.setupWhy != "" {
		if control.setupFromGopcua {
			t.Errorf("gopcua's own behaviour made this run unscoreable: %s",
				control.setupWhy)
			return
		}
		t.Fatalf("could not set the test up: %s", control.setupWhy)
	}
	if !control.renewalSeen {
		t.Fatalf("could not set the test up: the control arm's peer holds " +
			"no renewal, so the answer's token would be the channel's only " +
			"one, not an old one, and this run cannot be scored")
	}
	if !control.preExpiryResponseArrived {
		t.Fatalf("could not set the test up: the control arm's client did " +
			"not hand the live old-token answer to its caller, so its " +
			"rejection of the expired-token chunk is not grounded in " +
			"expiry and this run cannot be scored")
	}
	if !control.readArrived {
		t.Fatalf("could not set the test up: the control arm's scored Read " +
			"never reached the peer, so the instrument check never ran and " +
			"this run proves nothing about the scored arm")
	}
	if control.responseArrived {
		t.Fatalf("could not set the test up: the control arm's client handed " +
			"the expired-token response to its caller, so the instrument " +
			"cannot see a rejection at all and this run proves nothing about " +
			"the scored arm")
	}

	scored, teardown := runRejectionArm(scoredChannelID)
	t.Cleanup(teardown)
	if scored.setupWhy != "" {
		if scored.setupFromGopcua {
			t.Errorf("gopcua's own behaviour made this run unscoreable: %s",
				scored.setupWhy)
			return
		}
		t.Fatalf("could not set the test up: %s", scored.setupWhy)
	}
	if !scored.renewalSeen {
		t.Fatalf("could not set the test up: the scored arm's peer holds " +
			"no renewal, so the answer's token would be the channel's only " +
			"one, not an old one, and this run cannot be scored")
	}
	if !scored.preExpiryResponseArrived {
		t.Fatalf("could not set the test up: the scored arm's client did " +
			"not hand the live old-token answer to its caller, so a " +
			"client that rejects every chunk cannot be told from one that " +
			"rejected the expired token, and this run cannot be scored")
	}
	if !scored.readArrived {
		t.Errorf("the scored arm's expired-token Read never reached the " +
			"peer, so the arm scored nothing: the harness's setup check " +
			"reports no failure on the peer's side, so the client's own " +
			"behaviour stopped it, and that is scored as gopcua's " +
			"failure, never declined")
		return
	}

	switch {
	case scored.responseArrived:
		t.Errorf("the scored arm's client handed the expired-token " +
			"response to its caller, where the control arm's client rejected " +
			"its own, and \u00a76.7.7 orders the receiver to close the " +
			"communication channel on a failed Message verification")
	default:
		t.Logf("both arms rejected the old-token response: neither caller " +
			"saw the body")
	}
}

// rejectionArmOutcome is what one arm's run left within the row's
// reach. It holds no handle on the peer: the runner reads the
// peer's records after each stage of the arm's story has played,
// so the row never races a live connection for it.
type rejectionArmOutcome struct {
	// preExpiryResponseArrived reports whether the answer to the
	// arm's pre-expiry Read - a chunk the peer secured with the old
	// token while it was still live - reached the caller.
	preExpiryResponseArrived bool
	// responseArrived reports whether the answer to the arm's
	// scored Read - the same old token, now past its lifetime plus
	// the 25 % grace window - reached the caller.
	responseArrived bool
	renewalSeen     bool
	readArrived     bool
	// readsBefore is how many Reads the peer had recorded when the
	// arm sent its scored Read, which tells that Read's record apart
	// from the pre-expiry Read's. A count rather than a timestamp:
	// on Windows the clock can give the send and the peer's receipt
	// the same reading.
	readsBefore     int
	scoredSent      bool
	setupWhy        string
	setupFromGopcua bool
}

// runRejectionArm runs one arm of the row end to end: it mints the
// client's and the peer's keys in memory, starts a fresh peer
// speaking Basic256Sha256 in mode Sign through the signCodec below.
// The peer hands out an issue token carrying the channel id it is
// given and the token id issueTokenID and answers the client's own
// renewal with the token id renewTokenID on the same channel; the
// arm launches a fresh client - the real opcua.Client, driven
// through Dial and no further. The arm waits for the client's own
// renewal in the peer's records - due at 75 % of the issue token's
// lifetime, with renewalSlack for a late one - and then, still
// inside the old token's lifetime plus its 25 % grace window, sends
// the pre-expiry Read, bounded so its answer lands inside the
// window: the peer answers it with a chunk stamped with the old
// token and signed with the old token's keys, the token still live.
// The arm waits out the rest of the window and sends the scored
// Read, bounded by readBound: the peer answers it with a chunk
// stamped and signed with the same old, now expired token. Both
// answers are mapped by the script's answerToken hook, which never
// picks the renewed token, because §6.7.4 lets the client drop the
// old token on the first Message the new one secures and a
// rejection grounded in that instead of in expiry is not the
// behaviour the row scores. It returns what the arm's run left to
// score and the peer's teardown for the row to register with
// t.Cleanup; keys that could not be minted, a peer that could not
// be started or a client that could not be built is a decline
// whichever side caused it, reported through o.setupWhy and printed
// by the row with "could not set the test up", per ../README.md. The
// runner never aborts the test; it returns plain values.
func runRejectionArm(channelID uint32) (o rejectionArmOutcome,
	teardown func()) {
	clientKey, clientCert, why := mintSigningCert(
		"gopcua-securechannel-client")
	if why != "" {
		o.setupWhy = why
		return o, func() {}
	}
	peerKey, peerCert, why := mintSigningCert("gopcua-securechannel-peer")
	if why != "" {
		o.setupWhy = why
		return o, func() {}
	}

	// The peer stamps both tokens with its own clock.
	createdAt := time.Now()
	issue := channelToken{
		channelID: channelID,
		tokenID:   issueTokenID,
		createdAt: createdAt,
		lifetime:  uint32(tokenLifetime / time.Millisecond),
	}
	renew := channelToken{
		channelID: channelID,
		tokenID:   renewTokenID,
		createdAt: createdAt,
		lifetime:  uint32(renewalLifetime / time.Millisecond),
	}
	peer, teardown, why := startPeer(peerScript{
		issue: issue,
		renew: renew,
		answer: func(req ua.Request, reqID uint32) ua.Response {
			if r, ok := req.(*ua.ReadRequest); ok {
				return readResponse(r, reqID)
			}
			return nil
		},
		// The hook maps both the arm's Reads to the old token: the
		// pre-expiry answer carries a token that is still live, the
		// scored answer the same token now expired - each chunk carries
		// its token id and, under Sign, its signature is made with its
		// keys - and the renewed token answers nothing, because §6.7.4
		// lets the client drop the old token on the first Message the
		// new one secures.
		answerToken: func(ua.Request) *channelToken {
			return &issue
		},
		codec: newSignCodec(peerKey, peerCert),
	})
	if why != "" {
		o.setupWhy = why
		return o, teardown
	}

	client, err := opcua.NewClient("opc.tcp://"+peer.addr(),
		opcua.SecurityPolicy(ua.SecurityPolicyURIBasic256Sha256),
		opcua.SecurityMode(ua.MessageSecurityModeSign),
		opcua.PrivateKey(clientKey),
		opcua.Certificate(clientCert),
		opcua.RemoteCertificate(peerCert),
	)
	if err != nil {
		o.setupWhy = fmt.Sprintf("build the client: %v", err)
		return o, teardown
	}

	if why, fromGopcua := openChannel(client, peer, dialBound); why != "" {
		o.setupWhy, o.setupFromGopcua = why, fromGopcua
		return o, teardown
	}

	// The client's own renewal is due at 75 % of the issue token's
	// lifetime; wait for it in the peer's records before the
	// pre-expiry Read, because that Read is the instrument check's
	// positive half only once the channel holds both tokens.
	renewalDeadline := issue.createdAt.Add(tokenLifetime * 3 / 4).
		Add(renewalSlack)
	for !o.renewalSeen && time.Now().Before(renewalDeadline) {
		for _, opn := range peer.record().OPNs {
			if opn.Type == ua.SecurityTokenRequestTypeRenew {
				o.renewalSeen = true
				break
			}
		}
		if !o.renewalSeen {
			time.Sleep(10 * time.Millisecond)
		}
	}

	graceEnd := issue.createdAt.Add(tokenLifetime).Add(tokenLifetime / 4)
	windowEnd := graceEnd.Add(graceWait)
	if o.renewalSeen {
		// The pre-expiry Read: the peer answers with the old token
		// while it is still live, and the bound ends inside the
		// window so the answer is never verified against a token
		// the client has wiped on its own schedule.
		preBound := time.Until(graceEnd.Add(-preReadMargin))
		o.preExpiryResponseArrived = readBodyArrived(client, preBound)
		if d := time.Until(windowEnd); d > 0 {
			time.Sleep(d)
		}
		// The scored Read: the same old token on the answer, now
		// past its lifetime plus the 25 % grace window.
		o.readsBefore = countReads(peer.record().Requests)
		o.scoredSent = true
		o.responseArrived = readBodyArrived(client, readBound)
	}

	closeCtx, cancel := context.WithTimeout(context.Background(), dialBound)
	_ = client.Close(closeCtx)
	cancel()

	if o.scoredSent {
		o.readArrived = countReads(peer.record().Requests) > o.readsBefore
	}
	o.setupWhy, o.setupFromGopcua = peer.setupCheck()
	return o, teardown
}

// countReads is how many of the peer's recorded requests are Reads.
func countReads(reqs []requestRecord) int {
	n := 0
	for _, r := range reqs {
		if _, ok := r.Req.(*ua.ReadRequest); ok {
			n++
		}
	}
	return n
}

// mintSigningCert mints a self-signed 2048-bit RSA key and X.509
// certificate in memory, for the common name it is given: the
// smallest key Basic256Sha256 accepts, whose 256-byte signatures
// keep every chunk in the policy's one-byte padding scheme. Nothing
// is written to disk and no key material is committed: every run
// mints its own pair for the client and its own for the peer. It
// returns the key, the DER certificate, and "" - or, when the
// minting failed, the reason the run cannot be scored. A minting
// failure is a decline ("could not set the test up"): it comes from
// the environment's random source, not from gopcua.
func mintSigningCert(commonName string) (key *rsa.PrivateKey,
	cert []byte, why string) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, fmt.Sprintf("generate a 2048-bit RSA key: %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: commonName},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage: x509.KeyUsageDigitalSignature |
			x509.KeyUsageKeyEncipherment | x509.KeyUsageDataEncipherment,
	}
	cert, err = x509.CreateCertificate(rand.Reader, tmpl, tmpl,
		&key.PublicKey, key)
	if err != nil {
		return nil, nil, fmt.Sprintf("create the certificate: %v", err)
	}
	return key, cert, ""
}

// signCodec is the chunkCodec of SecurityPolicy Basic256Sha256 in
// mode Sign. It is the peer's half of the policy's algorithms - the
// asymmetric signing and encryption of the OPN exchange, the
// symmetric signing of every other chunk - with keys minted in
// memory at test time. It mirrors the byte layout the client's own
// signAndEncrypt and verifyAndDecrypt produce
// (uasc/secure_channel_instance.go): a chunk's signature covers
// the whole chunk and sits at its end, and an OPN chunk's plaintext
// is padded to a block boundary before it is encrypted to the
// receiver's public key. No arm scores the signature bytes.
type signCodec struct {
	// peerKey is the peer's own key: it signs the OPN responses and
	// decrypts the client's OPN requests. peerCert is the DER
	// certificate those responses carry, from which the client
	// derives the peer's public key.
	peerKey  *rsa.PrivateKey
	peerCert []byte

	// clientKey and clientThumb are the client's public key and the
	// thumbprint of its certificate, learned from the first OPN it
	// sends: the OPN responses are encrypted to that key and stamped
	// with that thumbprint.
	clientKey   *rsa.PublicKey
	clientThumb []byte

	// mu guards clientKey, clientThumb and syms; the harness calls
	// the codec from its serve goroutines, one per connection, so a
	// client that drops the channel and dials again reaches the
	// codec on a second one. syms holds each token's symmetric keys,
	// keyed by token id.
	mu   sync.Mutex
	syms map[uint32]*uapolicy.EncryptionAlgorithm
}

// newSignCodec builds the codec for one arm's peer from the peer's
// minted key and certificate.
func newSignCodec(peerKey *rsa.PrivateKey, peerCert []byte) *signCodec {
	return &signCodec{
		peerKey:  peerKey,
		peerCert: peerCert,
		syms:     make(map[uint32]*uapolicy.EncryptionAlgorithm),
	}
}

// decodeRequest decodes one chunk the client sent: an OPN chunk is
// decrypted with the peer's key, its signature verified against the
// certificate its security header carries and its padding stripped;
// a MSG or CLO chunk is verified with the symmetric keys of the token
// its security header names. It returns the service request the
// chunk carries and the request id from its sequence header.
func (c *signCodec) decodeRequest(frame []byte) (
	ua.Request, uint32, error) {
	chunk := &uasc.MessageChunk{}
	if _, err := chunk.Decode(frame); err != nil {
		return nil, 0, err
	}
	if chunk.MessageType == uasc.MessageTypeOpenSecureChannel {
		return c.decodeOPN(chunk, frame)
	}
	return c.decodeSymmetric(chunk, frame)
}

// decodeOPN decodes one OpenSecureChannelRequest, the mirror of what
// the client's own receive path does to the peer's OPN response: the
// encrypted region is decrypted with the peer's private key, the
// signature is verified over the header plus the plaintext - the
// signature does not cover the plaintext alone - and the policy's
// padding is stripped before the sequence header and the service
// body are decoded.
func (c *signCodec) decodeOPN(chunk *uasc.MessageChunk,
	frame []byte) (ua.Request, uint32, error) {
	ash := chunk.AsymmetricSecurityHeader
	if ash.SecurityPolicyURI != ua.SecurityPolicyURIBasic256Sha256 {
		return nil, 0, fmt.Errorf("the OPN chunk's security header names %q, "+
			"not Basic256Sha256", ash.SecurityPolicyURI)
	}
	clientCert, err := uapolicy.ParseCertificate(ash.SenderCertificate)
	if err != nil {
		return nil, 0, fmt.Errorf("parse the client's certificate from "+
			"the OPN: %w", err)
	}
	clientKey, ok := clientCert.PublicKey.(*rsa.PublicKey)
	if !ok {
		return nil, 0, fmt.Errorf("the client's certificate carries a %T "+
			"key, not RSA", clientCert.PublicKey)
	}
	c.mu.Lock()
	c.clientKey = clientKey
	c.clientThumb = uapolicy.Thumbprint(ash.SenderCertificate)
	c.mu.Unlock()

	algo, err := uapolicy.Asymmetric(ua.SecurityPolicyURIBasic256Sha256,
		c.peerKey, clientKey)
	if err != nil {
		return nil, 0, err
	}

	headerLength := 12 + ash.Len()
	b := make([]byte, len(frame))
	copy(b, frame)
	plaintext, err := algo.Decrypt(b[headerLength:])
	if err != nil {
		return nil, 0, fmt.Errorf("decrypt the OPN chunk's "+
			"encrypted region: %w", err)
	}
	b = append(b[:headerLength], plaintext...)
	sigLength := algo.RemoteSignatureLength()
	if len(b) < headerLength+sigLength {
		return nil, 0, fmt.Errorf("the OPN chunk is %d bytes, shorter than "+
			"its headers and signature", len(b))
	}
	messageToVerify := b[:len(b)-sigLength]
	signature := b[len(b)-sigLength:]
	if err := algo.VerifySignature(messageToVerify, signature); err != nil {
		return nil, 0, fmt.Errorf("verify the OPN chunk's signature: %w", err)
	}

	// Strip the policy's padding: one PaddingSize byte under the
	// 2048-bit keys the row mints, whose signatures run to exactly
	// 256 bytes.
	paddingLength := int(messageToVerify[len(messageToVerify)-1]) + 1
	if paddingLength > len(messageToVerify)-headerLength {
		return nil, 0, fmt.Errorf("the OPN chunk's padding (%d bytes) "+
			"overruns its plaintext", paddingLength)
	}
	return decodeServiceBody(
		messageToVerify[headerLength : len(messageToVerify)-paddingLength])
}

// decodeSymmetric decodes one symmetric chunk - a MSG or a CLO -
// under Sign: the chunk is neither padded nor encrypted, and the
// signature is the policy's HMAC at its end, over everything before
// it. The keys are those of the token the chunk's security header
// names.
func (c *signCodec) decodeSymmetric(chunk *uasc.MessageChunk,
	frame []byte) (ua.Request, uint32, error) {
	tokenID := chunk.SymmetricSecurityHeader.TokenID
	c.mu.Lock()
	sym := c.syms[tokenID]
	c.mu.Unlock()
	if sym == nil {
		return nil, 0, fmt.Errorf("no symmetric keys minted for the "+
			"token id %d", tokenID)
	}
	sigLength := sym.RemoteSignatureLength()
	headerLength := 12 + chunk.SymmetricSecurityHeader.Len()
	if len(frame) < headerLength+sigLength {
		return nil, 0, fmt.Errorf("the chunk is %d bytes, shorter than "+
			"its headers and signature", len(frame))
	}
	signature := frame[len(frame)-sigLength:]
	if err := sym.VerifySignature(frame[:len(frame)-sigLength],
		signature); err != nil {
		return nil, 0, fmt.Errorf("verify the chunk's signature under "+
			"the token id %d: %w", tokenID, err)
	}
	return decodeServiceBody(frame[headerLength : len(frame)-sigLength])
}

// decodeServiceBody decodes the sequence header and the service body
// of one decrypted chunk, and returns the request it carries with
// the request id from its sequence header.
func decodeServiceBody(data []byte) (ua.Request, uint32, error) {
	seq := uasc.NewSequenceHeader(0, 0)
	n, err := seq.Decode(data)
	if err != nil {
		return nil, 0, fmt.Errorf("decode the sequence header: %w", err)
	}
	_, svc, err := ua.DecodeService(data[n:])
	if err != nil {
		return nil, 0, err
	}
	req, ok := svc.(ua.Request)
	if !ok {
		return nil, 0, fmt.Errorf("the chunk's body is %T, "+
			"not a service request", svc)
	}
	return req, seq.RequestID, nil
}

// encodeOPN encodes a final OPN chunk answering the
// OpenSecureChannelRequest req with the given token, sequence number
// and request id: it mints a fresh ServerNonce, derives the token's
// symmetric keys off that nonce and the ClientNonce the request
// carries - the pair the policy defines a token's keys by - and
// remembers them under the token's id for the symmetric chunks that
// follow. The chunk is padded to a plaintext block boundary, its size
// field fixed, signed with the peer's key and encrypted to the
// client's public key: the mirror of what the client's own send path
// does to its OPN request.
func (c *signCodec) encodeOPN(token channelToken,
	req *ua.OpenSecureChannelRequest, seq, reqID uint32) ([]byte, error) {
	c.mu.Lock()
	clientKey, clientThumb := c.clientKey, c.clientThumb
	c.mu.Unlock()
	if clientKey == nil {
		return nil, fmt.Errorf("the OPN to answer arrived before the " +
			"client's certificate was learned")
	}
	algo, err := uapolicy.Asymmetric(ua.SecurityPolicyURIBasic256Sha256,
		c.peerKey, clientKey)
	if err != nil {
		return nil, err
	}
	serverNonce, err := algo.MakeNonce()
	if err != nil {
		return nil, fmt.Errorf("mint the ServerNonce: %w", err)
	}
	sym, err := uapolicy.Symmetric(ua.SecurityPolicyURIBasic256Sha256,
		serverNonce, req.ClientNonce)
	if err != nil {
		return nil, fmt.Errorf("derive the token's symmetric keys: %w", err)
	}
	c.mu.Lock()
	c.syms[token.tokenID] = sym
	c.mu.Unlock()

	resp := &ua.OpenSecureChannelResponse{
		ResponseHeader:        responseHeader(reqID, ua.StatusOK),
		ServerProtocolVersion: 0,
		SecurityToken: &ua.ChannelSecurityToken{
			ChannelID:       token.channelID,
			TokenID:         token.tokenID,
			CreatedAt:       token.createdAt,
			RevisedLifetime: token.lifetime,
		},
		ServerNonce: serverNonce,
	}
	m := &uasc.Message{
		MessageHeader: &uasc.MessageHeader{
			Header: uasc.NewHeader(uasc.MessageTypeOpenSecureChannel,
				uasc.ChunkTypeFinal, token.channelID),
			AsymmetricSecurityHeader: uasc.NewAsymmetricSecurityHeader(
				ua.SecurityPolicyURIBasic256Sha256, c.peerCert, clientThumb),
			SequenceHeader: uasc.NewSequenceHeader(seq, reqID),
		},
		TypeID:  ua.NewFourByteExpandedNodeID(0, ua.ServiceTypeID(resp)),
		Service: resp,
	}
	b, err := m.Encode()
	if err != nil {
		return nil, err
	}

	// Pad, fix the size, sign and encrypt the way the production
	// sender does, so the client verifies and decrypts the way the
	// production receiver does. The minted 2048-bit keys keep the
	// one-byte padding scheme.
	headerLength := 12 + m.AsymmetricSecurityHeader.Len()
	plaintextBlockSize := algo.PlaintextBlockSize()
	remainder := (len(b[headerLength:]) + algo.SignatureLength() + 1) %
		plaintextBlockSize
	paddingLength := 0
	if remainder != 0 {
		paddingLength = plaintextBlockSize - remainder
	}
	for i := 0; i <= paddingLength; i++ {
		b = append(b, byte(paddingLength))
	}
	// The size field is fixed before the chunk is signed, so the
	// signature covers exactly the bytes the receiver verifies.
	encryptedLength := ((len(b[headerLength:]) + algo.SignatureLength()) /
		plaintextBlockSize) * algo.BlockSize()
	binary.LittleEndian.PutUint32(b[4:], uint32(headerLength+encryptedLength))
	signature, err := algo.Signature(b)
	if err != nil {
		return nil, err
	}
	b = append(b, signature...)
	encrypted, err := algo.Encrypt(b[headerLength:])
	if err != nil {
		return nil, err
	}
	return append(b[:headerLength], encrypted...), nil
}

// encodeMSG encodes a final MSG chunk carrying resp under the given
// channel id and token id, with the given sequence number and
// request id, signed with the symmetric keys of the token the id
// names. Under Sign the chunk is neither padded nor encrypted.
func (c *signCodec) encodeMSG(channelID, tokenID, seq, reqID uint32,
	resp ua.Response) ([]byte, error) {
	c.mu.Lock()
	sym := c.syms[tokenID]
	c.mu.Unlock()
	if sym == nil {
		return nil, fmt.Errorf("no symmetric keys minted for the "+
			"token id %d", tokenID)
	}
	m := &uasc.Message{
		MessageHeader: &uasc.MessageHeader{
			Header: uasc.NewHeader(uasc.MessageTypeMessage,
				uasc.ChunkTypeFinal, channelID),
			SymmetricSecurityHeader: uasc.NewSymmetricSecurityHeader(tokenID),
			SequenceHeader:          uasc.NewSequenceHeader(seq, reqID),
		},
		TypeID:  ua.NewFourByteExpandedNodeID(0, ua.ServiceTypeID(resp)),
		Service: resp,
	}
	b, err := m.Encode()
	if err != nil {
		return nil, err
	}
	// The size field is fixed before the chunk is signed, so the
	// signature covers exactly the bytes the receiver verifies.
	headerLength := 12 + m.SymmetricSecurityHeader.Len()
	binary.LittleEndian.PutUint32(b[4:],
		uint32(headerLength+len(b[headerLength:])+sym.SignatureLength()))
	signature, err := sym.Signature(b)
	if err != nil {
		return nil, err
	}
	return append(b, signature...), nil
}
