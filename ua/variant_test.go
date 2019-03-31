// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package ua

import (
	"testing"
	"time"
)

func TestVariant(t *testing.T) {
	cases := []CodecTestCase{
		{
			Name:   "boolean",
			Struct: MustVariant(false),
			Bytes: []byte{
				// variant encoding mask
				0x01,
				// value
				0x00,
			},
		},
		{
			Name:   "int8",
			Struct: MustVariant(int8(-5)),
			Bytes: []byte{
				// variant encoding mask
				0x02,
				// value
				0xfb,
			},
		},
		{
			Name:   "uint8",
			Struct: MustVariant(uint8(5)),
			Bytes: []byte{
				// variant encoding mask
				0x03,
				// value
				0x05,
			},
		},
		{
			Name:   "int16",
			Struct: MustVariant(int16(-5)),
			Bytes: []byte{
				// variant encoding mask
				0x04,
				// value
				0xfb, 0xff,
			},
		},
		{
			Name:   "uint16",
			Struct: MustVariant(uint16(5)),
			Bytes: []byte{
				// variant encoding mask
				0x05,
				// value
				0x05, 0x00,
			},
		},
		{
			Name:   "int32",
			Struct: MustVariant(int32(-5)),
			Bytes: []byte{
				// variant encoding mask
				0x06,
				// value
				0xfb, 0xff, 0xff, 0xff,
			},
		},
		{
			Name:   "uint32",
			Struct: MustVariant(uint32(5)),
			Bytes: []byte{
				// variant encoding mask
				0x07,
				// value
				0x05, 0x00, 0x00, 0x00,
			},
		},
		{
			Name:   "int64",
			Struct: MustVariant(int64(-5)),
			Bytes: []byte{
				// variant encoding mask
				0x08,
				// value
				0xfb, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
			},
		},
		{
			Name:   "uint64",
			Struct: MustVariant(uint64(5)),
			Bytes: []byte{
				// variant encoding mask
				0x09,
				// value
				0x05, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			},
		},
		{
			Name:   "float32",
			Struct: MustVariant(float32(4.00067)),
			Bytes: []byte{
				// variant encoding mask
				0x0a,
				// value
				0x7d, 0x05, 0x80, 0x40,
			},
		},
		{
			Name:   "float64",
			Struct: MustVariant(float64(4.00067)),
			Bytes: []byte{
				// variant encoding mask
				0x0b,
				// value
				0x71, 0x5a, 0xf0, 0xa2, 0xaf, 0x0, 0x10, 0x40,
			},
		},
		{
			Name:   "string",
			Struct: MustVariant("abc"),
			Bytes: []byte{
				// variant encoding mask
				0x0c,
				// value
				0x03, 0x00, 0x00, 0x00,
				'a', 'b', 'c',
			},
		},
		{
			Name:   "DateTime",
			Struct: MustVariant(time.Date(2018, time.September, 17, 14, 28, 29, 112000000, time.UTC)),
			Bytes: []byte{
				// variant encoding mask
				0x0d,
				// value
				0x80, 0x3b, 0xe8, 0xb3, 0x92, 0x4e, 0xd4, 0x01,
			},
		},
		{
			Name:   "GUID",
			Struct: MustVariant(NewGUID("AAAABBBB-CCDD-EEFF-0101-0123456789AB")),
			Bytes: []byte{
				// variant encoding mask
				0x0e,
				// value
				0xbb, 0xbb, 0xaa, 0xaa, 0xdd, 0xcc, 0xff, 0xee,
				0xab, 0x89, 0x67, 0x45, 0x23, 0x01, 0x01, 0x01,
			},
		},
		{
			Name:   "ByteString",
			Struct: MustVariant([]byte{0x01, 0x02, 0x03}),
			Bytes: []byte{
				// variant encoding mask
				0x0f,
				// value
				0x03, 0x00, 0x00, 0x00,
				0x01, 0x02, 0x03,
			},
		},
		{
			Name:   "XmlElement",
			Struct: MustVariant(XmlElement("abc")),
			Bytes: []byte{
				// variant encoding mask
				0x10,
				// value
				0x03, 0x00, 0x00, 0x00,
				'a', 'b', 'c',
			},
		},
		{
			Name:   "NodeID",
			Struct: MustVariant(NewFourByteNodeID(1, 0xcafe)),
			Bytes: []byte{
				// variant encoding mask
				0x11,
				// node id mask
				0x01,
				// node id namespace
				0x01,
				// node id value
				0xfe, 0xca,
			},
		},
		{
			Name: "ExpandedNodeID",
			Struct: MustVariant(NewExpandedNodeID(
				true, false,
				NewTwoByteNodeID(0xff),
				"foobar", 0,
			)),
			Bytes: []byte{
				// variant encoding mask
				0x12,
				// expanded node id mask
				0x80,
				// expanded node id namespace
				0xff,
				// expanded node id value
				0x06, 0x00, 0x00, 0x00,
				0x66, 0x6f, 0x6f, 0x62, 0x61, 0x72,
			},
		},
		{
			Name:   "StatusCode",
			Struct: MustVariant(StatusCode(5)),
			Bytes: []byte{
				// variant encoding mask
				0x13,
				// value
				0x05, 0x00, 0x00, 0x00,
			},
		},
		{
			Name:   "QualifiedName",
			Struct: MustVariant(&QualifiedName{NamespaceIndex: 1, Name: "foobar"}),
			Bytes: []byte{
				// variant encoding mask
				0x14,
				// qualified name namespace
				0x01, 0x00,
				// qualified name
				0x06, 0x00, 0x00, 0x00,
				0x66, 0x6f, 0x6f, 0x62, 0x61, 0x72,
			},
		},
		{
			Name: "LocalizedText",
			Struct: MustVariant(&LocalizedText{
				EncodingMask: LocalizedTextText,
				Text:         "Gross value",
			}),
			Bytes: []byte{
				// variant encoding mask
				0x15,
				// localized text encoding mask
				0x02,
				// text length
				0x0b, 0x00, 0x00, 0x00,
				0x47, 0x72, 0x6f, 0x73, 0x73, 0x20, 0x76, 0x61, 0x6c, 0x75, 0x65,
			},
		},
		{
			Name: "ExtensionObjeject",
			Struct: MustVariant(NewExtensionObject(
				&AnonymousIdentityToken{PolicyID: "anonymous"},
			)),
			Bytes: []byte{
				// variant encoding mask
				0x16,
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
		{
			Name: "ExtensionObjeject - ServerStatusDataType",
			Struct: MustVariant(NewExtensionObject(
				&ServerStatusDataType{
					StartTime:   time.Date(2019, 3, 29, 19, 45, 3, 816525000, time.UTC), // Mar 29, 2019 20:45:03.816525000 CET
					CurrentTime: time.Date(2019, 3, 31, 8, 37, 14, 876798000, time.UTC), // Mar 31, 2019 10:37:14.876798000 CEST
					State:       ServerStateRunning,
					BuildInfo: &BuildInfo{
						ProductURI:       "http://open62541.org",
						ManufacturerName: "open62541",
						ProductName:      "open62541 OPC UA Server",
						SoftwareVersion:  "0.4.0-dev",
						BuildNumber:      "Mar  4 2019 15:22:43",
						BuildDate:        time.Time{},
					},
					SecondsTillShutdown: 0,
					ShutdownReason:      &LocalizedText{},
				},
			)),
			Bytes: []byte{
				// variant encoding mask
				0x16,
				// TypeID
				0x01, 0x00, 0x60, 0x03,
				// EncodingMask
				0x01,
				// Length
				0x86, 0x00, 0x00, 0x00,

				// ServerStatusDataType
				// StartTime
				0x02, 0xe1, 0x5b, 0xe7, 0x67, 0xe6, 0xd4, 0x01,
				// CurrentTime
				0xec, 0x62, 0x3c, 0xf1, 0x9c, 0xe7, 0xd4, 0x01,
				// State
				0x00, 0x00, 0x00, 0x00,

				// BuildInfo
				// ProductURI
				0x14, 0x00, 0x00, 0x00, // length
				0x68, 0x74, 0x74, 0x70, 0x3a, 0x2f, 0x2f, 0x6f,
				0x70, 0x65, 0x6e, 0x36, 0x32, 0x35, 0x34, 0x31,
				0x2e, 0x6f, 0x72, 0x67,
				// ManufacturerName
				0x09, 0x00, 0x00, 0x00, // length
				0x6f, 0x70, 0x65, 0x6e, 0x36, 0x32, 0x35, 0x34,
				0x31,
				// ProductName
				0x17, 0x00, 0x00, 0x00, //length
				0x6f, 0x70, 0x65, 0x6e, 0x36, 0x32, 0x35, 0x34,
				0x31, 0x20, 0x4f, 0x50, 0x43, 0x20, 0x55, 0x41,
				0x20, 0x53, 0x65, 0x72, 0x76, 0x65, 0x72,
				// SoftwareVersion
				0x09, 0x00, 0x00, 0x00, // length
				0x30, 0x2e, 0x34, 0x2e, 0x30, 0x2d, 0x64, 0x65,
				0x76,
				// BuildNumber
				0x14, 0x00, 0x00, 0x00, // length
				0x4d, 0x61, 0x72, 0x20, 0x20, 0x34, 0x20, 0x32,
				0x30, 0x31, 0x39, 0x20, 0x31, 0x35, 0x3a, 0x32,
				0x32, 0x3a, 0x34, 0x33,
				// BuildDate
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,

				// SecondsTillShutdown
				0x00, 0x00, 0x00, 0x00,
				// ShutdownReason
				0x00,
			},
		},
		{
			Name: "DataValue",
			Struct: MustVariant(&DataValue{
				EncodingMask: 0x01,
				Value:        MustVariant(float32(2.50025)),
			}),
			Bytes: []byte{
				// variant encoding mask
				0x17,
				// EncodingMask
				0x01,
				// Value
				0x0a,                   // type
				0x19, 0x04, 0x20, 0x40, // value
			},
		},
		{
			Name:   "Variant",
			Struct: MustVariant(MustVariant("abc")),
			Bytes: []byte{
				// variant encoding mask
				0x18,
				// inner variant encoding mask
				0x0c,
				// inner variant value
				0x03, 0x00, 0x00, 0x00,
				'a', 'b', 'c',
			},
		},
		{
			Name: "DiagnosticInfo",
			Struct: MustVariant(&DiagnosticInfo{
				EncodingMask: DiagnosticInfoSymbolicID,
				SymbolicID:   1,
			}),
			Bytes: []byte{
				// variant encoding mask
				0x19,
				0x01, 0x01, 0x00, 0x00, 0x00,
			},
		},
	}
	RunCodecTest(t, cases)
}

func TestVariantUnsupportedType(t *testing.T) {
	_, err := NewVariant(int(5))
	if err == nil {
		t.Fatal("got nil want err")
	}
}

func TestVariantBool(t *testing.T) {
	tests := []struct {
		v interface{}
		n bool
	}{
		{true, true},
		{"", false},
	}
	for _, tt := range tests {
		v, err := NewVariant(tt.v)
		if err != nil {
			t.Fatal(err)
		}
		if got, want := v.Bool(), tt.n; got != want {
			t.Fatalf("got %v want %v", got, want)
		}
	}
}

func TestVariantString(t *testing.T) {
	tests := []struct {
		v interface{}
		n string
	}{
		{"a", "a"},
		{&LocalizedText{Text: "a"}, "a"},
		{&QualifiedName{Name: "a"}, "a"},
		{int32(5), ""},
	}
	for _, tt := range tests {
		v, err := NewVariant(tt.v)
		if err != nil {
			t.Fatal(err)
		}
		if got, want := v.String(), tt.n; got != want {
			t.Fatalf("got %v want %v", got, want)
		}
	}
}

func TestVariantFloat(t *testing.T) {
	tests := []struct {
		v interface{}
		n float64
	}{
		{float32(5), 5},
		{float64(5), 5},
		{int32(5), 0},
	}
	for _, tt := range tests {
		v, err := NewVariant(tt.v)
		if err != nil {
			t.Fatal(err)
		}
		if got, want := v.Float(), tt.n; got != want {
			t.Fatalf("got %v want %v", got, want)
		}
	}
}

func TestVariantInt(t *testing.T) {
	tests := []struct {
		v interface{}
		n int64
	}{
		{int8(5), 5},
		{int16(5), 5},
		{int32(5), 5},
		{int64(5), 5},
		{"", 0},
	}
	for _, tt := range tests {
		v, err := NewVariant(tt.v)
		if err != nil {
			t.Fatal(err)
		}
		if got, want := v.Int(), tt.n; got != want {
			t.Fatalf("got %v want %v", got, want)
		}
	}
}

func TestVariantUint(t *testing.T) {
	tests := []struct {
		v interface{}
		n uint64
	}{
		{uint8(5), 5},
		{uint16(5), 5},
		{uint32(5), 5},
		{uint64(5), 5},
		{"", 0},
	}
	for _, tt := range tests {
		v, err := NewVariant(tt.v)
		if err != nil {
			t.Fatal(err)
		}
		if got, want := v.Uint(), tt.n; got != want {
			t.Fatalf("got %v want %v", got, want)
		}
	}
}

func TestVariantTime(t *testing.T) {
	tests := []struct {
		v interface{}
		n time.Time
	}{
		{time.Date(2019, 1, 1, 12, 13, 14, 0, time.UTC), time.Date(2019, 1, 1, 12, 13, 14, 0, time.UTC)},
		{"", time.Time{}},
	}
	for _, tt := range tests {
		v, err := NewVariant(tt.v)
		if err != nil {
			t.Fatal(err)
		}
		if got, want := v.Time(), tt.n; got != want {
			t.Fatalf("got %v want %v", got, want)
		}
	}
}
