// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uacp

import (
	"testing"

	"github.com/wmnsk/gopcua/utils/codectest"
)

func TestError(t *testing.T) {
	cases := []codectest.Case{
		{
			Struct: NewError(
				BadSecureChannelClosed, // Error
				"foobar",
			),
			Bytes: []byte{
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
	codectest.Run(t, cases, func(b []byte) (codectest.S, error) {
		v, err := DecodeError(b)
		if err != nil {
			return nil, err
		}
		v.Payload = nil
		return v, nil
	})
}
