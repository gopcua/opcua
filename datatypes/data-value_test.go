// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package datatypes

import (
	"testing"

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
}

func TestDecodeDataValue(t *testing.T) {
	for _, test := range dataValueTests {
		t.Run(test.description, func(t *testing.T) {
			v, err := DecodeDataValue(test.bytes)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(v, test.dv, opt); diff != "" {
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
			if diff := cmp.Diff(d, test.dv, opt); diff != "" {
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
