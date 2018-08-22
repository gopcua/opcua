// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package services

import (
	"testing"

	"github.com/wmnsk/gopcua/datatypes"
)

var testDiagnosticInfoBytes = [][]byte{
	{ // Nothing
		0x00,
	},
	{ // Has SymbolicID
		0x01, 0x01, 0x00, 0x00, 0x00,
	},
	{ // Has NamespaceURI
		0x02, 0x02, 0x00, 0x00, 0x00,
	},
	{ // Has LocalizedText
		0x04, 0x03, 0x00, 0x00, 0x00,
	},
	{ // Has Locale
		0x08, 0x04, 0x00, 0x00, 0x00,
	},
	{ // Has AdditionalInfo
		0x10, 0x06, 0x00, 0x00, 0x00, 0x66, 0x6f, 0x6f,
		0x62, 0x61, 0x72,
	},
	{ // Has InnerStatusCode
		0x20, 0x06, 0x00, 0x00, 0x00,
	},
	{ // Has InnerDiagnosticInfo
		0x40, 0x01, 0x07, 0x00, 0x00, 0x00,
	},
	{ // Has ALL
		0x7f,
		// SymbolicID
		0x01, 0x00, 0x00, 0x00,
		// NamespaceURI
		0x02, 0x00, 0x00, 0x00,
		// Locale
		0x04, 0x00, 0x00, 0x00,
		// LocalizedText
		0x03, 0x00, 0x00, 0x00,
		// AdditionalInfo
		0x06, 0x00, 0x00, 0x00, 0x66, 0x6f, 0x6f, 0x62, 0x61, 0x72,
		// InnerStatusCode
		0x06, 0x00, 0x00, 0x00,
		// InnerDiagnostics
		0x01, 0x07, 0x00, 0x00, 0x00,
	},
}

func TestDecodeDiagnosticInfo(t *testing.T) {
	t.Run("nothing", func(t *testing.T) {
		d, err := DecodeDiagnosticInfo(testDiagnosticInfoBytes[0])
		if err != nil {
			t.Fatalf("Failed to decode DiagnosticInfo: %s", err)
		}

		if d.EncodingMask != 0x00 {
			t.Errorf("EncodingMask doesn't match. Want: %x, Got %x", 0x00, d.EncodingMask)
		}
		t.Log(d.String())
	})
	t.Run("has-symbolic-id", func(t *testing.T) {
		d, err := DecodeDiagnosticInfo(testDiagnosticInfoBytes[1])
		if err != nil {
			t.Fatalf("Failed to decode DiagnosticInfo: %s", err)
		}

		switch {
		case d.EncodingMask != 0x01:
			t.Errorf("EncodingMask doesn't match. Want: %x, Got %x", 0x01, d.EncodingMask)
		case d.SymbolicID != 0x01:
			t.Errorf("SymbolicID doesn't match. Want: %x, Got %x", 0x01, d.SymbolicID)
		}
		t.Log(d.String())
	})
	t.Run("has-uri", func(t *testing.T) {
		d, err := DecodeDiagnosticInfo(testDiagnosticInfoBytes[2])
		if err != nil {
			t.Fatalf("Failed to decode DiagnosticInfo: %s", err)
		}

		switch {
		case d.EncodingMask != 0x02:
			t.Errorf("EncodingMask doesn't match. Want: %x, Got %x", 0x02, d.EncodingMask)
		case d.NamespaceURI != 0x02:
			t.Errorf("NamespaceURI doesn't match. Want: %x, Got %x", 0x02, d.NamespaceURI)
		}
		t.Log(d.String())
	})
	t.Run("has-text", func(t *testing.T) {
		d, err := DecodeDiagnosticInfo(testDiagnosticInfoBytes[3])
		if err != nil {
			t.Fatalf("Failed to decode DiagnosticInfo: %s", err)
		}

		switch {
		case d.EncodingMask != 0x04:
			t.Errorf("EncodingMask doesn't match. Want: %x, Got %x", 0x04, d.EncodingMask)
		case d.LocalizedText != 0x03:
			t.Errorf("LocalizedText doesn't match. Want: %x, Got %x", 0x03, d.LocalizedText)
		}
		t.Log(d.String())
	})
	t.Run("has-locale", func(t *testing.T) {
		d, err := DecodeDiagnosticInfo(testDiagnosticInfoBytes[4])
		if err != nil {
			t.Fatalf("Failed to decode DiagnosticInfo: %s", err)
		}

		switch {
		case d.EncodingMask != 0x08:
			t.Errorf("EncodingMask doesn't match. Want: %x, Got %x", 0x08, d.EncodingMask)
		case d.Locale != 0x04:
			t.Errorf("Locale doesn't match. Want: %x, Got %x", 0x04, d.Locale)
		}
		t.Log(d.String())
	})
	t.Run("has-info", func(t *testing.T) {
		d, err := DecodeDiagnosticInfo(testDiagnosticInfoBytes[5])
		if err != nil {
			t.Fatalf("Failed to decode DiagnosticInfo: %s", err)
		}

		switch {
		case d.EncodingMask != 0x10:
			t.Errorf("EncodingMask doesn't match. Want: %x, Got %x", 0x10, d.EncodingMask)
		case d.AdditionalInfo.Get() != "foobar":
			t.Errorf("Locale doesn't match. Want: %s, Got %s", "foobar", d.AdditionalInfo.Get())
		}
		t.Log(d.String())
	})
	t.Run("has-status", func(t *testing.T) {
		d, err := DecodeDiagnosticInfo(testDiagnosticInfoBytes[6])
		if err != nil {
			t.Fatalf("Failed to decode DiagnosticInfo: %s", err)
		}

		switch {
		case d.EncodingMask != 0x20:
			t.Errorf("EncodingMask doesn't match. Want: %x, Got %x", 0x20, d.EncodingMask)
		case d.InnerStatusCode != 0x06:
			t.Errorf("InnerStatusCode doesn't match. Want: %x, Got %x", 0x06, d.InnerStatusCode)
		}
		t.Log(d.String())
	})
	t.Run("has-diag", func(t *testing.T) {
		d, err := DecodeDiagnosticInfo(testDiagnosticInfoBytes[7])
		if err != nil {
			t.Fatalf("Failed to decode DiagnosticInfo: %s", err)
		}

		switch {
		case d.EncodingMask != 0x40:
			t.Errorf("EncodingMask doesn't match. Want: %x, Got %x", 0x40, d.EncodingMask)
		case d.InnerDiagnosticInfo.EncodingMask != 0x01:
			t.Errorf("Locale doesn't match. Want: %x, Got %x", 0x01, d.InnerDiagnosticInfo.EncodingMask)
		case d.InnerDiagnosticInfo.SymbolicID != 0x07:
			t.Errorf("SymbolicID doesn't match. Want: %x, Got %x", 0x07, d.InnerDiagnosticInfo.SymbolicID)
		}
		t.Log(d.String())
	})
	t.Run("has-all", func(t *testing.T) {
		d, err := DecodeDiagnosticInfo(testDiagnosticInfoBytes[8])
		if err != nil {
			t.Fatalf("Failed to decode DiagnosticInfo: %s", err)
		}

		switch {
		case d.EncodingMask != 0x7f:
			t.Errorf("EncodingMask doesn't match. Want: %x, Got %x", 0x7f, d.EncodingMask)
		case d.SymbolicID != 0x01:
			t.Errorf("SymbolicID doesn't match. Want: %x, Got %x", 0x01, d.SymbolicID)
		case d.NamespaceURI != 0x02:
			t.Errorf("NamespaceURI doesn't match. Want: %x, Got %x", 0x02, d.NamespaceURI)
		case d.Locale != 0x04:
			t.Errorf("Locale doesn't match. Want: %x, Got %x", 0x04, d.Locale)
		case d.LocalizedText != 0x03:
			t.Errorf("LocalizedText doesn't match. Want: %x, Got %x", 0x03, d.LocalizedText)
		case d.AdditionalInfo.Get() != "foobar":
			t.Errorf("AdditionalInfo doesn't match. Want: %s, Got %s", "foobar", d.AdditionalInfo.Get())
		case d.InnerStatusCode != 0x06:
			t.Errorf("InnerStatusCode doesn't match. Want: %x, Got %x", 0x06, d.InnerStatusCode)
		case d.InnerDiagnosticInfo.EncodingMask != 0x01:
			t.Errorf("InnerDiagnosticInfo.EncodingMask doesn't match. Want: %x, Got %x", 0x01, d.InnerDiagnosticInfo.EncodingMask)
		case d.InnerDiagnosticInfo.SymbolicID != 0x07:
			t.Errorf("SymbolicID doesn't match. Want: %x, Got %x", 0x07, d.InnerDiagnosticInfo.SymbolicID)
		}
		t.Log(d.String())
	})
}

func TestSerializeDiagnosticInfo(t *testing.T) {
	t.Run("nothing", func(t *testing.T) {
		d := NewNullDiagnosticInfo()

		serialized, err := d.Serialize()
		if err != nil {
			t.Fatalf("Failed to serialize DiagnosticInfo: %s", err)
		}

		for i, s := range serialized {
			x := testDiagnosticInfoBytes[0][i]
			if s != x {
				t.Errorf("Bytes doesn't match. Want: %#x, Got: %#x at %dth", x, s, i)
			}
		}
		t.Logf("%x", serialized)
	})
	t.Run("has-symbolic-id", func(t *testing.T) {
		d := NewDiagnosticInfo(
			true, false, false, false, false, false, false,
			1, 0, 0, 0, nil, 0, nil,
		)

		serialized, err := d.Serialize()
		if err != nil {
			t.Fatalf("Failed to serialize DiagnosticInfo: %s", err)
		}

		for i, s := range serialized {
			x := testDiagnosticInfoBytes[1][i]
			if s != x {
				t.Errorf("Bytes doesn't match. Want: %#x, Got: %#x at %dth", x, s, i)
			}
		}
		t.Logf("%x", serialized)
	})
	t.Run("has-uri", func(t *testing.T) {
		d := NewDiagnosticInfo(
			false, true, false, false, false, false, false,
			0, 2, 0, 0, nil, 0, nil,
		)

		serialized, err := d.Serialize()
		if err != nil {
			t.Fatalf("Failed to serialize DiagnosticInfo: %s", err)
		}

		for i, s := range serialized {
			x := testDiagnosticInfoBytes[2][i]
			if s != x {
				t.Errorf("Bytes doesn't match. Want: %#x, Got: %#x at %dth", x, s, i)
			}
		}
		t.Logf("%x", serialized)
	})
	t.Run("has-text", func(t *testing.T) {
		d := NewDiagnosticInfo(
			false, false, true, false, false, false, false,
			0, 0, 0, 3, nil, 0, nil,
		)

		serialized, err := d.Serialize()
		if err != nil {
			t.Fatalf("Failed to serialize DiagnosticInfo: %s", err)
		}

		for i, s := range serialized {
			x := testDiagnosticInfoBytes[3][i]
			if s != x {
				t.Errorf("Bytes doesn't match. Want: %#x, Got: %#x at %dth", x, s, i)
			}
		}
		t.Logf("%x", serialized)
	})
	t.Run("has-locale", func(t *testing.T) {
		d := NewDiagnosticInfo(
			false, false, false, true, false, false, false,
			0, 0, 4, 0, nil, 0, nil,
		)

		serialized, err := d.Serialize()
		if err != nil {
			t.Fatalf("Failed to serialize DiagnosticInfo: %s", err)
		}

		for i, s := range serialized {
			x := testDiagnosticInfoBytes[4][i]
			if s != x {
				t.Errorf("Bytes doesn't match. Want: %#x, Got: %#x at %dth", x, s, i)
			}
		}
		t.Logf("%x", serialized)
	})
	t.Run("has-info", func(t *testing.T) {
		d := NewDiagnosticInfo(
			false, false, false, false, true, false, false,
			0, 0, 0, 0,
			datatypes.NewString("foobar"),
			0, nil,
		)

		serialized, err := d.Serialize()
		if err != nil {
			t.Fatalf("Failed to serialize DiagnosticInfo: %s", err)
		}

		for i, s := range serialized {
			x := testDiagnosticInfoBytes[5][i]
			if s != x {
				t.Errorf("Bytes doesn't match. Want: %#x, Got: %#x at %dth", x, s, i)
			}
		}
		t.Logf("%x", serialized)
	})
	t.Run("has-status", func(t *testing.T) {
		d := NewDiagnosticInfo(
			false, false, false, false, false, true, false,
			0, 0, 0, 0, nil, 6, nil,
		)

		serialized, err := d.Serialize()
		if err != nil {
			t.Fatalf("Failed to serialize DiagnosticInfo: %s", err)
		}

		for i, s := range serialized {
			x := testDiagnosticInfoBytes[6][i]
			if s != x {
				t.Errorf("Bytes doesn't match. Want: %#x, Got: %#x at %dth", x, s, i)
			}
		}
		t.Logf("%x", serialized)
	})
	t.Run("has-diag", func(t *testing.T) {
		d := NewDiagnosticInfo(
			false, false, false, false, false, false, true,
			0, 0, 0, 0, nil, 0,
			NewDiagnosticInfo(
				true, false, false, false, false, false, false,
				7, 0, 0, 0, nil, 0, nil,
			),
		)

		serialized, err := d.Serialize()
		if err != nil {
			t.Fatalf("Failed to serialize DiagnosticInfo: %s", err)
		}

		for i, s := range serialized {
			x := testDiagnosticInfoBytes[7][i]
			if s != x {
				t.Errorf("Bytes doesn't match. Want: %#x, Got: %#x at %dth", x, s, i)
			}
		}
		t.Logf("%x", serialized)
	})
	t.Run("has-all", func(t *testing.T) {
		d := NewDiagnosticInfo(
			true, true, true, true, true, true, true,
			1, 2, 4, 3,
			datatypes.NewString("foobar"),
			6,
			NewDiagnosticInfo(
				true, false, false, false, false, false, false,
				7, 0, 0, 0, nil, 0, nil,
			),
		)

		serialized, err := d.Serialize()
		if err != nil {
			t.Fatalf("Failed to serialize DiagnosticInfo: %s", err)
		}

		for i, s := range serialized {
			x := testDiagnosticInfoBytes[8][i]
			if s != x {
				t.Errorf("Bytes doesn't match. Want: %#x, Got: %#x at %dth", x, s, i)
			}
		}
		t.Logf("%x", serialized)
	})
}
