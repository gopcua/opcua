// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uasc

import (
	"encoding/hex"
	"testing"
)

var testHeaderBytes = [][]byte{
	{ // Message message
		// MessageType: MSG
		0x4d, 0x53, 0x47,
		// Chunk Type: Final
		0x46,
		// MessageSize: 16
		0x10, 0x00, 0x00, 0x00,
		// SecureChannelID: 0
		0x00, 0x00, 0x00, 0x00,
		// dummy Payload
		0xde, 0xad, 0xbe, 0xef,
	},
	{},
	{},
}

func TestDecodeHeader(t *testing.T) {
	h, err := DecodeHeader(testHeaderBytes[0])
	if err != nil {
		t.Fatalf("Failed to decode Header: %s", err)
	}

	dummyStr := hex.EncodeToString(h.Payload)
	switch {
	case h.MessageTypeValue() != MessageTypeMessage:
		t.Errorf("MessageType doesn't match. Want: %s, Got: %s", MessageTypeMessage, h.MessageTypeValue())
	case h.ChunkTypeValue() != ChunkTypeFinal:
		t.Errorf("ChunkType doesn't match. Want: %s, Got: %s", ChunkTypeFinal, h.ChunkTypeValue())
	case h.MessageSize != 16:
		t.Errorf("MessageSize doesn't match. Want: %d, Got: %d", 16, h.MessageSize)
	case h.SecureChannelIDValue() != 0:
		t.Errorf("SecureChannelIDValue doesn't match. Want: %d, Got: %d", 0, h.MessageSize)
	case dummyStr != "deadbeef":
		t.Errorf("Paylaod doesn't match. Want: %s, Got: %s", "deadbeef", dummyStr)
	}
	t.Log(h.String())
}

func TestSerializeHeader(t *testing.T) {
	h := NewHeader(
		MessageTypeMessage,
		ChunkTypeFinal,
		0,
		[]byte{0xde, 0xad, 0xbe, 0xef},
	)

	serialized, err := h.Serialize()
	if err != nil {
		t.Fatalf("Failed to serialize Header: %s", err)
	}

	for i, s := range serialized {
		x := testHeaderBytes[0][i]
		if s != x {
			t.Errorf("Bytes doesn't match. Want: %#x, Got: %#x at %dth", x, s, i)
		}
	}
	t.Logf("%x", serialized)
}
