// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uacp

import (
	"testing"

	"github.com/gopcua/opcua/ua"
)

func TestError(t *testing.T) {
	cases := []ua.CodecTestCase{
		{
			Struct: NewError(
				BadSecureChannelClosed, // Error
				"foobar",
			),
			Bytes: []byte{
				// Error: BadSecureChannelClosed
				0x00, 0x00, 0x86, 0x80,
				// Reason: dummy
				0x06, 0x00, 0x00, 0x00, 0x66, 0x6f, 0x6f, 0x62, 0x61, 0x72,
			},
		},
	}
	ua.RunCodecTest(t, cases)
}
