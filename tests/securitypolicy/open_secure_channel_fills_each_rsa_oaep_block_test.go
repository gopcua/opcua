//go:build conformance

// A client opening a channel under SecurityPolicy Aes256_Sha256_RsaPss in
// mode SignAndEncrypt encrypts its OpenSecureChannel request in
// RSA-OAEP-SHA2-256 blocks, each carrying the most plaintext the
// algorithm allows: k - 2hLen - 2 bytes, where k is the receiver's
// modulus length in octets and hLen is SHA-256's 32-byte output - 190
// for the 2048-bit receiver key the harness mints.
//
// The clauses: OPC 10000-7 v1.04 §6.6.166, Table 186 "SecurityPolicy -
// Aes256-Sha256-RsaPss", lists the conformance unit
// "AsymmetricEncryptionAlgorithm_RSA-OAEP-SHA2-256" (Optional = False),
// defined as "The RSA encryption algorithm which is defined in
// https://tools.ietf.org/html/rfc3447. The RSAES-OAEP scheme is used.
// The hash algorithm is SHA2 with 256 bits and is described in
// https://tools.ietf.org/html/rfc6234. The mask generation algorithm
// also uses SHA2 with 256 bits. The URI is
// http://opcfoundation.org/UA/security/rsa-oaep-sha2-256." OPC 10000-6
// §6.7.2.5.1: "During encryption, a block with a size equal to
// PlainTextBlockSize is processed to produce a block with size equal to
// CipherTextBlockSize. These values depend on the encryption algorithm
// and may be the same." RFC 3447 §7.1.1 step 1b: "If mLen > k - 2hLen -
// 2, output "message too long" and stop." Part 7 v1.04 is quoted because
// version 1.05 moved the policy tables to the profiles website; v1.04 is
// the last text form. The expected 190 is computed in the row from that
// arithmetic and the minted key's size, and never read from uapolicy:
// uapolicy's constant is the bug this row guards against - it reserves
// 64 bytes for each hash where RFC 3447 reserves 32
// (RSAOAEPMinPaddingSHA256 = (2 * 64) + 2, uapolicy/crypto_rsaoaep.go:21,
// subtracted at uapolicy/policyAes256Sha256RsaPss.go:114), which leaves
// every block carrying 126 plaintext bytes.
//
// Neighbouring rows:
// client_nonce_under_security_policy_none_is_null_or_usable_test.go
// scores the other thing a SecurityPolicy hands the channel at open -
// the client nonce, under SecurityPolicy None, the policy at the other
// end of the line from this row's. tests/transport holds the HEL/ACK
// exchange that precedes the OPN; this row does not score it.
//
// The row runs against the scripted peer from harness.go: it accepts one
// connection, answers the ACK, records the OPN frame, and closes without
// answering it. The client under test is the real opcua.Client,
// launched by the shared launchClient with
// opcua.SecurityPolicy(ua.SecurityPolicyURIAes256Sha256RsaPss),
// opcua.SecurityMode(ua.MessageSecurityModeSignAndEncrypt),
// opcua.PrivateKey and opcua.Certificate carrying the minted client
// key and certificate, and opcua.RemoteCertificate carrying the peer's
// minted certificate, under a context bounded by the harness's
// handshake deadline; its Connect is
// expected to fail once the peer closes, and no arm asserts its error.
// See harness.go.
//
// The row decrypts every block of the OPN's encrypted region with the
// standard library's crypto/rsa.DecryptOAEP under SHA-256 and the peer's
// minted key - never with gopcua code - and asserts that every block
// decrypts and carries exactly 190 plaintext bytes. The encrypted
// region is every byte of the OPN chunk after the asymmetric security
// header - what uasc.MessageChunk.Decode leaves in Data: the sequence
// header, the request, the padding and the signature, all encrypted
// together - and every k-byte slice of it is one block. A region that
// is empty or not a whole number of k-byte blocks is an ordinary
// failure, and that arm is the guard against a pass on zero blocks. It
// declines, and does not score, only when no OPN frame arrived within
// the deadline; an OPN whose asymmetric security header names a policy
// other than Aes256_Sha256_RsaPss, or on which
// uasc.MessageChunk.Decode fails, can only be the client's bytes, so
// the row prints "gopcua's own behaviour made this run unscoreable"
// with t.Errorf, then returns.
// It deliberately asserts nothing about the asymmetric signature
// (RSA-PSS), the symmetric algorithms, the channel lifecycle - tokens,
// renewal - or the HEL/ACK exchange.
//
// The row fails on main, so it carries a knownBug constant, declared in
// the test function's body together with the arms that read it. The
// known-bug arm keys on a presence: every block decrypts and carries
// exactly k - 2*64 - 2 = 126 bytes. The literals are 32 and 64, the two
// hash lengths; k is the peer key's modulus length in octets - the peer
// is the receiver - so the row computes both sizes from the peer key's
// Size(), and with the 2048-bit key the harness mints they come to 190
// and 126; neither is read from uapolicy: reading the
// constant would make the known-bug arm move with the fix. That
// presence is itself the misbehaviour, so per ../README.md the row
// carries no extra arm for it.
// Any other outcome - a block that does not decrypt, mixed sizes,
// another size - is an ordinary failure. All blocks carrying 190 passes,
// and on main that pass is the hard error asking for the constant's
// deletion.
//
// No reference stack has been read for this behaviour.
package securitypolicy

import "testing"

func TestOpenSecureChannelFillsEachRsaOaepBlock(t *testing.T) {
	t.Skip("body not written yet")
}
