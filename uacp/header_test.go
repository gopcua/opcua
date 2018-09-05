// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uacp

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestDecodeHeader(t *testing.T) {
	cases := []struct {
		input []byte
		want  *Header
	}{
		{ // Normal Header
			[]byte{ // Hello message
				// MessageType: HEL
				0x48, 0x45, 0x4c,
				// Chunk Type: Final
				0x46,
				// MessageSize: 12
				0x0c, 0x00, 0x00, 0x00,
				// dummy Payload
				0xde, 0xad, 0xbe, 0xef,
			},
			NewHeader(
				MessageTypeHello,
				ChunkTypeFinal,
				[]byte{0xde, 0xad, 0xbe, 0xef},
			),
		},
	}

	for i, c := range cases {
		got, err := DecodeHeader(c.input)
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(got, c.want); diff != "" {
			t.Errorf("case #%d failed\n%s", i, diff)
		}
	}
}

func TestSerializeHeader(t *testing.T) {
	cases := []struct {
		input *Header
		want  []byte
	}{
		{ // Normal Header
			NewHeader(
				MessageTypeHello,
				ChunkTypeFinal,
				[]byte{0xde, 0xad, 0xbe, 0xef},
			),
			[]byte{ // Hello message
				// MessageType: HEL
				0x48, 0x45, 0x4c,
				// Chunk Type: Final
				0x46,
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

func TestHeaderLen(t *testing.T) {
	cases := []struct {
		input *Header
		want  int
	}{
		{ // Normal Header
			NewHeader(
				MessageTypeHello,
				ChunkTypeFinal,
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
