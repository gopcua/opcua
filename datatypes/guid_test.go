// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package datatypes

import (
	"testing"
)

var testGIUDBytes = [][]byte{
	{ // OK
		0xbb, 0xbb, 0xaa, 0xaa, 0xdd, 0xcc, 0xff, 0xee,
		0xab, 0x89, 0x67, 0x45, 0x23, 0x01, 0x01, 0x01,
	},
	{ // Too short
		0xaa, 0xaa, 0xbb, 0xbb, 0xcc, 0xdd, 0xee, 0xff,
	},
}

func TestDecodeGUID(t *testing.T) {
	g, err := DecodeGUID(testGIUDBytes[0])
	if err != nil {
		t.Fatalf("Failed to decode GUID: %s", err)
	}

	if g.String() != "AAAABBBB-CCDD-EEFF-0101-0123456789AB" {
		t.Errorf("Error decoding GUID: Want: %s, Got: %s", "AAAABBBB-CCDD-EEFF-0101-0123456789AB", g.String())
	}
}

func TestSerializeGUID(t *testing.T) {
	g := NewGUID("AAAABBBB-CCDD-EEFF-0101-0123456789AB")

	serialized, err := g.Serialize()
	if err != nil {
		t.Fatalf("Failed to serialize GUID: %s", err)
	}

	for i, s := range serialized {
		x := testGIUDBytes[0][i]
		if s != x {
			t.Errorf("Bytes doesn't match. Want: %#x, Got: %#x at %dth", x, s, i)
		}
	}
	t.Logf("%x", serialized)
}
