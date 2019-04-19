// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uasc

import (
	"testing"
)

func TestSymmetricSecurityHeader(t *testing.T) {
	cases := []CodecTestCase{
		{
			Name: "normal",
			Struct: NewSymmetricSecurityHeader(
				0x11223344,
			),
			Bytes: []byte{
				// TokenID
				0x44, 0x33, 0x22, 0x11,
			},
		}, {
			Name: "no-payload",
			Struct: NewSymmetricSecurityHeader(
				0x11223344,
			),
			Bytes: []byte{
				// TokenID
				0x44, 0x33, 0x22, 0x11,
			},
		},
	}
	RunCodecTest(t, cases)
}
