// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uasc

import (
	"testing"

	"github.com/wmnsk/gopcua/utils/codectest"
)

func TestSequenceHeader(t *testing.T) {
	cases := []codectest.Case{
		{
			Name: "normal",
			Struct: NewSequenceHeader(
				0x11223344,
				0x44332211,
				[]byte{0xde, 0xad, 0xbe, 0xef},
			),
			Bytes: []byte{
				// SequenceNumber
				0x44, 0x33, 0x22, 0x11,
				// RequestID
				0x11, 0x22, 0x33, 0x44,
				// dummy Payload
				0xde, 0xad, 0xbe, 0xef,
			},
		},
		{
			Name: "no-payload",
			Struct: NewSequenceHeader(
				0x11223344,
				0x44332211,
				nil,
			),
			Bytes: []byte{
				// SequenceNumber
				0x44, 0x33, 0x22, 0x11,
				// RequestID
				0x11, 0x22, 0x33, 0x44,
			},
		},
	}
	codectest.Run(t, cases, func(b []byte) (codectest.S, error) {
		return DecodeSequenceHeader(b)
	})
}
