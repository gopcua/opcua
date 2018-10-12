// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package datatypes

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

var diagnosticInfoCases = []struct {
	description string
	structured  *DiagnosticInfo
	serialized  []byte
}{
	{
		"nothing",
		NewNullDiagnosticInfo(),
		[]byte{0x00},
	},
	{
		"has-symbolic-id",
		NewDiagnosticInfo(
			true, false, false, false, false, false, false,
			1, 0, 0, 0, nil, 0, nil,
		),
		[]byte{
			0x01, 0x01, 0x00, 0x00, 0x00,
		},
	},
	{
		"has-namespace-uri",
		NewDiagnosticInfo(
			false, true, false, false, false, false, false,
			0, 2, 0, 0, nil, 0, nil,
		),
		[]byte{
			0x02, 0x02, 0x00, 0x00, 0x00,
		},
	},
	{
		"has-localized-text",
		NewDiagnosticInfo(
			false, false, true, false, false, false, false,
			0, 0, 0, 3, nil, 0, nil,
		),
		[]byte{
			0x04, 0x03, 0x00, 0x00, 0x00,
		},
	},
	{
		"has-locale",
		NewDiagnosticInfo(
			false, false, false, true, false, false, false,
			0, 0, 4, 0, nil, 0, nil,
		),
		[]byte{
			0x08, 0x04, 0x00, 0x00, 0x00,
		},
	},
	{
		"has-additional-info",
		NewDiagnosticInfo(
			false, false, false, false, true, false, false,
			0, 0, 0, 0,
			NewString("foobar"),
			0, nil,
		),
		[]byte{
			0x10, 0x06, 0x00, 0x00, 0x00, 0x66, 0x6f, 0x6f,
			0x62, 0x61, 0x72,
		},
	},
	{
		"has-inner-status-code",
		NewDiagnosticInfo(
			false, false, false, false, false, true, false,
			0, 0, 0, 0, nil, 6, nil,
		),
		[]byte{
			0x20, 0x06, 0x00, 0x00, 0x00,
		},
	},
	{
		"has-inner-diagnostic-info",
		NewDiagnosticInfo(
			false, false, false, false, false, false, true,
			0, 0, 0, 0, nil, 0,
			NewDiagnosticInfo(
				true, false, false, false, false, false, false,
				7, 0, 0, 0, nil, 0, nil,
			),
		),
		[]byte{
			0x40, 0x01, 0x07, 0x00, 0x00, 0x00,
		},
	},
	{
		"has-all",
		NewDiagnosticInfo(
			true, true, true, true, true, true, true,
			1, 2, 4, 3,
			NewString("foobar"),
			6,
			NewDiagnosticInfo(
				true, false, false, false, false, false, false,
				7, 0, 0, 0, nil, 0, nil,
			),
		),
		[]byte{
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
	},
}

func TestDecodeDiagnosticInfo(t *testing.T) {
	for _, c := range diagnosticInfoCases {
		got, err := DecodeDiagnosticInfo(c.serialized)
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(got, c.structured); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}

func TestSerializeDiagnosticInfo(t *testing.T) {
	for _, c := range diagnosticInfoCases {
		got, err := c.structured.Serialize()
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(got, c.serialized); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}

func TestDiagnosticInfoLen(t *testing.T) {
	for _, c := range diagnosticInfoCases {
		got := c.structured.Len()

		if diff := cmp.Diff(got, len(c.serialized)); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}

var diagnosticInfoArrayCases = []struct {
	description string
	structured  *DiagnosticInfoArray
	serialized  []byte
}{
	{
		"nothing",
		NewDiagnosticInfoArray(nil),
		[]byte{
			0x00, 0x00, 0x00, 0x00,
		},
	},
	{
		"1 null DiagnosticInfo",
		NewDiagnosticInfoArray([]*DiagnosticInfo{NewNullDiagnosticInfo()}),
		[]byte{
			0x01, 0x00, 0x00, 0x00,
			0x00,
		},
	},
	{
		"4 null DiagnosticInfo",
		NewDiagnosticInfoArray([]*DiagnosticInfo{
			NewNullDiagnosticInfo(),
			NewNullDiagnosticInfo(),
			NewNullDiagnosticInfo(),
			NewNullDiagnosticInfo(),
		}),
		[]byte{
			0x04, 0x00, 0x00, 0x00,
			0x00,
			0x00,
			0x00,
			0x00,
		},
	},
	{
		"1 null DiagnosticInfo & 1 DiagnosticInfo with SymbolicID",
		NewDiagnosticInfoArray([]*DiagnosticInfo{
			NewNullDiagnosticInfo(),
			NewDiagnosticInfo(
				true, false, false, false, false, false, false,
				1, 0, 0, 0, nil, 0, nil,
			),
		}),
		[]byte{
			0x02, 0x00, 0x00, 0x00,
			0x00,
			0x01, 0x01, 0x00, 0x00, 0x00,
		},
	},
}

func TestDecodeDiagnosticInfoArray(t *testing.T) {
	for _, c := range diagnosticInfoArrayCases {
		got, err := DecodeDiagnosticInfoArray(c.serialized)
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(got, c.structured); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}

func TestSerializeDiagnosticInfoArray(t *testing.T) {
	for _, c := range diagnosticInfoArrayCases {
		got, err := c.structured.Serialize()
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(got, c.serialized); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}

func TestDiagnosticInfoArrayLen(t *testing.T) {
	for _, c := range diagnosticInfoArrayCases {
		got := c.structured.Len()

		if diff := cmp.Diff(got, len(c.serialized)); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}
