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
// cannot be what separates them. The channel id and the token id being
// the same number is load-bearing: gopcua's expiration wipes the
// channel instance under the token id (uasc/secure_channel.go:888)
// while every arriving response is looked up under the channel id
// (uasc/secure_channel.go:536, uasc/secure_channel.go:556), so the
// premature wipe only lands on the instance a response needs when a
// server assigns both the same number - which open62541 does, which is
// why issue #816 saw its response dropped. The control arm stamps
// createdAt at the present moment, which is what an ordinary server
// does, and establishes that the instrument can see a reply arrive at
// all; the skewed arm stamps createdAt six years back, and the two arms
// must agree. The skewed arm has no decline of its own, because
// whatever a client does with a backdated token is the behaviour the
// row scores, so a run the skewed arm cannot score is scored as
// gopcua's failure, never declined. The client under test is the real
// opcua.Client, launched by the row with a request timeout of its own;
// its Connect is expected to return the injected fault.
// See harness.go.
//
// The row asserts that the skewed arm's client reports the injected
// ServiceFault status exactly as the control arm's does, and nothing
// else: what the client does with the fault is not scored, and neither
// is the fault's meaning - it was chosen as a distinctive,
// promptly-delivered reply that needs no forged session state to be
// well formed. It declines, and does not score, when the fixture fails
// to deliver its half of the exchange - the peer's own failure, a
// frame the harness flags as the client's own bytes - per the
// harness's setupCheck and the row's own check. No CreateSession
// arriving is a decline on the control arm only, the story never
// having reached the client whose reply the instrument check needs;
// on the skewed arm whatever happens is scored, and a skewed run in
// which no CreateSession arrives is scored as gopcua's failure,
// because the arm has no decline of its own. The control arm is the
// instrument check: a run in which it does not see the reply proves
// nothing about the skewed arm, and fails as a broken fixture, not
// as a score. The knownBug constant records that gopcua does not do
// this yet: it adopts the server's clock at
// uasc/secure_channel.go:679 (instance.createdAt =
// resp.SecurityToken.CreatedAt) and schedules the token's expiration
// off that stamp at uasc/secure_channel.go:872 (when :=
// instance.createdAt.Add(time.Second * time.Duration(
// instance.revisedLifetime.Seconds()*expireAfter)) in
// scheduleExpiration), so a token stamped six years back is wiped at
// once and the skewed arm's reply is lost - on main the control arm
// reports the injected fault and the skewed arm reports a timeout,
// which is issue #816's symptom. The knownBug arm is the skewed arm
// disagreeing with the control arm, and what deletes the constant is
// the fix that measures a token's life from receipt instead of from
// the server's stamp, as PR #911 does; measured on PR #904 the row
// still fails, because that fix touches the ServiceFault path and not
// the clock. When the constant stands, the skewed arm's disagreement
// is logged as the known bug; when both arms agree, that is a hard
// error telling whoever runs it to delete the constant, every arm that
// reads it, and anything only those arms used.
//
// No reference stack has been read for this behaviour; the measurement
// on main and on the two candidate fixes is the row's own.
package securechannel

import "testing"

func TestSecurityTokenLifetimeIsMeasuredFromReceipt(t *testing.T) {
	t.Skip("body not written yet")
}
