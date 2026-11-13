package uasc

import (
	"bytes"
	"io"
	"math"
	"testing"
	"time"

	"github.com/gopcua/opcua/id"
	"github.com/gopcua/opcua/ua"
	"github.com/gopcua/opcua/uacp"
	"github.com/gopcua/opcua/uapolicy"
	"github.com/stretchr/testify/require"
)

// TestReadChunkVerifiesSequenceNumber_Part6_672 verifies that the receive path
// checks the SequenceNumber of every chunk it reads: OPC 10000-6 §6.7.2.4: "A
// SequenceNumber shall not be reused for any TokenId. The SecurityToken
// lifetime shall be short enough to ensure that this never happens; however, if
// it does the receiver shall treat it as a transport error and force a
// reconnect. The SequenceNumber does not reset when a new TokenId is issued and
// it shall be incremented by exactly one for each MessageChunk sent." A chunk
// that reuses a number and one that skips a number are both rejected with
// Bad_SecurityChecksFailed (§6.7.7: "If these checks fail a
// Bad_SecurityChecksFailed error is reported"), while the well-formed chunks
// around them still reach the caller. The numbering continues across token
// renewals, honours the wrap-around of §6.7.2.4 (a sender may count up to
// MaxUint32 and wrap to any number below 1024), and a chunk dropped because
// its token is expired or unknown still consumes its number when it carries
// the expected successor.
func TestReadChunkVerifiesSequenceNumber_Part6_672(t *testing.T) {
	t.Run("reused number is rejected", func(t *testing.T) {
		s, peer := newSequenceNumberTestChannel(t)

		writeSequenceNumberChunk(t, peer, 1, 2)
		_, err := s.readChunk()
		require.NoError(t, err, "the first chunk must be read")

		writeSequenceNumberChunk(t, peer, 2, 2)
		_, err = s.readChunk()
		require.NoError(t, err, "the well-formed answer must reach the caller")

		writeSequenceNumberChunk(t, peer, 2, 3)
		_, err = s.readChunk()
		require.Equal(t, ua.StatusBadSecurityChecksFailed, err, "a reused SequenceNumber must be rejected")
	})

	t.Run("skipped number is rejected", func(t *testing.T) {
		s, peer := newSequenceNumberTestChannel(t)

		writeSequenceNumberChunk(t, peer, 1, 2)
		_, err := s.readChunk()
		require.NoError(t, err, "the first chunk must be read")

		writeSequenceNumberChunk(t, peer, 2, 2)
		_, err = s.readChunk()
		require.NoError(t, err, "the well-formed answer must reach the caller")

		writeSequenceNumberChunk(t, peer, 4, 3)
		_, err = s.readChunk()
		require.Equal(t, ua.StatusBadSecurityChecksFailed, err, "a skipped SequenceNumber must be rejected")
	})

	t.Run("numbering continues across token renewals", func(t *testing.T) {
		s, peer := newSequenceNumberTestChannel(t)

		writeSequenceNumberChunk(t, peer, 1, 2)
		_, err := s.readChunk()
		require.NoError(t, err)

		// The renewed token: the counter does not reset for it, so a
		// chunk that restarts the numbering is a reuse, and one that
		// continues it is well-formed.
		writeSequenceNumberChunk(t, peer, 1, 3)
		_, err = s.readChunk()
		require.Equal(t, ua.StatusBadSecurityChecksFailed, err, "a renewed TokenId must not restart the numbering")

		writeSequenceNumberChunk(t, peer, 2, 3)
		_, err = s.readChunk()
		require.NoError(t, err, "a chunk under the renewed token must be read if it continues the numbering")
	})

	t.Run("numbering continues past MaxUint32-1023 up to MaxUint32", func(t *testing.T) {
		s, peer := newSequenceNumberTestChannel(t)

		writeSequenceNumberChunk(t, peer, math.MaxUint32-1023, 2)
		_, err := s.readChunk()
		require.NoError(t, err, "the first chunk must be read")

		// A sender may keep counting up to MaxUint32 before wrapping
		// (§6.7.2.4: the number "shall not wrap around until it is greater
		// than 4 294 966 271 (UInt32.MaxValue - 1024)").
		for seq := uint32(math.MaxUint32 - 1022); ; seq++ {
			writeSequenceNumberChunk(t, peer, seq, 2)
			_, err = s.readChunk()
			require.NoError(t, err, "a peer may count past MaxUint32-1023 up to MaxUint32")
			if seq == math.MaxUint32 {
				break
			}
		}
	})

	t.Run("numbering wraps from MaxUint32-1023 to 1", func(t *testing.T) {
		s, peer := newSequenceNumberTestChannel(t)

		writeSequenceNumberChunk(t, peer, math.MaxUint32-1023, 2)
		_, err := s.readChunk()
		require.NoError(t, err, "the first chunk must be read")

		writeSequenceNumberChunk(t, peer, 1, 2)
		_, err = s.readChunk()
		require.NoError(t, err, "the number after the wrap shall be less than 1024; 1 is what gopcua itself sends (see nextSequenceNumber)")
	})

	t.Run("numbering wraps from MaxUint32 to 0", func(t *testing.T) {
		s, peer := newSequenceNumberTestChannel(t)

		writeSequenceNumberChunk(t, peer, math.MaxUint32, 2)
		_, err := s.readChunk()
		require.NoError(t, err, "the first chunk must be read")

		writeSequenceNumberChunk(t, peer, 0, 2)
		_, err = s.readChunk()
		require.NoError(t, err, "a peer that counts to MaxUint32 may wrap to 0")
	})

	t.Run("numbering wraps to 1023", func(t *testing.T) {
		s, peer := newSequenceNumberTestChannel(t)

		writeSequenceNumberChunk(t, peer, math.MaxUint32-1023, 2)
		_, err := s.readChunk()
		require.NoError(t, err, "the first chunk must be read")

		writeSequenceNumberChunk(t, peer, 1023, 2)
		_, err = s.readChunk()
		require.NoError(t, err, "the number after the wrap shall be less than 1024")
	})

	t.Run("numbering wraps to 1024 is rejected", func(t *testing.T) {
		s, peer := newSequenceNumberTestChannel(t)

		writeSequenceNumberChunk(t, peer, math.MaxUint32-1023, 2)
		_, err := s.readChunk()
		require.NoError(t, err, "the first chunk must be read")

		writeSequenceNumberChunk(t, peer, 1024, 2)
		_, err = s.readChunk()
		require.Equal(t, ua.StatusBadSecurityChecksFailed, err, "the number after the wrap shall be less than 1024")
	})

	t.Run("numbering wraps to 1 from below the wrap threshold is rejected", func(t *testing.T) {
		s, peer := newSequenceNumberTestChannel(t)

		writeSequenceNumberChunk(t, peer, 1000000, 2)
		_, err := s.readChunk()
		require.NoError(t, err, "the first chunk must be read")

		writeSequenceNumberChunk(t, peer, 1, 3)
		_, err = s.readChunk()
		require.Equal(t, ua.StatusBadSecurityChecksFailed, err, "a number below UInt32.MaxValue - 1024 may not wrap")
	})

	t.Run("chunk numbered 0 seeds the counter", func(t *testing.T) {
		s, peer := newSequenceNumberTestChannel(t)

		writeSequenceNumberChunk(t, peer, 0, 2)
		_, err := s.readChunk()
		require.NoError(t, err, "the first chunk seeds the counter with whatever number it carries")

		writeSequenceNumberChunk(t, peer, 0, 3)
		_, err = s.readChunk()
		require.Equal(t, ua.StatusBadSecurityChecksFailed, err, "a second chunk numbered 0 reuses the number; it must not re-seed the counter")

		writeSequenceNumberChunk(t, peer, 1, 3)
		_, err = s.readChunk()
		require.NoError(t, err, "the counter is seeded at 0, so its successor is 1")
	})

	t.Run("chunk with an expired or unknown token still consumes its number", func(t *testing.T) {
		// Sign, not SignAndEncrypt, so a chunk that fails verification
		// still has a plaintext sequence header to record.
		const uri = ua.SecurityPolicyURIBasic256Sha256
		s, peer := newSequenceNumberTestChannelWithMode(t, uri, ua.MessageSecurityModeSign)

		nonce := bytes.Repeat([]byte{0x5a}, 32)
		algo, err := uapolicy.Symmetric(uri, nonce, nonce)
		require.NoError(t, err)
		otherAlgo, err := uapolicy.Symmetric(uri, bytes.Repeat([]byte{0xa5}, 32), bytes.Repeat([]byte{0xa5}, 32))
		require.NoError(t, err)

		peer.write(t, 1, 2, uri, ua.MessageSecurityModeSign, algo)
		_, err = s.readChunk()
		require.NoError(t, err, "the first chunk must be read")

		// Signed with keys the channel does not know, the way a chunk
		// secured with an expired or unknown token fails verification.
		peer.write(t, 2, 3, uri, ua.MessageSecurityModeSign, otherAlgo)
		_, err = s.readChunk()
		require.Error(t, err, "a chunk with an unknown token must be rejected")

		peer.write(t, 3, 3, uri, ua.MessageSecurityModeSign, algo)
		_, err = s.readChunk()
		require.NoError(t, err, "the rejected chunk consumed its SequenceNumber, so the next one is no gap")
	})

	t.Run("an ignored chunk with an unexpected number does not move the counter", func(t *testing.T) {
		// Sign, not SignAndEncrypt, so a chunk that fails verification
		// still has a plaintext sequence header to record.
		const uri = ua.SecurityPolicyURIBasic256Sha256
		s, peer := newSequenceNumberTestChannelWithMode(t, uri, ua.MessageSecurityModeSign)

		nonce := bytes.Repeat([]byte{0x5a}, 32)
		algo, err := uapolicy.Symmetric(uri, nonce, nonce)
		require.NoError(t, err)
		otherAlgo, err := uapolicy.Symmetric(uri, bytes.Repeat([]byte{0xa5}, 32), bytes.Repeat([]byte{0xa5}, 32))
		require.NoError(t, err)

		peer.write(t, 1, 2, uri, ua.MessageSecurityModeSign, algo)
		_, err = s.readChunk()
		require.NoError(t, err, "the first chunk must be read")

		// Signed with keys the channel does not know and carrying a
		// number other than the expected successor, so the chunk is
		// dropped and must not move the counter either.
		peer.write(t, 100, 3, uri, ua.MessageSecurityModeSign, otherAlgo)
		_, err = s.readChunk()
		require.Error(t, err, "a chunk with an unknown token must be rejected")

		peer.write(t, 2, 3, uri, ua.MessageSecurityModeSign, algo)
		_, err = s.readChunk()
		require.NoError(t, err, "the dropped chunk left the counter alone, so the successor of the first chunk is no gap")
	})
}

// newSequenceNumberTestChannel returns a client SecureChannel reading MSG
// chunks under SecurityPolicy None, and a peer that scripts their
// SequenceNumbers. Each test builds a fresh channel, so the counter starts
// unseeded.
func newSequenceNumberTestChannel(t *testing.T) (*SecureChannel, *sequenceNumberPeer) {
	return newSequenceNumberTestChannelWithMode(t, ua.SecurityPolicyURINone, ua.MessageSecurityModeNone)
}

func newSequenceNumberTestChannelWithMode(t *testing.T, policyURI string, mode ua.MessageSecurityMode) (*SecureChannel, *sequenceNumberPeer) {
	t.Helper()

	serverTCP, clientTCP := newTestTCPConnPair(t)
	t.Cleanup(func() { _ = serverTCP.Close() })
	t.Cleanup(func() { _ = clientTCP.Close() })

	uacpConn, err := uacp.NewConn(serverTCP, &uacp.Acknowledge{
		ReceiveBufSize: 65535,
		SendBufSize:    65535,
		MaxMessageSize: 2 * 1024 * 1024,
		MaxChunkCount:  512,
	})
	require.NoError(t, err)

	s := &SecureChannel{
		c: uacpConn,
		cfg: &Config{
			SecurityPolicyURI: policyURI,
			SecurityMode:      mode,
			RequestTimeout:    time.Second,
		},
		instances: make(map[uint32][]*channelInstance),
	}
	instance := newChannelInstance(s)
	instance.state = channelActive
	instance.secureChannelID = 1
	instance.securityTokenID = 2
	if mode != ua.MessageSecurityModeNone {
		nonce := bytes.Repeat([]byte{0x5a}, 32)
		instance.algo, err = uapolicy.Symmetric(policyURI, nonce, nonce)
		require.NoError(t, err)
	}
	s.instances[1] = []*channelInstance{instance}
	s.activeInstance = instance

	return s, &sequenceNumberPeer{conn: clientTCP}
}

// sequenceNumberPeer writes single-chunk MSG responses to the channel under
// test, the way a server answers a request.
type sequenceNumberPeer struct {
	conn io.Writer
}

// write sends one final MSG chunk carrying the given SequenceNumber, signed
// and encrypted with algo when the mode asks for it.
func (p *sequenceNumberPeer) write(t *testing.T, seq, reqID uint32, policyURI string, mode ua.MessageSecurityMode, algo *uapolicy.EncryptionAlgorithm) {
	t.Helper()

	m := &Message{
		MessageHeader: &MessageHeader{
			Header:                  NewHeader(MessageTypeMessage, ChunkTypeFinal, 1),
			SymmetricSecurityHeader: NewSymmetricSecurityHeader(2),
			SequenceHeader:          NewSequenceHeader(seq, reqID),
		},
		TypeID: ua.NewFourByteExpandedNodeID(0, id.ReadResponse_Encoding_DefaultBinary),
		Service: &ua.ReadResponse{
			ResponseHeader: &ua.ResponseHeader{
				Timestamp:          time.Now(),
				RequestHandle:      reqID,
				ServiceDiagnostics: &ua.DiagnosticInfo{},
				StringTable:        []string{},
				AdditionalHeader:   ua.NewExtensionObject(nil),
			},
		},
	}
	chunks, err := m.EncodeChunks(math.MaxUint32)
	require.NoError(t, err)
	require.Len(t, chunks, 1)

	b := chunks[0]
	if mode != ua.MessageSecurityModeNone {
		signer := &channelInstance{
			sc:   &SecureChannel{cfg: &Config{SecurityMode: mode, SecurityPolicyURI: policyURI}},
			algo: algo,
		}
		b, err = signer.signAndEncrypt(m, b)
		require.NoError(t, err)
	}

	_, err = p.conn.Write(b)
	require.NoError(t, err)
}

func writeSequenceNumberChunk(t *testing.T, p *sequenceNumberPeer, seq, reqID uint32) {
	p.write(t, seq, reqID, ua.SecurityPolicyURINone, ua.MessageSecurityModeNone, nil)
}
