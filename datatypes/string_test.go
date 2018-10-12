// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package datatypes

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

var stringCases = []struct {
	description string
	structured  *String
	serialized  []byte
}{
	{
		"normal String: foobar",
		NewString("foobar"),
		[]byte{
			0x06, 0x00, 0x00, 0x00, 0x66, 0x6f, 0x6f, 0x62,
			0x61, 0x72,
		},
	},
	{
		"null String",
		NewString(""),
		[]byte{
			0xff, 0xff, 0xff, 0xff,
		},
	},
}

func TestDecodeString(t *testing.T) {
	for _, c := range stringCases {
		got, err := DecodeString(c.serialized)
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(got, c.structured); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}

func TestSerializeString(t *testing.T) {
	for _, c := range stringCases {
		got, err := c.structured.Serialize()
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(got, c.serialized); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}

func TestStringLen(t *testing.T) {
	for _, c := range stringCases {
		got := c.structured.Len()

		if diff := cmp.Diff(got, len(c.serialized)); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}

var stringArrayCases = []struct {
	description string
	structured  *StringArray
	serialized  []byte
}{
	{
		"normal String: foobar",
		NewStringArray([]string{"foo", "bar"}),
		[]byte{
			// ArraySize
			0x02, 0x00, 0x00, 0x00,
			// first String: "foo"
			0x03, 0x00, 0x00, 0x00, 0x66, 0x6f, 0x6f,
			// second String: "bar"
			0x03, 0x00, 0x00, 0x00, 0x62, 0x61, 0x72,
		},
	},
	{
		"null String",
		NewStringArray(nil),
		[]byte{
			// Empty StringArray
			0x00, 0x00, 0x00, 0x00,
		},
	},
}

func TestDecodeStringArray(t *testing.T) {
	for _, c := range stringArrayCases {
		got, err := DecodeStringArray(c.serialized)
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(got, c.structured); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}

func TestSerializeStringArray(t *testing.T) {
	for _, c := range stringArrayCases {
		got, err := c.structured.Serialize()
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(got, c.serialized); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}

func TestStringArrayLen(t *testing.T) {
	for _, c := range stringArrayCases {
		got := c.structured.Len()

		if diff := cmp.Diff(got, len(c.serialized)); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}
