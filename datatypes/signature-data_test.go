// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package datatypes

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

var signatureDataCases = []struct {
	description string
	structured  *SignatureData
	serialized  []byte
}{
	{
		"empty",
		NewSignatureData("", nil),
		[]byte{
			// Algorithm
			0xff, 0xff, 0xff, 0xff,
			// Signature
			0xff, 0xff, 0xff, 0xff,
		},
	},
	{
		"dummy-data",
		NewSignatureData("alg", []byte{0xde, 0xad, 0xbe, 0xef}),
		[]byte{
			// Algorithm
			0x03, 0x00, 0x00, 0x00, 0x61, 0x6c, 0x67,
			// Signature
			0x04, 0x00, 0x00, 0x00, 0xde, 0xad, 0xbe, 0xef,
		},
	},
}

func TestDecodeSignatureData(t *testing.T) {
	for _, c := range signatureDataCases {
		got, err := DecodeSignatureData(c.serialized)
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(got, c.structured); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}

func TestSerializeSignatureData(t *testing.T) {
	for _, c := range signatureDataCases {
		got, err := c.structured.Serialize()
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(got, c.serialized); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}

func TestSignatureDataLen(t *testing.T) {
	for _, c := range signatureDataCases {
		got := c.structured.Len()

		if diff := cmp.Diff(got, len(c.serialized)); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}
