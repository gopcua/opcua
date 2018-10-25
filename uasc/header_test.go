// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uasc

import (
	"testing"

	"github.com/wmnsk/gopcua/utils/codectest"
)

func TestHeader(t *testing.T) {
	cases := []codectest.Case{
		{
			Name: "normal",
			Struct: NewHeader(
				MessageTypeMessage,
				ChunkTypeFinal,
				0,
				[]byte{0xde, 0xad, 0xbe, 0xef},
			),
			Bytes: []byte{ // Message message
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
			Name: "no-payload",
			Struct: NewHeader(
				MessageTypeMessage,
				ChunkTypeFinal,
				0,
				nil,
			),
			Bytes: []byte{ // Message message
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
	codectest.Run(t, cases, func(b []byte) (codectest.S, error) {
		return DecodeHeader(b)
	})
}
