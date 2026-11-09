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
// separates them.
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
// declines, and does not score, when the fixture fails its half - the
// peer's own failure, a frame the harness flags as the client's own
// bytes, the channel never opening, the peer's OPN records holding no
// renewal: the records hold one
// entry per OpenSecureChannelRequest the peer answered, with the
// connection generation it arrived on, whether it asked for a new
// token or renewed the current one, and the moment it arrived, and
// without a renewal there is no new token id on the same channel
// for the scored answer to be old against - and it scores, never
// declines, a client whose own behaviour leaves the scored arm
// unscoreable. The row's Read never reaching the peer is scored by
// arm: on the scored arm it is scored as gopcua's failure - unhandled
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
// id, after which both arms reject and pass; measured on main and on
// the candidate fixes #911 and #904 the scored arm still fails,
// because neither touches the expiration's keying. When the constant
// stands, the scored arm's acceptance is logged as the known bug; when
// both arms reject, that is a hard error telling whoever runs it to
// delete the constant, every arm that reads it, and anything only
// those arms used.
//
// No reference stack has been read for this behaviour; the measurement
// on main and on the two candidate fixes is the row's own.
package securechannel

import "testing"

func TestExpiredSecurityTokenIsRejected(t *testing.T) {
	t.Skip("body not written yet")
}
