// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package datatypes

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

var booleanCases = []struct {
	description string
	structured  *Boolean
	serialized  []byte
}{
	{
		"true",
		NewBoolean(true),
		[]byte{0x01},
	},
	{
		"false",
		NewBoolean(false),
		[]byte{0x00},
	},
}

func TestDecodeBoolean(t *testing.T) {
	for _, c := range booleanCases {
		got, err := DecodeBoolean(c.serialized)
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(got, c.structured); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}

func TestSerializeBoolean(t *testing.T) {
	for _, c := range booleanCases {
		got, err := c.structured.Serialize()
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(got, c.serialized); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}

func TestBooleanLen(t *testing.T) {
	for _, c := range booleanCases {
		got := c.structured.Len()

		if diff := cmp.Diff(got, len(c.serialized)); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}
