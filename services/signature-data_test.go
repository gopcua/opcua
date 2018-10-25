// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package services

import (
	"encoding/hex"
	"testing"
)

var testSignatureDataBytes = [][]byte{
	{ // Empty
		// Algorithm
		0xff, 0xff, 0xff, 0xff,
		// Signature
		0xff, 0xff, 0xff, 0xff,
	},
	{ // dummy data
		// Algorithm
		0x03, 0x00, 0x00, 0x00, 0x61, 0x6c, 0x67,
		// Signature
		0x04, 0x00, 0x00, 0x00, 0xde, 0xad, 0xbe, 0xef,
	},
	{},
	{},
}

func TestDecodeSignatureData(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		s, err := DecodeSignatureData(testSignatureDataBytes[0])
		if err != nil {
			t.Fatalf("Failed to decode SignatureData: %s", err)
		}

		switch {
		case s.Algorithm.Get() != "":
			t.Errorf("Algorithm doesn't match. Want: %s, Got: %s", "", s.Algorithm.Get())
		case s.Signature.Get() != nil:
			t.Errorf("Signature doesn't match. Want: %v, Got: %v", nil, s.Signature.Get())
		}
		t.Log(s.String())
	})
	t.Run("with-dummy", func(t *testing.T) {
		s, err := DecodeSignatureData(testSignatureDataBytes[1])
		if err != nil {
			t.Fatalf("Failed to decode SignatureData: %s", err)
		}

		dummyStr := hex.EncodeToString(s.Signature.Get())
		switch {
		case s.Algorithm.Get() != "alg":
			t.Errorf("Algorithm doesn't match. Want: %s, Got: %s", "alg", s.Algorithm.Get())
		case dummyStr != "deadbeef":
			t.Errorf("Signature doesn't match. Want: %s, Got: %s", "deadbeef", dummyStr)
		}
		t.Log(s.String())
	})
}

func TestSerializeSignatureData(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		s := NewSignatureData("", nil)

		serialized, err := s.Serialize()
		if err != nil {
			t.Fatalf("Failed to serialize SignatureData: %s", err)
		}

		for i, s := range serialized {
			x := testSignatureDataBytes[0][i]
			if s != x {
				t.Errorf("Bytes doesn't match. Want: %#x, Got: %#x at %dth", x, s, i)
			}
		}
		t.Logf("%x", serialized)
	})
	t.Run("with-dummy", func(t *testing.T) {
		s := NewSignatureData("alg", []byte{0xde, 0xad, 0xbe, 0xef})

		serialized, err := s.Serialize()
		if err != nil {
			t.Fatalf("Failed to serialize SignatureData: %s", err)
		}

		for i, s := range serialized {
			x := testSignatureDataBytes[1][i]
			if s != x {
				t.Errorf("Bytes doesn't match. Want: %#x, Got: %#x at %dth", x, s, i)
			}
		}
		t.Logf("%x", serialized)
	})
}
