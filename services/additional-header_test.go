// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package services

import (
	"testing"

	"github.com/wmnsk/gopcua/datatypes"
)

var testAdditionalHeaderBytes = [][]byte{
	{ // No bodies.
		0x00, 0xff, 0x00,
	},
	{},
	{},
	{},
}

func TestDecodeAdditionalHeader(t *testing.T) {
	a, err := DecodeAdditionalHeader(testAdditionalHeaderBytes[0])
	if err != nil {
		t.Fatalf("Failed to decode AdditionalHeader: %s", err)
	}

	switch {
	case a.TypeID.HasNamespaceURI():
		t.Errorf("URI Flag doesn't match. Want: %v, Got: %v", false, a.TypeID.HasNamespaceURI())
	case a.TypeID.HasServerIndex():
		t.Errorf("Index Flag doesn't match. Want: %v, Got: %v", false, a.TypeID.HasServerIndex())
	case a.TypeID.EncodingMaskValue() != 0x00:
		t.Errorf("EncodingMask in TypeID doesn't match. Want: %x, Got: %x", 0x00, a.TypeID.EncodingMaskValue())
	case a.EncodingMask != 0x00:
		t.Errorf("EncodingMask doesn't match. Want: %x, Got: %x", 0x00, a.EncodingMask)
	}
}

func TestSerializeAdditionalHeader(t *testing.T) {
	a := NewAdditionalHeader(
		datatypes.NewExpandedNodeID(
			false, false,
			datatypes.NewTwoByteNodeID(255),
			"", 0,
		),
		0x00,
	)

	serialized, err := a.Serialize()
	if err != nil {
		t.Fatalf("Failed to serialize AdditionalHeader: %s", err)
	}

	for i, s := range serialized {
		x := testAdditionalHeaderBytes[0][i]
		if s != x {
			t.Errorf("Bytes doesn't match. Want: %#x, Got: %#x at %dth", x, s, i)
		}
	}
	t.Logf("%x", serialized)
}
