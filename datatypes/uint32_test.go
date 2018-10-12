// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package datatypes

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

var uint32ArrayCases = []struct {
	description string
	structured  *Uint32Array
	serialized  []byte
}{
	{
		"No contents",
		NewUint32Array(nil),
		[]byte{
			0x00, 0x00, 0x00, 0x00,
		},
	},
	{
		"1 value",
		NewUint32Array([]uint32{1}),
		[]byte{
			0x01, 0x00, 0x00, 0x00,
			0x01, 0x00, 0x00, 0x00,
		},
	},
	{
		"4 values",
		NewUint32Array([]uint32{1, 2, 3, 4}),
		[]byte{
			0x04, 0x00, 0x00, 0x00,
			0x01, 0x00, 0x00, 0x00,
			0x02, 0x00, 0x00, 0x00,
			0x03, 0x00, 0x00, 0x00,
			0x04, 0x00, 0x00, 0x00,
		},
	},
}

func TestDecodeUint32Array(t *testing.T) {
	for _, c := range uint32ArrayCases {
		got, err := DecodeUint32Array(c.serialized)
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(got, c.structured); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}

func TestSerializeUint32Array(t *testing.T) {
	for _, c := range uint32ArrayCases {
		got, err := c.structured.Serialize()
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(got, c.serialized); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}

func TestUint32ArrayLen(t *testing.T) {
	for _, c := range uint32ArrayCases {
		got := c.structured.Len()

		if diff := cmp.Diff(got, len(c.serialized)); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}
