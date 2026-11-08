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
// the nonce's length prefix and bytes straight out of the frame's own
// bytes, never through gopcua's decoder, per ../README.md: the offset
// is computed by walking OPC 10000-6's encoding rules, every length
// read from the frame itself, and the walk asserts it ends on the
// frame's last byte; a length prefix of 0 and one of -1 both leave no
// nonce bytes, §5.1.11's equality, quoted above. Each connection's
// nonce must be null, or else at least 32 bytes long; a non-null nonce
// must also differ from the other connection's nonce, and a null nonce
// counts as different from any non-null one. It declines, and does
// not score, when either connection's OPN did not arrive, whether the
// harness's `handshake` bound passed or the peer's exchange ended
// first - that is the row's own decline, and the setup's declines
// (starting the peer, building the client) apply as well. A frame the
// walk cannot follow - a length prefix that overruns it, a type ID
// that is not OpenSecureChannelRequest's, bytes left over at its end -
// can only be the client's bytes, and so are the frames the harness
// itself flags, per harness.go, so the row prints "gopcua's own
// behaviour made this run unscoreable" with t.Errorf, then returns.
// It deliberately asserts nothing about the rest of the OPN - the
// security header, the mode - and nothing about the channel
// lifecycle: §5.6.2.2's renewal sentence ("A new clientNonce shall be
// generated for each time a SecureChannel is renewed.") is the
// channel's business, not the nonce's.
//
// UaExpert sends a 1-byte 0x00 ClientNonce under None, per the Wireshark
// comparison in gopcua issue #894; the row fails that shape on purpose:
// OPC 10000-4 §5.6.2.3 says a Server shall check the minimum length of
// the Client nonce and return Bad_NonceInvalid below 32 bytes, so a
// conformant server may reject it, and OPC 10000-6 §6.7.4 asks for null
// under None as the recommended shape. Issue #894 itself reports
// gopcua's empty ClientNonce as the interop failure — some embedded
// servers drop the connection on receiving it — and asks for an opt-in
// option to send a non-empty one rather than a change to the default;
// the row's verdict stands either way, and a nonce sent under such an
// option meets the same floor. No reference stack's
// source has been read for this behaviour.
package securitypolicy

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"testing"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/ua"
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

// clientNonceFromOPNFrame walks the recorded OPN frame's own bytes by
// OPC 10000-6's encoding rules alone - never through gopcua's decoder
// - and returns the ClientNonce the OpenSecureChannelRequest it carries
// was given, with "" - or the reason the run cannot be scored, and
// true, because a frame the walk cannot follow - a length prefix that
// overruns it, a type ID that is not OpenSecureChannelRequest's, bytes
// left over at its end - can only be the client's bytes. Every length
// the walk reads comes from the frame itself, and the walk asserts
// that it ends on the frame's last byte - RequestedLifetime's 4 - so
// a wrong offset cannot score.
func clientNonceFromOPNFrame(frame []byte) (nonce []byte, why string, fromGopcua bool) {
	// The message header: the message type (3), the chunk type (1),
	// the message size (4) and the secure channel id (4) - 12 bytes.
	o, why := skip(frame, 0, 12)
	if why != "" {
		return nil, why, true
	}

	// The asymmetric security header: SecurityPolicyURI,
	// SenderCertificate and ReceiverCertificateThumbprint - three
	// length-prefixed ByteStrings, a prefix of -1 null.
	for i := 0; i < 3; i++ {
		if _, o, why = skipPrefixed(frame, o); why != "" {
			return nil, why, true
		}
	}

	// The sequence header: the sequence number and the request id,
	// two UInt32s - 8 bytes.
	if o, why = skip(frame, o, 8); why != "" {
		return nil, why, true
	}

	// The body's type ID: whatever NodeId encoding the client chose,
	// it must carry OpenSecureChannelRequest's encoding id - ns=0,
	// i=446, OpenSecureChannelRequest_Encoding_DefaultBinary. The
	// literal is in the row, never read from gopcua's id tables.
	var ns uint16
	var id uint64
	if ns, id, o, why = nodeID(frame, o); why != "" {
		return nil, why, true
	}
	if ns != 0 || id != 446 {
		return nil, fmt.Sprintf("the OPN's body carries type id ns=%d "+
			"i=%d, not OpenSecureChannelRequest's ns=0 i=446", ns, id), true
	}

	// The RequestHeader: the AuthenticationToken, a NodeId; the
	// Timestamp, 8 bytes; the RequestHandle and the
	// ReturnDiagnostics, two UInt32s; the AuditEntryID, a
	// length-prefixed String; the TimeoutHint, a UInt32; and the
	// AdditionalHeader - an ExtensionObject: a NodeId, then one
	// encoding byte, which the walk asserts is 0, the "no body is
	// encoded" of a null ExtensionObject, so the walk reads no body.
	if _, _, o, why = nodeID(frame, o); why != "" {
		return nil, why, true
	}
	if o, why = skip(frame, o, 8); why != "" {
		return nil, why, true
	}
	if o, why = skip(frame, o, 4); why != "" {
		return nil, why, true
	}
	if o, why = skip(frame, o, 4); why != "" {
		return nil, why, true
	}
	if _, o, why = skipPrefixed(frame, o); why != "" {
		return nil, why, true
	}
	if o, why = skip(frame, o, 4); why != "" {
		return nil, why, true
	}
	if _, _, o, why = nodeID(frame, o); why != "" {
		return nil, why, true
	}
	if o, why = skip(frame, o, 1); why != "" {
		return nil, why, true
	}
	if enc := frame[o-1]; enc != 0 {
		return nil, fmt.Sprintf("the OPN's AdditionalHeader carries "+
			"encoding %d, not the 0 of an ExtensionObject with no body", enc), true
	}

	// ClientProtocolVersion, RequestType and SecurityMode - three
	// UInt32s, 12 bytes.
	if o, why = skip(frame, o, 12); why != "" {
		return nil, why, true
	}

	// The ClientNonce: its Int32 length prefix, then that many bytes.
	// A prefix of 0 and one of -1 both leave no nonce bytes -
	// §5.1.11's equality, quoted in the header.
	var n int32
	if n, o, why = i32(frame, o); why != "" {
		return nil, why, true
	}
	if n < -1 {
		return nil, fmt.Sprintf("the OPN's ClientNonce carries length "+
			"prefix %d, neither a length nor the -1 of null", n), true
	}
	if n > 0 {
		if o, why = skip(frame, o, int(n)); why != "" {
			return nil, why, true
		}
		nonce = frame[o-int(n) : o]
	}

	// RequestedLifetime, a UInt32, is the body's last field, so the
	// walk must end on the frame's last byte; a walk that ends anywhere
	// else read the frame wrong and cannot score.
	if o+4 != len(frame) {
		return nil, fmt.Sprintf("the OPN carries %d bytes after the "+
			"ClientNonce, not RequestedLifetime's 4", len(frame)-o), true
	}
	return nonce, "", false
}

// i32 reads one Int32 - little-endian, the byte order OPC 10000-6
// encodes every integer in - from the frame at offset o, and returns
// its value and the offset past it, with "" - or, when the frame ends
// before the four bytes, the reason the walk cannot continue.
func i32(frame []byte, o int) (int32, int, string) {
	if o+4 > len(frame) {
		return 0, o, fmt.Sprintf("the frame ends %d bytes into the "+
			"Int32 at offset %d", o+4-len(frame), o)
	}
	return int32(binary.LittleEndian.Uint32(frame[o : o+4])), o + 4, ""
}

// skip advances the walk past n bytes of the frame and returns the
// offset past them, with "" - or, when the frame ends before them, the
// reason the walk cannot continue.
func skip(frame []byte, o, n int) (int, string) {
	if o+n > len(frame) {
		return o, fmt.Sprintf("the frame ends %d bytes into the %d "+
			"bytes at offset %d", o+n-len(frame), n, o)
	}
	return o + n, ""
}

// skipPrefixed reads one Int32 length prefix and advances the walk
// past the bytes it counts - the encoding String and ByteString share,
// where a prefix of -1 is null and counts no bytes - and returns the
// prefix and the offset past the bytes, with "" - or the reason the
// walk cannot continue: the frame ending mid-field, or a prefix that
// is neither a length nor the -1 of null, which no encoding rule
// allows.
func skipPrefixed(frame []byte, o int) (int32, int, string) {
	n, o, why := i32(frame, o)
	if why != "" {
		return n, o, why
	}
	if n < -1 {
		return n, o, fmt.Sprintf("the length prefix %d at offset %d is "+
			"neither a length nor the -1 of null", n, o-4)
	}
	if n < 0 {
		return n, o, ""
	}
	o, why = skip(frame, o, int(n))
	return n, o, why
}

// nodeID walks one NodeId, per OPC 10000-6's NodeId encoding table:
// a one-byte encoding format, then the namespace and the identifier
// the format names. It returns the namespace, the identifier (0 for
// the String, Guid and ByteString formats, whose identifiers carry no
// number to compare), the offset past the NodeId, and "" - or the
// reason the walk cannot continue.
func nodeID(frame []byte, o int) (uint16, uint64, int, string) {
	o, why := skip(frame, o, 1)
	if why != "" {
		return 0, 0, o, why
	}
	switch enc := frame[o-1]; enc {
	case 0x00: // TwoByte: namespace 0, a one-byte identifier.
		if o, why = skip(frame, o, 1); why != "" {
			return 0, 0, o, why
		}
		return 0, uint64(frame[o-1]), o, ""
	case 0x01: // FourByte: a one-byte namespace, a two-byte identifier.
		if o, why = skip(frame, o, 3); why != "" {
			return 0, 0, o, why
		}
		return uint16(frame[o-3]),
			uint64(binary.LittleEndian.Uint16(frame[o-2 : o])), o, ""
	case 0x02: // Numeric: a two-byte namespace, a four-byte identifier.
		if o, why = skip(frame, o, 6); why != "" {
			return 0, 0, o, why
		}
		return binary.LittleEndian.Uint16(frame[o-6 : o-4]),
			uint64(binary.LittleEndian.Uint32(frame[o-4 : o])), o, ""
	case 0x03, 0x05: // String, ByteString: a two-byte namespace,
		// then a length-prefixed identifier.
		if o, why = skip(frame, o, 2); why != "" {
			return 0, 0, o, why
		}
		ns := binary.LittleEndian.Uint16(frame[o-2 : o])
		if _, o, why = skipPrefixed(frame, o); why != "" {
			return 0, 0, o, why
		}
		return ns, 0, o, ""
	case 0x04: // Guid: a two-byte namespace, a 16-byte identifier.
		if o, why = skip(frame, o, 18); why != "" {
			return 0, 0, o, why
		}
		return binary.LittleEndian.Uint16(frame[o-18 : o-16]), 0, o, ""
	default:
		return 0, 0, o, fmt.Sprintf("the NodeId at offset %d carries "+
			"encoding %d, which no NodeId encoding rule names", o-1, enc)
	}
}
