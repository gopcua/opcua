// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package datatypes

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

var dataValueCases = []struct {
	description string
	structured  *DataValue
	serialized  []byte
}{
	{
		"value only",
		NewDataValue(
			true, false, false, false, false, false,
			NewVariant(NewFloat(2.50025)),
			0, time.Time{}, 0, time.Time{}, 0,
		),
		[]byte{
			// EncodingMask
			0x01,
			// Value(Float)
			0x0a, 0x19, 0x04, 0x20, 0x40},
	},
	{
		"value, source timestamp, server timestamp",
		NewDataValue(
			true, false, true, false, true, false,
			NewVariant(NewFloat(2.50017)),
			0, time.Date(2018, time.September, 17, 14, 28, 29, 112000000, time.UTC),
			0, time.Date(2018, time.September, 17, 14, 28, 29, 112000000, time.UTC),
			0,
		),
		[]byte{
			// EncodingMask
			0x0d,
			// Value(Float)
			0x0a, 0xc9, 0x02, 0x20, 0x40,
			// SourceTimestamp
			0x80, 0x3b, 0xe8, 0xb3, 0x92, 0x4e, 0xd4, 0x01,
			// ServerTimestamp
			0x80, 0x3b, 0xe8, 0xb3, 0x92, 0x4e, 0xd4, 0x01,
		},
	},
}

func TestDecodeDataValue(t *testing.T) {
	for _, c := range dataValueCases {
		got, err := DecodeDataValue(c.serialized)
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(got, c.structured); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}

func TestSerializeDataValue(t *testing.T) {
	for _, c := range dataValueCases {
		got, err := c.structured.Serialize()
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(got, c.serialized); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}

func TestDataValueLen(t *testing.T) {
	for _, c := range dataValueCases {
		got := c.structured.Len()

		if diff := cmp.Diff(got, len(c.serialized)); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}

var dataValueArrayCases = []struct {
	description string
	structured  *DataValueArray
	serialized  []byte
}{
	{
		"value only and value, source timestamp, server timestamp",
		NewDataValueArray([]*DataValue{
			&DataValue{
				EncodingMask: 0x01,
				Value:        NewVariant(NewFloat(2.50025)),
			},
			&DataValue{
				EncodingMask:    0x0d,
				Value:           NewVariant(NewFloat(2.50017)),
				SourceTimestamp: time.Date(2018, time.September, 17, 14, 28, 29, 112000000, time.UTC),
				ServerTimestamp: time.Date(2018, time.September, 17, 14, 28, 29, 112000000, time.UTC),
			},
		}),
		[]byte{
			// ArraySize
			0x02, 0x00, 0x00, 0x00,
			// EncodingMask
			0x01,
			// Variant(Float)
			0x0a, 0x19, 0x04, 0x20, 0x40,
			// EncodingMask
			0x0d,
			// Variant(Float)
			0x0a, 0xc9, 0x02, 0x20, 0x40,
			// SourceTimestamp
			0x80, 0x3b, 0xe8, 0xb3, 0x92, 0x4e, 0xd4, 0x01,
			// ServerTimestamp
			0x80, 0x3b, 0xe8, 0xb3, 0x92, 0x4e, 0xd4, 0x01,
		},
	},
}

func TestDecodeDataValueArray(t *testing.T) {
	for _, c := range dataValueArrayCases {
		got, err := DecodeDataValueArray(c.serialized)
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(got, c.structured); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}

func TestSerializeDataValueArray(t *testing.T) {
	for _, c := range dataValueArrayCases {
		got, err := c.structured.Serialize()
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(got, c.serialized); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}

func TestDataValueArrayLen(t *testing.T) {
	for _, c := range dataValueArrayCases {
		got := c.structured.Len()

		if diff := cmp.Diff(got, len(c.serialized)); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}
