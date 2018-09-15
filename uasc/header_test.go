// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uasc

import (
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
)

var msgHdrCases = []struct {
	description string
	structured  *Header
	serialized  []byte
}{
	{
		"normal",
		NewHeader(
			MessageTypeMessage,
			ChunkTypeFinal,
			0,
			[]byte{0xde, 0xad, 0xbe, 0xef},
		),
		[]byte{ // Message message
			// MessageType: MSG
			0x4d, 0x53, 0x47,
			// Chunk Type: Final
			0x46,
			// MessageSize: 16
			0x10, 0x00, 0x00, 0x00,
			// SecureChannelID: 0
			0x00, 0x00, 0x00, 0x00,
			// dummy Payload
			0xde, 0xad, 0xbe, 0xef,
		},
	}, {
		"no-payload",
		NewHeader(
			MessageTypeMessage,
			ChunkTypeFinal,
			0,
			nil,
		),
		[]byte{ // Message message
			// MessageType: MSG
			0x4d, 0x53, 0x47,
			// Chunk Type: Final
			0x46,
			// MessageSize: 12
			0x0c, 0x00, 0x00, 0x00,
			// SecureChannelID: 0
			0x00, 0x00, 0x00, 0x00,
		},
	},
}

func TestDecodeHeader(t *testing.T) { // option to regard []T{} and []T{nil} as equal
	// https://godoc.org/github.com/google/go-cmp/cmp#example-Option--EqualEmpty
	alwaysEqual := cmp.Comparer(func(_, _ interface{}) bool { return true })
	opt := cmp.FilterValues(func(x, y interface{}) bool {
		vx, vy := reflect.ValueOf(x), reflect.ValueOf(y)
		return (vx.IsValid() && vy.IsValid() && vx.Type() == vy.Type()) &&
			(vx.Kind() == reflect.Slice) && (vx.Len() == 0 && vy.Len() == 0)
	}, alwaysEqual)

	for _, c := range msgHdrCases {
		got, err := DecodeHeader(c.serialized)
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(got, c.structured, opt); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}

func TestSerializeHeader(t *testing.T) {
	for _, c := range msgHdrCases {
		got, err := c.structured.Serialize()
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(got, c.serialized); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}

func TestHeaderLen(t *testing.T) {
	for _, c := range msgHdrCases {
		got := c.structured.Len()

		if diff := cmp.Diff(got, len(c.serialized)); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}
