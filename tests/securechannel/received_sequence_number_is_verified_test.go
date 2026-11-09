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
// The row runs against the scripted peer from harness.go, under
// SecurityPolicy None, with the client under test built by the row as
// the real opcua.Client with AutoReconnect(false), so an error return
// is the whole observable. The peer answers the OPN with a response
// whose SequenceNumber is 1 - its first chunk on the channel - then
// answers the client's first request with a response chunk whose
// SequenceNumber is 1 again, a reuse within the same TokenId, and the
// client's second request with one whose SequenceNumber is 3, skipping
// 2, using the harness's msgSeqs script.
// See harness.go.
//
// The row asserts that the client does not hand the reused-number
// response to its caller, and that it does not hand the skipped-number
// response to its caller either: a receiver with no sequence check at
// all accepts both, so the second arm separates a partial check from
// none. It deliberately asserts nothing about which error the client
// returns, and nothing about whether it reconnects: with
// AutoReconnect(false) and no monitor, the error return is the whole
// observable, and the clause's "force a reconnect" is scored as the
// transport error the client raises, not as a dialling behaviour. It
// declines, and does not score, when the fixture fails its half - the
// peer's own failure, a frame the harness flags as the client's own
// bytes, the channel never opening, the client never sending the
// request whose answer is scored - and it scores, never declines, a
// client whose own behaviour leaves the run unscoreable. The knownBug
// constant records that gopcua does not do this yet, and no pull
// request fixes it: the receive path decodes the sequence header at
// uasc/secure_channel.go:520 (n, err := m.SequenceHeader.Decode(m.Data))
// but never compares the number it carries, a single-chunk message is
// returned without any look at it (uasc/secure_channel.go:1231), and
// the only read of a received SequenceNumber silently skips a
// duplicate within one chunked message instead of raising the
// transport error (uasc/secure_channel.go:1238), so a reused number
// and a skipped number are both accepted on every tree measured. The
// knownBug arm is the caller being handed a response whose
// SequenceNumber was reused or skipped - a presence a fixed client does
// not show - and what deletes the constant is a fix that verifies the
// SequenceNumber of every received chunk and raises the transport
// error; there is no pull request to name. When the constant stands,
// the acceptance is logged as the known bug; when both responses are
// rejected, that is a hard error telling whoever runs it to delete the
// constant, every arm that reads it, and anything only those arms
// used.
//
// No reference stack has been read for this behaviour; the measurement
// on the trees this suite was authored against is the row's own.
package securechannel

import "testing"

func TestReceivedSequenceNumberIsVerified(t *testing.T) {
	t.Skip("body not written yet")
}
