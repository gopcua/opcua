// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uacp

import (
	"testing"

	"github.com/wmnsk/gopcua/utils/codectest"
)

func TestAcknowledge(t *testing.T) {
	cases := []codectest.Case{
		{
			Struct: NewAcknowledge(
				0,     //Version
				65280, // ReceiveBufSize
				65535, // SendBufSize
				4000,  // MaxMessageSize
			),
			Bytes: []byte{
				// MessageType: ACK
				0x41, 0x43, 0x4b,
				// Chunk Type: F
				0x46,
				// MessageSize: 28
				0x1c, 0x00, 0x00, 0x00,
				// Version: 0
				0x00, 0x00, 0x00, 0x00,
				// ReceiveBufSize: 65280
				0x00, 0xff, 0x00, 0x00,
				// SendBufSize: 65535
				0xff, 0xff, 0x00, 0x00,
				// MaxMessageSize: 4000
				0xa0, 0x0f, 0x00, 0x00,
				// MaxChunkCount: 0
				0x00, 0x00, 0x00, 0x00,
			},
		},
	}
	codectest.Run(t, cases, func(b []byte) (codectest.S, error) {
		v, err := DecodeAcknowledge(b)
		if err != nil {
			return nil, err
		}
		v.Payload = nil
		return v, nil
	})
}
