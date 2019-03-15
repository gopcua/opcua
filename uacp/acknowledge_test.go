// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uacp

import (
	"testing"

	"github.com/gopcua/opcua/ua"
)

func TestAcknowledge(t *testing.T) {
	cases := []ua.CodecTestCase{
		{
			Struct: NewAcknowledge(
				0,     //Version
				65280, // ReceiveBufSize
				65535, // SendBufSize
				4000,  // MaxMessageSize
			),
			Bytes: []byte{
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
	ua.RunCodecTest(t, cases)
}
