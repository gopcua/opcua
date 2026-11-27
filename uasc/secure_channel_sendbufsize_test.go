// Copyright 2018-2020 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uasc

import (
	"context"
	"encoding/binary"
	"net"
	"testing"
	"time"

	"github.com/gopcua/opcua/ua"
	"github.com/gopcua/opcua/uacp"
	"github.com/stretchr/testify/require"
)

// uacpHeaderLen is the size of the UACP message header, uacp.hdrlen, which
// the uacp package does not export.
const uacpHeaderLen = 8

// The chunks the client sends are sized from the Acknowledge's SendBufferSize,
// which the handshake clamps to the ReceiveBufferSize the Hello requested
// (OPC 10000-6 §7.1.2.4); this test checks the clamp where it acts, on the
// wire, with a request large enough to need more than one chunk. A value with
// the top bit set, like the one this test's server answers with, is a
// negative int on 32-bit platforms; that path is not covered by this test.
func TestSecureChannelSendsWithTheSendBufferTheHelloOffered(t *testing.T) {
	const (
		// the buffer the client's Hello offers to receive: the largest chunk
		// the client asked to be sent, and the most this test lets a chunk
		// it sends weigh
		bufsize = uacp.DefaultReceiveBufSize // 0xffff
		// enough ReadValueIDs to span more than one chunk of that size
		nodesToRead = 4000
		// the Acknowledge this test's server answers with, larger than any
		// buffer the Hello requested
		oversizedSendBufSize = 0x80000000
	)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer ln.Close()
	endpoint := "opc.tcp://" + ln.Addr().String() + "/foo/bar"

	// what the server saw: the buffer the Hello requested, the weight of
	// every chunk of the request, and the request's node count
	type serverSaw struct {
		helloReceiveBufSize uint32
		chunkSizes          []uint32
		nodesRead           int
	}
	serverDone := make(chan serverSaw, 1)

	go func() {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		defer c.Close()

		// the server's own connection parameters, which bound what it reads
		srvConn, err := uacp.NewConn(c.(*net.TCPConn), &uacp.Acknowledge{
			ReceiveBufSize: bufsize,
			SendBufSize:    bufsize,
			MaxMessageSize: 2 * uacp.MB,
			MaxChunkCount:  uacp.DefaultMaxChunkCount,
		})
		if err != nil {
			return
		}

		var sizes []uint32

		// answer the Hello with an Acknowledge whose SendBufferSize is larger
		// than the ReceiveBufferSize the Hello requested
		b, err := srvConn.Receive()
		if err != nil {
			return
		}
		hel := new(uacp.Hello)
		if _, err := hel.Decode(b[uacpHeaderLen:]); err != nil {
			return
		}
		ack := &uacp.Acknowledge{
			ReceiveBufSize: bufsize,
			SendBufSize:    oversizedSendBufSize,
			MaxMessageSize: 2 * uacp.MB,
			MaxChunkCount:  uacp.DefaultMaxChunkCount,
		}
		if err := srvConn.Send("ACKF", ack); err != nil {
			return
		}

		sc, err := NewServerSecureChannel(endpoint, srvConn,
			&Config{
				SecurityPolicyURI: ua.SecurityPolicyURINone,
				SecurityMode:      ua.MessageSecurityModeNone,
			},
			make(chan error, 1), 1, 1, 1)
		if err != nil {
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// the OpenSecureChannel request: Receive answers it itself
		if msg := sc.Receive(ctx); msg.Err != nil {
			return
		}

		// the request, one chunk at a time, read at the transport: the
		// weight of a chunk is what this test is about, and the transport is
		// where a chunk larger than the buffer the Hello offered is refused
		//
		// A chunk is 8 bytes of UACP header, then 4 of SecureChannelId, 4 of
		// TokenId and 8 of SequenceHeader, and then the body (OPC 10000-6
		// §6.7.2).
		const chunkHeaderLen = 24 // uacpHeaderLen + 4 SecureChannelId + 4 TokenId + 8 SequenceHeader
		var (
			chunks [][]byte
			reqID  uint32
		)
		for {
			b, err := srvConn.Receive()
			if err != nil {
				return
			}
			if got := string(b[:3]); got != MessageTypeMessage {
				return
			}
			sizes = append(sizes, binary.LittleEndian.Uint32(b[4:8]))
			reqID = binary.LittleEndian.Uint32(b[20:24])
			chunks = append(chunks, b)
			if b[3] == ChunkTypeFinal {
				break
			}
		}

		// the chunks must add up to the whole request
		var body []byte
		for _, chunk := range chunks {
			body = append(body, chunk[chunkHeaderLen:]...)
		}
		_, svc, err := ua.DecodeService(body)
		if err != nil {
			return
		}
		req, ok := svc.(*ua.ReadRequest)
		if !ok {
			return
		}

		// answer it, so the client's send completes rather than timing out
		resp := &ua.ReadResponse{
			ResponseHeader: &ua.ResponseHeader{
				Timestamp:          time.Now(),
				RequestHandle:      req.RequestHeader.RequestHandle,
				ServiceDiagnostics: &ua.DiagnosticInfo{},
				StringTable:        []string{},
				AdditionalHeader:   ua.NewExtensionObject(nil),
				ServiceResult:      ua.StatusOK,
			},
		}
		if err := sc.SendResponseWithContext(ctx, reqID, resp); err != nil {
			return
		}

		serverDone <- serverSaw{
			helloReceiveBufSize: hel.ReceiveBufSize,
			chunkSizes:          sizes,
			nodesRead:           len(req.NodesToRead),
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	conn, err := uacp.Dial(ctx, endpoint)
	require.NoError(t, err)
	defer conn.Close()

	cfg := &Config{
		SecurityPolicyURI: ua.SecurityPolicyURINone,
		SecurityMode:      ua.MessageSecurityModeNone,
		RequestTimeout:    10 * time.Second,
		// long enough that the client does not renew the token mid-test:
		// the renewal is an OpenSecureChannel request of its own, which the
		// fake server above does not answer
		Lifetime: 3600000,
	}
	sc, err := NewSecureChannel(endpoint, conn, cfg, make(chan error, 1))
	require.NoError(t, err)
	defer sc.Close()

	require.NoError(t, sc.Open(ctx), "the handshake and the OpenSecureChannel exchange completed")
	require.LessOrEqual(t, conn.SendBufSize(), uint32(bufsize), "the client kept the send buffer the Hello offered")

	// a request that needs several chunks at the buffer the Hello offered
	req := &ua.ReadRequest{}
	for i := 0; i < nodesToRead; i++ {
		req.NodesToRead = append(req.NodesToRead, &ua.ReadValueID{
			NodeID:      ua.NewNumericNodeID(0, 2253),
			AttributeID: ua.AttributeIDValue,
		})
	}

	// the send must complete; the server's tally says what went on the wire
	sendErr := sc.SendRequest(ctx, req, nil, func(ua.Response) error { return nil })
	require.NoError(t, sendErr, "the client's send completed")

	select {
	case saw := <-serverDone:
		require.Equal(t, uint32(bufsize), saw.helloReceiveBufSize, "the Hello offered the default receive buffer")
		require.Equal(t, nodesToRead, saw.nodesRead, "the request arrived complete")
		require.GreaterOrEqual(t, len(saw.chunkSizes), 2, "the request spans more than one chunk")
		for i, size := range saw.chunkSizes {
			require.LessOrEqualf(t, size, uint32(bufsize), "chunk %d weighs more than the buffer the Hello offered", i)
		}
	case <-time.After(30 * time.Second):
		require.Fail(t, "the fake server never sent what it saw")
	}
}
