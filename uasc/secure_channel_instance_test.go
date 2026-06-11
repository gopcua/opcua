package uasc

import (
	"bytes"
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
