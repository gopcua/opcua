// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package datatypes

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

var readValueIDCases = []struct {
	description string
	structured  *ReadValueID
	serialized  []byte
}{
	{
		"normal",
		NewReadValueID(
			NewFourByteNodeID(0, 2256),
			IntegerIDValue,
			"", 0, "",
		),
		[]byte{
			0x01, 0x00, 0xd0, 0x08, 0x0d, 0x00, 0x00, 0x00,
			0xff, 0xff, 0xff, 0xff, 0x00, 0x00, 0xff, 0xff,
			0xff, 0xff,
		},
	},
}

func TestDecodeReadValueID(t *testing.T) {
	for _, c := range readValueIDCases {
		got, err := DecodeReadValueID(c.serialized)
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(got, c.structured); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}

func TestSerializeReadValueID(t *testing.T) {
	for _, c := range readValueIDCases {
		got, err := c.structured.Serialize()
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(got, c.serialized); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}

func TestReadValueIDLen(t *testing.T) {
	for _, c := range readValueIDCases {
		got := c.structured.Len()

		if diff := cmp.Diff(got, len(c.serialized)); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}

var readValueIDArrayCases = []struct {
	description string
	structured  *ReadValueIDArray
	serialized  []byte
}{
	{
		"normal",
		NewReadValueIDArray(
			[]*ReadValueID{
				{
					NodeID:       NewStringNodeID(1, "Temperature"),
					AttributeID:  IntegerIDNodeClass,
					IndexRange:   NewString(""),
					DataEncoding: NewQualifiedName(0, ""),
				},
				{
					NodeID:       NewStringNodeID(1, "Temperature"),
					AttributeID:  IntegerIDBrowseName,
					IndexRange:   NewString(""),
					DataEncoding: NewQualifiedName(0, ""),
				},
				{
					NodeID:       NewStringNodeID(1, "Temperature"),
					AttributeID:  IntegerIDDisplayName,
					IndexRange:   NewString(""),
					DataEncoding: NewQualifiedName(0, ""),
				},
			},
		),
		[]byte{
			0x03, 0x00, 0x00, 0x00, 0x03, 0x01, 0x00, 0x0b,
			0x00, 0x00, 0x00, 0x54, 0x65, 0x6d, 0x70, 0x65,
			0x72, 0x61, 0x74, 0x75, 0x72, 0x65, 0x02, 0x00,
			0x00, 0x00, 0xff, 0xff, 0xff, 0xff, 0x00, 0x00,
			0xff, 0xff, 0xff, 0xff, 0x03, 0x01, 0x00, 0x0b,
			0x00, 0x00, 0x00, 0x54, 0x65, 0x6d, 0x70, 0x65,
			0x72, 0x61, 0x74, 0x75, 0x72, 0x65, 0x03, 0x00,
			0x00, 0x00, 0xff, 0xff, 0xff, 0xff, 0x00, 0x00,
			0xff, 0xff, 0xff, 0xff, 0x03, 0x01, 0x00, 0x0b,
			0x00, 0x00, 0x00, 0x54, 0x65, 0x6d, 0x70, 0x65,
			0x72, 0x61, 0x74, 0x75, 0x72, 0x65, 0x04, 0x00,
			0x00, 0x00, 0xff, 0xff, 0xff, 0xff, 0x00, 0x00,
			0xff, 0xff, 0xff, 0xff,
		},
	},
}

func TestDecodeReadValueIDArray(t *testing.T) {
	for _, c := range readValueIDArrayCases {
		got, err := DecodeReadValueIDArray(c.serialized)
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(got, c.structured); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}

func TestSerializeReadValueIDArray(t *testing.T) {
	for _, c := range readValueIDArrayCases {
		got, err := c.structured.Serialize()
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(got, c.serialized); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}

func TestReadValueIDArrayLen(t *testing.T) {
	for _, c := range readValueIDArrayCases {
		got := c.structured.Len()

		if diff := cmp.Diff(got, len(c.serialized)); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}
