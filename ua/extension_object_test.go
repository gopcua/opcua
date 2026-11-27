// Copyright 2018-2020 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package ua

import (
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExtensionObject(t *testing.T) {
	cases := []CodecTestCase{
		{
			Name:   "anonymous-user-identity-token",
			Struct: NewExtensionObject(&AnonymousIdentityToken{PolicyID: "anonymous"}),
			Bytes: []byte{
				// TypeID
				0x01, 0x00, 0x41, 0x01,
				// EncodingMask
				0x01,
				// Length
				0x0d, 0x00, 0x00, 0x00,
				// AnonymousIdentityToken
				0x09, 0x00, 0x00, 0x00, 0x61, 0x6e, 0x6f, 0x6e, 0x79, 0x6d, 0x6f, 0x75, 0x73,
			},
		},
	}
	RunCodecTest(t, cases)
}

// OPC 10000-6 §5.2.2.15, Table 25: "The length shall be specified if the body is encoded" - a negative length cannot be satisfied (see ReadN).
func TestExtensionObjectDecodeRejectsNegativeInt32BodyLength(t *testing.T) {
	for _, n := range []uint32{0x80000000} {
		d := []byte{
			// TypeID
			0x00, 0x00,
			// EncodingMask: ExtensionObjectBinary
			0x01,
			// Length
			byte(n), byte(n >> 8), byte(n >> 16), byte(n >> 24),
		}
		var e ExtensionObject
		_, err := e.Decode(d)
		require.Equal(t, io.ErrUnexpectedEOF, err)
	}
}
