// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package datatypes

import (
	"testing"

	"github.com/wmnsk/gopcua/utils/codectest"
)

func TestVariant(t *testing.T) {
	cases := []codectest.Case{
		{
			Name:   "float",
			Struct: NewVariant(NewFloat(4.00067)),
			Bytes: []byte{
				// encoding mask
				0x0a,
				// value
				0x7d, 0x05, 0x80, 0x40,
			},
		},
		{
			Name:   "boolean",
			Struct: NewVariant(NewBoolean(false)),
			Bytes: []byte{
				// encoding mask
				0x01,
				// value
				0x00,
			},
		},
		{
			Name:   "localized text",
			Struct: NewVariant(NewLocalizedText("", "Gross value")),
			Bytes: []byte{
				// variant encoding mask
				0x15,
				// localized text encoding mask
				0x02,
				// text length
				0x0b, 0x00, 0x00, 0x00,
				0x47, 0x72, 0x6f, 0x73, 0x73, 0x20, 0x76, 0x61, 0x6c, 0x75, 0x65,
			},
		},
	}
	codectest.Run(t, cases, func(b []byte) (codectest.S, error) {
		return DecodeVariant(b)
	})
}
