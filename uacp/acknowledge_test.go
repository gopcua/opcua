// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uacp

import (
	"testing"
)

var testAcknowledgeBytes = [][]byte{
	{ // Acknowledge message
		// MessageType: ACK
		0x41, 0x43, 0x4b,
		// Chunk Type: F
		0x46,
		// MessageSize: 28
		0x1c, 0x00, 0x00, 0x00,
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
	},
	{},
	{},
}

func TestDecodeAcknowledge(t *testing.T) {
	h, err := DecodeAcknowledge(testAcknowledgeBytes[0])
	if err != nil {
		t.Fatalf("Failed to decode Acknowledge: %s", err)
	}

	switch {
	case h.MessageTypeValue() != MessageTypeAcknowledge:
		t.Errorf("MessageType doesn't match. Want: %s, Got: %s", MessageTypeAcknowledge, h.MessageTypeValue())
	case h.ChunkTypeValue() != ChunkTypeFinal:
		t.Errorf("ChunkType doesn't match. Want: %s, Got: %s", ChunkTypeFinal, h.ChunkTypeValue())
	case h.MessageSize != 28:
		t.Errorf("MessageSize doesn't match. Want: %d, Got: %d", 28, h.MessageSize)
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
	}
	t.Log(h)
}

func TestSerializeAcknowledge(t *testing.T) {
	h := NewAcknowledge(
		0,      //Version
		0xffff, // SendBufSize
		0xffff, // ReceiveBufSize
		4000,   // MaxMessageSize
	)

	serialized, err := h.Serialize()
	if err != nil {
		t.Fatalf("Failed to serialize Acknowledge: %s", err)
	}

	for i, s := range serialized {
		x := testAcknowledgeBytes[0][i]
		if s != x {
			t.Errorf("Bytes doesn't match. Want: %#x, Got: %#x at %dth", x, s, i)
		}
	}
	t.Logf("%x", serialized)
}
