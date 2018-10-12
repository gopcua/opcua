// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package datatypes

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

var signedSoftwareCertificateCases = []struct {
	description string
	structured  *SignedSoftwareCertificate
	serialized  []byte
}{
	{
		"empty",
		NewSignedSoftwareCertificate(nil, nil),
		[]byte{
			// CertificateData
			0xff, 0xff, 0xff, 0xff,
			// Signature
			0xff, 0xff, 0xff, 0xff,
		},
	},
	{
		"dummy-data",
		NewSignedSoftwareCertificate([]byte{0xca, 0xfe}, []byte{0xde, 0xad, 0xbe, 0xef}),
		[]byte{
			// CertificateData
			0x02, 0x00, 0x00, 0x00, 0xca, 0xfe,
			// Signature
			0x04, 0x00, 0x00, 0x00, 0xde, 0xad, 0xbe, 0xef,
		},
	},
}

func TestDecodeSignedSoftwareCertificate(t *testing.T) {
	for _, c := range signedSoftwareCertificateCases {
		got, err := DecodeSignedSoftwareCertificate(c.serialized)
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(got, c.structured); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}

func TestSerializeSignedSoftwareCertificate(t *testing.T) {
	for _, c := range signedSoftwareCertificateCases {
		got, err := c.structured.Serialize()
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(got, c.serialized); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}

func TestSignedSoftwareCertificateLen(t *testing.T) {
	for _, c := range signedSoftwareCertificateCases {
		got := c.structured.Len()

		if diff := cmp.Diff(got, len(c.serialized)); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}

var signedSoftwareCertificateArrayCases = []struct {
	description string
	structured  *SignedSoftwareCertificateArray
	serialized  []byte
}{
	{
		"empty",
		NewSignedSoftwareCertificateArray(nil),
		[]byte{
			// ArraySize
			0x00, 0x00, 0x00, 0x00,
		},
	},
	{
		"dummy-data",
		NewSignedSoftwareCertificateArray(
			[]*SignedSoftwareCertificate{
				NewSignedSoftwareCertificate([]byte{0xca, 0xfe}, []byte{0xde, 0xad, 0xbe, 0xef}),
				NewSignedSoftwareCertificate([]byte{0xca, 0xfe}, []byte{0xde, 0xad, 0xbe, 0xef}),
			},
		),
		[]byte{
			// ArraySize
			0x02, 0x00, 0x00, 0x00,
			// CertificateData
			0x02, 0x00, 0x00, 0x00, 0xca, 0xfe,
			// Signature
			0x04, 0x00, 0x00, 0x00, 0xde, 0xad, 0xbe, 0xef,
			// CertificateData
			0x02, 0x00, 0x00, 0x00, 0xca, 0xfe,
			// Signature
			0x04, 0x00, 0x00, 0x00, 0xde, 0xad, 0xbe, 0xef,
		},
	},
}

func TestDecodeSignedSoftwareCertificateArray(t *testing.T) {
	for _, c := range signedSoftwareCertificateArrayCases {
		got, err := DecodeSignedSoftwareCertificateArray(c.serialized)
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(got, c.structured); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}

func TestSerializeSignedSoftwareCertificateArray(t *testing.T) {
	for _, c := range signedSoftwareCertificateArrayCases {
		got, err := c.structured.Serialize()
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(got, c.serialized); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}

func TestSignedSoftwareCertificateArrayLen(t *testing.T) {
	for _, c := range signedSoftwareCertificateArrayCases {
		got := c.structured.Len()

		if diff := cmp.Diff(got, len(c.serialized)); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}
