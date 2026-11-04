// Copyright 2018-2020 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package ua

import (
	"slices"
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

func TestExtensionObjectUnknownType(t *testing.T) {
	// ns=2;i=1234 is not registered, so the body cannot be decoded.
	typeID := NewFourByteExpandedNodeID(2, 1234)
	body := []byte{0x2a, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	bytes := []byte{
		// TypeID
		0x01, 0x02, 0xd2, 0x04,
		// EncodingMask
		0x01,
		// Length
		0x08, 0x00, 0x00, 0x00,
		// Body
		0x2a, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	}

	RunCodecTest(t, []CodecTestCase{
		{
			Name: "unknown-type-keeps-body",
			Struct: &ExtensionObject{
				EncodingMask: ExtensionObjectBinary,
				TypeID:       typeID,
				body:         body,
			},
			Bytes: bytes,
		},
	})

	t.Run("body does not alias the input", func(t *testing.T) {
		b := slices.Clone(bytes)
		var e ExtensionObject
		_, err := e.Decode(b)
		require.NoError(t, err)
		require.Nil(t, e.Value)
		require.Equal(t, body, e.Body())

		b[9] = 0xff
		require.Equal(t, body, e.Body())
	})

	t.Run("decoded value has no body", func(t *testing.T) {
		e := &ExtensionObject{body: body}
		b, err := NewExtensionObject(&AnonymousIdentityToken{PolicyID: "anonymous"}).Encode()
		require.NoError(t, err)
		_, err = e.Decode(b)
		require.NoError(t, err)
		require.IsType(t, &AnonymousIdentityToken{}, e.Value)
		require.Nil(t, e.Body())
	})

	t.Run("empty object has no body", func(t *testing.T) {
		var e ExtensionObject
		_, err := e.Decode([]byte{0x00, 0x00, 0x00})
		require.NoError(t, err)
		require.Nil(t, e.Value)
		require.Nil(t, e.Body())
	})
}
