// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package connection

import (
	"encoding/hex"
	"testing"
)

var testHeaderBytes = [][]byte{
	{ // Hello message
		// MessageType: HEL
		0x48, 0x45, 0x4c,
		// Chunk Type: F
		0x46,
		// MessageSize: 70
		0x0c, 0x00, 0x00, 0x00,
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
	case h.MessageTypeString() != "HEL":
		t.Errorf("MessageType doesn't match. Want: %s, Got: %s", "HEL", h.MessageTypeString())
	case h.ChunkTypeString() != "F":
		t.Errorf("ChunkType doesn't match. Want: %s, Got: %s", "F", h.ChunkTypeString())
	case h.MessageSize != 12:
		t.Errorf("MessageSize doesn't match. Want: %d, Got: %d", 12, h.MessageSize)
	case dummyStr != "deadbeef":
		t.Errorf("Paylaod doesn't match. Want: %s, Got: %s", "deadbeef", dummyStr)
	}
}

func TestSerializeHeader(t *testing.T) {
	h := NewHeader(
		"HEL",
		"F",
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
