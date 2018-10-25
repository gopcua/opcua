// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package services

import (
	"testing"

	"github.com/wmnsk/gopcua/datatypes"
	"github.com/wmnsk/gopcua/utils/codectest"
)

func TestDiagnosticInfo(t *testing.T) {
	cases := []codectest.Case{
		{
			Name:   "Nothing",
			Struct: NewNullDiagnosticInfo(),
			Bytes: []byte{
				0x00,
			},
		},
		{
			Name: "Has SymbolicID",
			Struct: NewDiagnosticInfo(
				true, false, false, false, false, false, false,
				1, 0, 0, 0, nil, 0, nil,
			),
			Bytes: []byte{
				0x01, 0x01, 0x00, 0x00, 0x00,
			},
		},
		{
			Name: "Has NamespaceURI",
			Struct: NewDiagnosticInfo(
				false, true, false, false, false, false, false,
				0, 2, 0, 0, nil, 0, nil,
			),
			Bytes: []byte{
				0x02, 0x02, 0x00, 0x00, 0x00,
			},
		},
		{
			Name: "Has LocalizedText",
			Struct: NewDiagnosticInfo(
				false, false, true, false, false, false, false,
				0, 0, 0, 3, nil, 0, nil,
			),
			Bytes: []byte{
				0x04, 0x03, 0x00, 0x00, 0x00,
			},
		},
		{
			Name: "Has Locale",
			Struct: NewDiagnosticInfo(
				false, false, false, true, false, false, false,
				0, 0, 4, 0, nil, 0, nil,
			),
			Bytes: []byte{
				0x08, 0x04, 0x00, 0x00, 0x00,
			},
		},
		{
			Name: "Has AdditionalInfo",
			Struct: NewDiagnosticInfo(
				false, false, false, false, true, false, false,
				0, 0, 0, 0,
				datatypes.NewString("foobar"),
				0, nil,
			),
			Bytes: []byte{
				0x10, 0x06, 0x00, 0x00, 0x00, 0x66, 0x6f, 0x6f,
				0x62, 0x61, 0x72,
			},
		},
		{
			Name: "Has InnerStatusCode",
			Struct: NewDiagnosticInfo(
				false, false, false, false, false, true, false,
				0, 0, 0, 0, nil, 6, nil,
			),
			Bytes: []byte{
				0x20, 0x06, 0x00, 0x00, 0x00,
			},
		},
		{
			Name: "Has InnerDiagnosticInfo",
			Struct: NewDiagnosticInfo(
				false, false, false, false, false, false, true,
				0, 0, 0, 0, nil, 0,
				NewDiagnosticInfo(
					true, false, false, false, false, false, false,
					7, 0, 0, 0, nil, 0, nil,
				),
			),
			Bytes: []byte{
				0x40, 0x01, 0x07, 0x00, 0x00, 0x00,
			},
		},
		{
			Name: "Has all",
			Struct: NewDiagnosticInfo(
				true, true, true, true, true, true, true,
				1, 2, 4, 3,
				datatypes.NewString("foobar"),
				6,
				NewDiagnosticInfo(
					true, false, false, false, false, false, false,
					7, 0, 0, 0, nil, 0, nil,
				),
			),
			Bytes: []byte{
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
	codectest.Run(t, cases, func(b []byte) (codectest.S, error) {
		return DecodeDiagnosticInfo(b)
	})
}

func TestDiagnosticInfoArray(t *testing.T) {
	cases := []codectest.Case{
		{
			Name:   "Nothing",
			Struct: NewDiagnosticInfoArray(nil),
			Bytes: []byte{
				0x00, 0x00, 0x00, 0x00,
			},
		},
		{
			Name:   "1 null DiagnosticInfo",
			Struct: NewDiagnosticInfoArray([]*DiagnosticInfo{NewNullDiagnosticInfo()}),
			Bytes: []byte{
				0x01, 0x00, 0x00, 0x00,
				0x00,
			},
		},
		{
			Name: "4 null DiagnosticInfo",
			Struct: NewDiagnosticInfoArray([]*DiagnosticInfo{
				NewNullDiagnosticInfo(),
				NewNullDiagnosticInfo(),
				NewNullDiagnosticInfo(),
				NewNullDiagnosticInfo(),
			}),
			Bytes: []byte{
				0x04, 0x00, 0x00, 0x00,
				0x00,
				0x00,
				0x00,
				0x00,
			},
		},
		{
			Name: "1 null DiagnosticInfo & 1 DiagnosticInfo with SymbolicID",
			Struct: NewDiagnosticInfoArray([]*DiagnosticInfo{
				NewNullDiagnosticInfo(),
				NewDiagnosticInfo(
					true, false, false, false, false, false, false,
					1, 0, 0, 0, nil, 0, nil,
				),
			}),
			Bytes: []byte{
				0x02, 0x00, 0x00, 0x00,
				0x00,
				0x01, 0x01, 0x00, 0x00, 0x00,
			},
		},
	}
	codectest.Run(t, cases, func(b []byte) (codectest.S, error) {
		return DecodeDiagnosticInfoArray(b)
	})
}
