package uasc

import (
	"context"
	"encoding/binary"
	"io"
	"math"
	"testing"
	"time"

	"github.com/gopcua/opcua/id"
	"github.com/gopcua/opcua/ua"
	"github.com/gopcua/opcua/uacp"
	"github.com/stretchr/testify/require"
)

// msgChunkPrefixLen is the on-wire size of the headers that precede the
// body of a MSG chunk: message header (12), symmetric security header (4)
// and sequence header (8).
const msgChunkPrefixLen = 24

// partialChunkTestTimeout bounds how long a test waits for Receive to
// answer the chunks a peer wrote. A bound that never answers is the
// defect these tests guard against, so waiting forever must fail the
// test instead of hanging.
const partialChunkTestTimeout = 5 * time.Second

// TestReceiveRejectsMessageAboveMaxMessageSize_Part6_7124 guards the
// byte bound per unfinished message (OPC 10000-6 §7.1.2.4): a request id
// whose chunks exceed the negotiated MaxMessageSize is refused at the
// chunk that crosses the limit instead of buffering the rest.
func TestReceiveRejectsMessageAboveMaxMessageSize_Part6_7124(t *testing.T) {
	s, peer := newPartialChunkChannel(t, server, nil)
	defer peer.Close()

	res := startReceive(t, s, 1)

	// 35 chunks of 60000 body bytes add up to 2100000 bytes, more than
	// the 2 MB MaxMessageSize the channel negotiated, all for one
	// request id. The receiver must refuse them while it still holds
	// only the negotiated limit's worth of bytes.
	body := make([]byte, 60000)
	for i := 0; i < 35; i++ {
		peer.writeChunk(t, ChunkTypeIntermediate, 1, body)
	}

	msg := recvPartialMessage(t, res)
	require.Error(t, msg.Err, "a message above the negotiated MaxMessageSize must be reported")

	s.chunksMu.Lock()
	defer s.chunksMu.Unlock()
	require.Empty(t, s.chunks, "the chunks of the refused message must not stay buffered")
}

// TestReceiveRejectsTooManyUnfinishedMessages_Part6_673 guards the cap
// in number of request ids with buffered chunks (OPC 10000-6 §6.7.3):
// more unfinished messages than maxUnfinishedMessages are refused, in
// chunks as well as in bytes.
func TestReceiveRejectsTooManyUnfinishedMessages_Part6_673(t *testing.T) {
	s, peer := newPartialChunkChannel(t, server, nil)
	defer peer.Close()

	res := startReceive(t, s, 1)

	// maxUnfinishedMessages request ids with an unfinished message is
	// the cap; the next request id must be refused. Every body is
	// empty, so a bound that only counts chunk bytes never notices them.
	for reqID := uint32(1); reqID <= maxUnfinishedMessages+1; reqID++ {
		peer.writeChunk(t, ChunkTypeIntermediate, reqID, nil)
	}

	msg := recvPartialMessage(t, res)
	require.Error(t, msg.Err, "more unfinished messages than the per-channel cap must be reported")
	require.Contains(t, msg.Err.Error(), "too many unfinished messages",
		"the error must name the per-channel cap on request ids with an unfinished message")

	s.chunksMu.Lock()
	defer s.chunksMu.Unlock()
	require.Empty(t, s.chunks, "no chunk of a refused request id may stay buffered")
}

// TestReceiveAcceptsMaxMessageSizeAboveInt32_Part6_7124 guards the
// bound against 32-bit integer conversion: a MaxMessageSize of 2^31
// or above is a legal UInt32 (OPC 10000-6 §7.1.2.4) and must not
// overflow an int comparison on 32-bit platforms.
func TestReceiveAcceptsMaxMessageSizeAboveInt32_Part6_7124(t *testing.T) {
	s, peer := newPartialChunkChannel(t, server, &uacp.Acknowledge{
		ReceiveBufSize: 65535,
		SendBufSize:    65535,
		MaxMessageSize: math.MaxUint32,
		MaxChunkCount:  512,
	})
	defer peer.Close()

	res := startReceive(t, s, 1)

	// The first intermediate chunk of a message must be buffered, not
	// refused by an overflowed limit.
	peer.writeChunk(t, ChunkTypeIntermediate, 1, []byte("partial request payload"))

	require.Eventually(t, func() bool {
		s.chunksMu.Lock()
		defer s.chunksMu.Unlock()
		return len(s.chunks[1]) == 1
	}, partialChunkTestTimeout, 10*time.Millisecond,
		"a MaxMessageSize of 2^32-1 must not refuse the first intermediate chunk of a message")

	// A message may buffer up to that limit's worth of bytes, which on
	// a 32-bit platform an int conversion cannot even represent.
	peer.writeChunk(t, ChunkTypeIntermediate, 1, make([]byte, 60000))

	require.Eventually(t, func() bool {
		s.chunksMu.Lock()
		defer s.chunksMu.Unlock()
		return len(s.chunks[1]) == 2
	}, partialChunkTestTimeout, 10*time.Millisecond,
		"chunks of a message below a MaxMessageSize of 2^32-1 must be buffered")

	// Nothing may have been reported while the message assembled below
	// the negotiated limit.
	select {
	case msg := <-res:
		t.Fatalf("no error expected while buffering below a MaxMessageSize of 2^32-1, got: %v", msg.Err)
	default:
	}
}

// TestReceiveBufferedChunkDataIsACopy_Part6_672 checks that a chunk the
// receiver buffers for a later final chunk holds only its own bytes
// (OPC 10000-6 §6.7.2.2 Table 57 sizes a chunk by its MessageSize), not
// the whole receive buffer it was decoded from.
func TestReceiveBufferedChunkDataIsACopy_Part6_672(t *testing.T) {
	s, peer := newPartialChunkChannel(t, server, nil)
	defer peer.Close()

	startReceive(t, s, 1)

	body := []byte("partial request payload")
	peer.writeChunk(t, ChunkTypeIntermediate, 7, body)

	require.Eventually(t, func() bool {
		s.chunksMu.Lock()
		defer s.chunksMu.Unlock()
		return len(s.chunks[7]) == 1
	}, partialChunkTestTimeout, 10*time.Millisecond, "the intermediate chunk must be buffered for its request id")

	s.chunksMu.Lock()
	stored := s.chunks[7][0]
	s.chunksMu.Unlock()

	require.Equal(t, body, stored.Data, "the buffered chunk must carry the body it was sent with")
	require.LessOrEqual(t, cap(stored.Data), 1024,
		"a buffered chunk must hold its own MessageSize bytes, not the whole ReceiveBufSize buffer it was decoded from")
}

// TestReceiveDropsChunksForUnknownRequestIDs_Part6_6724 checks that a
// client drops chunks for request ids it never sent (OPC 10000-6
// §6.7.2.4 Table 60) instead of buffering them.
func TestReceiveDropsChunksForUnknownRequestIDs_Part6_6724(t *testing.T) {
	s, peer := newPartialChunkChannel(t, client, nil)
	defer peer.Close()

	res := startReceive(t, s, 1)

	for reqID := uint32(1); reqID <= 10; reqID++ {
		peer.writeChunk(t, ChunkTypeIntermediate, reqID, []byte("never sent"))
	}
	// An abort chunk for an eleventh, equally unknown request id makes
	// Receive return, so every chunk above has been handled by then.
	abort := make([]byte, 8)
	binary.LittleEndian.PutUint32(abort, uint32(ua.StatusBadTimeout))
	peer.writeChunk(t, ChunkTypeError, 11, abort)

	msg := recvPartialMessage(t, res)
	require.Equal(t, uint32(11), msg.RequestID)

	s.chunksMu.Lock()
	defer s.chunksMu.Unlock()
	require.Empty(t, s.chunks, "a client must not buffer chunks for request ids it never sent")
}

// TestReceiveAssemblesClientChunkedResponse_Part6_673 checks that a
// client assembles a response split into several chunks for a request
// it has in flight: OPC 10000-6 §6.7.3 allows a message to span chunks,
// and only the merged body is the whole response.
func TestReceiveAssemblesClientChunkedResponse_Part6_673(t *testing.T) {
	s, peer := newPartialChunkChannel(t, client, nil)
	defer peer.Close()

	// Register a handler for the request id, the way a request in
	// flight does.
	const reqID = 7
	s.handlersMu.Lock()
	s.handlers[reqID] = make(chan *MessageBody, 1)
	s.handlersMu.Unlock()

	res := startReceive(t, s, 1)

	// A response split into four chunks: three intermediate and a
	// final one. No chunk's body decodes on its own, so only the
	// merged body can answer the request.
	want := &ua.CreateSubscriptionResponse{
		ResponseHeader:            &ua.ResponseHeader{RequestHandle: 1},
		SubscriptionID:            42,
		RevisedPublishingInterval: 1,
		RevisedLifetimeCount:      2,
		RevisedMaxKeepAliveCount:  3,
	}
	body := ua.NewBuffer(nil)
	body.WriteStruct(ua.NewFourByteExpandedNodeID(0, id.CreateSubscriptionResponse_Encoding_DefaultBinary))
	body.WriteStruct(want)
	require.NoError(t, body.Error())

	b := body.Bytes()
	n := len(b) / 4
	peer.writeChunk(t, ChunkTypeIntermediate, reqID, b[:n])
	peer.writeChunk(t, ChunkTypeIntermediate, reqID, b[n:2*n])
	peer.writeChunk(t, ChunkTypeIntermediate, reqID, b[2*n:3*n])
	peer.writeChunk(t, ChunkTypeFinal, reqID, b[3*n:])

	msg := recvPartialMessage(t, res)
	require.Equal(t, uint32(reqID), msg.RequestID)
	require.NoError(t, msg.Err, "the chunks of a response for a request in flight must assemble")

	got, ok := msg.Response().(*ua.CreateSubscriptionResponse)
	require.True(t, ok, "the merged chunks must decode into the response")
	require.Equal(t, want.SubscriptionID, got.SubscriptionID)
	require.Equal(t, want.RevisedPublishingInterval, got.RevisedPublishingInterval)
	require.Equal(t, want.RevisedLifetimeCount, got.RevisedLifetimeCount)
	require.Equal(t, want.RevisedMaxKeepAliveCount, got.RevisedMaxKeepAliveCount)
}

// TestReceiveAssemblesInterleavedPartialMessages_Part6_673 guards the
// legal traffic the bound must not break: interleaved chunks of several
// requests in flight at once (OPC 10000-6 §6.7.3) must each reach their
// own message whole.
func TestReceiveAssemblesInterleavedPartialMessages_Part6_673(t *testing.T) {
	s, peer := newPartialChunkChannel(t, server, nil)
	defer peer.Close()

	recv := func() *MessageBody {
		return recvPartialMessage(t, startReceive(t, s, 1))
	}

	// Two requests in flight at once, their chunks interleaved. The
	// final chunk of request 1 assembles its own chunks only. The
	// body is not a decodable service, so the merged message comes back
	// with an error; the state of request 2's buffer is what matters.
	peer.writeChunk(t, ChunkTypeIntermediate, 1, []byte("one-a"))
	peer.writeChunk(t, ChunkTypeIntermediate, 2, []byte("two-a"))
	peer.writeChunk(t, ChunkTypeFinal, 1, []byte("one-b"))

	msg := recv()
	require.Equal(t, uint32(1), msg.RequestID)
	require.Error(t, msg.Err)

	s.chunksMu.Lock()
	require.Len(t, s.chunks[2], 1, "the chunks of the other request in flight must be kept")
	require.Equal(t, []byte("two-a"), s.chunks[2][0].Data, "the buffered chunk must carry the body it was sent with")
	require.Len(t, s.chunks, 1, "only the unfinished request may be buffered")
	s.chunksMu.Unlock()

	// Finish request 2 and confirm its own chunks reached it.
	peer.writeChunk(t, ChunkTypeIntermediate, 2, []byte("two-b"))
	peer.writeChunk(t, ChunkTypeFinal, 2, []byte("two-c"))

	msg = recv()

	require.Equal(t, uint32(2), msg.RequestID)
	require.Error(t, msg.Err)

	s.chunksMu.Lock()
	defer s.chunksMu.Unlock()
	require.Empty(t, s.chunks, "a finished request must release its buffered chunks")
}

// TestReceiveAssemblesTwoLargeInterleavedMessages_Part6_7124 guards
// the byte bound's scope: MaxMessageSize (OPC 10000-6 §7.1.2.4) is per
// message, so two messages below the limit may buffer more than one
// limit together without being refused.
func TestReceiveAssemblesTwoLargeInterleavedMessages_Part6_7124(t *testing.T) {
	s, peer := newPartialChunkChannel(t, server, nil)
	defer peer.Close()

	const chunksPerMessage = 20
	const chunkBodyLen = 60000 // 20 chunks of body = 1200000 bytes < 2097152

	res := startReceive(t, s, 2)

	// Two messages, each 1200000 body bytes and so below the negotiated
	// MaxMessageSize of 2 MB, their chunks interleaved. Together they
	// buffer 2400000 bytes, more than one MaxMessageSize, which a
	// per-channel byte budget would refuse.
	body := make([]byte, chunkBodyLen)
	for i := 0; i < chunksPerMessage; i++ {
		peer.writeChunk(t, ChunkTypeIntermediate, 1, body)
		peer.writeChunk(t, ChunkTypeIntermediate, 2, body)
	}
	peer.writeChunk(t, ChunkTypeFinal, 1, []byte("one-end"))
	peer.writeChunk(t, ChunkTypeFinal, 2, []byte("two-end"))

	// Both messages must reassemble whole. Their bodies are not a
	// decodable service, so each comes back with a decoding error; what
	// matters is that neither is refused for the bytes it buffers.
	for _, wantReqID := range []uint32{1, 2} {
		msg := recvPartialMessage(t, res)
		require.Equal(t, wantReqID, msg.RequestID, "both interleaved messages must reassemble")
		require.Error(t, msg.Err, "the merged body is not a decodable service")
		require.NotContains(t, msg.Err.Error(), "too many",
			"two messages that each stay below MaxMessageSize must not be refused, whatever they buffer together")
	}

	s.chunksMu.Lock()
	defer s.chunksMu.Unlock()
	require.Empty(t, s.chunks, "finished messages must release their buffered chunks")
}

// startReceive runs Receive n times in the background, one after the
// other, and returns the results on the returned channel.
func startReceive(t *testing.T, s *SecureChannel, n int) <-chan *MessageBody {
	t.Helper()

	res := make(chan *MessageBody, n)
	go func() {
		for i := 0; i < n; i++ {
			res <- s.Receive(context.Background())
		}
	}()
	return res
}

// recvPartialMessage waits for one result of Receive and fails the test
// if none arrives in time.
func recvPartialMessage(t *testing.T, res <-chan *MessageBody) *MessageBody {
	t.Helper()

	select {
	case msg := <-res:
		return msg
	case <-time.After(partialChunkTestTimeout):
		t.Fatalf("Receive did not answer within %s", partialChunkTestTimeout)
		return nil
	}
}

// newPartialChunkChannel returns a SecureChannel that reads MSG chunks
// under SecurityPolicy None and a peer that scripts chunked messages.
// kind selects the channel's role: a server accepts any request id, a
// client only answers to requests it sent. The caller must keep the
// peer alive until the test ends, e.g. with defer peer.Close().
func newPartialChunkChannel(t *testing.T, kind channelKind, ack *uacp.Acknowledge) (*SecureChannel, *partialChunkPeer) {
	t.Helper()

	if ack == nil {
		ack = &uacp.Acknowledge{
			ReceiveBufSize: 65535,
			SendBufSize:    65535,
			MaxMessageSize: 2 * 1024 * 1024,
			MaxChunkCount:  512,
		}
	}

	sechanTCP, peerTCP := newTestTCPConnPair(t)

	uacpConn, err := uacp.NewConn(sechanTCP, ack)
	require.NoError(t, err)

	s := &SecureChannel{
		c:    uacpConn,
		kind: kind,
		cfg: &Config{
			SecurityPolicyURI: ua.SecurityPolicyURINone,
			SecurityMode:      ua.MessageSecurityModeNone,
			RequestTimeout:    time.Second,
		},
		instances: make(map[uint32][]*channelInstance),
		chunks:    make(map[uint32][]*MessageChunk),
		handlers:  make(map[uint32]chan *MessageBody),
	}
	instance := newChannelInstance(s)
	instance.state = channelActive
	instance.secureChannelID = 1
	instance.securityTokenID = 2
	s.instances[1] = []*channelInstance{instance}
	s.activeInstance = instance

	return s, &partialChunkPeer{conn: peerTCP}
}

// partialChunkPeer writes MSG chunks under SecurityPolicy None, drawing a
// strictly increasing SequenceNumber so the chunks pass the receive
// path's sequence check.
type partialChunkPeer struct {
	conn io.Writer
	seq  uint32
}

// Close closes the peer's connection. Tests defer it to keep the peer,
// and with it its TCP connection, alive for the whole test: net sets a
// finalizer on the connection that closes it, so a peer that goes out of
// scope early makes the channel under test read io.EOF.
func (p *partialChunkPeer) Close() error {
	if c, ok := p.conn.(io.Closer); ok {
		return c.Close()
	}
	return nil
}

// writeChunk writes one MSG chunk with the given chunk type, request id
// and body under SecurityPolicy None.
func (p *partialChunkPeer) writeChunk(t *testing.T, chunkType byte, reqID uint32, body []byte) {
	t.Helper()

	p.seq++
	b := make([]byte, msgChunkPrefixLen+len(body))
	copy(b, MessageTypeMessage)
	b[3] = chunkType
	binary.LittleEndian.PutUint32(b[4:], uint32(len(b))) // MessageSize
	binary.LittleEndian.PutUint32(b[8:], 1)              // SecureChannelID
	binary.LittleEndian.PutUint32(b[12:], 2)             // TokenID
	binary.LittleEndian.PutUint32(b[16:], p.seq)
	binary.LittleEndian.PutUint32(b[20:], reqID)
	copy(b[msgChunkPrefixLen:], body)

	_, err := p.conn.Write(b)
	require.NoError(t, err)
}
