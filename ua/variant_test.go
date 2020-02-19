// Copyright 2018-2020 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package ua

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/gopcua/opcua/errors"

	"github.com/pascaldekloe/goe/verify"
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
			Struct: MustVariant(NewGUID("72962B91-FA75-4AE6-8D28-B404DC7DAF63")),
			Bytes: []byte{
				// variant encoding mask
				0x0e,
				// data1 (inverse order)
				0x91, 0x2b, 0x96, 0x72,
				// data2 (inverse order)
				0x75, 0xfa,
				// data3 (inverse order)
				0xe6, 0x4a,
				// data4 (same order)
				0x8d, 0x28, 0xb4, 0x04, 0xdc, 0x7d, 0xaf, 0x63,
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
			Name:   "XMLElement",
			Struct: MustVariant(XMLElement("abc")),
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
				// mask
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
				// mask
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
				// mask
				0x01,
				// value
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
		{
			Name:   "[]uint32",
			Struct: MustVariant([]uint32{1, 2, 3}),
			Bytes: []byte{
				// variant encoding mask
				0x87,
				// array length
				0x03, 0x00, 0x00, 0x00,
				// array values
				0x01, 0x00, 0x00, 0x00,
				0x02, 0x00, 0x00, 0x00,
				0x03, 0x00, 0x00, 0x00,
			},
		},
		{
			Name:   "[3][2]uint32",
			Struct: MustVariant([][]uint32{{1, 1}, {2, 2}, {3, 3}}),
			Bytes: []byte{
				// variant encoding mask
				0xc7,
				// array length
				0x06, 0x00, 0x00, 0x00,
				// array values
				0x01, 0x00, 0x00, 0x00,
				0x01, 0x00, 0x00, 0x00,
				0x02, 0x00, 0x00, 0x00,
				0x02, 0x00, 0x00, 0x00,
				0x03, 0x00, 0x00, 0x00,
				0x03, 0x00, 0x00, 0x00,
				// array dimensions length
				0x02, 0x00, 0x00, 0x00,
				// array dimensions
				0x03, 0x00, 0x00, 0x00,
				0x02, 0x00, 0x00, 0x00,
			},
		},
		{
			Name: "[3][2][1]uint32",
			Struct: MustVariant([][][]uint32{
				{{1}, {1}},
				{{2}, {2}},
				{{3}, {3}},
			}),
			Bytes: []byte{
				// variant encoding mask
				0xc7,
				// array length
				0x06, 0x00, 0x00, 0x00,
				// array values
				0x01, 0x00, 0x00, 0x00,
				0x01, 0x00, 0x00, 0x00,
				0x02, 0x00, 0x00, 0x00,
				0x02, 0x00, 0x00, 0x00,
				0x03, 0x00, 0x00, 0x00,
				0x03, 0x00, 0x00, 0x00,
				// array dimensions length
				0x03, 0x00, 0x00, 0x00,
				// array dimensions
				0x03, 0x00, 0x00, 0x00,
				0x02, 0x00, 0x00, 0x00,
				0x01, 0x00, 0x00, 0x00,
			},
		},
		{
			Name: "[0][0][0]uint32",
			Struct: MustVariant([][][]uint32{
				{{}, {}},
				{{}, {}},
				{{}, {}},
			}),
			Bytes: []byte{
				// variant encoding mask
				0xc7,
				// array length
				0x00, 0x00, 0x00, 0x00,
				// array values
				// array dimensions length
				0x03, 0x00, 0x00, 0x00,
				// array dimensions
				0x03, 0x00, 0x00, 0x00,
				0x02, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00,
			},
		},
	}
	RunCodecTest(t, cases)
}

func TestMustVariant(t *testing.T) {
	t.Run("int", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Fatalf("MustVariant(int) did not panic")
			}
		}()
		MustVariant(int(5))
	})
	t.Run("uint", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Fatalf("MustVariant(uint) did not panic")
			}
		}()
		MustVariant(uint(5))
	})
}

func TestArray(t *testing.T) {
	t.Run("one-dimension", func(t *testing.T) {
		v := MustVariant([]uint32{1, 2, 3})
		if got, want := v.ArrayLength(), int32(3); got != want {
			t.Fatalf("got length %d want %d", got, want)
		}
		if got, want := v.EncodingMask(), byte(TypeIDUint32|VariantArrayValues); got != want {
			t.Fatalf("got mask %d want %d", got, want)
		}
		verify.Values(t, "", v.ArrayDimensions(), []int32{})
	})
	t.Run("multi-dimension", func(t *testing.T) {
		v := MustVariant([][]uint32{{1, 1}, {2, 2}, {3, 3}})
		if got, want := v.ArrayLength(), int32(6); got != want {
			t.Fatalf("got length %d want %d", got, want)
		}
		if got, want := v.EncodingMask(), byte(TypeIDUint32|VariantArrayValues|VariantArrayDimensions); got != want {
			t.Fatalf("got mask %d want %d", got, want)
		}
		verify.Values(t, "", v.ArrayDimensions(), []int32{3, 2})
	})
	t.Run("unbalanced", func(t *testing.T) {
		b := []byte{
			// variant encoding mask
			0xc7,
			// array length
			0x03, 0x00, 0x00, 0x00,
			// array values
			0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00,
			// array dimensions length
			0x02, 0x00, 0x00, 0x00,
			// array dimensions
			0x03, 0x00, 0x00, 0x00,
			0x02, 0x00, 0x00, 0x00,
		}

		_, err := Decode(b, MustVariant([]uint32{0}))
		if got, want := err, errUnbalancedSlice; !errors.Equal(got, want) {
			t.Fatalf("got error %#v want %#v", got, want)
		}
	})
	t.Run("length negative", func(t *testing.T) {
		b := []byte{
			// variant encoding mask
			0x87,
			// array length
			0xff, 0xff, 0xff, 0xff, // -1
		}

		_, err := Decode(b, MustVariant([]uint32{0}))
		if got, want := err, StatusBadEncodingLimitsExceeded; !errors.Equal(got, want) {
			t.Fatalf("got error %#v want %#v", got, want)
		}
	})
	t.Run("length too big", func(t *testing.T) {
		b := []byte{
			// variant encoding mask
			0x87,
			// array length
			0xff, 0xff, 0x01, 0x00,
			// array values
			0x00, 0x00, 0x00, 0x00,
		}

		_, err := Decode(b, MustVariant([]uint32{0}))
		if got, want := err, StatusBadEncodingLimitsExceeded; !errors.Equal(got, want) {
			t.Fatalf("got error %v want %v", err, StatusBadEncodingLimitsExceeded)
		}
	})
	t.Run("dimensions length negative", func(t *testing.T) {
		b := []byte{
			// variant encoding mask
			0xc7,
			// array length
			0x02, 0x00, 0x00, 0x00,
			// array values
			0x01, 0x00, 0x00, 0x00,
			0x01, 0x00, 0x00, 0x00,
			// array dimesions length
			0xff, 0xff, 0xff, 0xff, // -1
			// array dimesions
			0x01, 0x00, 0x00, 0x00,
			0x01, 0x00, 0x00, 0x00,
		}

		_, err := Decode(b, MustVariant([]uint32{0}))
		if got, want := err, StatusBadEncodingLimitsExceeded; !errors.Equal(got, want) {
			t.Fatalf("got error %#v want %#v", got, want)
		}
	})
	t.Run("dimensions negative", func(t *testing.T) {
		b := []byte{
			// variant encoding mask
			0xc7,
			// array length
			0x02, 0x00, 0x00, 0x00,
			// array values
			0x01, 0x00, 0x00, 0x00,
			0x01, 0x00, 0x00, 0x00,
			// array dimesions length
			0x02, 0x00, 0x00, 0x00,
			// array dimesions
			0x01, 0x00, 0x00, 0x00,
			0xff, 0xff, 0xff, 0xff, // -1
		}

		_, err := Decode(b, MustVariant([]uint32{0}))
		if got, want := err, StatusBadEncodingLimitsExceeded; !errors.Equal(got, want) {
			t.Fatalf("got error %#v want %#v", got, want)
		}
	})
}

func TestSet(t *testing.T) {
	tests := []struct {
		v   interface{}
		va  *Variant
		err error
	}{
		{
			v: []byte{0xca, 0xfe},
			va: &Variant{
				mask:  byte(TypeIDByteString),
				value: []byte{0xca, 0xfe},
			},
		},
		{
			v: [][]byte{{0xca, 0xfe}, {0xaf, 0xfe}},
			va: &Variant{
				mask:        byte(VariantArrayValues | TypeIDByteString),
				arrayLength: 2,
				value:       [][]byte{{0xca, 0xfe}, {0xaf, 0xfe}},
			},
		},
		{
			v: int32(5),
			va: &Variant{
				mask:  byte(TypeIDInt32),
				value: int32(5),
			},
		},
		{
			v: []int32{5},
			va: &Variant{
				mask:        byte(VariantArrayValues | TypeIDInt32),
				arrayLength: 1,
				value:       []int32{5},
			},
		},
		{
			v: [][]int32{{5}, {5}, {5}},
			va: &Variant{
				mask:                  byte(VariantArrayDimensions | VariantArrayValues | TypeIDInt32),
				arrayLength:           3,
				arrayDimensionsLength: 2,
				arrayDimensions:       []int32{3, 1},
				value:                 [][]int32{{5}, {5}, {5}},
			},
		},
		{
			v: [][][]int32{
				{{}, {}},
				{{}, {}},
				{{}, {}},
			},
			va: &Variant{
				mask:                  byte(VariantArrayDimensions | VariantArrayValues | TypeIDInt32),
				arrayLength:           0,
				arrayDimensionsLength: 3,
				arrayDimensions:       []int32{3, 2, 0},
				value: [][][]int32{
					{{}, {}},
					{{}, {}},
					{{}, {}},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%T", tt.v), func(t *testing.T) {
			va, err := NewVariant(tt.v)
			if got, want := err, tt.err; got != want {
				t.Fatalf("got error %v want %v", got, want)
			}
			verify.Values(t, "variant", va, tt.va)
		})
	}
}

func TestSliceDim(t *testing.T) {
	tests := []struct {
		v   interface{}
		et  reflect.Type
		dim []int32
		len int32
		err error
	}{
		// happy flows
		{
			v:   "a",
			et:  reflect.TypeOf(""),
			dim: nil,
			len: 1,
		},
		{
			v:   1,
			et:  reflect.TypeOf(int(0)),
			dim: nil,
			len: 1,
		},
		{
			v:   []int{},
			et:  reflect.TypeOf(int(0)),
			dim: []int32{0},
			len: 0,
		},
		{
			v:   []int{1, 2, 3},
			et:  reflect.TypeOf(int(0)),
			dim: []int32{3},
			len: 3,
		},
		{
			v:   [][]int{{1, 1}, {2, 2}, {3, 3}},
			et:  reflect.TypeOf(int(0)),
			dim: []int32{3, 2},
			len: 6,
		},
		{
			v:   [][]int{{}, {}, {}},
			et:  reflect.TypeOf(int(0)),
			dim: []int32{3, 0},
			len: 0,
		},

		// error flows
		{
			v:   [][]int{{1, 1}, {2, 2, 2}, {3}},
			err: errUnbalancedSlice,
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%T", tt.v), func(t *testing.T) {
			et, dim, len, err := sliceDim(reflect.ValueOf(tt.v))
			if got, want := err, tt.err; got != want {
				t.Fatalf("got error %v want %v", got, want)
			}
			if got, want := et, tt.et; got != want {
				t.Fatalf("got type %v want %v", got, want)
			}
			if got, want := dim, tt.dim; !reflect.DeepEqual(got, want) {
				t.Fatalf("got dimensions %v want %v", got, want)
			}
			if got, want := len, tt.len; got != want {
				t.Fatalf("got len %v want %v", got, want)
			}
		})
	}
}

func TestVariantUnsupportedType(t *testing.T) {
	tests := []interface{}{int(5), uint(5)}
	for _, v := range tests {
		t.Run(fmt.Sprintf("%T", v), func(t *testing.T) {
			if _, err := NewVariant(v); err == nil {
				t.Fatal("got nil want err")
			}
		})
	}
}

func TestVariantValueMethod(t *testing.T) {
	if got, want := MustVariant(int32(5)).Value().(int32), int32(5); got != want {
		t.Fatalf("got %d want %d", got, want)
	}
}

func TestVariantValueHelpers(t *testing.T) {
	tests := []struct {
		v    interface{}
		want interface{}
		fn   func(v *Variant) interface{}
	}{
		// bool
		{
			v:    int32(5),
			want: false,
			fn:   func(v *Variant) interface{} { return v.Bool() },
		},
		{
			v:    false,
			want: false,
			fn:   func(v *Variant) interface{} { return v.Bool() },
		},
		{
			v:    true,
			want: true,
			fn:   func(v *Variant) interface{} { return v.Bool() },
		},

		// string
		{
			v:    false,
			want: "",
			fn:   func(v *Variant) interface{} { return v.String() },
		},
		{
			v:    "a",
			want: "a",
			fn:   func(v *Variant) interface{} { return v.String() },
		},
		{
			v:    XMLElement("a"),
			want: "a",
			fn:   func(v *Variant) interface{} { return v.String() },
		},
		{
			v:    &LocalizedText{Text: "a"},
			want: "a",
			fn:   func(v *Variant) interface{} { return v.String() },
		},
		{
			v:    &QualifiedName{Name: "a"},
			want: "a",
			fn:   func(v *Variant) interface{} { return v.String() },
		},

		// float
		{
			v:    false,
			want: float64(0),
			fn:   func(v *Variant) interface{} { return v.Float() },
		},
		{
			v:    float32(5),
			want: float64(5),
			fn:   func(v *Variant) interface{} { return v.Float() },
		},
		{
			v:    float64(5),
			want: float64(5),
			fn:   func(v *Variant) interface{} { return v.Float() },
		},

		// int
		{
			v:    false,
			want: int64(0),
			fn:   func(v *Variant) interface{} { return v.Int() },
		},
		{
			v:    int8(5),
			want: int64(5),
			fn:   func(v *Variant) interface{} { return v.Int() },
		},
		{
			v:    int16(5),
			want: int64(5),
			fn:   func(v *Variant) interface{} { return v.Int() },
		},
		{
			v:    int32(5),
			want: int64(5),
			fn:   func(v *Variant) interface{} { return v.Int() },
		},
		{
			v:    int64(5),
			want: int64(5),
			fn:   func(v *Variant) interface{} { return v.Int() },
		},

		// uint
		{
			v:    false,
			want: uint64(0),
			fn:   func(v *Variant) interface{} { return v.Uint() },
		},
		{
			v:    uint8(5),
			want: uint64(5),
			fn:   func(v *Variant) interface{} { return v.Uint() },
		},
		{
			v:    uint16(5),
			want: uint64(5),
			fn:   func(v *Variant) interface{} { return v.Uint() },
		},
		{
			v:    uint32(5),
			want: uint64(5),
			fn:   func(v *Variant) interface{} { return v.Uint() },
		},
		{
			v:    uint64(5),
			want: uint64(5),
			fn:   func(v *Variant) interface{} { return v.Uint() },
		},

		// ByteString
		{
			v:    false,
			want: ([]byte)(nil),
			fn:   func(v *Variant) interface{} { return v.ByteString() },
		},
		{
			v:    []byte("abc"),
			want: []byte("abc"),
			fn:   func(v *Variant) interface{} { return v.ByteString() },
		},

		// DataValue
		{
			v:    false,
			want: (*DataValue)(nil),
			fn:   func(v *Variant) interface{} { return v.DataValue() },
		},
		{
			v:    &DataValue{Status: StatusBad},
			want: &DataValue{Status: StatusBad},
			fn:   func(v *Variant) interface{} { return v.DataValue() },
		},

		// DiagnosticInfo
		{
			v:    false,
			want: (*DiagnosticInfo)(nil),
			fn:   func(v *Variant) interface{} { return v.DiagnosticInfo() },
		},
		{
			v:    &DiagnosticInfo{SymbolicID: 5},
			want: &DiagnosticInfo{SymbolicID: 5},
			fn:   func(v *Variant) interface{} { return v.DiagnosticInfo() },
		},

		// ExpandedNodeID
		{
			v:    false,
			want: (*ExpandedNodeID)(nil),
			fn:   func(v *Variant) interface{} { return v.ExpandedNodeID() },
		},
		{
			v:    &ExpandedNodeID{NamespaceURI: "abc"},
			want: &ExpandedNodeID{NamespaceURI: "abc"},
			fn:   func(v *Variant) interface{} { return v.ExpandedNodeID() },
		},

		// ExtensionObject
		{
			v:    false,
			want: (*ExtensionObject)(nil),
			fn:   func(v *Variant) interface{} { return v.ExtensionObject() },
		},
		{
			v:    &ExtensionObject{Value: "abc"},
			want: &ExtensionObject{Value: "abc"},
			fn:   func(v *Variant) interface{} { return v.ExtensionObject() },
		},

		// GUID
		{
			v:    false,
			want: (*GUID)(nil),
			fn:   func(v *Variant) interface{} { return v.GUID() },
		},
		{
			v:    NewGUID("abc"),
			want: NewGUID("abc"),
			fn:   func(v *Variant) interface{} { return v.GUID() },
		},

		// LocalizedText
		{
			v:    false,
			want: (*LocalizedText)(nil),
			fn:   func(v *Variant) interface{} { return v.LocalizedText() },
		},
		{
			v:    &LocalizedText{Text: "abc"},
			want: &LocalizedText{Text: "abc"},
			fn:   func(v *Variant) interface{} { return v.LocalizedText() },
		},

		// NodeID
		{
			v:    false,
			want: (*NodeID)(nil),
			fn:   func(v *Variant) interface{} { return v.NodeID() },
		},
		{
			v:    NewFourByteNodeID(1, 2),
			want: NewFourByteNodeID(1, 2),
			fn:   func(v *Variant) interface{} { return v.NodeID() },
		},

		// QualifiedName
		{
			v:    false,
			want: (*QualifiedName)(nil),
			fn:   func(v *Variant) interface{} { return v.QualifiedName() },
		},
		{
			v:    &QualifiedName{Name: "a"},
			want: &QualifiedName{Name: "a"},
			fn:   func(v *Variant) interface{} { return v.QualifiedName() },
		},

		// StatusCode
		{
			v:    false,
			want: StatusBadTypeMismatch,
			fn:   func(v *Variant) interface{} { return v.StatusCode() },
		},
		{
			v:    StatusBad,
			want: StatusBad,
			fn:   func(v *Variant) interface{} { return v.StatusCode() },
		},

		// time.Time
		{
			v:    false,
			want: time.Time{},
			fn:   func(v *Variant) interface{} { return v.Time() },
		},
		{
			v:    time.Date(2019, 1, 1, 12, 13, 14, 0, time.UTC),
			want: time.Date(2019, 1, 1, 12, 13, 14, 0, time.UTC),
			fn:   func(v *Variant) interface{} { return v.Time() },
		},

		// Variant
		{
			v:    false,
			want: (*Variant)(nil),
			fn:   func(v *Variant) interface{} { return v.Variant() },
		},
		{
			v:    MustVariant("abc"),
			want: MustVariant("abc"),
			fn:   func(v *Variant) interface{} { return v.Variant() },
		},

		// XMLElement
		{
			v:    false,
			want: XMLElement(""),
			fn:   func(v *Variant) interface{} { return v.XMLElement() },
		},
		{
			v:    XMLElement("a"),
			want: XMLElement("a"),
			fn:   func(v *Variant) interface{} { return v.XMLElement() },
		},

		// []string
		{
			v:    []string{"a", "b", "c"},
			want: "",
			fn:   func(v *Variant) interface{} { return v.String() },
		},

		// []bool
		{
			v:    []bool{true, true, true},
			want: false,
			fn:   func(v *Variant) interface{} { return v.Bool() },
		},

		// []float64
		{
			v:    []float64{1, 2, 3},
			want: float64(0),
			fn:   func(v *Variant) interface{} { return v.Float() },
		},

		// []int64
		{
			v:    []int64{1, 2, 3},
			want: int64(0),
			fn:   func(v *Variant) interface{} { return v.Int() },
		},

		// []uint64
		{
			v:    []uint64{1, 2, 3},
			want: uint64(0),
			fn:   func(v *Variant) interface{} { return v.Uint() },
		},

		// [][]byte
		{
			v:    [][]byte{{'x', 'y', 'z'}},
			want: ([]byte)(nil),
			fn:   func(v *Variant) interface{} { return v.ByteString() },
		},

		// []*DataValue
		{
			v:    []*DataValue{{Status: StatusBad}},
			want: (*DataValue)(nil),
			fn:   func(v *Variant) interface{} { return v.DataValue() },
		},

		// []*DiagnosticInfo
		{
			v:    []*DiagnosticInfo{{AdditionalInfo: "nop"}},
			want: (*DiagnosticInfo)(nil),
			fn:   func(v *Variant) interface{} { return v.DiagnosticInfo() },
		},

		// []*ExpandedNodeID
		{
			v:    []*ExpandedNodeID{{NamespaceURI: "abc"}},
			want: (*ExpandedNodeID)(nil),
			fn:   func(v *Variant) interface{} { return v.ExpandedNodeID() },
		},

		// []*ExtensionObject
		{
			v:    []*ExtensionObject{{Value: "abc"}},
			want: (*ExtensionObject)(nil),
			fn:   func(v *Variant) interface{} { return v.ExtensionObject() },
		},

		// []*GUID
		{
			v:    []*GUID{NewGUID("abcd")},
			want: (*GUID)(nil),
			fn:   func(v *Variant) interface{} { return v.GUID() },
		},

		// []*LocalizedText
		{
			v:    []*LocalizedText{{Text: "abc"}},
			want: (*LocalizedText)(nil),
			fn:   func(v *Variant) interface{} { return v.LocalizedText() },
		},

		// []*NodeID
		{
			v:    []*NodeID{NewFourByteNodeID(1, 2)},
			want: (*NodeID)(nil),
			fn:   func(v *Variant) interface{} { return v.NodeID() },
		},

		// []*QualifiedName
		{
			v:    []*QualifiedName{{Name: "a"}},
			want: (*QualifiedName)(nil),
			fn:   func(v *Variant) interface{} { return v.QualifiedName() },
		},

		// []*StatusCode
		{
			v:    []StatusCode{StatusOK, StatusBad},
			want: StatusBadTypeMismatch,
			fn:   func(v *Variant) interface{} { return v.StatusCode() },
		},

		// []time.Time
		{
			v:    []time.Time{time.Date(2019, 1, 1, 12, 13, 14, 0, time.UTC)},
			want: time.Time{},
			fn:   func(v *Variant) interface{} { return v.Time() },
		},

		// []*Variant
		{
			v:    []*Variant{MustVariant("abc")},
			want: (*Variant)(nil),
			fn:   func(v *Variant) interface{} { return v.Variant() },
		},

		// []XMLElement
		{
			v:    []XMLElement{XMLElement("a")},
			want: XMLElement(""),
			fn:   func(v *Variant) interface{} { return v.XMLElement() },
		},
	}
	for i, tt := range tests {
		name := fmt.Sprintf("test-%d %T -> %T", i, tt.v, tt.want)
		t.Run(name, func(t *testing.T) {
			verify.Values(t, "", tt.fn(MustVariant(tt.v)), tt.want)
		})
	}
}

func TestDecodeInvalidType(t *testing.T) {
	b := []byte{
		// variant encoding mask
		0x20, // invalid type id
	}

	v := &Variant{}
	_, err := v.Decode(b)
	if got, want := err, errors.New("invalid type id: 32"); !errors.Equal(got, want) {
		t.Fatalf("got error %s want %s", got, want)
	}
}
