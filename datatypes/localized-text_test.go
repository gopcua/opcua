// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package datatypes

import (
	"testing"
)

var testLocalizedTextBytes = [][]byte{
	{ // nothing
		0x00,
	},
	{ // has locale
		0x01,
		0x03, 0x00, 0x00, 0x00, 0x66, 0x6f, 0x6f,
	},
	{ // has text
		0x02,
		0x03, 0x00, 0x00, 0x00, 0x62, 0x61, 0x72,
	},
	{ // has both
		0x03,
		0x03, 0x00, 0x00, 0x00, 0x66, 0x6f, 0x6f,
		// second String: "bar"
		0x03, 0x00, 0x00, 0x00, 0x62, 0x61, 0x72,
	},
}

func TestDecodeLocalizedText(t *testing.T) {
	t.Run("nothing", func(t *testing.T) {
		t.Parallel()
		l, err := DecodeLocalizedText(testLocalizedTextBytes[0])
		if err != nil {
			t.Fatalf("Failed to decode LocalizedText: %s", err)
		}

		switch {
		case l.EncodingMask != 0x00:
			t.Errorf("EncodingMask doesn't match. Want: %x, Got: %x", 0x00, l.EncodingMask)
		case l.Locale != nil:
			t.Errorf("Locale doesn't match. Want: %v, Got: %v", nil, l.Locale)
		case l.Text != nil:
			t.Errorf("Text doesn't match. Want: %v, Got: %v", nil, l.Text)
		}
	})
	t.Run("has-locale", func(t *testing.T) {
		t.Parallel()
		l, err := DecodeLocalizedText(testLocalizedTextBytes[1])
		if err != nil {
			t.Fatalf("Failed to decode LocalizedText: %s", err)
		}

		switch {
		case l.EncodingMask != 0x01:
			t.Errorf("EncodingMask doesn't match. Want: %x, Got: %x", 0x01, l.EncodingMask)
		case l.Locale.Get() != "foo":
			t.Errorf("Locale doesn't match. Want: %x, Got: %x", "foo", l.Locale.Get())
		case l.Text != nil:
			t.Errorf("Text doesn't match. Want: %v, Got: %v", nil, l.Text)
		}
	})
	t.Run("has-text", func(t *testing.T) {
		t.Parallel()
		l, err := DecodeLocalizedText(testLocalizedTextBytes[2])
		if err != nil {
			t.Fatalf("Failed to decode LocalizedText: %s", err)
		}

		switch {
		case l.EncodingMask != 0x02:
			t.Errorf("EncodingMask doesn't match. Want: %x, Got: %x", 0x02, l.EncodingMask)
		case l.Locale != nil:
			t.Errorf("Locale doesn't match. Want: %v, Got: %v", nil, l.Locale)
		case l.Text.Get() != "bar":
			t.Errorf("Text doesn't match. Want: %x, Got: %x", "bar", l.Text.Get())
		}
	})
	t.Run("has-both", func(t *testing.T) {
		t.Parallel()
		l, err := DecodeLocalizedText(testLocalizedTextBytes[3])
		if err != nil {
			t.Fatalf("Failed to decode LocalizedText: %s", err)
		}

		switch {
		case l.EncodingMask != 0x03:
			t.Errorf("EncodingMask doesn't match. Want: %x, Got: %x", 0x03, l.EncodingMask)
		case l.Locale.Get() != "foo":
			t.Errorf("Locale doesn't match. Want: %x, Got: %x", "foo", l.Locale.Get())
		case l.Text.Get() != "bar":
			t.Errorf("Text doesn't match. Want: %x, Got: %x", "bar", l.Text.Get())
		}
	})
}

func TestSerializeLocalizedText(t *testing.T) {
	t.Run("nothing", func(t *testing.T) {
		t.Parallel()
		l := NewLocalizedText("", "")

		serialized, err := l.Serialize()
		if err != nil {
			t.Fatalf("Failed to serialize LocalizedText: %s", err)
		}

		for i, s := range serialized {
			x := testLocalizedTextBytes[0][i]
			if s != x {
				t.Errorf("Bytes doesn't match. Want: %#x, Got: %#x at %dth", x, s, i)
			}
		}
	})
	t.Run("has-locale", func(t *testing.T) {
		t.Parallel()
		l := NewLocalizedText("foo", "")

		serialized, err := l.Serialize()
		if err != nil {
			t.Fatalf("Failed to serialize LocalizedText: %s", err)
		}

		for i, s := range serialized {
			x := testLocalizedTextBytes[1][i]
			if s != x {
				t.Errorf("Bytes doesn't match. Want: %#x, Got: %#x at %dth", x, s, i)
			}
		}
	})
	t.Run("has-text", func(t *testing.T) {
		t.Parallel()
		l := NewLocalizedText("", "bar")

		serialized, err := l.Serialize()
		if err != nil {
			t.Fatalf("Failed to serialize LocalizedText: %s", err)
		}

		for i, s := range serialized {
			x := testLocalizedTextBytes[2][i]
			if s != x {
				t.Errorf("Bytes doesn't match. Want: %#x, Got: %#x at %dth", x, s, i)
			}
		}
	})
	t.Run("has-both", func(t *testing.T) {
		t.Parallel()
		l := NewLocalizedText("foo", "bar")

		serialized, err := l.Serialize()
		if err != nil {
			t.Fatalf("Failed to serialize LocalizedText: %s", err)
		}

		for i, s := range serialized {
			x := testLocalizedTextBytes[3][i]
			if s != x {
				t.Errorf("Bytes doesn't match. Want: %#x, Got: %#x at %dth", x, s, i)
			}
		}
	})
}
