// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package datatypes

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/wmnsk/gopcua/id"
)

var variantTests = []struct {
	description string
	in          []byte
	expected    *Variant
	length      int
}{
	{
		description: "float",
		in:          []byte{0x0a, 0x7d, 0x05, 0x80, 0x40},
		expected:    NewVariant(NewFloat(4.00067)),
		length:      5,
	},
	{
		description: "boolean",
		in:          []byte{0x01, 0x00},
		expected:    NewVariant(NewBoolean(false)),
		length:      2,
	},
	{
		description: "localized text",
		in: []byte{
			0x15, 0x02, 0x0b, 0x00, 0x00, 0x00, 0x47, 0x72,
			0x6f, 0x73, 0x73, 0x20, 0x76, 0x61, 0x6c, 0x75,
			0x65,
		},
		expected: NewVariant(NewLocalizedText("", "Gross value")),
		length:   17,
	},
}

func TestNewVariant(t *testing.T) {
	f := NewFloat(3.1415926)
	v := NewVariant(f)
	expected := &Variant{
		EncodingMask: id.Float,
		Value:        f,
	}
	if diff := cmp.Diff(v, expected); diff != "" {
		t.Error(diff)
	}
}

func TestDecodeVariant(t *testing.T) {
	for _, test := range variantTests {
		t.Run(test.description, func(t *testing.T) {
			v, err := DecodeVariant(test.in)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(v, test.expected, opt); diff != "" {
				t.Error(diff)
			}
		})
	}
}

func TestVariantDecodeFromBytes(t *testing.T) {
	for _, test := range variantTests {
		t.Run(test.description, func(t *testing.T) {
			v := &Variant{}
			if err := v.DecodeFromBytes(test.in); err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(v, test.expected, opt); diff != "" {
				t.Error(diff)
			}
		})
	}
}

func TestVariantSerialize(t *testing.T) {
	for _, test := range variantTests {
		t.Run(test.description, func(t *testing.T) {
			b, err := test.expected.Serialize()
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(b, test.in); diff != "" {
				t.Error(diff)
			}
		})
	}
}

func TestVariantSerializeTo(t *testing.T) {
	for _, test := range variantTests {
		t.Run(test.description, func(t *testing.T) {
			b := make([]byte, test.expected.Len())
			if err := test.expected.SerializeTo(b); err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(b, test.in); diff != "" {
				t.Error(diff)
			}
		})
	}
}

func TestVariantLen(t *testing.T) {
	for _, test := range variantTests {
		t.Run(test.description, func(t *testing.T) {
			if test.expected.Len() != test.length {
				t.Errorf("Len doesn't match. Want: %d, Got: %d", test.length, test.expected.Len())
			}
		})
	}
}
