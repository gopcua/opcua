// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package datatypes

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

var localizedTextCases = []struct {
	description string
	structured  *LocalizedText
	serialized  []byte
}{
	{
		"nothing",
		NewLocalizedText("", ""),
		[]byte{
			0x00,
		},
	},
	{
		"has locale",
		NewLocalizedText("foo", ""),
		[]byte{
			0x01,
			0x03, 0x00, 0x00, 0x00, 0x66, 0x6f, 0x6f,
		},
	},
	{
		"has text",
		NewLocalizedText("", "bar"),
		[]byte{
			0x02,
			0x03, 0x00, 0x00, 0x00, 0x62, 0x61, 0x72,
		},
	},
	{
		"has both",
		NewLocalizedText("foo", "bar"),
		[]byte{
			0x03,
			0x03, 0x00, 0x00, 0x00, 0x66, 0x6f, 0x6f,
			0x03, 0x00, 0x00, 0x00, 0x62, 0x61, 0x72,
		},
	},
}

func TestDecodeLocalizedText(t *testing.T) {
	for _, c := range localizedTextCases {
		got, err := DecodeLocalizedText(c.serialized)
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(got, c.structured); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}

func TestSerializeLocalizedText(t *testing.T) {
	for _, c := range localizedTextCases {
		got, err := c.structured.Serialize()
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(got, c.serialized); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}

func TestLocalizedTextLen(t *testing.T) {
	for _, c := range localizedTextCases {
		got := c.structured.Len()

		if diff := cmp.Diff(got, len(c.serialized)); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}
