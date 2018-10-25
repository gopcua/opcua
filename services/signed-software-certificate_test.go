// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package services

import (
	"encoding/hex"
	"testing"
)

var testSignedSoftwareCertificateBytes = [][]byte{
	{ // Empty
		// CertificateData
		0xff, 0xff, 0xff, 0xff,
		// Signature
		0xff, 0xff, 0xff, 0xff,
	},
	{ // dummy data
		// CertificateData
		0x02, 0x00, 0x00, 0x00, 0xca, 0xfe,
		// Signature
		0x04, 0x00, 0x00, 0x00, 0xde, 0xad, 0xbe, 0xef,
	},
	{ // Empty Array
		// ArraySize
		0x00, 0x00, 0x00, 0x00,
	},
	{ // Array with dummy data
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
	{},
}

func TestDecodeSignedSoftwareCertificate(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		s, err := DecodeSignedSoftwareCertificate(testSignedSoftwareCertificateBytes[0])
		if err != nil {
			t.Fatalf("Failed to decode SignedSoftwareCertificate: %s", err)
		}

		switch {
		case s.CertificateData.Get() != nil:
			t.Errorf("CertificateData doesn't match. Want: %v, Got: %v", nil, s.CertificateData.Get())
		case s.Signature.Get() != nil:
			t.Errorf("Signature doesn't match. Want: %v, Got: %v", nil, s.Signature.Get())
		}
		t.Log(s.String())
	})
	t.Run("with-dummy", func(t *testing.T) {
		s, err := DecodeSignedSoftwareCertificate(testSignedSoftwareCertificateBytes[1])
		if err != nil {
			t.Fatalf("Failed to decode SignedSoftwareCertificate: %s", err)
		}

		dummyCert := hex.EncodeToString(s.CertificateData.Get())
		dummySign := hex.EncodeToString(s.Signature.Get())
		switch {
		case dummyCert != "cafe":
			t.Errorf("CertificateData doesn't match. Want: %s, Got: %s", "cafe", dummyCert)
		case dummySign != "deadbeef":
			t.Errorf("Signature doesn't match. Want: %s, Got: %s", "deadbeef", dummySign)
		}
		t.Log(s.String())
	})
	t.Run("array-empty", func(t *testing.T) {
		s, err := DecodeSignedSoftwareCertificateArray(testSignedSoftwareCertificateBytes[2])
		if err != nil {
			t.Fatalf("Failed to decode SignedSoftwareCertificateArray: %s", err)
		}

		if s.ArraySize != 0 {
			t.Errorf("ArraySize doesn't match. Want: %d, Got: %d", 0, s.ArraySize)
		}
		t.Log(s)
	})
	t.Run("array-with-dummy", func(t *testing.T) {
		s, err := DecodeSignedSoftwareCertificateArray(testSignedSoftwareCertificateBytes[3])
		if err != nil {
			t.Fatalf("Failed to decode SignedSoftwareCertificateArray: %s", err)
		}

		for _, cert := range s.Certificates {
			dummyCert := hex.EncodeToString(cert.CertificateData.Get())
			dummySign := hex.EncodeToString(cert.Signature.Get())
			switch {
			case dummyCert != "cafe":
				t.Errorf("CertificateData doesn't match. Want: %s, Got: %s", "cafe", dummyCert)
			case dummySign != "deadbeef":
				t.Errorf("Signature doesn't match. Want: %s, Got: %s", "deadbeef", dummySign)
			}
			t.Log(cert.String())
		}
	})
}

func TestSerializeSignedSoftwareCertificate(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		s := NewSignedSoftwareCertificate(nil, nil)

		serialized, err := s.Serialize()
		if err != nil {
			t.Fatalf("Failed to serialize SignedSoftwareCertificate: %s", err)
		}

		for i, s := range serialized {
			x := testSignedSoftwareCertificateBytes[0][i]
			if s != x {
				t.Errorf("Bytes doesn't match. Want: %#x, Got: %#x at %dth", x, s, i)
			}
		}
		t.Logf("%x", serialized)
	})
	t.Run("with-dummy", func(t *testing.T) {
		s := NewSignedSoftwareCertificate([]byte{0xca, 0xfe}, []byte{0xde, 0xad, 0xbe, 0xef})

		serialized, err := s.Serialize()
		if err != nil {
			t.Fatalf("Failed to serialize SignedSoftwareCertificate: %s", err)
		}

		for i, s := range serialized {
			x := testSignedSoftwareCertificateBytes[1][i]
			if s != x {
				t.Errorf("Bytes doesn't match. Want: %#x, Got: %#x at %dth", x, s, i)
			}
		}
		t.Logf("%x", serialized)
	})
	t.Run("array-with-dummy", func(t *testing.T) {
		s := NewSignedSoftwareCertificateArray(nil)

		serialized, err := s.Serialize()
		if err != nil {
			t.Fatalf("Failed to serialize SignedSoftwareCertificateArray: %s", err)
		}

		for i, s := range serialized {
			x := testSignedSoftwareCertificateBytes[2][i]
			if s != x {
				t.Errorf("Bytes doesn't match. Want: %#x, Got: %#x at %dth", x, s, i)
			}
		}
		t.Logf("%x", serialized)
	})
	t.Run("array-with-dummy", func(t *testing.T) {
		s := NewSignedSoftwareCertificateArray(
			[]*SignedSoftwareCertificate{
				NewSignedSoftwareCertificate([]byte{0xca, 0xfe}, []byte{0xde, 0xad, 0xbe, 0xef}),
				NewSignedSoftwareCertificate([]byte{0xca, 0xfe}, []byte{0xde, 0xad, 0xbe, 0xef}),
			},
		)

		serialized, err := s.Serialize()
		if err != nil {
			t.Fatalf("Failed to serialize SignedSoftwareCertificateArray: %s", err)
		}

		for i, s := range serialized {
			x := testSignedSoftwareCertificateBytes[3][i]
			if s != x {
				t.Errorf("Bytes doesn't match. Want: %#x, Got: %#x at %dth", x, s, i)
			}
		}
		t.Logf("%x", serialized)
	})
}
