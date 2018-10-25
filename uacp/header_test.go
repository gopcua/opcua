// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uacp

import (
	"testing"

	"github.com/wmnsk/gopcua/utils/codectest"
)

func TestHeader(t *testing.T) {
	cases := []codectest.Case{
		{
			Struct: NewHeader(
				MessageTypeHello,
				ChunkTypeFinal,
				[]byte{0xde, 0xad, 0xbe, 0xef},
			),
			Bytes: []byte{ // Hello message
				// MessageType: HEL
				0x48, 0x45, 0x4c,
				// Chunk Type: Final
				0x46,
				// MessageSize: 12
				0x0c, 0x00, 0x00, 0x00,
				// dummy Payload
				0xde, 0xad, 0xbe, 0xef,
			},
		},
	}
	codectest.Run(t, cases, func(b []byte) (codectest.S, error) {
		return DecodeHeader(b)
	})
}
