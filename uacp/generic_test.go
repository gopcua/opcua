// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uacp

import (
	"encoding/hex"
	"testing"
)

var testGenericBytes = [][]byte{
	{ // Undefined type of message
		// MessageType: XXX
		0x58, 0x58, 0x58,
		// Chunk Type: X
		0x58,
		// MessageSize: 12
		0x0c, 0x00, 0x00, 0x00,
		// dummy Payload
		0xde, 0xad, 0xbe, 0xef,
	},
	{},
	{},
}

func TestDecodeGeneric(t *testing.T) {
	h, err := DecodeGeneric(testGenericBytes[0])
	if err != nil {
		t.Fatalf("Failed to decode Generic: %s", err)
	}

	dummyStr := hex.EncodeToString(h.Payload)
	switch {
	case h.MessageTypeValue() != "XXX":
		t.Errorf("MessageType doesn't match. Want: %s, Got: %s", "XXX", h.MessageTypeValue())
	case h.ChunkTypeValue() != "X":
		t.Errorf("ChunkType doesn't match. Want: %s, Got: %s", "X", h.ChunkTypeValue())
	case h.MessageSize != 12:
		t.Errorf("MessageSize doesn't match. Want: %d, Got: %d", 12, h.MessageSize)
	case dummyStr != "deadbeef":
		t.Errorf("Paylaod doesn't match. Want: %s, Got: %s", "deadbeef", dummyStr)
	}
}

func TestSerializeGeneric(t *testing.T) {
	h := NewGeneric(
		"XXX",
		"X",
		[]byte{0xde, 0xad, 0xbe, 0xef},
	)

	serialized, err := h.Serialize()
	if err != nil {
		t.Fatalf("Failed to serialize Generic: %s", err)
	}

	for i, s := range serialized {
		x := testGenericBytes[0][i]
		if s != x {
			t.Errorf("Bytes doesn't match. Want: %#x, Got: %#x at %dth", x, s, i)
		}
	}
	t.Logf("%x", serialized)
}
