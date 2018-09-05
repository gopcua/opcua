// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uacp

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestDecodeError(t *testing.T) {
	cases := []struct {
		input []byte
		want  *Error
	}{
		{ // Normal Error
			[]byte{
				// MessageType: ERR
				0x45, 0x52, 0x52,
				// Chunk Type: F
				0x46,
				// MessageSize: 22
				0x16, 0x00, 0x00, 0x00,
				// Error: BadSecureChannelClosed
				0x00, 0x00, 0x86, 0x80,
				// Reason: dummy
				0x06, 0x00, 0x00, 0x00, 0x66, 0x6f, 0x6f, 0x62, 0x61, 0x72,
			},
			NewError(
				BadSecureChannelClosed, // Error
				"foobar",
			),
		},
	}

	for i, c := range cases {
		got, err := DecodeError(c.input)
		if err != nil {
			t.Fatal(err)
		}

		got.Payload = nil
		if diff := cmp.Diff(got, c.want); diff != "" {
			t.Errorf("case #%d failed\n%s", i, diff)
		}
	}
}

func TestSerializeError(t *testing.T) {
	cases := []struct {
		input *Error
		want  []byte
	}{
		{ // Normal Error
			NewError(
				BadSecureChannelClosed, // Error
				"foobar",
			),
			[]byte{
				// MessageType: ERR
				0x45, 0x52, 0x52,
				// Chunk Type: F
				0x46,
				// MessageSize: 22
				0x16, 0x00, 0x00, 0x00,
				// Error: BadSecureChannelClosed
				0x00, 0x00, 0x86, 0x80,
				// Reason: dummy
				0x06, 0x00, 0x00, 0x00, 0x66, 0x6f, 0x6f, 0x62, 0x61, 0x72,
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

func TestErrorLen(t *testing.T) {
	cases := []struct {
		input *Error
		want  int
	}{
		{ // Normal Error
			NewError(
				BadSecureChannelClosed, // Error
				"foobar",
			),
			22,
		},
	}

	for i, c := range cases {
		got := c.input.Len()
		if diff := cmp.Diff(got, c.want); diff != "" {
			t.Errorf("case #%d failed\n%s", i, diff)
		}
	}
}
