// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package ua

import (
	"testing"
)

func TestSignedSoftwareCertificate(t *testing.T) {
	cases := []CodecTestCase{
		{
			Name:   "empty",
			Struct: &SignedSoftwareCertificate{},
			Bytes: []byte{
				// CertificateData
				0xff, 0xff, 0xff, 0xff,
				// Signature
				0xff, 0xff, 0xff, 0xff,
			},
		},
		{
			Name: "dummy data",
			Struct: &SignedSoftwareCertificate{
				CertificateData: []byte{0xca, 0xfe},
				Signature:       []byte{0xde, 0xad, 0xbe, 0xef},
			},
			Bytes: []byte{
				// CertificateData
				0x02, 0x00, 0x00, 0x00, 0xca, 0xfe,
				// Signature
				0x04, 0x00, 0x00, 0x00, 0xde, 0xad, 0xbe, 0xef,
			},
		},
	}
	RunCodecTest(t, cases)
}

func TestSignedSoftwareCertificateArray(t *testing.T) {
	cases := []CodecTestCase{
		{
			Name:   "empty",
			Struct: []*SignedSoftwareCertificate{},
			Bytes: []byte{
				// ArraySize
				0x00, 0x00, 0x00, 0x00,
			},
		},
		{
			Name: "dummy data",
			Struct: []*SignedSoftwareCertificate{
				&SignedSoftwareCertificate{
					CertificateData: []byte{0xca, 0xfe},
					Signature:       []byte{0xde, 0xad, 0xbe, 0xef},
				},
				&SignedSoftwareCertificate{
					CertificateData: []byte{0xca, 0xfe},
					Signature:       []byte{0xde, 0xad, 0xbe, 0xef},
				},
			},
			Bytes: []byte{
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
	RunCodecTest(t, cases)
}
