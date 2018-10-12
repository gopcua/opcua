// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package datatypes

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

var guidCases = []struct {
	description string
	structured  *GUID
	serialized  []byte
}{
	{
		"ok",
		NewGUID("AAAABBBB-CCDD-EEFF-0101-0123456789AB"),
		[]byte{
			0xbb, 0xbb, 0xaa, 0xaa, 0xdd, 0xcc, 0xff, 0xee,
			0xab, 0x89, 0x67, 0x45, 0x23, 0x01, 0x01, 0x01,
		},
	},
}

func TestDecodeGUID(t *testing.T) {
	for _, c := range guidCases {
		got, err := DecodeGUID(c.serialized)
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(got, c.structured); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}

func TestSerializeGUID(t *testing.T) {
	for _, c := range guidCases {
		got, err := c.structured.Serialize()
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(got, c.serialized); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}

func TestGUIDLen(t *testing.T) {
	for _, c := range guidCases {
		got := c.structured.Len()

		if diff := cmp.Diff(got, len(c.serialized)); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}
