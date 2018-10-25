// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uasc

import (
	"testing"

	"github.com/wmnsk/gopcua/utils/codectest"
)

func TestSymmetricSecurityHeader(t *testing.T) {
	cases := []codectest.Case{
		{
			Name: "normal",
			Struct: NewSymmetricSecurityHeader(
				0x11223344,
				[]byte{0xde, 0xad, 0xbe, 0xef},
			),
			Bytes: []byte{
				// TokenID
				0x44, 0x33, 0x22, 0x11,
				// dummy Payload
				0xde, 0xad, 0xbe, 0xef,
			},
		}, {
			Name: "no-payload",
			Struct: NewSymmetricSecurityHeader(
				0x11223344,
				nil,
			),
			Bytes: []byte{
				// TokenID
				0x44, 0x33, 0x22, 0x11,
			},
		},
	}
	codectest.Run(t, cases, func(b []byte) (codectest.S, error) {
		return DecodeSymmetricSecurityHeader(b)
	})
}
