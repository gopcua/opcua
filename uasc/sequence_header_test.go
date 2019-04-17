// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uasc

import (
	"testing"
)

func TestSequenceHeader(t *testing.T) {
	cases := []CodecTestCase{
		{
			Name: "normal",
			Struct: NewSequenceHeader(
				0x11223344,
				0x44332211,
			),
			Bytes: []byte{
				// SequenceNumber
				0x44, 0x33, 0x22, 0x11,
				// RequestID
				0x11, 0x22, 0x33, 0x44,
			},
		},
		{
			Name: "no-payload",
			Struct: NewSequenceHeader(
				0x11223344,
				0x44332211,
			),
			Bytes: []byte{
				// SequenceNumber
				0x44, 0x33, 0x22, 0x11,
				// RequestID
				0x11, 0x22, 0x33, 0x44,
			},
		},
	}
	RunCodecTest(t, cases)
}
