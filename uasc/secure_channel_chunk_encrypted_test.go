package uasc

import (
	"bytes"
	"context"
	"encoding/binary"
	"testing"
	"time"

	"github.com/gopcua/opcua/ua"
	"github.com/gopcua/opcua/uacp"
	"github.com/gopcua/opcua/uapolicy"
	"github.com/stretchr/testify/require"
)

// TODO: extend chunked-send test coverage:
//   - round-trip: feed the multi-chunk stream back through Receive/mergeChunks/
//     verifyAndDecrypt and assert the decoded payload is byte-identical (the
//     tests below assert framing and size, not payload integrity);
//   - Sign-only mode (MessageSecurityModeSign): only None (see
//     TestSendResponseWithContextWritesAllChunks) and SignAndEncrypt are covered;
//   - boundary body sizes: exactly maxBodySize, maxBodySize+1, and an exact
//     multiple of maxBodySize (which yields an empty final chunk);
//   - concurrent SendResponseWithContext calls must not interleave chunks (the
//     per-instance lock added alongside writeMessageChunks).

// TestServerChunkedResponseEncryptedFitsChunk_Part6_672 drives the real server
// send path (SendResponseWithContext -> sendResponseWithContext ->
// writeMessageChunks) under an encrypting security policy and reads the framing
// off a TCP socket. PR #861's own regression test runs SecurityMode None, so it
// never exercises the per-chunk sign/encrypt path where the #781/#824 padding
// math lives; this covers it for every symmetric policy.
//
// OPC UA Part 6 v1.05 §6.7.2: a Message larger than one MessageChunk is split
// into multiple chunks; each chunk including its security footer must not exceed
// the negotiated chunk size (SendBufferSize), intermediate chunks are 'C' and the
// last is 'F'. (The per-chunk SequenceNumber is monotonic but is inside the
// encrypted region here, so TestSendResponseWithContextWritesAllChunks asserts
// that in None mode; this test asserts the size/framing invariant under crypto.)
func TestServerChunkedResponseEncryptedFitsChunk_Part6_672(t *testing.T) {
	const bufSize = 64 * 1024
	nonce := bytes.Repeat([]byte{0x5a}, 32)

	for _, uri := range symmetricPolicyURIs {
		t.Run(uri, func(t *testing.T) {
			serverTCP, clientTCP := newTestTCPConnPair(t)
			defer serverTCP.Close()
			defer clientTCP.Close()

			uacpConn, err := uacp.NewConn(serverTCP, &uacp.Acknowledge{
				ReceiveBufSize: bufSize,
				SendBufSize:    bufSize,
				MaxMessageSize: 8 * 1024 * 1024,
				MaxChunkCount:  1024,
			})
			require.NoError(t, err)

			algo, err := uapolicy.Symmetric(uri, nonce, nonce)
			require.NoError(t, err)

			s := &SecureChannel{
				c: uacpConn,
				cfg: &Config{
					SecurityPolicyURI: uri,
					SecurityMode:      ua.MessageSecurityModeSignAndEncrypt,
					RequestTimeout:    time.Second,
				},
				instances: make(map[uint32][]*channelInstance),
			}
			instance := newChannelInstance(s)
			instance.algo = algo
			instance.state = channelActive
			instance.secureChannelID = 1
			instance.securityTokenID = 1
			instance.SetMaximumBodySize(int(s.c.SendBufSize()))
			s.activeInstance = instance

			readDone := make(chan readChunksResult, 1)
			go func() { readDone <- readChunksUntilFinal(clientTCP) }()

			// A response a few times larger than one body forces several chunks.
			valueBytes := make([]byte, int(instance.maxBodySize)*3)
			for i := range valueBytes {
				valueBytes[i] = byte(i)
			}
			v, err := ua.NewVariant(valueBytes)
			require.NoError(t, err)
			dv := &ua.DataValue{Value: v}
			dv.UpdateMask()
			resp := &ua.ReadResponse{
				ResponseHeader: &ua.ResponseHeader{
					Timestamp:          time.Now(),
					RequestHandle:      1,
					ServiceDiagnostics: &ua.DiagnosticInfo{},
					StringTable:        []string{},
					AdditionalHeader:   ua.NewExtensionObject(nil),
				},
				Results: []*ua.DataValue{dv},
			}

			require.NoError(t, s.SendResponseWithContext(context.Background(), 1, resp))

			select {
			case result := <-readDone:
				require.NoError(t, result.err)
				chunks := result.chunks
				require.Greater(t, len(chunks), 1, "response should span multiple chunks")
				for i, chunk := range chunks {
					require.Equal(t, MessageTypeMessage, string(chunk[:3]))
					wantType := byte(ChunkTypeIntermediate)
					if i == len(chunks)-1 {
						wantType = ChunkTypeFinal
					}
					require.Equal(t, wantType, chunk[3], "chunk type")
					// The core #824-interaction invariant on the server multi-chunk
					// path: every encrypted chunk fits the negotiated chunk size.
					require.LessOrEqual(t, len(chunk), bufSize, "encrypted chunk exceeds negotiated chunk size")
					require.EqualValues(t, len(chunk), binary.LittleEndian.Uint32(chunk[4:8]), "MessageSize header must equal chunk length")
				}
			case <-time.After(chunkedResponseTestTimeout):
				t.Fatal("timed out waiting for response chunks")
			}
		})
	}
}
