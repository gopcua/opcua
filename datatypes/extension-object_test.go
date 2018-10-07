// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package datatypes

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

var extensionObjectCases = []struct {
	description string
	structured  *ExtensionObject
	serialized  []byte
}{
	{
		"anonymous-user-identity-token",
		NewExtensionObject(
			0x01, NewAnonymousIdentityToken("anonymous"),
		),
		[]byte{
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

func TestDecodeExtensionObject(t *testing.T) {
	for _, c := range extensionObjectCases {
		got, err := DecodeExtensionObject(c.serialized)
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(got, c.structured); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}

func TestSerializeExtensionObject(t *testing.T) {
	for _, c := range extensionObjectCases {
		got, err := c.structured.Serialize()
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(got, c.serialized); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}

func TestExtensionObjectLen(t *testing.T) {
	for _, c := range extensionObjectCases {
		got := c.structured.Len()

		if diff := cmp.Diff(got, len(c.serialized)); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}
