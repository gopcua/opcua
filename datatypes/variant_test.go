// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package datatypes

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

var variantCases = []struct {
	description string
	structured  *Variant
	serialized  []byte
}{
	{
		"float",
		NewVariant(NewFloat(4.00067)),
		[]byte{0x0a, 0x7d, 0x05, 0x80, 0x40},
	},
	{
		"boolean",
		NewVariant(NewBoolean(false)),
		[]byte{0x01, 0x00},
	},
	{
		"localized text",
		NewVariant(NewLocalizedText("", "Gross value")),
		[]byte{
			0x15, 0x02, 0x0b, 0x00, 0x00, 0x00, 0x47, 0x72,
			0x6f, 0x73, 0x73, 0x20, 0x76, 0x61, 0x6c, 0x75,
			0x65,
		},
	},
}

func TestDecodeVariant(t *testing.T) {
	for _, c := range variantCases {
		got, err := DecodeVariant(c.serialized)
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(got, c.structured); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}

func TestSerializeVariant(t *testing.T) {
	for _, c := range variantCases {
		got, err := c.structured.Serialize()
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(got, c.serialized); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}

func TestVariantLen(t *testing.T) {
	for _, c := range variantCases {
		got := c.structured.Len()

		if diff := cmp.Diff(got, len(c.serialized)); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}
