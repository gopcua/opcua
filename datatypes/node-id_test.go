// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package datatypes

import (
	"encoding/hex"
	"testing"
)

var testNodeIDBytes = [][]byte{
	{ // TwoByte
		0x00, 0xff,
	},
	{ // FourByte
		0x01, 0x00, 0xfe, 0xca,
	},
	{ // Numeric
		0x02, 0x0a, 0x00, 0xef, 0xbe, 0xad, 0xde,
	},
	{ // String
		0x03, 0xff, 0x00, 0x06, 0x00, 0x00, 0x00, 0x66,
		0x6f, 0x6f, 0x62, 0x61, 0x72,
	},
	{ // GUID
		0x04, 0x34, 0x12,
		0xbb, 0xbb, 0xaa, 0xaa, 0xdd, 0xcc, 0xff, 0xee,
		0xab, 0x89, 0x67, 0x45, 0x23, 0x01, 0x01, 0x01,
	},
	{ // Opaque
		0x05, 0x00, 0x80, 0x04, 0x00, 0x00, 0x00, 0xde,
		0xad, 0xbe, 0xef,
	},
}

func TestDecodeNodeID(t *testing.T) {
	t.Run("TwoByte", func(t *testing.T) {
		t.Parallel()
		n, err := DecodeNodeID(testNodeIDBytes[0])
		if err != nil {
			t.Fatalf("Failed to decode NodeID: %s", err)
		}

		two, ok := n.(*TwoByteNodeID)
		if !ok {
			t.Errorf("Failed to assert type. Want: %s, Got: %T", "*TwoByteNodeID", n)
		}

		switch {
		case two.EncodingMask != TypeTwoByte:
			t.Errorf("EncodingMask doesn't match. Want: %d, Got: %d", TypeTwoByte, two.EncodingMask)
		case two.Identifier != 0xff:
			t.Errorf("Identifier doesn't match. Want: %x, Got: %x", 0xff, two.Identifier)
		}
		t.Log(two.String())
	})
	t.Run("FourByte", func(t *testing.T) {
		t.Parallel()
		n, err := DecodeNodeID(testNodeIDBytes[1])
		if err != nil {
			t.Fatalf("Failed to decode NodeID: %s", err)
		}

		four, ok := n.(*FourByteNodeID)
		if !ok {
			t.Errorf("Failed to assert type. Want: %s, Got: %T", "*FourByteNodeID", n)
		}

		switch {
		case four.EncodingMask != TypeFourByte:
			t.Errorf("EncodingMask doesn't match. Want: %d, Got: %d", TypeFourByte, four.EncodingMask)
		case four.Namespace != 0:
			t.Errorf("Namespace doesn't match. Want: %x, Got: %x", 0, four.Namespace)
		case four.Identifier != 0xcafe:
			t.Errorf("Identifier doesn't match. Want: %x, Got: %x", 0xcafe, four.Identifier)
		}
		t.Log(four.String())
	})
	t.Run("Numeric", func(t *testing.T) {
		t.Parallel()
		n, err := DecodeNodeID(testNodeIDBytes[2])
		if err != nil {
			t.Fatalf("Failed to decode NodeID: %s", err)
		}

		num, ok := n.(*NumericNodeID)
		if !ok {
			t.Errorf("Failed to assert type. Want: %s, Got: %T", "*NumericNodeID", n)
		}

		switch {
		case num.EncodingMask != TypeNumeric:
			t.Errorf("EncodingMask doesn't match. Want: %d, Got: %d", TypeNumeric, num.EncodingMask)
		case num.Namespace != 10:
			t.Errorf("Namespace doesn't match. Want: %x, Got: %x", 10, num.Namespace)
		case num.Identifier != 0xdeadbeef:
			t.Errorf("Identifier doesn't match. Want: %x, Got: %x", 0xdeadbeef, num.Identifier)
		}
		t.Log(num.String())
	})
	t.Run("String", func(t *testing.T) {
		t.Parallel()
		n, err := DecodeNodeID(testNodeIDBytes[3])
		if err != nil {
			t.Fatalf("Failed to decode NodeID: %s", err)
		}

		str, ok := n.(*StringNodeID)
		if !ok {
			t.Errorf("Failed to assert type. Want: %s, Got: %T", "*StringNodeID", n)
		}

		switch {
		case str.EncodingMask != TypeString:
			t.Errorf("EncodingMask doesn't match. Want: %d, Got: %d", TypeString, str.EncodingMask)
		case str.Namespace != 255:
			t.Errorf("Namespace doesn't match. Want: %x, Got: %x", 255, str.Namespace)
		case str.Length != 6:
			t.Errorf("Length doesn't match. Want: %x, Got: %x", 6, str.Length)
		case str.Value() != "foobar":
			t.Errorf("Identifier doesn't match. Want: %s, Got: %s", "foobar", str.Value())
		}
		t.Log(str.String())
	})
	t.Run("GUID", func(t *testing.T) {
		t.Parallel()
		n, err := DecodeNodeID(testNodeIDBytes[4])
		if err != nil {
			t.Fatalf("Failed to decode NodeID: %s", err)
		}

		guid, ok := n.(*GUIDNodeID)
		if !ok {
			t.Fatalf("Failed to assert type. Want: %s, Got: %T", "*GUIDNodeID", n)
		}

		switch {
		case guid.EncodingMask != TypeGUID:
			t.Errorf("EncodingMask doesn't match. Want: %d, Got: %d", TypeGUID, guid.EncodingMask)
		case guid.Namespace != 4660:
			t.Errorf("Namespace doesn't match. Want: %x, Got: %x", 4660, guid.Namespace)
		case guid.Value() != "AAAABBBB-CCDD-EEFF-0101-0123456789AB":
			t.Errorf("Identifier doesn't match. Want: %s, Got: %s", "AAAABBBB-CCDD-EEFF-0101-0123456789AB", guid.Value())
		}
		t.Log(guid.String())
	})
	t.Run("Opaque", func(t *testing.T) {
		t.Parallel()
		n, err := DecodeNodeID(testNodeIDBytes[5])
		if err != nil {
			t.Fatalf("Failed to decode NodeID: %s", err)
		}

		opq, ok := n.(*OpaqueNodeID)
		if !ok {
			t.Errorf("Failed to assert type. Want: %s, Got: %T", "*OpaqueNodeID", n)
		}

		dummyStr := hex.EncodeToString(opq.Identifier)
		switch {
		case opq.EncodingMask != TypeOpaque:
			t.Errorf("EncodingMask doesn't match. Want: %d, Got: %d", TypeOpaque, opq.EncodingMask)
		case opq.Namespace != 32768:
			t.Errorf("Namespace doesn't match. Want: %x, Got: %x", 32768, opq.Namespace)
		case opq.Length != 4:
			t.Errorf("Length doesn't match. Want: %x, Got: %x", 4, opq.Length)
		case dummyStr != "deadbeef":
			t.Errorf("Identifier doesn't match. Want: %s, Got: %s", "deadbeef", dummyStr)
		}
		t.Log(opq.String())
	})
}
func TestSerializeNodeID(t *testing.T) {
	t.Run("TwoByte", func(t *testing.T) {
		t.Parallel()
		n := NewTwoByteNodeID(0xff)

		serialized, err := n.Serialize()
		if err != nil {
			t.Fatalf("Failed to serizlize TwoByteNodeID: %s", err)
		}

		for i, s := range serialized {
			x := testNodeIDBytes[0][i]
			if s != x {
				t.Errorf("Bytes doesn't match. Want: %#x, Got: %#x at %dth", x, s, i)
			}
		}
		t.Logf("%x", serialized)
	})
	t.Run("FourByte", func(t *testing.T) {
		t.Parallel()
		n := NewFourByteNodeID(0, 0xcafe)

		serialized, err := n.Serialize()
		if err != nil {
			t.Fatalf("Failed to serizlize FourByteNodeID: %s", err)
		}

		for i, s := range serialized {
			x := testNodeIDBytes[1][i]
			if s != x {
				t.Errorf("Bytes doesn't match. Want: %#x, Got: %#x at %dth", x, s, i)
			}
		}
		t.Logf("%x", serialized)
	})
	t.Run("Numeric", func(t *testing.T) {
		t.Parallel()
		n := NewNumericNodeID(10, 0xdeadbeef)

		serialized, err := n.Serialize()
		if err != nil {
			t.Fatalf("Failed to serizlize NumericNodeID: %s", err)
		}

		for i, s := range serialized {
			x := testNodeIDBytes[2][i]
			if s != x {
				t.Errorf("Bytes doesn't match. Want: %#x, Got: %#x at %dth", x, s, i)
			}
		}
		t.Logf("%x", serialized)
	})
	t.Run("String", func(t *testing.T) {
		t.Parallel()
		n := NewStringNodeID(255, "foobar")

		serialized, err := n.Serialize()
		if err != nil {
			t.Fatalf("Failed to serizlize StringNodeID: %s", err)
		}

		for i, s := range serialized {
			x := testNodeIDBytes[3][i]
			if s != x {
				t.Errorf("Bytes doesn't match. Want: %#x, Got: %#x at %dth", x, s, i)
			}
		}
		t.Logf("%x", serialized)
	})
	t.Run("GUID", func(t *testing.T) {
		t.Parallel()
		n := NewGUIDNodeID(4660, "AAAABBBB-CCDD-EEFF-0101-0123456789AB")

		serialized, err := n.Serialize()
		if err != nil {
			t.Fatalf("Failed to serizlize GUIDNodeID: %s", err)
		}

		for i, s := range serialized {
			x := testNodeIDBytes[4][i]
			if s != x {
				t.Errorf("Bytes doesn't match. Want: %#x, Got: %#x at %dth", x, s, i)
			}
		}
		t.Logf("%x", serialized)
	})
	t.Run("Opaque", func(t *testing.T) {
		t.Parallel()
		n := NewOpaqueNodeID(32768, []byte{0xde, 0xad, 0xbe, 0xef})

		serialized, err := n.Serialize()
		if err != nil {
			t.Fatalf("Failed to serizlize OpaqueNodeID: %s", err)
		}

		for i, s := range serialized {
			x := testNodeIDBytes[5][i]
			if s != x {
				t.Errorf("Bytes doesn't match. Want: %#x, Got: %#x at %dth", x, s, i)
			}
		}
		t.Logf("%x", serialized)
	})
}
