// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package utils

import "testing"

var testBytes = [][]byte{
	{
		0xde,
	},
	{
		0xde, 0xad,
	},
	{
		0xde, 0xad, 0xbe,
	},
	{
		0xde, 0xad, 0xbe, 0xef,
	},
	{
		0xad, 0xbe, 0xef,
	},
	{
		0xbe, 0xef,
	},
	{
		0xef,
	},
}

func TestUint24To32(t *testing.T) {
	for i, b := range testBytes {
		r := Uint24To32(b)
		switch len(b) {
		case 3:
			if i == 2 && r != 0xdeadbe {
				t.Errorf("Error in conversion. Expected: %#x, Got: %#x", 0xdeadbe, r)
			}
			if i == 4 && r != 0xadbeef {
				t.Errorf("Error in conversion. Expected: %#x, Got: %#x", 0xadbeef, r)
			}
		default:
			if r != 0 {
				t.Fatalf("Uint24To32 should always be 0 when the input byte length is not 3, but got %#x", r)
			}
		}
		t.Logf("%x", r)
	}
}

func TestUint32To24(t *testing.T) {
	r := Uint32To24(uint32(0xdeadbeef))

	for i, b := range r {
		x := testBytes[4][i] // 0xad, 0xbe, 0xef,
		if b != x {
			t.Errorf("Error in conversion. Expected: %#x, Got: %#x", x, b)
		}
	}
	t.Logf("%x", r)
}
