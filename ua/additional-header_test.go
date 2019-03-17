// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package ua

import (
	"testing"
)

func TestAdditionalHeader(t *testing.T) {
	cases := []CodecTestCase{
		{
			Name: "no-body",
			Struct: NewAdditionalHeader(
				NewExpandedNodeID(
					false, false,
					NewTwoByteNodeID(255),
					"", 0,
				),
				0x00,
			),
			Bytes: []byte{0x00, 0xff, 0x00},
		},
	}
	RunCodecTest(t, cases)
}
