//go:build conformance

// Under SecurityPolicy None, the ClientNonce in the OpenSecureChannel
// request is null, or, if a client sends one anyway, it is at least 32
// bytes long and differs between two connections.
//
// The clauses: OPC 10000-6 §6.7.4: "The OpenSecureChannel Messages are not
// signed or encrypted if the SecurityMode is None. The Nonces are ignored and
// should be set to null." OPC 10000-4 §5.6.2.2, clientNonce: "A random number
// that shall not be used in any other request." and "This parameter shall have
// a length equal to the SecureChannelNonceLength defined for the SecurityPolicy
// in OPC 10000-7." OPC 10000-4 §5.6.2.3, Bad_NonceInvalid: "A Server shall
// check the minimum length of the Client nonce and return this status if the
// length is below 32 bytes." OPC 10000-7 v1.04 gives no
// SecureChannelNonceLength for None - the profile's limits unit
// SecurtyPolicy_None_Limits (sic) lists only "DerivedSignatureKeyLength: 0" -
// so §5.6.2.2's length rule has no number to apply here, and the row
// rests on §5.6.2.3's server-side floor, with §6.7.4's null as the
// recommended shape. What counts as a null nonce is settled by the encoding
// rules, not by this row: OPC 10000-6 §5.2.2.7, ByteString: "If the length of
// the byte string is -1 then the byte string is 'null'." OPC 10000-6 §5.1.11,
// Null, Empty and Zero-Length Arrays, which covers ByteStrings ("ByteStrings
// are arrays of Bytes for purposes of this discussion"): "For this reason,
// null, empty and zero length arrays are semantically the same for all
// DataEncodings." "When testing for equality, applications shall treat null and
// empty arrays as equal." That is why the row counts a ClientNonce whose length
// prefix is 0 and one whose length prefix is -1 alike: both are null. gopcua
// passes this row as it stands: measured on this tree, its OPN under None
// carries a zero-length ClientNonce - a length prefix of 0, no nonce bytes.
//
// Neighbouring rows:
// open_secure_channel_fills_each_rsa_oaep_block_test.go scores the
// other thing a SecurityPolicy hands the channel at open - the
// plaintext block size, under Aes256_Sha256_RsaPss, the policy at the
// other end of the line from this row's. tests/transport holds the
// HEL/ACK exchange that precedes the OPN; this row does not score it.
//
// The row runs against the scripted peer from harness.go, twice - a
// fresh peer and a fresh client each time - so it reads the OPN from
// two separate connections and can check that a non-null nonce is not
// reused. The client under test is the real
// opcua.Client, launched by the shared launchClient with
// opcua.SecurityPolicy(ua.SecurityPolicyURINone) and
// opcua.SecurityMode(ua.MessageSecurityModeNone), under a context
// bounded by the harness's handshake deadline; its Connect is expected
// to fail once the peer closes, and no arm asserts its error.
// See harness.go.
//
// The row asserts, from the nonce bytes as they arrive on the wire -
// read by the scripted peer, not through uapolicy.NonceLength() - that
// the ClientNonce is null - a length prefix of 0 or -1, no nonce
// bytes. If nonce bytes are sent anyway, they must be at least 32
// bytes long and different between the two connections. The row reads
// the nonce by decoding the recorded frame with uasc and the request
// body with ua.DecodeService into a *ua.OpenSecureChannelRequest, then
// reading ClientNonce; that is acceptable because the verdict rests on
// len and bytes.Equal of the nonce bytes, and the decoder turning a
// length prefix of 0 and one of -1 into the same empty value is
// §5.1.11's equality, quoted above. Each connection's nonce must be
// null, or else at least 32 bytes long; a non-null nonce must also
// differ from the other connection's nonce, and a null nonce counts as
// different from any non-null one. It declines, and does not score,
// when either connection's OPN did not arrive, whether the harness's
// `handshake` bound passed or the peer's exchange ended first - that is
// the row's own decline, and the setup's declines (starting the peer,
// building the client) apply as well. A frame that does not decode - its
// chunk, its sequence header or its request body - or whose body is not
// an OpenSecureChannelRequest, can only be the client's bytes, and so are
// the frames the harness itself flags, per harness.go, so the row
// prints "gopcua's own behaviour made this run unscoreable" with
// t.Errorf, then returns. It deliberately asserts nothing about the
// rest of the OPN - the security header, the mode - and nothing about
// the channel lifecycle: §5.6.2.2's renewal sentence ("A new
// clientNonce shall be generated for each time a SecureChannel is
// renewed.") is the channel's business, not the nonce's.
//
// UaExpert sends a 1-byte 0x00 ClientNonce under None, per the Wireshark
// comparison in gopcua issue #894; the row fails that shape on purpose:
// OPC 10000-4 §5.6.2.3 says a Server shall check the minimum length of
// the Client nonce and return Bad_NonceInvalid below 32 bytes, so a
// conformant server may reject it, and OPC 10000-6 §6.7.4 asks for null
// under None as the recommended shape. No reference stack's
// source has been read for this behaviour.
package securitypolicy

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/ua"
	"github.com/gopcua/opcua/uasc"
)

func TestClientNonceUnderSecurityPolicyNoneIsNullOrUsable(t *testing.T) {
	// nonceFloor is OPC 10000-4 §5.6.2.3's 32-byte minimum, quoted in the
	// row's header. The literal is in the row, never read from uapolicy,
	// whose NonceLength() is 0 for None.
	const nonceFloor = 32

	var nonces [2][]byte
	for i := range nonces {
		nonce, teardown, why, fromGopcua := clientNonceFromOneConnection()
		t.Cleanup(teardown)
		if why != "" {
			if fromGopcua {
				t.Errorf("gopcua's own behaviour made this run unscoreable: %s", why)
				return
			}
			t.Fatalf("could not set the test up: %s", why)
		}
		nonces[i] = nonce
	}

	// Each connection on its own: the ClientNonce is null - no nonce bytes,
	// the length prefix 0 and -1 alike, per §5.1.11's equality, quoted in
	// the header - or it is at least nonceFloor bytes long. A nonce that
	// is neither is an ordinary failure.
	for i, nonce := range nonces {
		if len(nonce) == 0 {
			continue
		}
		if len(nonce) < nonceFloor {
			t.Errorf("connection %d's ClientNonce is a %d-byte nonce: OPC "+
				"10000-4 §5.6.2.3 says a Server shall check the minimum length of "+
				"the Client nonce and return Bad_NonceInvalid below %d bytes, so a "+
				"conformant server may reject it, and OPC 10000-6 §6.7.4 asks for "+
				"null under None (\"The Nonces are ignored and should be set to "+
				"null.\")",
				i+1, len(nonce), nonceFloor)
		}
	}

	// Across the two connections: a non-null ClientNonce is "a random
	// number that shall not be used in any other request" (OPC 10000-4
	// §5.6.2.2), so it must differ between them. The len guards leave
	// null nonces out of the check, so a null nonce never counts as
	// reused.
	if len(nonces[0]) > 0 && len(nonces[1]) > 0 && bytes.Equal(nonces[0], nonces[1]) {
		t.Errorf("both connections carried the same %d-byte ClientNonce, but "+
			"the clientNonce is \"a random number that shall not be used in any "+
			"other request\" (OPC 10000-4 §5.6.2.2)", len(nonces[0]))
	}

	if t.Failed() {
		return
	}

	// The pass log states what the row saw, the way the sibling rows log
	// theirs.
	t.Logf("connection 1's ClientNonce is %s, connection 2's is %s",
		nonceDesc(nonces[0]), nonceDesc(nonces[1]))
}

// nonceDesc says what a recorded ClientNonce was: null, or how many
// bytes of nonce the client sent anyway.
func nonceDesc(n []byte) string {
	if len(n) == 0 {
		return "null (no nonce bytes)"
	}
	return fmt.Sprintf("%d bytes of nonce", len(n))
}

// clientNonceFromOneConnection runs one whole connection against the
// scripted peer: it starts the peer, launches the client under
// SecurityPolicy None and SecurityMode None, waits for the OPN within
// the harness's handshake bound, and returns the ClientNonce the client
// put in it, the peer's teardown for the row to register, and "" - or
// the reason the run cannot be scored, and fromGopcua, which side
// caused it, in awaitOpenSecureChannel's sense.
func clientNonceFromOneConnection() (nonce []byte, teardown func(), why string, fromGopcua bool) {
	peer, teardown, why := startScriptedPeer()
	if why != "" {
		return nil, teardown, why, false
	}

	why = launchClient("opc.tcp://"+peer.addr(),
		opcua.SecurityPolicy(ua.SecurityPolicyURINone),
		opcua.SecurityMode(ua.MessageSecurityModeNone),
	)
	if why != "" {
		return nil, teardown, why, false
	}

	frame, why, fromGopcua := peer.awaitOpenSecureChannel(handshake)
	if why != "" {
		return nil, teardown, why, fromGopcua
	}

	nonce, why, fromGopcua = clientNonceFromOPNFrame(frame)
	return nonce, teardown, why, fromGopcua
}

// clientNonceFromOPNFrame decodes the recorded OPN frame with uasc and
// its request body with ua.DecodeService, and returns the
// ClientNonce of the OpenSecureChannelRequest it carries, with "" - or
// the reason the run cannot be scored, and true, because a frame that
// does not decode - its chunk, its sequence header or its request body
// - or whose body is not an OpenSecureChannelRequest can only be the
// client's bytes.
func clientNonceFromOPNFrame(frame []byte) (nonce []byte, why string, fromGopcua bool) {
	chunk := &uasc.MessageChunk{}
	if _, err := chunk.Decode(frame); err != nil {
		return nil, fmt.Sprintf("the OPN frame does not decode as a message chunk: %v", err), true
	}

	seq := &uasc.SequenceHeader{}
	n, err := seq.Decode(chunk.Data)
	if err != nil {
		return nil, fmt.Sprintf("the OPN's sequence header does not decode: %v", err), true
	}

	_, svc, err := ua.DecodeService(chunk.Data[n:])
	if err != nil {
		return nil, fmt.Sprintf("the OPN's request body does not decode: %v", err), true
	}

	req, ok := svc.(*ua.OpenSecureChannelRequest)
	if !ok {
		return nil, fmt.Sprintf("the OPN's request body is a %T, not an OpenSecureChannelRequest", svc), true
	}
	return req.ClientNonce, "", false
}
