//go:build conformance

// Package securechannel holds the gopcua secure channel tests: one
// behaviour per file, with the clause it checks in the file's opening
// comment. This file holds the fixture they all run against; it is not
// a row.
//
// The fixture is a scripted TCP peer, built on net and not on gopcua's
// uacp server, that plays the server's half of the secure channel: it
// answers the HEL with the ACK, answers the OPN with a security token
// whose channel id, token id, createdAt and lifetime the row chose -
// and answers the client's own renewal OPN the same way - and answers
// the client's MSG requests with the response bodies the row's script
// picks, stamped with the sequence numbers the row picked. Rows 1, 2
// and 4 run under SecurityPolicy None, so every chunk is in the clear
// and the None codec below encodes and decodes them; the expired-token
// row runs under Basic256Sha256/Sign, and the codec for that policy -
// the asymmetric and symmetric signing - lives with that row, because
// no second row calls it.
//
// The peer accepts more than one connection, each served on its own
// generation: the ServiceFault row scores a client that drops the
// channel and dials again, and what it does on the way back is visible
// only if the peer answers the new connection too. The listen, the
// HEL/ACK opening of each connection, and the split of a failed read
// into one that merely ended and one that only the client's own bytes
// can explain are the core the package shares with the transport and
// securitypolicy packages, in tests/internal/scriptedpeer.
//
// Every helper here returns plain values. A helper does not call
// t.Errorf or t.Fatalf; a row's test function turns a non-empty reason
// string into the failure or the decline, and it, not the fixture,
// registers the teardown with t.Cleanup.
//
// The package name is securechannel, as the directory is; the
// production code it drives lives in github.com/gopcua/opcua and its
// uasc package, on the same line as OPC 10000-6's SecureChannel.
package securechannel

import (
	"errors"
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	"github.com/gopcua/opcua/tests/internal/scriptedpeer"
	"github.com/gopcua/opcua/ua"
	"github.com/gopcua/opcua/uacp"
	"github.com/gopcua/opcua/uasc"
)

// bound caps every read the peer makes on an open connection: how long
// it waits for the client's next chunk. It only lets a slow machine
// finish; it never waits out a broken one, because the teardown closes
// the connection and the read ends. A read that the bound ends is a
// fact the rows read in the peer's close records, not a failure of
// either side on its own.
const bound = 10 * time.Second

// peerAck is the Acknowledge the scripted peer answers the HEL with:
// protocol version 0, 65535-byte buffers in both directions, and no
// message-size or chunk-count limit.
var peerAck = &uacp.Acknowledge{
	Version:        0,
	ReceiveBufSize: 65535,
	SendBufSize:    65535,
}

// channelToken is the security token the peer hands out in one
// OpenSecureChannelResponse: the SecureChannelId and TokenId it
// carries, the createdAt it is stamped with, and its revised lifetime
// in milliseconds. Which clock stamped createdAt is one row's whole
// subject; which ids the token carries are another row's.
type channelToken struct {
	channelID uint32
	tokenID   uint32
	createdAt time.Time
	lifetime  uint32
}

// chunkCodec decodes the chunks the client sends and encodes the chunks
// the peer answers with, under one security policy. The None codec
// below is the default, and the rows under None share it; a row under a
// signing policy - the expired-token row, under Basic256Sha256/Sign -
// supplies its own, holding the policy's asymmetric and symmetric
// signing in its own file because no second row calls it. The codec is
// not the library grading itself: the rows score what the client does
// with the chunks, and a row that needs the chunk's own bytes decodes
// them itself, never through the codec.
type chunkCodec interface {
	// decodeRequest decodes one chunk the client sent and returns the
	// service request it carries, with the request id from its sequence
	// header.
	decodeRequest(frame []byte) (req ua.Request, reqID uint32, err error)

	// encodeOPN encodes a final OPN chunk answering an
	// OpenSecureChannelRequest with the given token, sequence number
	// and request id.
	encodeOPN(token channelToken, seq, reqID uint32) ([]byte, error)

	// encodeMSG encodes a final MSG chunk carrying resp under the given
	// channel id and token id, with the given sequence number and
	// request id.
	encodeMSG(channelID, tokenID, seq, reqID uint32, resp ua.Response) ([]byte, error)
}

// noneCodec is the chunkCodec of SecurityPolicy None: every chunk is in
// the clear, so decoding is uasc's own chunk and service decode and
// encoding is uasc's own message encode, which moves bytes the rows
// never score.
type noneCodec struct{}

// decodeRequest decodes a clear chunk: the whole secure-conversation
// message, headers and service body.
func (noneCodec) decodeRequest(frame []byte) (ua.Request, uint32, error) {
	var m uasc.Message
	if _, err := m.Decode(frame); err != nil {
		return nil, 0, err
	}
	req, ok := m.Service.(ua.Request)
	if !ok {
		return nil, 0, fmt.Errorf("the chunk's body is %T, not a service request", m.Service)
	}
	return req, m.SequenceHeader.RequestID, nil
}

// encodeOPN encodes an OpenSecureChannelResponse handing out token: a
// null policy URI, no certificates, and the response header built by
// responseHeader. The sequence number and the request id are exactly
// the caller's, whatever they are.
func (noneCodec) encodeOPN(token channelToken, seq, reqID uint32) ([]byte, error) {
	resp := &ua.OpenSecureChannelResponse{
		ResponseHeader:        responseHeader(reqID, ua.StatusOK),
		ServerProtocolVersion: 0,
		SecurityToken: &ua.ChannelSecurityToken{
			ChannelID:       token.channelID,
			TokenID:         token.tokenID,
			CreatedAt:       token.createdAt,
			RevisedLifetime: token.lifetime,
		},
		ServerNonce: nil,
	}
	m := &uasc.Message{
		MessageHeader: &uasc.MessageHeader{
			Header:                   uasc.NewHeader(uasc.MessageTypeOpenSecureChannel, uasc.ChunkTypeFinal, token.channelID),
			AsymmetricSecurityHeader: uasc.NewAsymmetricSecurityHeader(ua.SecurityPolicyURINone, nil, nil),
			SequenceHeader:           uasc.NewSequenceHeader(seq, reqID),
		},
		TypeID:  ua.NewFourByteExpandedNodeID(0, ua.ServiceTypeID(resp)),
		Service: resp,
	}
	return m.Encode()
}

// encodeMSG encodes a final MSG chunk carrying resp under the given
// channel id and token id, with the sequence number and the request id
// exactly the caller's.
func (noneCodec) encodeMSG(channelID, tokenID, seq, reqID uint32, resp ua.Response) ([]byte, error) {
	m := &uasc.Message{
		MessageHeader: &uasc.MessageHeader{
			Header:                  uasc.NewHeader(uasc.MessageTypeMessage, uasc.ChunkTypeFinal, channelID),
			SymmetricSecurityHeader: uasc.NewSymmetricSecurityHeader(tokenID),
			SequenceHeader:          uasc.NewSequenceHeader(seq, reqID),
		},
		TypeID:  ua.NewFourByteExpandedNodeID(0, ua.ServiceTypeID(resp)),
		Service: resp,
	}
	return m.Encode()
}

// peerScript scripts the peer. Every field is chosen by the row that
// runs against the peer, so the numbers the rows score come from the
// script and never from the library under test.
type peerScript struct {
	// issue is the token in the first OpenSecureChannelResponse on each
	// connection; renew is the token in every renewal response after
	// that. A client that drops the channel and dials again opens a new
	// connection, whose first OPN is answered with issue.
	issue channelToken
	renew channelToken

	// answer says how to answer one service request that arrived as a
	// MSG chunk: it returns the response body to send, or nil to answer
	// nothing. It is called on the peer's serve goroutine, so it must
	// not block past the exchange it delays - the lifetime row holds
	// the answer it scores on for its own delay, which is part of its
	// stimulus and not a bound the harness owns.
	answer func(req ua.Request, reqID uint32) ua.Response

	// msgSeqs, when set, are the SequenceNumbers the peer stamps on its
	// MSG responses, in the order it answers them, in place of its own
	// counter; the sequence-number row uses them to violate OPC
	// 10000-6's sender rules on the wire. When unset, the peer is a
	// well-behaved sender that increments its counter by exactly one
	// per chunk.
	msgSeqs []uint32

	// codec is the security policy the peer speaks; nil means
	// SecurityPolicy None.
	codec chunkCodec
}

// requestRecord is one service request the peer decoded off the wire,
// with the connection generation it arrived on. Conn counts accepted
// connections from 1.
type requestRecord struct {
	Conn int
	Req  ua.Request
	At   time.Time
}

// acceptRecord is one TCP connection the peer accepted.
type acceptRecord struct {
	Conn int
	At   time.Time
}

// closeRecord is one connection whose reads ended - a close, a reset,
// an EOF - with the moment the peer noticed.
type closeRecord struct {
	Conn int
	At   time.Time
}

// peerRecord is what the peer saw, frozen while its connections may
// still be live: the connections it accepted, the requests it decoded
// with the generation each arrived on, and the connections whose reads
// ended. The ServiceFault row reads all three to tell a client that
// stayed on one connection from one that came back on a second.
type peerRecord struct {
	Accepts  []acceptRecord
	Requests []requestRecord
	Closed   []closeRecord
}

// scriptedPeer is the scripted TCP peer: it accepts connections until
// the listener closes in teardown, and plays the script on each. What
// the script answers is the rows' business; what the peer records -
// accepts, requests, ends of reads - is what the rows read back.
type scriptedPeer struct {
	lis    net.Listener
	script peerScript

	mu       sync.Mutex
	accepts  []acceptRecord
	requests []requestRecord
	closed   []closeRecord
	conns    []net.Conn
	nextConn int
	fixture  string // the peer's own failure, a decline
	client   string // the client's own bytes, gopcua's behaviour
}

// startPeer brings the peer up on the loopback interface, on the shared
// core's listener (tests/internal/scriptedpeer). It returns the peer and
// a teardown for the listener and every connection it accepted - or,
// when the peer could not be started, the reason the run cannot be
// scored. The row's test function, not the fixture, registers the
// teardown with t.Cleanup.
func startPeer(script peerScript) (*scriptedPeer, func(), string) {
	l, err := scriptedpeer.Listen("127.0.0.1")
	if err != nil {
		return nil, func() {}, fmt.Sprintf("listen on the loopback interface: %v", err)
	}
	p := &scriptedPeer{lis: l, script: script}
	go p.serve()
	teardown := func() {
		_ = l.Close()
		p.mu.Lock()
		defer p.mu.Unlock()
		for _, c := range p.conns {
			_ = c.Close()
		}
	}
	return p, teardown, ""
}

// addr is the host:port the peer listens on.
func (p *scriptedPeer) addr() string { return p.lis.Addr().String() }

// serve accepts connections until the listener closes in teardown and
// serves each on its own goroutine, so a client that drops one
// connection and dials again is served on a second generation while the
// first is still draining.
func (p *scriptedPeer) serve() {
	for {
		c, err := p.lis.Accept()
		if err != nil {
			return // the listener was closed in teardown
		}
		p.mu.Lock()
		p.nextConn++
		connID := p.nextConn
		p.conns = append(p.conns, c)
		p.accepts = append(p.accepts, acceptRecord{Conn: connID, At: time.Now()})
		p.mu.Unlock()
		go p.serveConn(connID, c)
	}
}

// serveConn plays the script on one connection: the shared HEL/ACK
// exchange, then the client's chunks until the connection's reads end.
// An OPN is answered with the script's token - issue for a request of
// type Issue, renew for a renewal - and a MSG is recorded and handed to
// the script's answer, whose response is stamped with the peer's next
// sequence number, or the next of msgSeqs when the script set them. A
// CLO is recorded and never answered; the connection's end is still
// read afterwards, so the row sees the connection go. Every encode
// failure and every write the peer could not complete is the peer's
// own, a decline that records the connection's close and nothing
// else - a write the peer could not complete is never the client's
// bytes; the opening exchange fails the same way on the peer's side,
// with scriptedpeer.HelloAck's ErrPeerSide. Every chunk the codec
// cannot decode, and every frame off script, can only be the client's
// own bytes, and the setup check reports it as gopcua's behaviour.
func (p *scriptedPeer) serveConn(connID int, c net.Conn) {
	codec := p.script.codec
	if codec == nil {
		codec = noneCodec{}
	}

	uc, err := scriptedpeer.HelloAck(c.(*net.TCPConn), peerAck)
	if err != nil {
		// The three outcomes of the opening exchange: the peer's own
		// failure records the connection's close and nothing else, so
		// the row declines; a frame that is not a HEL is the client's
		// own bytes; a receive error keeps the ended-connection split.
		var off *scriptedpeer.OffScriptError
		switch {
		case errors.Is(err, scriptedpeer.ErrPeerSide):
			p.noteClose(connID)
		case errors.As(err, &off):
			p.noteClientBytes(off)
		default:
			p.noteReadEnd(connID, err)
		}
		return
	}

	token := p.script.issue
	seq := uint32(0)
	answered := 0
	for {
		_ = uc.SetReadDeadline(time.Now().Add(bound))
		frame, err := uc.Receive()
		if err != nil {
			p.noteReadEnd(connID, err)
			return
		}

		switch string(frame[:3]) {
		case uasc.MessageTypeOpenSecureChannel:
			req, reqID, err := codec.decodeRequest(frame)
			if err != nil {
				p.noteClientBytes(fmt.Errorf("decode an OPN chunk: %w", err))
				return
			}
			opn, ok := req.(*ua.OpenSecureChannelRequest)
			if !ok {
				p.noteClientBytes(fmt.Errorf("an OPN chunk carried %T, not an OpenSecureChannelRequest", req))
				return
			}
			token = p.script.renew
			if opn.RequestType == ua.SecurityTokenRequestTypeIssue {
				token = p.script.issue
			}
			seq++
			resp, err := codec.encodeOPN(token, seq, reqID)
			if err != nil {
				p.noteFixture(fmt.Errorf("encode the OpenSecureChannelResponse: %w", err))
				return
			}
			if _, err := uc.Write(resp); err != nil {
				p.noteClose(connID)
				return
			}

		case uasc.MessageTypeMessage:
			req, reqID, err := codec.decodeRequest(frame)
			if err != nil {
				p.noteClientBytes(fmt.Errorf("decode a MSG chunk: %w", err))
				return
			}
			p.noteRequest(connID, req)
			if p.script.answer == nil {
				continue
			}
			resp := p.script.answer(req, reqID)
			if resp == nil {
				continue
			}
			s := seq + 1
			if answered < len(p.script.msgSeqs) {
				s = p.script.msgSeqs[answered]
			}
			answered++
			chunk, err := codec.encodeMSG(token.channelID, token.tokenID, s, reqID, resp)
			if err != nil {
				p.noteFixture(fmt.Errorf("encode a MSG response: %w", err))
				return
			}
			if _, err := uc.Write(chunk); err != nil {
				p.noteClose(connID)
				return
			}
			seq = s

		case uasc.MessageTypeCloseSecureChannel:
			req, _, err := codec.decodeRequest(frame)
			if err != nil {
				p.noteClientBytes(fmt.Errorf("decode a CLO chunk: %w", err))
				return
			}
			p.noteRequest(connID, req)
			// No response is defined for a CLO; keep reading so the
			// connection's end is recorded when the client closes it.

		default:
			p.noteClientBytes(fmt.Errorf("a %q frame arrived, where the script expected OPN, MSG or CLO", string(frame[:4])))
			return
		}
	}
}

// noteRequest records one request the peer decoded, with the
// connection generation it arrived on.
func (p *scriptedPeer) noteRequest(connID int, req ua.Request) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.requests = append(p.requests, requestRecord{Conn: connID, Req: req, At: time.Now()})
}

// noteClose records that connID's exchange ended - a write the peer
// could not complete, or the opening exchange failing on the peer's
// side - and nothing else: the row reads the close in its records, and
// the run stays a decline, because a write the peer could not complete
// is never the client's bytes.
func (p *scriptedPeer) noteClose(connID int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.closed = append(p.closed, closeRecord{Conn: connID, At: time.Now()})
}

// noteReadEnd records that connID's reads ended with err: a close
// record for the row to read - and, unless the connection merely ended
// or the bound passed, the client's own bytes, which the setup check
// reports as gopcua's behaviour. The ended-connection split is the
// shared scriptedpeer.ConnectionEnded; a read deadline is the bound
// passing, which can close on either side and stays a fact the row
// reads in the close records.
func (p *scriptedPeer) noteReadEnd(connID int, err error) {
	p.noteClose(connID)
	if scriptedpeer.ConnectionEnded(err) || errors.Is(err, os.ErrDeadlineExceeded) {
		return
	}
	p.noteClientBytes(err)
}

// noteFixture records the peer's own failure - the reason the run
// cannot be scored and the fixture caused it. Only the first is kept;
// the first is the one the setup check reports.
func (p *scriptedPeer) noteFixture(err error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.fixture == "" {
		p.fixture = err.Error()
	}
}

// noteClientBytes records a failure only the client's own bytes can
// explain - a chunk the codec cannot decode, a frame off script, bytes
// the uacp framing rejects - which the setup check reports as gopcua's
// behaviour, not as a decline. Only the first is kept.
func (p *scriptedPeer) noteClientBytes(err error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.client == "" {
		p.client = err.Error()
	}
}

// record freezes what the peer saw. It copies under the one lock, so
// the record describes one instant of the run.
func (p *scriptedPeer) record() peerRecord {
	p.mu.Lock()
	defer p.mu.Unlock()
	return peerRecord{
		Accepts:  append([]acceptRecord(nil), p.accepts...),
		Requests: append([]requestRecord(nil), p.requests...),
		Closed:   append([]closeRecord(nil), p.closed...),
	}
}

// setupCheck is the check the rows share: it returns "" when the
// fixture's own exchange went clean, or the reason the run cannot be
// scored and which side caused it - the peer's own failure (false, a
// decline the row prints with "could not set the test up") or the
// client's own bytes (true, gopcua's behaviour, which the row prints
// with "gopcua's own behaviour made this run unscoreable" and t.Errorf,
// then returns). What it does not cover, the row's own check does: a
// stage the client never reached - no OPN, no session request - is the
// row's to word, because which stages matter is the row's own story.
func (p *scriptedPeer) setupCheck() (why string, fromGopcua bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.fixture != "" {
		return p.fixture, false
	}
	if p.client != "" {
		return p.client, true
	}
	return "", false
}

// responseHeader builds the ResponseHeader every scripted service
// response carries: the request's id echoed as its handle - no row
// scores the handle - the given status, empty diagnostics, and a null
// AdditionalHeader. The pointer fields are filled in because the
// reflective encoder writes nothing for a nil pointer, which would
// corrupt the byte stream.
func responseHeader(handle uint32, status ua.StatusCode) *ua.ResponseHeader {
	return &ua.ResponseHeader{
		Timestamp:          time.Now(),
		RequestHandle:      handle,
		ServiceResult:      status,
		ServiceDiagnostics: &ua.DiagnosticInfo{},
		StringTable:        []string{},
		AdditionalHeader:   ua.NewExtensionObject(nil),
	}
}
