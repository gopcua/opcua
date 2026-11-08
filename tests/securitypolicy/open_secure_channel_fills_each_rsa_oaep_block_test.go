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
// declines, and does not score, when no OPN frame arrived within
// the deadline, the harness's `handshake` constant as the wait - that
// is the row's own decline, and the harness's setup declines (minting,
// starting the peer, building the client) apply as well; an OPN
// whose asymmetric security header names a policy other than
// Aes256_Sha256_RsaPss, or on which
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

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"testing"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/ua"
	"github.com/gopcua/opcua/uasc"
)

func TestOpenSecureChannelFillsEachRsaOaepBlock(t *testing.T) {
	// knownBug records that gopcua does not do this yet; the text is logged
	// in place of a failure. ../README.md says when to delete it. It carries
	// no issue link.
	const knownBug = "uapolicy reserves 64 bytes for each SHA-256 hash " +
		"where RFC 3447 §7.1.1 reserves 32 (RSAOAEPMinPaddingSHA256 = " +
		"(2 * 64) + 2, uapolicy/crypto_rsaoaep.go:21, subtracted at " +
		"uapolicy/policyAes256Sha256RsaPss.go:114), so every " +
		"RSA-OAEP-SHA2-256 block carries 126 plaintext bytes where the " +
		"arithmetic allows 190"

	clientKey, clientCert, why := mintCert("gopcua-conformance-client")
	if why != "" {
		t.Fatalf("could not set the test up: %s", why)
	}
	peerKey, peerCert, why := mintCert("gopcua-conformance-peer")
	if why != "" {
		t.Fatalf("could not set the test up: %s", why)
	}

	peer, teardown, why := startScriptedPeer()
	t.Cleanup(teardown)
	if why != "" {
		t.Fatalf("could not set the test up: %s", why)
	}

	_, _, why = launchClient("opc.tcp://"+peer.addr(),
		opcua.SecurityPolicy(ua.SecurityPolicyURIAes256Sha256RsaPss),
		opcua.SecurityMode(ua.MessageSecurityModeSignAndEncrypt),
		opcua.PrivateKey(clientKey),
		opcua.Certificate(clientCert),
		opcua.RemoteCertificate(peerCert),
	)
	if why != "" {
		t.Fatalf("could not set the test up: %s", why)
	}

	frame, why, fromGopcua := peer.awaitOpenSecureChannel(handshake)
	if why != "" {
		if fromGopcua {
			t.Errorf("gopcua's own behaviour made this run unscoreable: %s", why)
			return
		}
		t.Fatalf("could not set the test up: %s", why)
	}

	// The row slices the frame with uasc, which only moves bytes it never
	// scores: the verdict below rests on the standard library's
	// DecryptOAEP and on the literal hash lengths.
	chunk := &uasc.MessageChunk{}
	if _, err := chunk.Decode(frame); err != nil {
		t.Errorf("gopcua's own behaviour made this run unscoreable: the OPN "+
			"frame does not decode as a message chunk: %v", err)
		return
	}
	if uri := chunk.AsymmetricSecurityHeader.SecurityPolicyURI; uri != ua.SecurityPolicyURIAes256Sha256RsaPss {
		t.Errorf("gopcua's own behaviour made this run unscoreable: the OPN's "+
			"asymmetric security header names policy %q, not Aes256_Sha256_RsaPss", uri)
		return
	}

	// k is the receiver's modulus length in octets - the peer is the
	// receiver, and the row decrypts with the peer's minted key. The two
	// hash lengths are literals: RFC 3447 §7.1.1's 32, and the 64 per hash
	// uapolicy's constant reserves. Neither size is read from uapolicy,
	// so the known-bug arm cannot move with the fix.
	k := peerKey.Size()
	const hLen = 32
	const bugHLen = 64
	want := k - 2*hLen - 2
	knownBugSize := k - 2*bugHLen - 2

	// chunk.Data is the whole encrypted region: sequence header, request,
	// padding and signature, all encrypted together. A region that is
	// empty or not a whole number of k-byte blocks is an ordinary
	// failure - the guard against a pass on zero blocks.
	data := chunk.Data
	if len(data) == 0 || len(data)%k != 0 {
		t.Errorf("the OPN's encrypted region is %d bytes, not a whole "+
			"number of %d-byte blocks", len(data), k)
		return
	}
	blocks := len(data) / k

	sizes := make([]int, 0, blocks)
	for i := 0; i < blocks; i++ {
		pt, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, peerKey,
			data[i*k:(i+1)*k], nil)
		if err != nil {
			t.Errorf("block %d of %d does not decrypt under the peer's key: %v",
				i+1, blocks, err)
			return
		}
		sizes = append(sizes, len(pt))
	}

	atWant, atKnownBugSize := 0, 0
	for _, n := range sizes {
		switch n {
		case want:
			atWant++
		case knownBugSize:
			atKnownBugSize++
		}
	}

	switch {
	case atWant == blocks:
		// Every block carries the most plaintext the algorithm allows.
		// On main that pass is the hard error asking for the constant's
		// deletion; with the constant deleted it is the row's pass and
		// has nothing to report.
		if knownBug != "" {
			t.Errorf("this now passes (all %d blocks carry %d plaintext "+
				"bytes, not %d) - delete the knownBug constant, every arm "+
				"that reads it, and anything only those arms used, in the "+
				"commit that fixed it", blocks, want, knownBugSize)
		}
	case atKnownBugSize == blocks:
		if knownBug != "" {
			t.Logf("known bug: %s (all %d blocks carry %d plaintext "+
				"bytes, not %d)", knownBug, blocks, knownBugSize, want)
		} else {
			t.Errorf("all %d blocks carry %d plaintext bytes, not %d",
				blocks, knownBugSize, want)
		}
	default:
		// Mixed sizes, or a size that is neither the RFC's nor the bug's.
		t.Errorf("the %d blocks carry mixed or wrong plaintext sizes "+
			"(%d bytes each), not %d bytes each", blocks, sizes, want)
	}
}
