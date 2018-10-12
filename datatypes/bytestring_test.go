// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package datatypes

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

var byteStringCases = []struct {
	description string
	structured  *ByteString
	serialized  []byte
}{
	{
		"normal",
		NewByteString([]byte{0xde, 0xad, 0xbe, 0xef}),
		[]byte{
			0x04, 0x00, 0x00, 0x00,
			0xde, 0xad, 0xbe, 0xef,
		},
	},
}

func TestDecodeByteString(t *testing.T) {
	for _, c := range byteStringCases {
		got, err := DecodeByteString(c.serialized)
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(got, c.structured); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}

func TestSerializeByteString(t *testing.T) {
	for _, c := range byteStringCases {
		got, err := c.structured.Serialize()
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(got, c.serialized); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}

func TestByteStringLen(t *testing.T) {
	for _, c := range byteStringCases {
		got := c.structured.Len()

		if diff := cmp.Diff(got, len(c.serialized)); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}
