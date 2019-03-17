// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package ua

import (
	"testing"
)

func TestExpandedNodeID(t *testing.T) {
	cases := []CodecTestCase{
		{
			Name: "Without optional fields",
			Struct: NewExpandedNodeID(
				false, false,
				NewTwoByteNodeID(0xff),
				"", 0,
			),
			Bytes: []byte{
				0x00, 0xff,
			},
		},
		{
			Name: "With NamespaceURI",
			Struct: NewExpandedNodeID(
				true, false,
				NewTwoByteNodeID(0xff),
				"foobar", 0,
			),
			Bytes: []byte{
				0x80, 0xff, 0x06, 0x00, 0x00, 0x00, 0x66, 0x6f,
				0x6f, 0x62, 0x61, 0x72,
			},
		},
		{
			Name: "With ServerIndex",
			Struct: NewExpandedNodeID(
				false, true,
				NewTwoByteNodeID(0xff),
				"", 32768,
			),
			Bytes: []byte{ // With ServerIndex
				0x40, 0xff, 0x00, 0x80, 0x00, 0x00,
			},
		},
		{
			Name: "With NamespaceURI and ServerIndex",
			Struct: NewExpandedNodeID(
				true, true,
				NewTwoByteNodeID(0xff),
				"foobar", 32768,
			),
			Bytes: []byte{
				0xc0, 0xff, 0x06, 0x00, 0x00, 0x00, 0x66, 0x6f,
				0x6f, 0x62, 0x61, 0x72, 0x00, 0x80, 0x00, 0x00,
			},
		},
	}
	RunCodecTest(t, cases)
}
