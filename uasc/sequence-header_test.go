// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uasc

import (
	"encoding/hex"
	"testing"
)

var testSequenceHeaderBytes = [][]byte{
	{
		// SequenceNumber
		0x44, 0x33, 0x22, 0x11,
		// RequestID
		0x11, 0x22, 0x33, 0x44,
		// dummy Payload
		0xde, 0xad, 0xbe, 0xef,
	},
	{},
	{},
}

func TestDecodeSequenceHeader(t *testing.T) {
	s, err := DecodeSequenceHeader(testSequenceHeaderBytes[0])
	if err != nil {
		t.Fatalf("Failed to decode SequenceHeader: %s", err)
	}

	dummyStr := hex.EncodeToString(s.Payload)
	switch {
	case s.SequenceNumber != 0x11223344:
		t.Errorf("SequenceNumber doesn't match. Want: %x, Got: %x", 0x11223344, s.SequenceNumber)
	case s.RequestID != 0x44332211:
		t.Errorf("RequestID doesn't match. Want: %x, Got: %x", 0x44332211, s.RequestID)
	case dummyStr != "deadbeef":
		t.Errorf("Payload doesn't match. Want: %s, Got: %s", "deadbeef", dummyStr)
	}
	t.Log(s.String())
}

func TestSerializeSequenceHeader(t *testing.T) {
	a := NewSequenceHeader(
		0x11223344,
		0x44332211,
		[]byte{0xde, 0xad, 0xbe, 0xef},
	)

	serialized, err := a.Serialize()
	if err != nil {
		t.Fatalf("Failed to serialize Header: %s", err)
	}

	for i, s := range serialized {
		x := testSequenceHeaderBytes[0][i]
		if s != x {
			t.Errorf("Bytes doesn't match. Want: %#x, Got: %#x at %dth", x, s, i)
		}
	}
	t.Logf("%x", serialized)
}
