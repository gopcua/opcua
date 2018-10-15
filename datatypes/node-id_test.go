// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package datatypes

import (
	"encoding/hex"
	"errors"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
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

		identStr := hex.EncodeToString(n.GetIdentifier())

		switch {
		case two.EncodingMask != TypeTwoByte:
			t.Errorf("EncodingMask doesn't match. Want: %d, Got: %d", TypeTwoByte, two.EncodingMask)
		case two.Identifier != 0xff:
			t.Errorf("Identifier doesn't match. Want: %x, Got: %x", 0xff, two.Identifier)
		case identStr != "ff":
			t.Errorf("GetIdentifier doesn't match. Want: %s, Got: %s", "ff", identStr)
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

		identStr := hex.EncodeToString(n.GetIdentifier())

		switch {
		case four.EncodingMask != TypeFourByte:
			t.Errorf("EncodingMask doesn't match. Want: %d, Got: %d", TypeFourByte, four.EncodingMask)
		case four.Namespace != 0:
			t.Errorf("Namespace doesn't match. Want: %x, Got: %x", 0, four.Namespace)
		case four.Identifier != 0xcafe:
			t.Errorf("Identifier doesn't match. Want: %x, Got: %x", 0xcafe, four.Identifier)
		case identStr != "feca":
			t.Errorf("GetIdentifier doesn't match. Want: %s, Got: %s", "feca", identStr)
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

		identStr := hex.EncodeToString(n.GetIdentifier())

		switch {
		case num.EncodingMask != TypeNumeric:
			t.Errorf("EncodingMask doesn't match. Want: %d, Got: %d", TypeNumeric, num.EncodingMask)
		case num.Namespace != 10:
			t.Errorf("Namespace doesn't match. Want: %x, Got: %x", 10, num.Namespace)
		case num.Identifier != 0xdeadbeef:
			t.Errorf("Identifier doesn't match. Want: %x, Got: %x", 0xdeadbeef, num.Identifier)
		case identStr != "efbeadde":
			t.Errorf("GetIdentifier doesn't match. Want: %s, Got: %s", "efbeadde", identStr)
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

		identStr := hex.EncodeToString(n.GetIdentifier())

		switch {
		case str.EncodingMask != TypeString:
			t.Errorf("EncodingMask doesn't match. Want: %d, Got: %d", TypeString, str.EncodingMask)
		case str.Namespace != 255:
			t.Errorf("Namespace doesn't match. Want: %x, Got: %x", 255, str.Namespace)
		case str.Length != 6:
			t.Errorf("Length doesn't match. Want: %x, Got: %x", 6, str.Length)
		case str.Value() != "foobar":
			t.Errorf("Identifier doesn't match. Want: %s, Got: %s", "foobar", str.Value())
		case identStr != "666f6f626172":
			t.Errorf("GetIdentifier doesn't match. Want: %s, Got: %s", "666f6f626172", identStr)
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

		identStr := hex.EncodeToString(n.GetIdentifier())

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
		case identStr != "bbbbaaaaddccffeeab89674523010101":
			t.Errorf("GetIdentifier doesn't match. Want: %s, Got: %s", "bbbbaaaaddccffeeab89674523010101", identStr)
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

		identStr := hex.EncodeToString(n.GetIdentifier())

		switch {
		case opq.EncodingMask != TypeOpaque:
			t.Errorf("EncodingMask doesn't match. Want: %d, Got: %d", TypeOpaque, opq.EncodingMask)
		case opq.Namespace != 32768:
			t.Errorf("Namespace doesn't match. Want: %x, Got: %x", 32768, opq.Namespace)
		case opq.Length != 4:
			t.Errorf("Length doesn't match. Want: %x, Got: %x", 4, opq.Length)
		case identStr != "deadbeef":
			t.Errorf("Identifier doesn't match. Want: %s, Got: %s", "deadbeef", identStr)
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
			t.Fatalf("Failed to serialize TwoByteNodeID: %s", err)
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
			t.Fatalf("Failed to serialize FourByteNodeID: %s", err)
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
			t.Fatalf("Failed to serialize NumericNodeID: %s", err)
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
			t.Fatalf("Failed to serialize StringNodeID: %s", err)
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
			t.Fatalf("Failed to serialize GUIDNodeID: %s", err)
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
			t.Fatalf("Failed to serialize OpaqueNodeID: %s", err)
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

func TestParseNodeID(t *testing.T) {
	cases := []struct {
		s   string
		n   NodeID
		err error
	}{
		// happy flows
		{s: "", n: NewTwoByteNodeID(0)},
		{s: "ns=0;i=1", n: NewTwoByteNodeID(1)},
		{s: "ns=1;i=2", n: NewFourByteNodeID(1, 2)},
		{s: "ns=256;i=2", n: NewNumericNodeID(256, 2)},
		{s: "ns=1;i=65536", n: NewNumericNodeID(1, 65536)},
		{s: "ns=65535;i=65536", n: NewNumericNodeID(65535, 65536)},
		{s: "ns=1;g=5eac051c-c313-43d7-b790-24aa2c3cfd37", n: NewGUIDNodeID(1, "5eac051c-c313-43d7-b790-24aa2c3cfd37")},
		{s: "ns=1;b=YWJj", n: NewOpaqueNodeID(1, []byte{'a', 'b', 'c'})},
		{s: "ns=1;s=a", n: NewStringNodeID(1, "a")},
		{s: "ns=1;a", n: NewStringNodeID(1, "a")},

		// error flows
		{s: "i=1", err: errors.New("invalid node id: i=1")},
		{s: "nsu=abc;i=1", err: errors.New("namespace urls are not supported: nsu=abc;i=1")},
		{s: "ns=65536;i=1", err: errors.New("namespace id out of range (0..65535): ns=65536;i=1")},
		{s: "ns=abc;i=1", err: errors.New("invalid namespace id: ns=abc;i=1")},
		{s: "ns=1;i=abc", err: errors.New("invalid numeric id: ns=1;i=abc")},
		{s: "ns=1;i=4294967296", err: errors.New("numeric id out of range (0..2^32-1): ns=1;i=4294967296")},
		{s: "ns=1;g=x", err: errors.New("invalid guid node id: ns=1;g=x")},
		{s: "ns=1;b=aW52YWxp%ZA==", err: errors.New("invalid opaque node id: ns=1;b=aW52YWxp%ZA==")},
	}

	for _, c := range cases {
		t.Run(c.s, func(t *testing.T) {
			n, err := ParseNodeID(c.s)
			if got, want := err, c.err; !reflect.DeepEqual(got, want) {
				t.Fatalf("got error %v want %v", got, want)
			}
			if got, want := n, c.n; !cmp.Equal(got, want) {
				t.Fatal(cmp.Diff(got, want))
			}
		})
	}
}
