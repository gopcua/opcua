// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package datatypes

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

var nodeIDCases = []struct {
	description string
	structured  NodeID
	serialized  []byte
}{
	{
		"TwoByte",
		NewTwoByteNodeID(0xff),
		[]byte{
			0x00, 0xff,
		},
	},
	{
		"FourByte",
		NewFourByteNodeID(0, 0xcafe),
		[]byte{
			0x01, 0x00, 0xfe, 0xca,
		},
	},
	{
		"Numeric",
		NewNumericNodeID(10, 0xdeadbeef),
		[]byte{
			0x02, 0x0a, 0x00, 0xef, 0xbe, 0xad, 0xde,
		},
	},
	{
		"String",
		NewStringNodeID(255, "foobar"),
		[]byte{
			0x03, 0xff, 0x00, 0x06, 0x00, 0x00, 0x00, 0x66,
			0x6f, 0x6f, 0x62, 0x61, 0x72,
		},
	},
	{
		"GUID",
		NewGUIDNodeID(4660, "AAAABBBB-CCDD-EEFF-0101-0123456789AB"),
		[]byte{
			0x04, 0x34, 0x12,
			0xbb, 0xbb, 0xaa, 0xaa, 0xdd, 0xcc, 0xff, 0xee,
			0xab, 0x89, 0x67, 0x45, 0x23, 0x01, 0x01, 0x01,
		},
	},
	{
		"Opaque",
		NewOpaqueNodeID(32768, []byte{0xde, 0xad, 0xbe, 0xef}),
		[]byte{
			0x05, 0x00, 0x80, 0x04, 0x00, 0x00, 0x00, 0xde,
			0xad, 0xbe, 0xef,
		},
	},
}

func TestDecodeNodeID(t *testing.T) {
	for _, c := range nodeIDCases {
		got, err := DecodeNodeID(c.serialized)
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(got, c.structured); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}

func TestSerializeNodeID(t *testing.T) {
	for _, c := range nodeIDCases {
		got, err := c.structured.Serialize()
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(got, c.serialized); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}

func TestNodeIDLen(t *testing.T) {
	for _, c := range nodeIDCases {
		got := c.structured.Len()

		if diff := cmp.Diff(got, len(c.serialized)); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}
