// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uacp

import (
	"testing"
)

var testErrorBytes = [][]byte{
	{
		// MessageType: ERR
		0x45, 0x52, 0x52,
		// Chunk Type: F
		0x46,
		// MessageSize: 22
		0x16, 0x00, 0x00, 0x00,
		// Error: BadSecureChannelClosed
		0x00, 0x00, 0x86, 0x80,
		// ReasonSize
		0x06, 0x00, 0x00, 0x00,
		// Reason: dummy
		0x66, 0x6f, 0x6f, 0x62, 0x61, 0x72,
	},
	{},
	{},
}

func TestDecodeError(t *testing.T) {
	h, err := DecodeError(testErrorBytes[0])
	if err != nil {
		t.Fatalf("Failed to decode Error: %s", err)
	}

	reason := string(h.Reason)
	switch {
	case h.MessageTypeValue() != MessageTypeError:
		t.Errorf("MessageType doesn't match. Want: %s, Got: %s", MessageTypeError, h.MessageTypeValue())
	case h.ChunkTypeValue() != ChunkTypeFinal:
		t.Errorf("ChunkType doesn't match. Want: %s, Got: %s", ChunkTypeFinal, h.ChunkTypeValue())
	case h.MessageSize != 22:
		t.Errorf("MessageSize doesn't match. Want: %d, Got: %d", 22, h.MessageSize)
	case h.Error != BadSecureChannelClosed:
		t.Errorf("Error doesn't match. Want: %d, Got: %d", BadSecureChannelClosed, h.Error)
	case h.ReasonSize != 6:
		t.Errorf("ReasonSize doesn't match. Want: %d, Got: %d", 6, h.ReasonSize)
	case reason != "foobar":
		t.Errorf("Reason doesn't match. Want: %s, Got: %s", "foobar", reason)
	}
	t.Log(h)
}

func TestSerializeError(t *testing.T) {
	h := NewError(
		BadSecureChannelClosed, // Error
		"foobar",
	)

	serialized, err := h.Serialize()
	if err != nil {
		t.Fatalf("Failed to serialize Error: %s", err)
	}

	for i, s := range serialized {
		x := testErrorBytes[0][i]
		if s != x {
			t.Errorf("Bytes doesn't match. Want: %#x, Got: %#x at %dth", x, s, i)
		}
	}
	t.Logf("%x", serialized)
}
