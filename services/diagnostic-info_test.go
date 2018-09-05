// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package services

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/wmnsk/gopcua/datatypes"
)

func TestDecodeDiagnosticInfo(t *testing.T) {
	cases := []struct {
		input []byte
		want  *DiagnosticInfo
	}{
		{ // Nothing
			[]byte{
				0x00,
			},
			NewNullDiagnosticInfo(),
		},
		{ // Has SymbolicID
			[]byte{
				0x01, 0x01, 0x00, 0x00, 0x00,
			},
			NewDiagnosticInfo(
				true, false, false, false, false, false, false,
				1, 0, 0, 0, nil, 0, nil,
			),
		},
		{ // Has NamespaceURI
			[]byte{
				0x02, 0x02, 0x00, 0x00, 0x00,
			},
			NewDiagnosticInfo(
				false, true, false, false, false, false, false,
				0, 2, 0, 0, nil, 0, nil,
			),
		},
		{ // Has LocalizedText
			[]byte{
				0x04, 0x03, 0x00, 0x00, 0x00,
			},
			NewDiagnosticInfo(
				false, false, true, false, false, false, false,
				0, 0, 0, 3, nil, 0, nil,
			),
		},
		{ // Has Locale
			[]byte{
				0x08, 0x04, 0x00, 0x00, 0x00,
			},
			NewDiagnosticInfo(
				false, false, false, true, false, false, false,
				0, 0, 4, 0, nil, 0, nil,
			),
		},
		{ // Has AdditionalInfo
			[]byte{
				0x10, 0x06, 0x00, 0x00, 0x00, 0x66, 0x6f, 0x6f,
				0x62, 0x61, 0x72,
			},
			NewDiagnosticInfo(
				false, false, false, false, true, false, false,
				0, 0, 0, 0,
				datatypes.NewString("foobar"),
				0, nil,
			),
		},
		{ // Has InnerStatusCode
			[]byte{
				0x20, 0x06, 0x00, 0x00, 0x00,
			},
			NewDiagnosticInfo(
				false, false, false, false, false, true, false,
				0, 0, 0, 0, nil, 6, nil,
			),
		},
		{ // Has InnerDiagnosticInfo
			[]byte{
				0x40, 0x01, 0x07, 0x00, 0x00, 0x00,
			},
			NewDiagnosticInfo(
				false, false, false, false, false, false, true,
				0, 0, 0, 0, nil, 0,
				NewDiagnosticInfo(
					true, false, false, false, false, false, false,
					7, 0, 0, 0, nil, 0, nil,
				),
			),
		},
		{ // Has ALL
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
			NewDiagnosticInfo(
				true, true, true, true, true, true, true,
				1, 2, 4, 3,
				datatypes.NewString("foobar"),
				6,
				NewDiagnosticInfo(
					true, false, false, false, false, false, false,
					7, 0, 0, 0, nil, 0, nil,
				),
			),
		},
	}

	for i, c := range cases {
		got, err := DecodeDiagnosticInfo(c.input)
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(got, c.want); diff != "" {
			t.Errorf("case #%d failed\n%s", i, diff)
		}
	}
}

func TestSerializeDiagnosticInfo(t *testing.T) {
	cases := []struct {
		input *DiagnosticInfo
		want  []byte
	}{
		{ // Nothing
			NewNullDiagnosticInfo(),
			[]byte{
				0x00,
			},
		},
		{ // Has SymbolicID
			NewDiagnosticInfo(
				true, false, false, false, false, false, false,
				1, 0, 0, 0, nil, 0, nil,
			),
			[]byte{
				0x01, 0x01, 0x00, 0x00, 0x00,
			},
		},
		{ // Has NamespaceURI
			NewDiagnosticInfo(
				false, true, false, false, false, false, false,
				0, 2, 0, 0, nil, 0, nil,
			),
			[]byte{
				0x02, 0x02, 0x00, 0x00, 0x00,
			},
		},
		{ // Has LocalizedText
			NewDiagnosticInfo(
				false, false, true, false, false, false, false,
				0, 0, 0, 3, nil, 0, nil,
			),
			[]byte{
				0x04, 0x03, 0x00, 0x00, 0x00,
			},
		},
		{ // Has Locale
			NewDiagnosticInfo(
				false, false, false, true, false, false, false,
				0, 0, 4, 0, nil, 0, nil,
			),
			[]byte{
				0x08, 0x04, 0x00, 0x00, 0x00,
			},
		},
		{ // Has AdditionalInfo
			NewDiagnosticInfo(
				false, false, false, false, true, false, false,
				0, 0, 0, 0,
				datatypes.NewString("foobar"),
				0, nil,
			),
			[]byte{
				0x10, 0x06, 0x00, 0x00, 0x00, 0x66, 0x6f, 0x6f,
				0x62, 0x61, 0x72,
			},
		},
		{ // Has InnerStatusCode
			NewDiagnosticInfo(
				false, false, false, false, false, true, false,
				0, 0, 0, 0, nil, 6, nil,
			),
			[]byte{
				0x20, 0x06, 0x00, 0x00, 0x00,
			},
		},
		{ // Has InnerDiagnosticInfo
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
		{ // Has ALL
			NewDiagnosticInfo(
				true, true, true, true, true, true, true,
				1, 2, 4, 3,
				datatypes.NewString("foobar"),
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

	for i, c := range cases {
		got, err := c.input.Serialize()
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(got, c.want); diff != "" {
			t.Errorf("case #%d failed\n%s", i, diff)
		}
	}
}
func TestDiagnosticInfoLen(t *testing.T) {
	cases := []struct {
		input *DiagnosticInfo
		want  int
	}{
		{ // Nothing
			NewNullDiagnosticInfo(),
			1,
		},
		{ // Has SymbolicID
			NewDiagnosticInfo(
				true, false, false, false, false, false, false,
				1, 0, 0, 0, nil, 0, nil,
			),
			5,
		},
		{ // Has NamespaceURI
			NewDiagnosticInfo(
				false, true, false, false, false, false, false,
				0, 2, 0, 0, nil, 0, nil,
			),
			5,
		},
		{ // Has LocalizedText
			NewDiagnosticInfo(
				false, false, true, false, false, false, false,
				0, 0, 0, 3, nil, 0, nil,
			),
			5,
		},
		{ // Has Locale
			NewDiagnosticInfo(
				false, false, false, true, false, false, false,
				0, 0, 4, 0, nil, 0, nil,
			),
			5,
		},
		{ // Has AdditionalInfo
			NewDiagnosticInfo(
				false, false, false, false, true, false, false,
				0, 0, 0, 0,
				datatypes.NewString("foobar"),
				0, nil,
			),
			11,
		},
		{ // Has InnerStatusCode
			NewDiagnosticInfo(
				false, false, false, false, false, true, false,
				0, 0, 0, 0, nil, 6, nil,
			),
			5,
		},
		{ // Has InnerDiagnosticInfo
			NewDiagnosticInfo(
				false, false, false, false, false, false, true,
				0, 0, 0, 0, nil, 0,
				NewDiagnosticInfo(
					true, false, false, false, false, false, false,
					7, 0, 0, 0, nil, 0, nil,
				),
			),
			6,
		},
		{ // Has ALL
			NewDiagnosticInfo(
				true, true, true, true, true, true, true,
				1, 2, 4, 3,
				datatypes.NewString("foobar"),
				6,
				NewDiagnosticInfo(
					true, false, false, false, false, false, false,
					7, 0, 0, 0, nil, 0, nil,
				),
			),
			36,
		},
	}

	for i, c := range cases {
		got := c.input.Len()
		if diff := cmp.Diff(got, c.want); diff != "" {
			t.Errorf("case #%d failed\n%s", i, diff)
		}
	}
}

func TestDecodeDiagnosticInfoArray(t *testing.T) {
	cases := []struct {
		input []byte
		want  *DiagnosticInfoArray
	}{
		{ // Nothing
			[]byte{
				0x00, 0x00, 0x00, 0x00,
			},
			NewDiagnosticInfoArray(nil),
		},
		{ // 1 null DiagnosticInfo
			[]byte{
				0x01, 0x00, 0x00, 0x00,
				0x00,
			},
			NewDiagnosticInfoArray([]*DiagnosticInfo{NewNullDiagnosticInfo()}),
		},
		{ // 4 null DiagnosticInfo
			[]byte{
				0x04, 0x00, 0x00, 0x00,
				0x00,
				0x00,
				0x00,
				0x00,
			},
			NewDiagnosticInfoArray([]*DiagnosticInfo{
				NewNullDiagnosticInfo(),
				NewNullDiagnosticInfo(),
				NewNullDiagnosticInfo(),
				NewNullDiagnosticInfo(),
			}),
		},
		{ // 1 null DiagnosticInfo & 1 DiagnosticInfo with SymbolicID
			[]byte{
				0x02, 0x00, 0x00, 0x00,
				0x00,
				0x01, 0x01, 0x00, 0x00, 0x00,
			},
			NewDiagnosticInfoArray([]*DiagnosticInfo{
				NewNullDiagnosticInfo(),
				NewDiagnosticInfo(
					true, false, false, false, false, false, false,
					1, 0, 0, 0, nil, 0, nil,
				),
			}),
		},
	}

	for i, c := range cases {
		got, err := DecodeDiagnosticInfoArray(c.input)
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(got, c.want); diff != "" {
			t.Errorf("case #%d failed\n%s", i, diff)
		}
	}
}

func TestSerializeDiagnosticInfoArray(t *testing.T) {
	cases := []struct {
		input *DiagnosticInfoArray
		want  []byte
	}{
		{ // Nothing
			NewDiagnosticInfoArray(nil),
			[]byte{
				0x00, 0x00, 0x00, 0x00,
			},
		},
		{ // 1 null DiagnosticInfo
			NewDiagnosticInfoArray([]*DiagnosticInfo{NewNullDiagnosticInfo()}),
			[]byte{
				0x01, 0x00, 0x00, 0x00,
				0x00,
			},
		},
		{ // 4 null DiagnosticInfo
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
		{ // 1 null DiagnosticInfo & 1 DiagnosticInfo with SymbolicID
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

	for i, c := range cases {
		got, err := c.input.Serialize()
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(got, c.want); diff != "" {
			t.Errorf("case #%d failed\n%s", i, diff)
		}
	}
}

func TestDiagnosticInfoArrayLen(t *testing.T) {
	cases := []struct {
		input *DiagnosticInfoArray
		want  int
	}{
		{ // Nothing
			NewDiagnosticInfoArray(nil),
			4,
		},
		{ // 1 null DiagnosticInfo
			NewDiagnosticInfoArray([]*DiagnosticInfo{NewNullDiagnosticInfo()}),
			5,
		},
		{ // 4 null DiagnosticInfo
			NewDiagnosticInfoArray([]*DiagnosticInfo{
				NewNullDiagnosticInfo(),
				NewNullDiagnosticInfo(),
				NewNullDiagnosticInfo(),
				NewNullDiagnosticInfo(),
			}),
			8,
		},
		{ // 1 null DiagnosticInfo & 1 DiagnosticInfo with SymbolicID
			NewDiagnosticInfoArray([]*DiagnosticInfo{
				NewNullDiagnosticInfo(),
				NewDiagnosticInfo(
					true, false, false, false, false, false, false,
					1, 0, 0, 0, nil, 0, nil,
				),
			}),
			10,
		},
	}

	for i, c := range cases {
		got := c.input.Len()
		if diff := cmp.Diff(got, c.want); diff != "" {
			t.Errorf("case #%d failed\n%s", i, diff)
		}
	}
}
