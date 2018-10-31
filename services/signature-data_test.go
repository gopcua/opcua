// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package services

import (
	"testing"

	"github.com/wmnsk/gopcua/utils/codectest"
)

func TestSignatureData(t *testing.T) {
	cases := []codectest.Case{
		{
			Name:   "empty",
			Struct: NewSignatureData("", nil),
			Bytes: []byte{
				// Algorithm
				0xff, 0xff, 0xff, 0xff,
				// Signature
				0xff, 0xff, 0xff, 0xff,
			},
		},
		{
			Name:   "dummy data",
			Struct: NewSignatureData("alg", []byte{0xde, 0xad, 0xbe, 0xef}),
			Bytes: []byte{
				// Algorithm
				0x03, 0x00, 0x00, 0x00, 0x61, 0x6c, 0x67,
				// Signature
				0x04, 0x00, 0x00, 0x00, 0xde, 0xad, 0xbe, 0xef,
			},
		},
	}
	codectest.Run(t, cases, func(b []byte) (codectest.S, error) {
		return DecodeSignatureData(b)
	})
}
