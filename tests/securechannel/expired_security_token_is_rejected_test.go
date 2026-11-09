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
// drew them and the ServerNonce of the response that carried them,
// so the harness hands the codec the decoded OPN, and the row's codec
// mints each token's keys off that pair. The row runs in two arms that
// differ only in the ids the peer hands out, each arm on a peer of its
// own with a client of its own. Each arm's client is the real
// opcua.Client, driven through Dial and no further: Connect would send
// CreateSession, ActivateSession and a Read of its own, each of
// which would consume answers the script has not chosen for it, and
// the row's one scored request is a raw Read it sends and bounds
// itself. The peer hands out a short token and answers the client's
// own renewal with a new token id on the same channel - a number
// the issue token never carried, so the channel holds two tokens to
// tell apart - and the row's Read goes out only after the old token's
// lifetime plus 25 % has passed on the client's clock, so the
// stimulus is expired on any tree that measures the window correctly.
// The peer answers that Read with a
// response chunk whose symmetric security header carries the old,
// expired token id and whose signature is made with the old token's
// keys, chosen with the harness's answerToken script: the scored
// arm's server assigned a channel id that differs from the token
// id, the control arm's the same number, and the ids are all that
// separates them. The arm then sends a second Read, which the peer
// answers with a chunk secured with the renewed token - the first
// Message from the server secured with the new SecurityToken, the
// event OPC 10000-6 §6.7.4 names - and its body must reach the
// caller on both arms: on the control arm it is the instrument
// check's positive half, a client that accepts a chunk the live
// token secures, so its rejection of the old-token chunk is
// grounded in the token; on the scored arm it is the precondition
// for the deletion demand, because a client that rejects every
// chunk - a channel dead for every token, against the continuation
// duty §6.7.4 asks - would otherwise read the same as the fix. A
// client that closes the channel on the failed check, as §6.7.7
// orders, leaves the renewed answer undelivered too, and the row
// declines it: the demand is earned by a fix that keeps the
// channel alive, as the candidate rekeying does.
// See harness.go.
//
// The row asserts that the scored arm's client does not hand the
// old-token response to its caller - per the README's definition of
// the phrase, the response body never arrives at the caller; on
// these unfixed trees that shows as the arm's Read returning its own
// timeout error, and a fixed client returns the closed channel's
// error instead, which is why the arm keys on the body - and that
// the control arm's client does reject it: the control arm is the
// instrument check - on this very tree the channel id = token id
// arm's expiration happens to remove the right instance, so it shows
// that the instrument can see a rejection at all, and a run in which
// it does not see one proves nothing about the scored arm. It
// declines, and does not score, when the fixture fails its half or
// a stage the story needs never ran, whichever side caused it -
// the peer's own failure, the channel never opening, the peer's OPN
// records holding no renewal, the renewed-token answer never
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
// for the scored answer to be old against. The row's Read never
// reaching the peer is scored by arm: on the scored arm it is scored
// as gopcua's failure - unhandled
// it would read as a rejection, the body never arriving, a false
// pass, and the harness's setupCheck already holds the peer's half
// clean, so a Read that never arrives can only be the client's own
// doing - and on the control arm it is a decline, because the
// instrument check never ran: a control run whose Read never reached
// the peer has no rejection to see, and proves nothing about the
// scored arm. The knownBug
// constant, for issue #941, records that gopcua does not do this yet:
// scheduleExpiration empties and rebuilds the instance map under the
// TOKEN id (uasc/secure_channel.go:888, uasc/secure_channel.go:890,
// uasc/secure_channel.go:900) while every arriving chunk is looked up
// under the CHANNEL id (uasc/secure_channel.go:536,
// uasc/secure_channel.go:556), so when the server assigns the two
// different numbers the expired instance is never removed and the
// old-token signature still verifies. The knownBug arm is the scored
// arm accepting the old-token response, a presence a fixed client does
// not show - the control arm passing shows the row is not scoring an
// instrument that rejects nothing - and what deletes the constant is
// keying scheduleExpiration by the channel id instead of the token
// id, after which both arms reject the old-token response and hand
// the renewed-token answer to their callers, and only such a run
// carries the demand - a client that rejects both answers is a
// decline, never a fix; measured on main and on
// the candidate fixes #911 and #904 the scored arm still logs the
// known bug, because neither touches the expiration's keying. When
// the constant stands, the scored arm's acceptance is logged as the
// known bug; when both arms reject, that is a hard error telling
// whoever runs it to delete the constant, every arm that reads it,
// and anything only those arms used.
//
// No reference stack has been read for this behaviour; the measurement
// on main and on the two candidate fixes is the row's own.
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
	"sync/atomic"
	"testing"
	"time"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/ua"
	"github.com/gopcua/opcua/uapolicy"
	"github.com/gopcua/opcua/uasc"
)

// tokenLifetime is the short lifetime the peer's issue token carries.
// Measured on this tree the client renews at half of it and wipes
// the expired instance at its end - the 75 % and 125 % factors
// truncate to whole seconds in uasc - so the arm's Read goes out
// 900 ms past the wipe, not the 400 ms graceWait sits past the
// window the specification names, and the stimulus is expired on
// any tree that measures the window correctly either way. Two
// seconds keep the arm's
// whole wait short while leaving half a lifetime between the renewal
// and the expiry, so a renewal that runs late on a loaded machine
// still lands before the window the Read waits out.
const tokenLifetime = 2 * time.Second

// renewalLifetime is the lifetime the renewal token carries: a full
// minute, so neither the renewed token's own renewal at 75 % nor its
// expiry at 125 % falls inside the arm.
const renewalLifetime = time.Minute

// graceWait is how far past the issue token's lifetime plus its 25 %
// grace window the arm waits before it sends the scored Read, and
// the window is waited out from the stamp the peer put on the
// token: the unfixed tree schedules the expiration from that stamp,
// and a tree that measures from receipt ends its window a handshake
// later, which the same wait covers. The wipe is a timer on a
// goroutine, and the wait gives the goroutine time to fire and its
// wipe to land on a loaded machine; a delay that outruns the wait
// on the scored arm alone logs the known bug on a fixed tree, which
// the row cannot tell apart.
const graceWait = 400 * time.Millisecond

// dialBound bounds each arm's Dial and the client's close at the
// arm's end: above the milliseconds a healthy HEL/ACK and OPN
// exchange takes, so a loaded machine finishes inside it, and short
// enough that a channel wedged past its own exchange cannot hold the
// arm.
const dialBound = 10 * time.Second

// readBound is the bound on the arm's Reads. On these unfixed trees
// a rejection shows as the Read returning this bound's own timeout
// error - a rejected chunk reaches no handler - so the bound is
// what a rejected answer costs the arm; a healthy answer crosses
// the loopback in milliseconds, and one slower than the bound is
// indistinguishable from a rejection on both arms. readBound sits
// under the harness's read bound (harness.go), so the peer's
// connection is never released for idling before the Read returns.
const readBound = 3 * time.Second

// The token ids the peer hands out: the issue token's id, and the
// new id the renewal carries - a number the issue token never had,
// so the channel holds two tokens to tell apart.
const (
	issueTokenID = 15
	renewTokenID = 16
)

// The channel ids the arms differ in: the control arm's server
// assigns the channel id and the token id the same number, so on
// this very tree the expiration happens to remove the right
// instance, and the scored arm's assigns the two different numbers.
// The ids are all that separates the arms.
const (
	controlChannelID = 15
	scoredChannelID  = 20
)

func TestExpiredSecurityTokenIsRejected(t *testing.T) {
	// knownBug records that gopcua does not do this yet; the text is
	// logged in place of a failure. ../README.md says when to delete
	// it.
	const knownBug = "scheduleExpiration empties and rebuilds the " +
		"instance map under the token id while every arriving chunk is " +
		"looked up under the channel id, so a server that assigns the " +
		"two different numbers leaves the expired instance in place and " +
		"the old-token signature still verifies (issue #941)"

	// The control arm first: the server assigns the channel id and
	// the token id the same number, and on this very tree the
	// expiration happens to remove the right instance, so the arm
	// is the instrument check - a run in which it sees no
	// rejection proves nothing about the scored arm. Its four
	// declines below - no renewal in the peer's records, the
	// arm's Read never reaching the peer, the old-token response
	// handed to the caller, the renewed-token answer not handed
	// to it - each leaves the check unrun or its rejection
	// ungrounded, whichever side caused them.
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
	if !control.readArrived {
		t.Fatalf("could not set the test up: the control arm's Read never " +
			"reached the peer, so the instrument check never ran and this " +
			"run proves nothing about the scored arm")
	}
	if control.responseArrived {
		t.Fatalf("could not set the test up: the control arm's client handed " +
			"the old-token response to its caller, so the instrument cannot " +
			"see a rejection at all and this run proves nothing about the " +
			"scored arm")
	}
	if !control.renewedResponseArrived {
		t.Fatalf("could not set the test up: the control arm's client did " +
			"not hand the renewed-token answer to its caller, so the " +
			"instrument check never ran")
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
	if !scored.readArrived {
		t.Errorf("the scored arm's Read never reached the peer, so the arm " +
			"scored nothing: the harness's setup check reports no " +
			"failure on the peer's side, so the client's own behaviour " +
			"stopped it, and that is scored as gopcua's failure, never " +
			"declined")
		return
	}

	switch {
	case scored.responseArrived && knownBug != "":
		t.Logf("known bug: %s (the scored arm's client, whose server "+
			"assigned the channel id %d and the token id %d, handed its "+
			"caller the response chunk stamped with the old, expired "+
			"token id %d and signed with its keys, while the control "+
			"arm's client, whose server assigned both the same number, "+
			"rejected its own)",
			knownBug, scoredChannelID, issueTokenID, issueTokenID)
	case scored.responseArrived:
		t.Errorf("the scored arm's client handed the old-token response to " +
			"its caller, where the control arm's client rejected its own, " +
			"and no knownBug constant stands")
	case !scored.renewedResponseArrived:
		t.Fatalf("could not set the test up: the scored arm's client did " +
			"not hand the renewed-token answer to its caller, so its " +
			"rejection of the old-token response cannot be told from a " +
			"channel dead for every token, and this run cannot be scored")
	case knownBug != "":
		t.Errorf("this now passes (both arms rejected the old-token " +
			"response, so its body never arrived at either caller, and " +
			"both carried the renewed-token answer, so neither channel " +
			"was dead) - delete the knownBug constant, every arm that " +
			"reads it, and anything only those arms used, in the commit " +
			"that fixed it")
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
	responseArrived        bool
	renewedResponseArrived bool
	renewalSeen            bool
	readArrived            bool
	setupWhy               string
	setupFromGopcua        bool
}

// runRejectionArm runs one arm of the row end to end: it mints the
// client's and the peer's keys in memory, starts a fresh peer
// speaking Basic256Sha256 in mode Sign through the signCodec below.
// The peer hands out an issue token carrying the channel id it is
// given and the token id issueTokenID and answers the client's own
// renewal with the token id renewTokenID on the same channel; the
// arm launches a fresh client - the real opcua.Client, driven
// through Dial and no further. The arm waits out the issue token's
// lifetime plus its 25 % grace window, then sends two Reads, each
// bounded by readBound: the peer answers the first with a
// response chunk stamped with the old, expired token and signed
// with the old token's keys and the second with a chunk secured
// with the renewed token, mapped by the script's answerToken
// hook. It returns what the arm's run left to score and the
// peer's teardown for the row to register with t.Cleanup; keys
// that could not be minted, a peer that could not be started or
// a client that could not be built is a decline whichever side
// caused it, reported through o.setupWhy and printed by the row
// with "could not set the test up", per ../README.md. The runner
// never aborts the test; it returns plain values.
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
	// readAnswers is counted on a serve goroutine, so it must be atomic.
	var readAnswers atomic.Uint32
	peer, teardown, why := startPeer(peerScript{
		issue: issue,
		renew: renew,
		answer: func(req ua.Request, reqID uint32) ua.Response {
			if r, ok := req.(*ua.ReadRequest); ok {
				return readResponse(r, reqID)
			}
			return nil
		},
		// The hook maps the arm's Reads to their tokens: the scored
		// answer is stamped with the old, expired token - the chunk
		// carries its token id and, under Sign, its signature is made
		// with its keys - and the second Read's answer with the
		// renewed one.
		answerToken: func(ua.Request) *channelToken {
			if readAnswers.Add(1) == 1 {
				return &issue
			}
			return &renew
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

	dialCtx, cancel := context.WithTimeout(context.Background(), dialBound)
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

	windowEnd := issue.createdAt.Add(tokenLifetime).
		Add(tokenLifetime / 4).Add(graceWait)
	if d := time.Until(windowEnd); d > 0 {
		time.Sleep(d)
	}

	// The client's own renewal has had the whole window to arrive.
	for _, opn := range peer.record().OPNs {
		if opn.Type == ua.SecurityTokenRequestTypeRenew {
			o.renewalSeen = true
			break
		}
	}
	if o.renewalSeen {
		read := func() bool {
			readCtx, cancel := context.WithTimeout(context.Background(), readBound)
			res, _ := client.Read(readCtx, &ua.ReadRequest{
				NodesToRead: []*ua.ReadValueID{{
					NodeID:      ua.NewNumericNodeID(0, 2255),
					AttributeID: ua.AttributeIDValue,
				}},
			})
			cancel()
			// The arm keys on the body, never on which error the Read
			// returned along the way.
			return res != nil
		}
		o.responseArrived = read()
		o.renewedResponseArrived = read()
	}

	closeCtx, cancel := context.WithTimeout(context.Background(), dialBound)
	_ = client.Close(closeCtx)
	cancel()

	for _, r := range peer.record().Requests {
		if _, ok := r.Req.(*ua.ReadRequest); ok {
			o.readArrived = true
		}
	}
	o.setupWhy, o.setupFromGopcua = peer.setupCheck()
	return o, teardown
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
		return nil, 0, fmt.Errorf(
			"parse the client's certificate from the OPN: %w", err)
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
		return nil, 0, fmt.Errorf(
			"decrypt the OPN chunk's encrypted region: %w", err)
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
		return nil, 0, fmt.Errorf(
			"no symmetric keys minted for the token id %d", tokenID)
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
		return nil, 0, fmt.Errorf(
			"the chunk's body is %T, not a service request", svc)
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
		return nil, fmt.Errorf(
			"no symmetric keys minted for the token id %d", tokenID)
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
