// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uacp

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestDecodeGeneric(t *testing.T) {
	cases := []struct {
		input []byte
		want  *Generic
	}{
		{ // Normal Generic (undefined type)
			[]byte{
				// MessageType: XXX
				0x58, 0x58, 0x58,
				// Chunk Type: X
				0x58,
				// MessageSize: 12
				0x0c, 0x00, 0x00, 0x00,
				// dummy Payload
				0xde, 0xad, 0xbe, 0xef,
			},
			NewGeneric(
				"XXX",
				"X",
				[]byte{0xde, 0xad, 0xbe, 0xef},
			),
		},
	}

	for i, c := range cases {
		got, err := DecodeGeneric(c.input)
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(got, c.want); diff != "" {
			t.Errorf("case #%d failed\n%s", i, diff)
		}
	}
}

func TestSerializeGeneric(t *testing.T) {
	cases := []struct {
		input *Generic
		want  []byte
	}{
		{ // Normal Generic (undefined type)
			NewGeneric(
				"XXX",
				"X",
				[]byte{0xde, 0xad, 0xbe, 0xef},
			),
			[]byte{
				// MessageType: XXX
				0x58, 0x58, 0x58,
				// Chunk Type: X
				0x58,
				// MessageSize: 12
				0x0c, 0x00, 0x00, 0x00,
				// dummy Payload
				0xde, 0xad, 0xbe, 0xef,
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

func TestGenericLen(t *testing.T) {
	cases := []struct {
		input *Generic
		want  int
	}{
		{ // Normal Generic (undefined type)
			NewGeneric(
				"XXX",
				"X",
				[]byte{0xde, 0xad, 0xbe, 0xef},
			),
			12,
		},
	}

	for i, c := range cases {
		got := c.input.Len()
		if diff := cmp.Diff(got, c.want); diff != "" {
			t.Errorf("case #%d failed\n%s", i, diff)
		}
	}
}
