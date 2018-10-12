// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package datatypes

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

var expandedNodeIDCases = []struct {
	description string
	structured  *ExpandedNodeID
	serialized  []byte
}{
	{
		"Without optional fields",
		NewExpandedNodeID(
			false, false,
			NewTwoByteNodeID(0xff),
			"", 0,
		),
		[]byte{
			0x00, 0xff,
		},
	},
	{
		"With NamespaceURI",
		NewExpandedNodeID(
			true, false,
			NewTwoByteNodeID(0xff),
			"foobar", 0,
		),
		[]byte{
			0x80, 0xff, 0x06, 0x00, 0x00, 0x00, 0x66, 0x6f,
			0x6f, 0x62, 0x61, 0x72,
		},
	},
	{
		"With ServerIndex",
		NewExpandedNodeID(
			false, true,
			NewTwoByteNodeID(0xff),
			"", 32768,
		),
		[]byte{
			0x40, 0xff, 0x00, 0x80, 0x00, 0x00,
		},
	},
	{
		"With NamespaceURI and ServerIndex",
		NewExpandedNodeID(
			true, true,
			NewTwoByteNodeID(0xff),
			"foobar", 32768,
		),
		[]byte{
			0xc0, 0xff, 0x06, 0x00, 0x00, 0x00, 0x66, 0x6f,
			0x6f, 0x62, 0x61, 0x72, 0x00, 0x80, 0x00, 0x00,
		},
	},
}

func TestDecodeExpandedNodeID(t *testing.T) {
	for _, c := range expandedNodeIDCases {
		got, err := DecodeExpandedNodeID(c.serialized)
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(got, c.structured); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}

func TestSerializeExpandedNodeID(t *testing.T) {
	for _, c := range expandedNodeIDCases {
		got, err := c.structured.Serialize()
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(got, c.serialized); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}

func TestExpandedNodeIDLen(t *testing.T) {
	for _, c := range expandedNodeIDCases {
		got := c.structured.Len()

		if diff := cmp.Diff(got, len(c.serialized)); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}
