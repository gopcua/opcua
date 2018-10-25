// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package services

import (
	"testing"

	"github.com/wmnsk/gopcua/utils/codectest"
)

func TestSignedSoftwareCertificate(t *testing.T) {
	cases := []codectest.Case{
		{
			Name:   "empty",
			Struct: NewSignedSoftwareCertificate(nil, nil),
			Bytes: []byte{
				// CertificateData
				0xff, 0xff, 0xff, 0xff,
				// Signature
				0xff, 0xff, 0xff, 0xff,
			},
		},
		{
			Name:   "dummy data",
			Struct: NewSignedSoftwareCertificate([]byte{0xca, 0xfe}, []byte{0xde, 0xad, 0xbe, 0xef}),
			Bytes: []byte{
				// CertificateData
				0x02, 0x00, 0x00, 0x00, 0xca, 0xfe,
				// Signature
				0x04, 0x00, 0x00, 0x00, 0xde, 0xad, 0xbe, 0xef,
			},
		},
	}
	codectest.Run(t, cases, func(b []byte) (codectest.S, error) {
		return DecodeSignedSoftwareCertificate(b)
	})
}

func TestSignedSoftwareCertificateArray(t *testing.T) {
	cases := []codectest.Case{
		{
			Name:   "empty",
			Struct: NewSignedSoftwareCertificateArray(nil),
			Bytes: []byte{
				// ArraySize
				0x00, 0x00, 0x00, 0x00,
			},
		},
		{
			Name: "dummy data",
			Struct: NewSignedSoftwareCertificateArray(
				[]*SignedSoftwareCertificate{
					NewSignedSoftwareCertificate([]byte{0xca, 0xfe}, []byte{0xde, 0xad, 0xbe, 0xef}),
					NewSignedSoftwareCertificate([]byte{0xca, 0xfe}, []byte{0xde, 0xad, 0xbe, 0xef}),
				},
			),
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
	codectest.Run(t, cases, func(b []byte) (codectest.S, error) {
		return DecodeSignedSoftwareCertificateArray(b)
	})
}
