// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package datatypes

import (
	"testing"

	"github.com/wmnsk/gopcua/utils/codectest"
)

func TestLocalizedText(t *testing.T) {
	cases := []codectest.Case{
		{
			Name:   "nothing",
			Struct: NewLocalizedText("", ""),
			Bytes:  []byte{0x00},
		},
		{
			Name:   "has-locale",
			Struct: NewLocalizedText("foo", ""),
			Bytes: []byte{
				0x01,
				0x03, 0x00, 0x00, 0x00, 0x66, 0x6f, 0x6f,
			},
		},
		{
			Name:   "has-text",
			Struct: NewLocalizedText("", "bar"),
			Bytes: []byte{
				0x02,
				0x03, 0x00, 0x00, 0x00, 0x62, 0x61, 0x72,
			},
		},
		{
			Name:   "has-both",
			Struct: NewLocalizedText("foo", "bar"),
			Bytes: []byte{
				0x03,
				0x03, 0x00, 0x00, 0x00, 0x66, 0x6f, 0x6f,
				// second String: "bar"
				0x03, 0x00, 0x00, 0x00, 0x62, 0x61, 0x72,
			},
		},
	}
	codectest.Run(t, cases, func(b []byte) (codectest.S, error) {
		return DecodeLocalizedText(b)
	})
}
