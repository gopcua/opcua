package uasc

import (
	"bytes"
	"crypto/rsa"
	"crypto/x509"
	"encoding/binary"
	"strconv"
	"testing"

	"github.com/gopcua/opcua/ua"
	"github.com/gopcua/opcua/uapolicy"
	"github.com/stretchr/testify/require"
)

func symmetricInstance(t *testing.T, uri string) *channelInstance {
	t.Helper()

	nonce := bytes.Repeat([]byte{0x5a}, 32)
	algo, err := uapolicy.Symmetric(uri, nonce, nonce)
	require.NoError(t, err, "error: create symmetric algorithm")

	return &channelInstance{
		sc:   &SecureChannel{cfg: &Config{SecurityMode: ua.MessageSecurityModeSignAndEncrypt}},
		algo: algo,
	}
}

// symmetricChunk returns the raw bytes of a MSG chunk (header, symmetric
// security header, sequence header, body) and the matching Message needed
// by signAndEncrypt.
func symmetricChunk(body []byte) ([]byte, *Message) {
	m := &Message{
		MessageHeader: &MessageHeader{
			Header:                  NewHeader(MessageTypeMessage, ChunkTypeFinal, 1),
			SymmetricSecurityHeader: NewSymmetricSecurityHeader(1),
			SequenceHeader:          NewSequenceHeader(1, 1),
		},
	}

	b := make([]byte, 0, 24+len(body))
	b = append(b, 'M', 'S', 'G', 'F')
	b = append(b, 0, 0, 0, 0) // message size, set by signAndEncrypt
	b = append(b, 1, 0, 0, 0) // secure channel id
	b = append(b, 1, 0, 0, 0) // security token id
	b = append(b, 1, 0, 0, 0) // sequence number
	b = append(b, 1, 0, 0, 0) // request id
	b = append(b, body...)
	return b, m
}

var symmetricPolicyURIs = []string{
	ua.SecurityPolicyURIBasic128Rsa15,
	ua.SecurityPolicyURIBasic256,
	ua.SecurityPolicyURIBasic256Sha256,
	ua.SecurityPolicyURIAes128Sha256RsaOaep,
	ua.SecurityPolicyURIAes256Sha256RsaPss,
}

// TestMaxBodySizeFitsChunk reproduces the bug from
// https://github.com/gopcua/opcua/issues/781: a body of exactly maxBodySize
// must produce an encrypted chunk that does not exceed the negotiated chunk
// size, for every security policy and in particular for the default chunk
// size of 65535 which is not a multiple of the AES block size.
func TestMaxBodySizeFitsChunk(t *testing.T) {
	chunkSizes := []int{4096, 8192, 65521, 65535, 65536}

	for _, uri := range symmetricPolicyURIs {
		for _, chunkSize := range chunkSizes {
			t.Run(testName(uri, chunkSize), func(t *testing.T) {
				c := symmetricInstance(t, uri)
				c.SetMaximumBodySize(chunkSize)

				body := bytes.Repeat([]byte{0xab}, int(c.maxBodySize))
				b, m := symmetricChunk(body)
				want := make([]byte, len(b)-16)
				copy(want, b[16:])

				cipher, err := c.signAndEncrypt(m, b)
				require.NoError(t, err, "error: message encrypt")
				require.LessOrEqual(t, len(cipher), chunkSize, "encrypted chunk exceeds negotiated chunk size")

				// a body one byte larger must no longer fit, i.e. maxBodySize
				// is not needlessly conservative
				b2, m2 := symmetricChunk(append(body, 0xab))
				cipher2, err := c.signAndEncrypt(m2, b2)
				require.NoError(t, err, "error: message encrypt")
				require.Greater(t, len(cipher2), chunkSize, "maxBodySize is more conservative than necessary")

				// round trip
				mc := new(MessageChunk)
				_, err = mc.Decode(cipher)
				require.NoError(t, err, "error: message decode")

				plain, err := c.verifyAndDecrypt(mc, cipher)
				require.NoError(t, err, "error: message decrypt")
				require.Equal(t, want, plain, "plaintext not equal")
			})
		}
	}
}

// TestPaddingBlockAligned reproduces the second bug discussed in
// https://github.com/gopcua/opcua/pull/824: when the plaintext including the
// PaddingSize byte is already block-aligned, no padding bytes must be added
// beyond the PaddingSize byte itself. The old code added a full extra block.
func TestPaddingBlockAligned(t *testing.T) {
	for _, uri := range symmetricPolicyURIs {
		t.Run(uri, func(t *testing.T) {
			c := symmetricInstance(t, uri)

			blockSize := c.algo.PlaintextBlockSize()
			sigLen := c.algo.SignatureLength()

			// sequence header (8) + body + signature + 1 PaddingSize byte
			// is a multiple of the block size
			bodyLen := 4*blockSize - 8 - sigLen - 1
			for bodyLen < 0 {
				bodyLen += blockSize
			}
			b, m := symmetricChunk(bytes.Repeat([]byte{0xab}, bodyLen))

			cipher, err := c.signAndEncrypt(m, b)
			require.NoError(t, err, "error: message encrypt")

			// 16 byte header + already aligned plaintext + PaddingSize byte + signature
			want := 16 + 8 + bodyLen + 1 + sigLen
			require.Equal(t, want, len(cipher), "block-aligned plaintext got more than the PaddingSize byte of padding")

			mc := new(MessageChunk)
			_, err = mc.Decode(cipher)
			require.NoError(t, err, "error: message decode")

			plain, err := c.verifyAndDecrypt(mc, cipher)
			require.NoError(t, err, "error: message decrypt")
			require.Equal(t, b[16:], plain, "plaintext not equal")
		})
	}
}

func testName(uri string, chunkSize int) string {
	return uri[len("http://opcfoundation.org/UA/SecurityPolicy#"):] + "/" + strconv.Itoa(chunkSize)
}

// TestVerifyAndDecrypt_Part6_677_ChunkShorterThanItsSignature feeds verifyAndDecrypt
// a chunk whose body is shorter than the signature the security policy appends to
// every message chunk.
//
// OPC 10000-6 §6.7.7 forbids interpreting the chunk before the signature
// is verified and requires the failed check to be reported as
// Bad_SecurityChecksFailed. A chunk too short to hold the signature cannot
// validate it, so the receiver must report the error instead of reading the
// signature out of the buffer, which slices past its start.
//
// A Sign channel receives these chunks in the clear.
func TestVerifyAndDecrypt_Part6_677_ChunkShorterThanItsSignature(t *testing.T) {
	const uri = ua.SecurityPolicyURIBasic256Sha256

	// Equal nonces make the signing and verifying HMAC keys identical, so a
	// single channelInstance can sign a message and verify its own signature.
	nonce := bytes.Repeat([]byte{0x5a}, 32)
	algo, err := uapolicy.Symmetric(uri, nonce, nonce)
	require.NoError(t, err, "error: create symmetric algorithm")

	c := &channelInstance{
		sc: &SecureChannel{cfg: &Config{
			SecurityMode:      ua.MessageSecurityModeSign,
			SecurityPolicyURI: uri,
		}},
		algo: algo,
	}

	// 16 bytes of header and a 4 byte body: 20 bytes where the policy appends a
	// 32 byte signature, so there is no signature to check.
	b := []byte{
		// Message Header: MSG / Final / MessageSize / SecureChannelID
		0x4d, 0x53, 0x47, 0x46, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		// SymmetricSecurityHeader: TokenID
		0x01, 0x00, 0x00, 0x00,
		// Body: 4 bytes instead of the 32 byte signature the policy appends
		0x01, 0x02, 0x03, 0x04,
	}

	chunk := &MessageChunk{MessageHeader: &MessageHeader{
		Header:                  &Header{MessageType: MessageTypeMessage, ChunkType: ChunkTypeFinal},
		SymmetricSecurityHeader: &SymmetricSecurityHeader{TokenID: 1},
	}}

	_, err = c.verifyAndDecrypt(chunk, b)
	require.ErrorIs(t, err, ua.StatusBadSecurityChecksFailed,
		"a chunk without room for its signature must be rejected, not sliced")
}

// TestVerifyAndDecrypt_Part6_6725_PaddingLongerThanTheBody feeds verifyAndDecrypt
// a chunk that carries a PaddingSize naming more padding than the chunk holds.
// The signature verifies, so this test pins the padding guard alone.
//
// OPC 10000-6 §6.7.2.5, Table 61: the PaddingSize is "The number of padding bytes
// (not including the byte for the PaddingSize)" and the Padding is added "to the
// end of the Message to ensure length of the data to encrypt is an integer multiple
// of the encryption block size", so padding can never reach past the start of the
// body. A PaddingSize that does is not padding, and §6.7.7 requires the
// failed check to be reported as Bad_SecurityChecksFailed.
func TestVerifyAndDecrypt_Part6_6725_PaddingLongerThanTheBody(t *testing.T) {
	c := symmetricInstance(t, ua.SecurityPolicyURIBasic256Sha256)

	header := []byte{
		// Message Header: MSG / Final / MessageSize / SecureChannelID
		0x4d, 0x53, 0x47, 0x46, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		// SymmetricSecurityHeader: TokenID
		0x01, 0x00, 0x00, 0x00,
	}

	// A 16 byte body whose last byte is the PaddingSize the sender chose: 255
	// padding bytes plus the PaddingSize byte itself, in a body of 16.
	body := bytes.Repeat([]byte{0x00}, 16)
	body[15] = 0xff

	plain := append(append([]byte(nil), header...), body...)
	sig, err := c.algo.Signature(plain)
	require.NoError(t, err, "error: message sign")

	// The receiver decrypts everything after the 16 byte header, so sign the
	// whole chunk but encrypt only body and signature.
	enc, err := c.algo.Encrypt(append(body, sig...))
	require.NoError(t, err, "error: message encrypt")
	b := append(append([]byte(nil), header...), enc...)

	chunk := &MessageChunk{MessageHeader: &MessageHeader{
		Header:                  &Header{MessageType: MessageTypeMessage, ChunkType: ChunkTypeFinal},
		SymmetricSecurityHeader: &SymmetricSecurityHeader{TokenID: 1},
	}}

	_, err = c.verifyAndDecrypt(chunk, b)
	require.ErrorIs(t, err, ua.StatusBadSecurityChecksFailed,
		"padding that does not fit in the chunk must be rejected, not sliced off")
}

// TestVerifyAndDecrypt_Part6_6725_AsymmetricPaddingLongerThanTheBody is the
// asymmetric arm of the padding check above: an OpenSecureChannel response whose
// signature verifies and whose last plaintext byte claims more padding than the
// body holds.
//
// verifyAndDecrypt runs before the SenderCertificate is compared to the
// configured one, so the padding cannot be checked at certificate validation
// time. A symmetric-only test would leave this arm unexercised, so this one
// drives the asymmetric path end to end: a decrypting, signed OPN chunk, a
// signature that verifies, and then a PaddingSize that reaches past the start
// of the body.
//
// The expected result is the one OPC 10000-6 §6.7.7 states for a failed security
// check, the same status the symmetric test above expects.
func TestVerifyAndDecrypt_Part6_6725_AsymmetricPaddingLongerThanTheBody(t *testing.T) {
	const uri = ua.SecurityPolicyURIBasic256Sha256

	// A self-signed certificate the response is signed with.
	testCertDER, testKey := genThumbprintTestCert(t, "gopcua padding test")
	testCert, err := x509.ParseCertificate(testCertDER)
	require.NoError(t, err, "error: parse test certificate")
	testPub, ok := testCert.PublicKey.(*rsa.PublicKey)
	require.True(t, ok, "test certificate has no RSA key")

	// The client's channel instance, as readChunk builds it from the incoming
	// OPN: the local key is the client's, the remote key comes from the
	// SenderCertificate, which here is the test certificate's. Since this test
	// crafts the chunk and verifies it in one place, both keys are the test
	// certificate's, like the equal nonces of the symmetric test above.
	algo, err := uapolicy.Asymmetric(uri, testKey, testPub)
	require.NoError(t, err, "error: create asymmetric algorithm")

	c := &channelInstance{
		sc: &SecureChannel{cfg: &Config{
			SecurityMode:      ua.MessageSecurityModeSignAndEncrypt,
			SecurityPolicyURI: uri,
		}},
		algo: algo,
	}

	// An OPN chunk header: 12 bytes of message header plus the asymmetric
	// security header, which carries the test certificate the way an
	// OpenSecureChannel response does.
	ash := NewAsymmetricSecurityHeader(uri, testCertDER, bytes.Repeat([]byte{0x5a}, 20))
	hdr := NewHeader(MessageTypeOpenSecureChannel, ChunkTypeFinal, 1)
	hbuf := ua.NewBuffer(nil)
	hbuf.WriteStruct(hdr)
	hbuf.WriteStruct(ash)
	require.NoError(t, hbuf.Error(), "error: encode chunk header")
	header := hbuf.Bytes()

	headerLength := 12 + ash.Len()
	require.Equal(t, headerLength, len(header), "encoded header length")

	// The plaintext after the header: a sequence header, a short body, two
	// padding bytes, and then the PaddingSize byte, which claims 255 padding
	// bytes plus itself in the 13 bytes of sequence header, body and padding
	// that precede it.
	plain := []byte{
		// SequenceHeader: SequenceNumber / RequestID
		0x01, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00,
		// Body: anything, it is never reached
		0xde, 0xad, 0xbe, 0xef,
		// Padding, and a PaddingSize that does not fit in the body
		0x00, 0x00, 0xff,
	}

	// The signature and the plaintext are encrypted together, and the RSA-OAEP
	// block math fixes the chunk's MessageSize before it is signed.
	sigLen := algo.RemoteSignatureLength()
	require.Equal(t, 256, sigLen, "a 2048-bit key signs with 256 bytes")
	nblocks := (len(plain) + sigLen + algo.PlaintextBlockSize() - 1) / algo.PlaintextBlockSize()
	encLen := nblocks * algo.BlockSize()

	b := append(append([]byte(nil), header...), plain...)
	binary.LittleEndian.PutUint32(b[4:8], uint32(headerLength+encLen)) // MessageSize

	sig, err := algo.Signature(b)
	require.NoError(t, err, "error: sign with the test key")
	require.Len(t, sig, sigLen, "signature length")

	enc, err := algo.Encrypt(append(append([]byte(nil), plain...), sig...))
	require.NoError(t, err, "error: encrypt with the test key")
	require.Len(t, enc, encLen, "ciphertext length")

	chunk := append(append([]byte(nil), b[:headerLength]...), enc...)

	// Decode the chunk the way readChunk does, so that the AsymmetricSecurityHeader
	// verifyAndDecrypt measures the header with is the one on the wire.
	mc := new(MessageChunk)
	_, err = mc.Decode(chunk)
	require.NoError(t, err, "error: decode chunk header")
	require.NotNil(t, mc.AsymmetricSecurityHeader, "the chunk must decode as asymmetric")

	_, err = c.verifyAndDecrypt(mc, chunk)
	require.ErrorIs(t, err, ua.StatusBadSecurityChecksFailed,
		"an asymmetric chunk whose padding does not fit must be rejected, not sliced")
}
