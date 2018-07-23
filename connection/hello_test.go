// Copyright 2018 gopc-ua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package connection

import (
	"testing"
)

var testHelloBytes = [][]byte{
	{ // Hello message
		// MessageType: HEL
		0x48, 0x45, 0x4c,
		// Chunk Type: F
		0x46,
		// MessageSize: 70
		0x46, 0x00, 0x00, 0x00,
		// Version: 0
		0x00, 0x00, 0x00, 0x00,
		// ReceiveBufSize: 65535
		0xff, 0xff, 0x00, 0x00,
		// SendBufSize: 65535
		0xff, 0xff, 0x00, 0x00,
		// MaxMessageSize: 4000
		0xa0, 0x0f, 0x00, 0x00,
		// MaxChunkCount: 0
		0x00, 0x00, 0x00, 0x00,
		// Reserved
		0x2f, 0x00, 0x00, 0x00,
		// EndPointURL
		0x6f, 0x70, 0x63, 0x2e, 0x74, 0x63, 0x70, 0x3a,
		0x2f, 0x2f, 0x77, 0x6f, 0x77, 0x2e, 0x69, 0x74,
		0x73, 0x2e, 0x65, 0x61, 0x73, 0x79, 0x3a, 0x31,
		0x31, 0x31, 0x31, 0x31, 0x2f, 0x55, 0x41, 0x2f,
		0x53, 0x65, 0x72, 0x76, 0x65, 0x72,
	},
	{},
	{},
}

func TestDecodeHello(t *testing.T) {
	h, err := DecodeHello(testHelloBytes[0])
	if err != nil {
		t.Fatalf("Failed to decode Hello: %s", err)
	}

	switch {
	case h.MessageTypeString() != "HEL":
		t.Errorf("MessageType doesn't match. Want: %s, Got: %s", "HEL", h.MessageTypeString())
	case h.ChunkTypeString() != "F":
		t.Errorf("ChunkType doesn't match. Want: %s, Got: %s", "F", h.ChunkTypeString())
	case h.MessageSize != 70:
		t.Errorf("MessageSize doesn't match. Want: %d, Got: %d", 70, h.MessageSize)
	case h.Version != 0:
		t.Errorf("Version doesn't match. Want: %d, Got: %d", 0, h.Version)
	case h.SendBufSize != 65535:
		t.Errorf("SendBufSize doesn't match. Want: %d, Got: %d", 65535, h.SendBufSize)
	case h.ReceiveBufSize != 65535:
		t.Errorf("ReceiveBufSize doesn't match. Want: %d, Got: %d", 65535, h.ReceiveBufSize)
	case h.MaxMessageSize != 4000:
		t.Errorf("MaxMessageSize doesn't match. Want: %d, Got: %d", 4000, h.MaxMessageSize)
	case h.MaxChunkCount != 0:
		t.Errorf("MaxChunkCount doesn't match. Want: %d, Got: %d", 0, h.MaxChunkCount)
	case h.EndPointURLString() != "opc.tcp://wow.its.easy:11111/UA/Server":
		t.Errorf("EndPointURLString doesn't match. Want: %s, Got: %s", "opc.tcp://wow.its.easy:11111/UA/Server", h.EndPointURLString())
	}
	t.Log(h)
}

func TestSerializeHello(t *testing.T) {
	h := NewHello(
		0,      //Version
		0xffff, // SendBufSize
		0xffff, // ReceiveBufSize
		4000,   // MaxMessageSize
		"opc.tcp://wow.its.easy:11111/UA/Server", // EndPointURL
	)

	serialized, err := h.Serialize()
	if err != nil {
		t.Fatalf("Failed to serialize Hello: %s", err)
	}

	for i, s := range serialized {
		x := testHelloBytes[0][i]
		if s != x {
			t.Errorf("Bytes doesn't match. Want: %#x, Got: %#x at %dth", x, s, i)
		}
	}
	t.Logf("%x", serialized)
}
