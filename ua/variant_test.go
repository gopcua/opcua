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
			Struct: MustVariant(NewQualifiedName(1, "foobar")),
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
			Name:   "LocalizedText",
			Struct: MustVariant(&LocalizedText{Text: "Gross value"}),
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
				NewAnonymousIdentityToken("anonymous"),
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
			Struct: MustVariant(NewDiagnosticInfo(
				true, false, false, false, false, false, false,
				1, 0, 0, 0, "", 0, nil,
			)),
			Bytes: []byte{
				// variant encoding mask
				0x19,
				0x01, 0x01, 0x00, 0x00, 0x00,
			},
		},
	}
	RunCodecTest(t, cases)
}
