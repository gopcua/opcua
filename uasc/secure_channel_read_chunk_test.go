package uasc

import (
	"encoding/binary"
	"io"
	"testing"

	"github.com/gopcua/opcua/ua"
	"github.com/gopcua/opcua/uacp"
	"github.com/stretchr/testify/require"
)

// TestReadChunkRefusesAChunkShorterThanItsHeader_Part6_672 feeds readChunk a
// chunk that stops before the 12 byte secure conversation header is complete.
//
// uacp.Conn.Receive reads the eight byte uacp transport header first, checks
// only that the MessageSize it carries is between those eight bytes and the
// negotiated receive buffer, and returns b[:MessageSize]. A peer that
// acknowledges a receive buffer of eight bytes can therefore hand readChunk a
// slice of exactly eight bytes. With an eight-byte receive buffer there is no
// capacity past the chunk; with a larger one the unchecked slice would read
// stale bytes from the same buffer.
//
// OPC 10000-6 §6.7.2 defines the secure conversation header that every
// MessageChunk starts with; a chunk that ends before that header is complete
// cannot be decoded, and readChunk already reports every other chunk it cannot
// decode as ua.StatusBadDecodingError, so this one is too.
func TestReadChunkRefusesAChunkShorterThanItsHeader_Part6_672(t *testing.T) {
	// "MSGF" plus a MessageSize of 8: the chunk ends right after the uacp
	// header, five bytes before the secure conversation header is complete.
	// MessageSize is the only value the uacp layer checks, and 8 is the
	// smallest it accepts.
	chunk := make([]byte, 8)
	chunk[0], chunk[1], chunk[2], chunk[3] = 'M', 'S', 'G', 'F'
	binary.LittleEndian.PutUint32(chunk[4:], uint32(len(chunk)))

	// With an eight-byte receive buffer there is no capacity past the chunk.
	t.Run("receive buffer of eight bytes", func(t *testing.T) {
		testReadChunkShortChunk(t, chunk, 8)
	})

	// A receive buffer larger than the chunk leaves capacity past it, where
	// the unchecked slice would read stale bytes from the same buffer.
	t.Run("receive buffer larger than the chunk", func(t *testing.T) {
		testReadChunkShortChunk(t, chunk, 65535)
	})
}

func testReadChunkShortChunk(t *testing.T, chunk []byte, recvBufSize uint32) {
	t.Helper()

	serverTCP, clientTCP := newTestTCPConnPair(t)
	defer serverTCP.Close()
	defer clientTCP.Close()

	conn, err := uacp.NewConn(clientTCP, &uacp.Acknowledge{
		ReceiveBufSize: recvBufSize,
		SendBufSize:    recvBufSize,
		MaxMessageSize: 2 * 1024 * 1024,
		MaxChunkCount:  512,
	})
	require.NoError(t, err, "error: create uacp conn")

	s, err := NewSecureChannel("opc.tcp://localhost", conn, &Config{
		SecurityPolicyURI: ua.SecurityPolicyURINone,
		SecurityMode:      ua.MessageSecurityModeNone,
	}, make(chan error, 1))
	require.NoError(t, err, "error: create secure channel")

	_, err = serverTCP.Write(chunk)
	require.NoError(t, err, "error: write short chunk")

	_, err = s.readChunk()
	require.ErrorIs(t, err, ua.StatusBadDecodingError,
		"a chunk shorter than its header must be refused, not sliced")
}

// TestReadChunkReturnsAMessageTooLargeError feeds readChunk a chunk whose
// MessageSize exceeds the receive buffer. uacp rejects it with a 'message too
// large' error that readChunk must pass on unchanged: folding it into io.EOF
// turns a misbehaving peer into a bare 'EOF' for the caller.
func TestReadChunkReturnsAMessageTooLargeError(t *testing.T) {
	serverTCP, clientTCP := newTestTCPConnPair(t)
	defer serverTCP.Close()
	defer clientTCP.Close()

	// A receive buffer of eight bytes is the smallest uacp accepts.
	const recvBufSize = 8
	conn, err := uacp.NewConn(clientTCP, &uacp.Acknowledge{
		ReceiveBufSize: recvBufSize,
		SendBufSize:    recvBufSize,
		MaxMessageSize: 2 * 1024 * 1024,
		MaxChunkCount:  512,
	})
	require.NoError(t, err, "error: create uacp conn")

	s, err := NewSecureChannel("opc.tcp://localhost", conn, &Config{
		SecurityPolicyURI: ua.SecurityPolicyURINone,
		SecurityMode:      ua.MessageSecurityModeNone,
	}, make(chan error, 1))
	require.NoError(t, err, "error: create secure channel")

	// "MSGF" plus a MessageSize twice the receive buffer: uacp rejects
	// the chunk after reading the transport header, before any chunk bytes.
	chunk := make([]byte, 2*recvBufSize)
	chunk[0], chunk[1], chunk[2], chunk[3] = 'M', 'S', 'G', 'F'
	binary.LittleEndian.PutUint32(chunk[4:], uint32(len(chunk)))

	_, err = serverTCP.Write(chunk)
	require.NoError(t, err, "error: write oversized chunk")

	_, err = s.readChunk()
	require.Error(t, err, "an oversized chunk must be reported")
	require.NotErrorIs(t, err, io.EOF, "an oversized chunk is not an end of stream")
	require.Contains(t, err.Error(), "message too large")
}
