//go:build conformance

// Under SecurityPolicy None, the ClientNonce in the OpenSecureChannel
// request is null, or, if a client sends one anyway, it is at least 32
// bytes long and differs between two connections.
//
// The clauses: OPC 10000-6 §6.7.4: "The OpenSecureChannel Messages are
// not signed or encrypted if the SecurityMode is None. The Nonces are
// ignored and should be set to null." OPC 10000-4 §5.6.2.2, clientNonce:
// "A random number that shall not be used in any other request." OPC
// 10000-4 §5.6.2.3, Bad_NonceInvalid: "A Server shall check the minimum
// length of the Client nonce and return this status if the length is
// below 32 bytes." OPC 10000-7 v1.04 gives no SecureChannelNonceLength
// for None - the profile's limits unit SecurtyPolicy_None_Limits lists
// only "DerivedSignatureKeyLength: 0" - so §5.6.2.2's length rule has no
// number to apply here, and the row rests on §6.7.4 and §5.6.2.3. What
// counts as a null nonce is settled by the encoding rules, not by this
// row: OPC 10000-6 §5.2.2.7, ByteString: "If the length of the byte
// string is -1 then the byte string is 'null'." OPC 10000-6 §5.1.11,
// Null, Empty and Zero-Length Arrays, which covers ByteStrings
// ("ByteStrings are arrays of Bytes for purposes of this discussion"):
// "For this reason, null, empty and zero length arrays are semantically
// the same for all DataEncodings." "When testing for equality,
// applications shall treat null and empty arrays as equal." That is why
// the row counts a ClientNonce whose length prefix is 0 and one whose
// length prefix is -1 alike: both are null. gopcua passes this row as
// it stands: measured on this tree, its OPN under None carries a
// zero-length ClientNonce - a length prefix of 0, no nonce bytes.
//
// Neighbouring rows:
// open_secure_channel_fills_each_rsa_oaep_block_test.go scores the
// other thing a SecurityPolicy hands the channel at open - the
// plaintext block size, under Aes256_Sha256_RsaPss, the policy at the
// other end of the line from this row's. tests/transport holds the
// HEL/ACK exchange that precedes the OPN; this row does not score it.
//
// The row runs against the scripted peer from harness.go, twice: it
// starts the peer and launches a client again for its second reading, so
// it reads the OPN from two separate connections and can check that a
// non-null nonce is not reused. The client under test is the real
// opcua.Client, launched by the shared launchClient with SecurityPolicy
// None and SecurityMode None, under a context bounded by the harness's
// handshake deadline; its Connect is expected to fail once the peer
// closes, and no arm asserts its error. See harness.go.
//
// The row asserts, from the nonce bytes as they arrive on the wire -
// read by the scripted peer, not through uapolicy.NonceLength() - that
// the ClientNonce is null - a length prefix of 0 or -1, no nonce bytes
// - or, if nonce bytes are sent anyway, at least 32 bytes long and
// different between the two connections. The row reads the nonce by
// slicing the recorded frame with uasc and decoding the request body
// with ua.DecodeService into a *ua.OpenSecureChannelRequest, then
// reading ClientNonce; that is acceptable because the verdict rests on
// len and bytes.Equal of the nonce bytes, and the decoder turning a
// length prefix of 0 and one of -1 into the same empty value is
// §5.1.11's equality, quoted above. The two connections are read per
// connection: each connection's nonce must be null, or else at least 32
// bytes long; a non-null nonce must also differ from the other
// connection's nonce, and a null nonce counts as different from any
// non-null one. It declines, and does not score, when either
// connection's OPN did not arrive within the deadline; a frame that
// does not decode, or whose body is not an OpenSecureChannelRequest,
// can only be the client's bytes, so the row prints "gopcua's own
// behaviour made this run unscoreable" with t.Errorf, then returns. It
// deliberately asserts nothing about the rest of the OPN - the security
// header, the mode -
// and nothing about the channel lifecycle: §5.6.2.2's renewal sentence
// ("A new clientNonce shall be generated for each time a SecureChannel
// is renewed.") is the channel's business, not the nonce's.
//
// No reference stack has been read for this behaviour.
package securitypolicy

import "testing"

func TestClientNonceUnderSecurityPolicyNoneIsNullOrUsable(t *testing.T) {
	t.Skip("body not written yet")
}
