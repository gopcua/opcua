// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package ua

import (
	"testing"
)

func TestLocalizedText(t *testing.T) {
	cases := []CodecTestCase{
		{
			Name:   "nothing",
			Struct: &LocalizedText{},
			Bytes:  []byte{0x00},
		},
		{
			Name:   "has-locale",
			Struct: &LocalizedText{Locale: "foo"},
			Bytes: []byte{
				0x01,
				0x03, 0x00, 0x00, 0x00, 0x66, 0x6f, 0x6f,
			},
		},
		{
			Name:   "has-text",
			Struct: &LocalizedText{Text: "bar"},
			Bytes: []byte{
				0x02,
				0x03, 0x00, 0x00, 0x00, 0x62, 0x61, 0x72,
			},
		},
		{
			Name:   "has-both",
			Struct: &LocalizedText{Locale: "foo", Text: "bar"},
			Bytes: []byte{
				0x03,
				0x03, 0x00, 0x00, 0x00, 0x66, 0x6f, 0x6f,
				// second String: "bar"
				0x03, 0x00, 0x00, 0x00, 0x62, 0x61, 0x72,
			},
		},
	}
	RunCodecTest(t, cases)
}
