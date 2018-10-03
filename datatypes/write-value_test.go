// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package datatypes

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

var writeValueCases = []struct {
	description string
	structured  *WriteValue
	serialized  []byte
}{
	{
		"normal",
		NewWriteValue(
			NewFourByteNodeID(0, 2256),
			IntegerIDValue,
			"",
			NewDataValue(
				true, false, true, false, true, false,
				NewVariant(NewFloat(2.50017)),
				0,
				time.Date(2018, time.September, 17, 14, 28, 29, 112000000, time.UTC),
				0,
				time.Date(2018, time.September, 17, 14, 28, 29, 112000000, time.UTC),
				0,
			),
		),
		[]byte{
			// NodeID
			0x01, 0x00, 0xd0, 0x08,
			// AttributeID
			0x0d, 0x00, 0x00, 0x00,
			// IndexRange
			0xff, 0xff, 0xff, 0xff,
			// Value
			0x0d, 0x0a, 0xc9, 0x02, 0x20, 0x40, 0x80, 0x3b,
			0xe8, 0xb3, 0x92, 0x4e, 0xd4, 0x01, 0x80, 0x3b,
			0xe8, 0xb3, 0x92, 0x4e, 0xd4, 0x01,
		},
	},
}

func TestDecodeWriteValue(t *testing.T) {
	for _, c := range writeValueCases {
		got, err := DecodeWriteValue(c.serialized)
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(got, c.structured); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}

func TestSerializeWriteValue(t *testing.T) {
	for _, c := range writeValueCases {
		got, err := c.structured.Serialize()
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(got, c.serialized); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}

func TestWriteValueLen(t *testing.T) {
	for _, c := range writeValueCases {
		got := c.structured.Len()

		if diff := cmp.Diff(got, len(c.serialized)); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}

var writeValueArrayCases = []struct {
	description string
	structured  *WriteValueArray
	serialized  []byte
}{
	{
		"normal",
		NewWriteValueArray(
			[]*WriteValue{
				NewWriteValue(
					NewFourByteNodeID(0, 2256),
					IntegerIDValue,
					"",
					NewDataValue(
						true, false, true, false, true, false,
						NewVariant(NewFloat(2.50017)),
						0,
						time.Date(2018, time.September, 17, 14, 28, 29, 112000000, time.UTC),
						0,
						time.Date(2018, time.September, 17, 14, 28, 29, 112000000, time.UTC),
						0,
					),
				),
				NewWriteValue(
					NewFourByteNodeID(0, 2256),
					IntegerIDValue,
					"",
					NewDataValue(
						true, false, true, false, true, false,
						NewVariant(NewFloat(2.50017)),
						0,
						time.Date(2018, time.September, 17, 14, 28, 29, 112000000, time.UTC),
						0,
						time.Date(2018, time.September, 17, 14, 28, 29, 112000000, time.UTC),
						0,
					),
				),
			},
		),
		[]byte{
			// ArraySize
			0x02, 0x00, 0x00, 0x00,
			// NodeID
			0x01, 0x00, 0xd0, 0x08,
			// AttributeID
			0x0d, 0x00, 0x00, 0x00,
			// IndexRange
			0xff, 0xff, 0xff, 0xff,
			// Value
			0x0d, 0x0a, 0xc9, 0x02, 0x20, 0x40, 0x80, 0x3b,
			0xe8, 0xb3, 0x92, 0x4e, 0xd4, 0x01, 0x80, 0x3b,
			0xe8, 0xb3, 0x92, 0x4e, 0xd4, 0x01,
			// NodeID
			0x01, 0x00, 0xd0, 0x08,
			// AttributeID
			0x0d, 0x00, 0x00, 0x00,
			// IndexRange
			0xff, 0xff, 0xff, 0xff,
			// Value
			0x0d, 0x0a, 0xc9, 0x02, 0x20, 0x40, 0x80, 0x3b,
			0xe8, 0xb3, 0x92, 0x4e, 0xd4, 0x01, 0x80, 0x3b,
			0xe8, 0xb3, 0x92, 0x4e, 0xd4, 0x01,
		},
	},
}

func TestDecodeWriteValueArray(t *testing.T) {
	for _, c := range writeValueArrayCases {
		got, err := DecodeWriteValueArray(c.serialized)
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(got, c.structured); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}

func TestSerializeWriteValueArray(t *testing.T) {
	for _, c := range writeValueArrayCases {
		got, err := c.structured.Serialize()
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(got, c.serialized); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}

func TestWriteValueArrayLen(t *testing.T) {
	for _, c := range writeValueArrayCases {
		got := c.structured.Len()

		if diff := cmp.Diff(got, len(c.serialized)); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}
