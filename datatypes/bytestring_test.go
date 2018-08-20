// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package datatypes

import (
	"encoding/hex"
	"testing"
)

var testByteStringBytes = [][]byte{
	{
		0x04, 0x00, 0x00, 0x00,
		0xde, 0xad, 0xbe, 0xef,
	},
}

func TestDecodeByteString(t *testing.T) {
	b, err := DecodeByteString(testByteStringBytes[0])
	if err != nil {
		t.Fatalf("Failed to decode ByteString: %s", err)
	}

	str := hex.EncodeToString(b.Get())
	switch {
	case b.Length != 4:
		t.Errorf("Length doesn't match. Want: %d, Got: %d", 4, b.Length)
	case str != "deadbeef":
		t.Errorf("Value doesn't match. Want: %s, Got: %s", "deadbeef", str)
	}
}

func TestSerializeByteString(t *testing.T) {
	b := NewByteString([]byte{0xde, 0xad, 0xbe, 0xef})

	serialized, err := b.Serialize()
	if err != nil {
		t.Fatalf("Failed to serizlize ByteString: %s", err)
	}

	for i, s := range serialized {
		x := testByteStringBytes[0][i]
		if s != x {
			t.Errorf("Bytes doesn't match. Want: %#x, Got: %#x at %dth", x, s, i)
		}
	}
	t.Logf("%x", serialized)
}
