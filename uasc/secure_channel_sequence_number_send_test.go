// Copyright 2018-2020 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uasc

import (
	"context"
	"encoding/binary"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/gopcua/opcua/errors"
	"github.com/gopcua/opcua/id"
	"github.com/gopcua/opcua/ua"
	"github.com/gopcua/opcua/uacp"
	"github.com/stretchr/testify/require"
)

// All instances of a channel, i.e. all security tokens, draw their
// SequenceNumbers from one counter of the channel itself. OPC 10000-6
// §6.7.2.4 requires that a SequenceNumber "shall not be reused for any
// TokenId" and "does not reset when a new TokenId is issued".
func TestInstancesShareSequenceNumberCounter(t *testing.T) {
	s := &SecureChannel{cfg: &Config{}}
	oldInstance := newChannelInstance(s)
	newInstance := newChannelInstance(s)

	// the old instance sends two chunks, then the renewed one sends one
	var numbers []uint32
	for _, inst := range []*channelInstance{oldInstance, oldInstance, newInstance} {
		inst.Lock()
		numbers = append(numbers, inst.nextSequenceNumber())
		inst.Unlock()
	}

	require.Equal(t, []uint32{1, 2, 3}, numbers,
		"chunks of two instances of one channel must carry consecutive, distinct SequenceNumbers")
}

// TestInstancesSendSequenceNumbersInAssignedOrder drives two instances of
// one channel, the tokens before and after a renewal, concurrently against
// a connection the test reads from. A message draws its SequenceNumbers when
// it is built and its chunks are written afterwards, so a sender on the old
// instance and one on the new instance must not interleave: the numbers on
// the wire have to be strictly consecutive, or the receiving side rejects a
// chunk as a gap or a reuse.
func TestInstancesSendSequenceNumbersInAssignedOrder(t *testing.T) {
	const (
		instances          = 2 // the tokens before and after a renewal
		sendersPerInstance = 3 // concurrent senders per instance
		messagesPerSender  = 5 // multi-chunk messages per sender
		maxBodySize        = 4 * 1024
	)

	serverTCP, clientTCP := newTestTCPConnPair(t)
	t.Cleanup(func() { _ = serverTCP.Close() })
	t.Cleanup(func() { _ = clientTCP.Close() })

	uacpConn, err := uacp.NewConn(serverTCP, &uacp.Acknowledge{
		ReceiveBufSize: 64 * 1024,
		SendBufSize:    64 * 1024,
		MaxMessageSize: 2 * 1024 * 1024,
		MaxChunkCount:  1024,
	})
	require.NoError(t, err)

	s := &SecureChannel{
		c: uacpConn,
		cfg: &Config{
			SecurityPolicyURI: ua.SecurityPolicyURINone,
			SecurityMode:      ua.MessageSecurityModeNone,
			RequestTimeout:    chunkedResponseTestTimeout,
		},
		instances: make(map[uint32][]*channelInstance),
	}

	var sendInstances []*channelInstance
	for i := 0; i < instances; i++ {
		instance := newChannelInstance(s)
		instance.state = channelActive
		instance.secureChannelID = 1
		instance.securityTokenID = uint32(i + 1)
		instance.maxBodySize = maxBodySize
		sendInstances = append(sendInstances, instance)
		s.instances[1] = append(s.instances[1], instance)
	}
	s.activeInstance = sendInstances[0]

	// a request body of several chunks, so every message also runs the
	// per-chunk SequenceNumber patching on the send path
	req := &ua.ReadRequest{}
	req.SetHeader(&ua.RequestHeader{})
	for i := 0; i < 1000; i++ {
		req.NodesToRead = append(req.NodesToRead, &ua.ReadValueID{
			NodeID:      ua.NewNumericNodeID(0, 2253),
			AttributeID: ua.AttributeIDValue,
		})
	}

	// probe how many chunks one message spans so the test knows when the
	// last chunk of the last message has arrived
	probe := sendInstances[0].newMessage(req, id.ReadRequest_Encoding_DefaultBinary, 1)
	probeChunks, err := probe.EncodeChunks(maxBodySize)
	require.NoError(t, err)
	require.Greater(t, len(probeChunks), 1, "the test needs multi-chunk messages")
	s.sendSequenceNumber = 0 // the probe drew from the counter; start the sends at 1

	wantChunks := instances * sendersPerInstance * messagesPerSender * len(probeChunks)

	numbers := make(chan uint32)
	readErr := make(chan error, 1)
	go func() {
		// read raw chunks off the wire and hand their SequenceNumbers to
		// the test in the order they reached the connection
		defer close(numbers)
		if err := clientTCP.SetReadDeadline(time.Now().Add(chunkedResponseTestTimeout)); err != nil {
			readErr <- err
			return
		}
		for {
			hdr := make([]byte, 8)
			if _, err := io.ReadFull(clientTCP, hdr); err != nil {
				readErr <- err
				return
			}
			size := binary.LittleEndian.Uint32(hdr[4:8])
			if size < 12 {
				readErr <- io.ErrUnexpectedEOF
				return
			}
			rest := make([]byte, int(size)-8)
			if _, err := io.ReadFull(clientTCP, rest); err != nil {
				readErr <- err
				return
			}
			// 8 bytes of the message header plus 4 each of channel id and
			// token id precede the SequenceNumber (OPC 10000-6 §6.7.2)
			select {
			case numbers <- binary.LittleEndian.Uint32(rest[8:12]):
			case <-time.After(chunkedResponseTestTimeout):
				readErr <- errors.New("timed out handing a SequenceNumber to the test")
				return
			}
		}
	}()

	var wg sync.WaitGroup
	for _, instance := range sendInstances {
		for j := 0; j < sendersPerInstance; j++ {
			wg.Add(1)
			go func(instance *channelInstance) {
				defer wg.Done()
				reqID := s.nextRequestID()
				for k := 0; k < messagesPerSender; k++ {
					_, err := s.sendAsyncWithTimeout(context.Background(), req, reqID+uint32(k), instance, nil, false, chunkedResponseTestTimeout)
					if err != nil {
						t.Errorf("sendAsyncWithTimeout: %v", err)
						return
					}
				}
			}(instance)
		}
	}
	defer wg.Wait()

	var got []uint32
	for len(got) < wantChunks {
		select {
		case n, ok := <-numbers:
			if !ok {
				t.Fatal("the reader stopped before every chunk arrived")
			}
			got = append(got, n)
		case err := <-readErr:
			t.Fatalf("reading chunks: %v", err)
		case <-time.After(chunkedResponseTestTimeout):
			t.Fatalf("timed out after %d of %d chunks", len(got), wantChunks)
		}
	}

	require.Len(t, got, wantChunks)
	for i, n := range got {
		require.EqualValues(t, i+1, n,
			"the SequenceNumber of wire chunk %d: chunks must reach the connection in the order their numbers were assigned", i)
	}
}
