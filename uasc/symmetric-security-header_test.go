// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uasc

import (
	"encoding/hex"
	"testing"
)

var testSymmetricSecurityHeaderBytes = [][]byte{
	{
		// TokenID
		0x44, 0x33, 0x22, 0x11,
		// dummy Payload
		0xde, 0xad, 0xbe, 0xef,
	},
	{},
	{},
}

func TestDecodeSymmetricSecurityHeader(t *testing.T) {
	s, err := DecodeSymmetricSecurityHeader(testSymmetricSecurityHeaderBytes[0])
	if err != nil {
		t.Fatalf("Failed to decode SymmetricSecurityHeader: %s", err)
	}

	dummyStr := hex.EncodeToString(s.Payload)
	switch {
	case s.TokenID != 0x11223344:
		t.Errorf("TokenID doesn't match. Want: %x, Got: %x", 0x11223344, s.TokenID)
	case dummyStr != "deadbeef":
		t.Errorf("Payload doesn't match. Want: %s, Got: %s", "deadbeef", dummyStr)
	}
	t.Log(s.String())
}

func TestSerializeSymmetricSecurityHeader(t *testing.T) {
	a := NewSymmetricSecurityHeader(
		0x11223344,
		[]byte{0xde, 0xad, 0xbe, 0xef},
	)

	serialized, err := a.Serialize()
	if err != nil {
		t.Fatalf("Failed to serialize Header: %s", err)
	}

	for i, s := range serialized {
		x := testSymmetricSecurityHeaderBytes[0][i]
		if s != x {
			t.Errorf("Bytes doesn't match. Want: %#x, Got: %#x at %dth", x, s, i)
		}
	}
	t.Logf("%x", serialized)
}
