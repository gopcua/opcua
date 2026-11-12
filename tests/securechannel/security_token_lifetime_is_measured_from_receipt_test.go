//go:build conformance

// A SecurityToken whose createdAt is stamped on a clock the client does
// not share does not stop the client using the channel: the client
// measures the token's life on its own clock, so a token stamped years
// in the past costs it nothing. A server whose real-time clock never
// got set - an embedded controller without a battery or NTP - stamps
// createdAt years back, and issue #816 is that machine: the client
// logged a token it had just received as expiring years earlier, then
// lost the response that followed.
//
// The clauses: OPC 10000-4 §5.6.2.2, response parameters, createdAt:
// "The time when the SecurityToken was created." and revisedLifetime:
// "The lifetime of the SecurityToken in milliseconds. The UTC
// expiration time for the token may be calculated by adding the
// lifetime to the createdAt time. The revised lifetime shall be used by
// the Client to renew a SecureChannel before it expires even if the
// MessageSecurityMode is NONE." That formula is arithmetic on one clock:
// createdAt is stamped by the server, the deadline is enforced by the
// client, and nothing in the clause makes the two the same clock. OPC
// 10000-4 §5.6.2.1's allowances are proportions of the lifetime, not
// instants: "Clients should request a new SecurityToken after 75 % of
// its lifetime has elapsed." and "Clients should accept Messages
// secured by an expired SecurityToken for up to 25 % of the token
// lifetime." Elapsed, for a client, can only mean elapsed on the
// client's clock.
//
// Neighbouring rows: expired_security_token_is_rejected_test.go scores
// the other end of a token's life - what the client owes a token that
// has expired, where this row scores what a token stamped on a foreign
// clock must not cost it; service_fault_leaves_the_channel_open_test.go
// and received_sequence_number_is_verified_test.go score the channel's
// answer to a fault and to a bad chunk, where this row scores its
// treatment of a well-formed answer arriving after a skewed stamp.
//
// The row runs against the scripted peer from harness.go, twice - a
// fresh peer and a fresh client each time, under SecurityPolicy None.
// Each arm's peer hands out a token with a ten-minute lifetime - so
// neither arm is near a genuine expiry - carrying the channel id and
// the token id 15, and answers the client's CreateSession with a
// ServiceFault after a delay applied to both arms alike, so the delay
// cannot be what separates them, and answers any Read that arrives
// with the harness's readResponse. No Read arrives in this story -
// the fault answers CreateSession, and the client's own Read of the
// namespace array is sent only after a session is activated - so the
// case stands guard for a client that reads earlier. The channel id
// and the token id being the same number is load-bearing: gopcua's
// expiration wipes the channel instance under the token id
// (uasc/secure_channel.go:952-954) while every arriving response is
// looked up under the channel id (uasc/secure_channel.go:579,
// uasc/secure_channel.go:603), so the premature wipe only lands on
// the instance a response needs when a server assigns both the same
// number - which is what the open62541 server in issue #816 did,
// which is why its response was dropped.
// The control arm stamps createdAt at the present moment, which is
// what an ordinary server does, and establishes that the instrument
// can see a reply arrive at all; the skewed arm stamps createdAt six
// years back, and the two arms must agree. The client under test is
// the real opcua.Client, launched by the row with a request timeout
// of its own; its Connect is expected to return the injected fault,
// and the fault it returns is compared with the injected status
// through the harness's isServiceFault.
// See harness.go.
//
// The row asserts that the skewed arm's client reports the injected
// ServiceFault status exactly as the control arm's does, and nothing
// else: what the client does with the fault is not scored, and neither
// is the fault's meaning - it was chosen as a distinctive,
// promptly-delivered reply that needs no forged session state to be
// well formed. It declines, and does not score, when the fixture
// fails to deliver its half of the exchange - the peer's own failure
// - per the harness's setupCheck and the row's own check. A frame
// the harness flags as the client's own bytes is not a decline: the
// setup check reports it as gopcua's own behaviour, and the row
// scores the run as gopcua's failure with t.Errorf, never declines
// it. No CreateSession arriving is a decline on the control arm
// only, the story never having reached the client whose reply the
// instrument check needs; on the skewed arm whatever happens is
// scored, and a skewed run in which no CreateSession arrives is
// scored as gopcua's failure,
// because the arm has no decline of its own. The control arm is the
// instrument check: a run in which it does not see the reply proves
// nothing about the skewed arm, and fails as a broken fixture, not
// as a score. The row passes on main since PR #911: scheduleExpiration
// no longer takes the token's timer from the server's createdAt.
// It measures the token's remaining life on the client's own clock,
// scheduling the wipe for revisedLifetime x 1.25 counted from receipt
// (uasc/secure_channel.go:936, when := time.Second * time.Duration(
// instance.revisedLifetime.Seconds()*expireAfter)) and running that
// duration on a plain timer (uasc/secure_channel.go:940, t :=
// time.NewTimer(when)), so a stamp six years back costs the skewed
// arm nothing within the run: both arms' clients return the injected
// fault, and that agreement is the outcome the row scores.
//
// No reference stack has been read for this behaviour; the measurement
// on main before and after PRs #911 and #904 merged is the row's own.
package securechannel

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/ua"
)

// faultStatus is the ServiceFault status the row's script answers the
// client's CreateSession with. It is Bad, so it travels as a
// ServiceFault per OPC 10000-4 §7.34, and no path in the client
// generates it on its own - the skewed arm's lost reply shows up as
// Bad_Timeout - so a returned fault is unambiguously the injected one.
var faultStatus = ua.StatusBadTooManyOperations

// createSessionDelay is how long the peer holds the ServiceFault back
// before sending it, on both arms alike so it cannot be what separates
// them. The skewed arm's premature wipe is scheduled the moment the
// OPN answer is handled and fires microseconds later; the delay holds
// the answer past that wipe on any machine, so the reply always meets
// the wiped instance map on an unfixed tree and the outcome never
// rests on the race between the two.
const createSessionDelay = 500 * time.Millisecond

// requestTimeout is the request timeout the row launches its client
// with. It sits well above createSessionDelay, so the control arm's
// delayed fault is never mistaken for a lost reply, and low enough
// that the skewed arm's lost reply comes back as its own Bad_Timeout
// in seconds rather than the ten-second default.
const requestTimeout = 2 * time.Second

// armBound is the context deadline the row bounds each arm's Connect
// with: above the skewed arm's own latest scoring return - the
// request timeout plus uasc's 250ms leniency - with room for a loaded
// machine, so it never cuts a scoring run short, and short enough
// that a client wedged past its own timers cannot hold the row.
const armBound = 5 * time.Second

func TestSecurityTokenLifetimeIsMeasuredFromReceipt(t *testing.T) {
	// The control arm is the instrument check, and a run in which it
	// never saw the reply proves nothing about the skewed arm: its two
	// not-seeing-the-reply failures below - no CreateSession arriving,
	// a Connect return other than the injected fault - are declines,
	// whichever side caused them.
	control, teardown := runArm(time.Now())
	t.Cleanup(teardown)
	if control.setupWhy != "" {
		if control.setupFromGopcua {
			t.Errorf("gopcua's own behaviour made this run unscoreable: %s",
				control.setupWhy)
			return
		}
		t.Fatalf("could not set the test up: %s", control.setupWhy)
	}
	if !control.createSession {
		t.Fatalf("could not set the test up: no CreateSession arrived on " +
			"the control arm, so the client whose reply the instrument check " +
			"needs never sent it")
	}
	if !isServiceFault(control.connectErr, faultStatus) {
		t.Fatalf("could not set the test up: the control arm's client "+
			"returned %v, not the injected ServiceFault %s, so the instrument "+
			"check never ran and the skewed arm's run proves nothing",
			control.connectErr, faultStatus)
	}

	// The skewed arm has no decline of its own: whatever a client does
	// with a backdated token is the behaviour the row scores.
	skewed, teardown := runArm(time.Now().Add(-6 * 365 * 24 * time.Hour))
	t.Cleanup(teardown)
	if skewed.setupWhy != "" {
		if skewed.setupFromGopcua {
			t.Errorf("gopcua's own behaviour made this run unscoreable: %s",
				skewed.setupWhy)
			return
		}
		t.Fatalf("could not set the test up: %s", skewed.setupWhy)
	}

	skewFault := isServiceFault(skewed.connectErr, faultStatus)

	switch {
	case !skewed.createSession:
		t.Errorf("the skewed arm's client returned %v without ever "+
			"sending a CreateSession. The skewed arm has no decline of its "+
			"own: whatever a client does with a backdated token that stops "+
			"the client short of the story's one request is scored as "+
			"gopcua's failure", skewed.connectErr)
	case skewFault:
		t.Logf("the skewed arm's client returned the injected ServiceFault "+
			"%s exactly as the control arm's client did", faultStatus)
	default:
		t.Errorf("the skewed arm's client returned %v where the control "+
			"arm's client returned the injected ServiceFault %s",
			skewed.connectErr, faultStatus)
	}
}

// armOutcome is what one arm's run left within the row's reach. It
// holds no handle on the peer: runArm reads the peer's records once,
// after the arm's run is over, so the outcome is one instant of the
// run and the row never races a live connection for it.
type armOutcome struct {
	connectErr      error
	createSession   bool
	setupWhy        string
	setupFromGopcua bool
}

// runArm runs one arm of the row end to end: a fresh peer handing
// every connection the token stamped with createdAt - the channel id
// and the token id both 15, the ten-minute lifetime the header
// names - and a fresh client launched under SecurityPolicy None with
// the row's request timeout. runArm drives the client's Connect to its
// return within armBound. The script answers the client's
// CreateSession with a ServiceFault carrying faultStatus after
// createSessionDelay, answers any Read that arrives with the
// harness's readResponse - no Read arrives in this story, the fault
// answering CreateSession first - and answers nothing else. It
// returns what the arm's run left to score and the peer's teardown
// for the row to register with t.Cleanup; a peer that could not be
// started or a client that could not be built is a decline whichever
// side caused it, reported through o.setupWhy and printed by the row
// with "could not set the test up", per ../README.md.
func runArm(createdAt time.Time) (o armOutcome, teardown func()) {
	token := channelToken{
		channelID: 15,
		tokenID:   15,
		createdAt: createdAt,
		lifetime:  600000, // ten minutes, in milliseconds
	}
	peer, teardown, why := startPeer(peerScript{
		issue: token,
		renew: token,
		answer: func(req ua.Request, reqID uint32) ua.Response {
			switch r := req.(type) {
			case *ua.CreateSessionRequest:
				time.Sleep(createSessionDelay)
				return &ua.ServiceFault{
					ResponseHeader: responseHeader(reqID, faultStatus),
				}
			case *ua.ReadRequest:
				return readResponse(r, reqID)
			}
			return nil
		},
	})
	if why != "" {
		o.setupWhy = why
		return o, teardown
	}

	client, err := opcua.NewClient("opc.tcp://"+peer.addr(),
		opcua.SecurityPolicy(ua.SecurityPolicyURINone),
		opcua.SecurityMode(ua.MessageSecurityModeNone),
		opcua.RequestTimeout(requestTimeout),
	)
	if err != nil {
		o.setupWhy = fmt.Sprintf("build the client: %v", err)
		return o, teardown
	}

	ctx, cancel := context.WithTimeout(context.Background(), armBound)
	defer cancel()
	o.connectErr = client.Connect(ctx)
	_ = client.Close(ctx)

	for _, r := range peer.record().Requests {
		if _, ok := r.Req.(*ua.CreateSessionRequest); ok {
			o.createSession = true
			break
		}
	}
	o.setupWhy, o.setupFromGopcua = peer.setupCheck()
	return o, teardown
}
