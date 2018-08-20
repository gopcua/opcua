// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package datatypes

import (
	"testing"
)

var testStringBytes = [][]byte{
	{ // normal String: "foobar"
		0x06, 0x00, 0x00, 0x00, 0x66, 0x6f, 0x6f, 0x62,
		0x61, 0x72,
	},
	{ // null String
		0xff, 0xff, 0xff, 0xff,
	},
	{ // StringArray
		// ArraySize
		0x02, 0x00, 0x00, 0x00,
		// first String: "foo"
		0x03, 0x00, 0x00, 0x00, 0x66, 0x6f, 0x6f,
		// second String: "bar"
		0x03, 0x00, 0x00, 0x00, 0x62, 0x61, 0x72,
	},
}

func TestDecodeString(t *testing.T) {
	t.Run("normal", func(t *testing.T) {
		str, err := DecodeString(testStringBytes[0])
		if err != nil {
			t.Fatalf("Failed to decode String: %s", err)
		}

		switch {
		case str.Length != 6:
			t.Errorf("Length doesn't match. Want: %d, Got: %d", 6, str.Length)
		case str.Get() != "foobar":
			t.Errorf("Value doesn't match. Want: %s, Got: %s", "foobar", str.Get())
		}
		t.Log(str.Get())
	})
	t.Run("null", func(t *testing.T) {
		str, err := DecodeString(testStringBytes[1])
		if err != nil {
			t.Fatalf("Failed to decode String: %s", err)
		}

		switch {
		case str.Length != -1:
			t.Errorf("Length doesn't match. Want: %d, Got: %d", -1, str.Length)
		case str.Get() != "":
			t.Errorf("Value doesn't match. Want: %s, Got: %s", "", str.Get())
		}
		t.Log(str.Get())
	})
}

func TestDecodeStringArray(t *testing.T) {
	t.Run("normal", func(t *testing.T) {
		strs, err := DecodeStringArray(testStringBytes[2])
		if err != nil {
			t.Fatalf("Failed to decode StringArray: %s", err)
		}

		switch {
		case strs.ArraySize != 2:
			t.Errorf("ArraySize doesn't match. Want: %d, Got: %d", 2, strs.ArraySize)
		case strs.Strings[0].Length != 3:
			t.Errorf("Length doesn't match. Want: %d, Got: %d", 3, strs.Strings[0].Length)
		case strs.Strings[0].Get() != "foo":
			t.Errorf("Value doesn't match. Want: %s, Got: %s", "foo", strs.Strings[0].Get())
		case strs.Strings[1].Length != 3:
			t.Errorf("Length doesn't match. Want: %d, Got: %d", 3, strs.Strings[1].Length)
		case strs.Strings[1].Get() != "bar":
			t.Errorf("Value doesn't match. Want: %s, Got: %s", "bar", strs.Strings[1].Get())
		}
		for _, ss := range strs.Strings {
			t.Log(ss.Get())
		}
	})
}

func TestSerializeString(t *testing.T) {
	t.Run("normal", func(t *testing.T) {
		t.Parallel()
		str := NewString("foobar")

		serialized, err := str.Serialize()
		if err != nil {
			t.Fatalf("Failed to serialize String: %s", err)
		}

		for i, s := range serialized {
			x := testStringBytes[0][i]
			if s != x {
				t.Errorf("Bytes doesn't match. Want: %#x, Got: %#x at %dth", x, s, i)
			}
		}
		t.Logf("%x", serialized)
	})
	t.Run("null", func(t *testing.T) {
		t.Parallel()
		str := NewString("")

		serialized, err := str.Serialize()
		if err != nil {
			t.Fatalf("Failed to serialize String: %s", err)
		}

		for i, s := range serialized {
			x := testStringBytes[1][i]
			if s != x {
				t.Errorf("Bytes doesn't match. Want: %#x, Got: %#x at %dth", x, s, i)
			}
		}
		t.Logf("%x", serialized)
	})
}

func TestSerializeStringArray(t *testing.T) {
	t.Run("normal", func(t *testing.T) {
		t.Parallel()
		strs := NewStringArray([]string{"foo", "bar"})

		serialized, err := strs.Serialize()
		if err != nil {
			t.Fatalf("Failed to serialize StringArray: %s", err)
		}

		for i, s := range serialized {
			x := testStringBytes[2][i]
			if s != x {
				t.Errorf("Bytes doesn't match. Want: %#x, Got: %#x at %dth", x, s, i)
			}
		}
		t.Logf("%x", serialized)
	})
}
