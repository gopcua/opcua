// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package datatypes

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

var dataValueTests = []struct {
	description string
	bytes       []byte
	dv          *DataValue
	length      int
}{
	{
		description: "value only",
		bytes:       []byte{0x01, 0x0a, 0x19, 0x04, 0x20, 0x40},
		dv: &DataValue{
			EncodingMask: 0x01,
			Value:        NewVariant(NewFloat(2.50025)),
		},
		length: 6,
	},
	{
		description: "value, source timestamp, server timestamp",
		bytes: []byte{
			0x0d, 0x0a, 0xc9, 0x02, 0x20, 0x40, 0x80, 0x3b,
			0xe8, 0xb3, 0x92, 0x4e, 0xd4, 0x01, 0x80, 0x3b,
			0xe8, 0xb3, 0x92, 0x4e, 0xd4, 0x01,
		},
		dv: &DataValue{
			EncodingMask:    0x0d,
			Value:           NewVariant(NewFloat(2.50017)),
			SourceTimestamp: time.Date(2018, time.September, 17, 14, 28, 29, 112000000, time.UTC),
			ServerTimestamp: time.Date(2018, time.September, 17, 14, 28, 29, 112000000, time.UTC),
		},
		length: 22,
	},
}

func TestDecodeDataValue(t *testing.T) {
	for _, test := range dataValueTests {
		t.Run(test.description, func(t *testing.T) {
			v, err := DecodeDataValue(test.bytes)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(v, test.dv); diff != "" {
				t.Error(diff)
			}
		})
	}
}

func TestDataValueDecodeFromBytes(t *testing.T) {
	for _, test := range dataValueTests {
		t.Run(test.description, func(t *testing.T) {
			d := &DataValue{}
			if err := d.DecodeFromBytes(test.bytes); err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(d, test.dv); diff != "" {
				t.Error(diff)
			}
		})
	}
}

func TestDataValueSerialize(t *testing.T) {
	for _, test := range dataValueTests {
		t.Run(test.description, func(t *testing.T) {
			b, err := test.dv.Serialize()
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(b, test.bytes); diff != "" {
				t.Error(diff)
			}
		})
	}
}

func TestDataValueSerializeTo(t *testing.T) {
	for _, test := range dataValueTests {
		t.Run(test.description, func(t *testing.T) {
			b := make([]byte, test.dv.Len())
			if err := test.dv.SerializeTo(b); err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(b, test.bytes); diff != "" {
				t.Error(diff)
			}
		})
	}
}

func TestDataValueLen(t *testing.T) {
	for _, test := range dataValueTests {
		t.Run(test.description, func(t *testing.T) {
			if test.dv.Len() != test.length {
				t.Errorf("Len doesn't match. Want: %d, Got: %d", test.length, test.dv.Len())
			}
		})
	}
}

var dataValueArrayTests = []struct {
	description string
	bytes       []byte
	dva         *DataValueArray
	length      int
}{
	{
		description: "value only and value, source timestamp, server timestamp",
		bytes: []byte{
			0x02, 0x00, 0x00, 0x00, 0x01, 0x0a, 0x19, 0x04,
			0x20, 0x40, 0x0d, 0x0a, 0xc9, 0x02, 0x20, 0x40, 0x80, 0x3b,
			0xe8, 0xb3, 0x92, 0x4e, 0xd4, 0x01, 0x80, 0x3b,
			0xe8, 0xb3, 0x92, 0x4e, 0xd4, 0x01,
		},
		dva: NewDataValueArray([]*DataValue{
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
		length: 32,
	},
}

func TestDecodeDataValueArray(t *testing.T) {
	for _, test := range dataValueArrayTests {
		t.Run(test.description, func(t *testing.T) {
			v, err := DecodeDataValueArray(test.bytes)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(v, test.dva); diff != "" {
				t.Error(diff)
			}
		})
	}
}

func TestDataValueArrayDecodeFromBytes(t *testing.T) {
	for _, test := range dataValueArrayTests {
		t.Run(test.description, func(t *testing.T) {
			d := &DataValueArray{}
			if err := d.DecodeFromBytes(test.bytes); err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(d, test.dva); diff != "" {
				t.Error(diff)
			}
		})
	}
}

func TestDataValueArraySerialize(t *testing.T) {
	for _, test := range dataValueArrayTests {
		t.Run(test.description, func(t *testing.T) {
			b, err := test.dva.Serialize()
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(b, test.bytes); diff != "" {
				t.Error(diff)
			}
		})
	}
}

func TestDataValueArraySerializeTo(t *testing.T) {
	for _, test := range dataValueArrayTests {
		t.Run(test.description, func(t *testing.T) {
			b := make([]byte, test.dva.Len())
			if err := test.dva.SerializeTo(b); err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(b, test.bytes); diff != "" {
				t.Error(diff)
			}
		})
	}
}

func TestDataValueArrayLen(t *testing.T) {
	for _, test := range dataValueArrayTests {
		t.Run(test.description, func(t *testing.T) {
			if test.dva.Len() != test.length {
				t.Errorf("Len doesn't match. Want: %d, Got: %d", test.length, test.dva.Len())
			}
		})
	}
}
