// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package datatypes

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestDecodeUint32Array(t *testing.T) {
	cases := []struct {
		input []byte
		want  *Uint32Array
	}{
		{ // No contents
			[]byte{
				0x00, 0x00, 0x00, 0x00,
			},
			NewUint32Array(nil),
		},
		{ // 1 value
			[]byte{
				0x01, 0x00, 0x00, 0x00,
				0x01, 0x00, 0x00, 0x00,
			},
			NewUint32Array([]uint32{1}),
		},
		{ // 4 values
			[]byte{
				0x04, 0x00, 0x00, 0x00,
				0x01, 0x00, 0x00, 0x00,
				0x02, 0x00, 0x00, 0x00,
				0x03, 0x00, 0x00, 0x00,
				0x04, 0x00, 0x00, 0x00,
			},
			NewUint32Array([]uint32{1, 2, 3, 4}),
		},
	}

	for i, c := range cases {
		got, err := DecodeUint32Array(c.input)
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(got, c.want); diff != "" {
			t.Errorf("case #%d failed\n%s", i, diff)
		}
	}
}

func TestSerializeUint32Array(t *testing.T) {
	cases := []struct {
		input *Uint32Array
		want  []byte
	}{
		{ // No contents
			NewUint32Array(nil),
			[]byte{
				0x00, 0x00, 0x00, 0x00,
			},
		},
		{ // 1 value
			NewUint32Array([]uint32{1}),
			[]byte{
				0x01, 0x00, 0x00, 0x00,
				0x01, 0x00, 0x00, 0x00,
			},
		},
		{ // 4 values
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

	for i, c := range cases {
		got, err := c.input.Serialize()
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(got, c.want); diff != "" {
			t.Errorf("case #%d failed\n%s", i, diff)
		}
	}
}

func TestUint32ArrayLen(t *testing.T) {
	cases := []struct {
		input *Uint32Array
		want  int
	}{
		{ // No contents
			NewUint32Array(nil),
			4,
		},
		{ // 1 value
			NewUint32Array([]uint32{1}),
			8,
		},
		{ // 4 values
			NewUint32Array([]uint32{1, 2, 3, 4}),
			20,
		},
	}

	for i, c := range cases {
		got := c.input.Len()
		if diff := cmp.Diff(got, c.want); diff != "" {
			t.Errorf("case #%d failed\n%s", i, diff)
		}
	}
}
