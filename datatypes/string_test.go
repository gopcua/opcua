// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package datatypes

import (
	"testing"

	"github.com/wmnsk/gopcua/utils/codectest"
)

func TestString(t *testing.T) {
	cases := []codectest.Case{
		{
			Name:   "normal",
			Struct: NewString("foobar"),
			Bytes: []byte{
				0x06, 0x00, 0x00, 0x00,
				0x66, 0x6f, 0x6f, 0x62, 0x61, 0x72,
			},
		},
		{
			Name:   "null",
			Struct: NewString(""),
			Bytes: []byte{
				0xff, 0xff, 0xff, 0xff,
			},
		},
	}
	codectest.Run(t, cases, func(b []byte) (codectest.S, error) {
		return DecodeString(b)
	})
}

func TestStringArray(t *testing.T) {
	cases := []codectest.Case{
		{
			Name:   "normal",
			Struct: NewStringArray([]string{"foo", "bar"}),
			Bytes: []byte{
				// ArraySize
				0x02, 0x00, 0x00, 0x00,
				// first String: "foo"
				0x03, 0x00, 0x00, 0x00, 0x66, 0x6f, 0x6f,
				// second String: "bar"
				0x03, 0x00, 0x00, 0x00, 0x62, 0x61, 0x72,
			},
		},
		{
			Name:   "empty",
			Struct: NewStringArray([]string{}),
			Bytes: []byte{
				0x00, 0x00, 0x00, 0x00,
			},
		},
		// todo(fs): this should return a length of -1 to show the difference
		// todo(fs): between nil and empty
		// {
		// 	Name:   "null",
		// 	Struct: NewStringArray(nil),
		// 	Bytes: []byte{
		// 		0xff, 0xff, 0xff, 0xff,
		// 	},
		// },
	}
	codectest.Run(t, cases, func(b []byte) (codectest.S, error) {
		return DecodeStringArray(b)
	})
}
