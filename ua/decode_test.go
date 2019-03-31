// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package ua

import (
	"bytes"
	"reflect"
	"testing"
	"time"
)

type A struct {
	V uint32
}

type B struct {
	A *A
	S []*A
}

func TestCodec(t *testing.T) {
	tests := []struct {
		name string
		v    interface{}
		b    []byte
	}{
		{
			name: "bool:true",
			v:    &struct{ V bool }{true},
			b:    []byte{0x01},
		},
		{
			name: "bool:false",
			v:    &struct{ V bool }{false},
			b:    []byte{0x00},
		},
		{
			name: "int8",
			v:    &struct{ V int8 }{-5},
			b:    []byte{0xfb},
		},
		{
			name: "uint8",
			v:    &struct{ V uint8 }{5},
			b:    []byte{0x05},
		},
		{
			name: "int16",
			v:    &struct{ V int16 }{-5},
			b:    []byte{0xfb, 0xff},
		},
		{
			name: "uint16",
			v:    &struct{ V uint16 }{0x1234},
			b:    []byte{0x34, 0x12},
		},
		{
			name: "int32",
			v:    &struct{ V int32 }{-5},
			b:    []byte{0xfb, 0xff, 0xff, 0xff},
		},
		{
			name: "uint32",
			v:    &struct{ V uint32 }{0x12345678},
			b:    []byte{0x78, 0x56, 0x34, 0x12},
		},
		{
			name: "int64",
			v:    &struct{ V int64 }{-5},
			b:    []byte{0xfb, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
		},
		{
			name: "uint64",
			v:    &struct{ V uint64 }{0x1234567890abcdef},
			b:    []byte{0xef, 0xcd, 0xab, 0x90, 0x78, 0x56, 0x34, 0x12},
		},
		{
			name: "float32",
			v:    &struct{ V float32 }{1.234},
			b:    []byte{0xb6, 0xf3, 0x9d, 0x3f},
		},
		// todo(fs): this test will fail since NaN is defined as f != f
		// todo(fs): need to refactor the test
		// {
		// 	name: "float32-NaN",
		// 	v:    &struct{ V float32 }{float32(math.NaN())},
		// 	b:    []byte{0x00, 0x00, 0xc0, 0xff},
		// },
		{
			name: "float64",
			v:    &struct{ V float64 }{-1.234},
			b:    []byte{0x58, 0x39, 0xb4, 0xc8, 0x76, 0xbe, 0xf3, 0xbf},
		},
		// todo(fs): this test will fail since NaN is defined as f != f
		// todo(fs): need to refactor the test
		// {
		// 	name: "float64-NaN",
		// 	v:    &struct{ V float64 }{math.NaN()},
		// 	b:    []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xf8, 0xff},
		// },
		{
			name: "[]uint32",
			v:    &struct{ V []uint32 }{[]uint32{0x1234, 0x4567}},
			b: []byte{
				// length
				0x02, 0x00, 0x00, 0x00,
				// elem 1
				0x34, 0x12, 0x00, 0x00,
				// elem 2
				0x67, 0x45, 0x00, 0x00,
			},
		},
		{
			name: "string",
			v:    &struct{ V string }{"abc"},
			b: []byte{
				// length
				0x03, 0x00, 0x00, 0x00,

				// value
				'a', 'b', 'c',
			},
		},
		{
			name: "empty string",
			v:    &struct{ V string }{""},
			b: []byte{
				// length
				0xff, 0xff, 0xff, 0xff,
			},
		},
		{
			name: "ByteString",
			v:    &struct{ V []byte }{[]byte{0x01, 0x02, 0x03}},
			b: []byte{
				// length
				0x03, 0x00, 0x00, 0x00,

				// value
				0x01, 0x02, 0x03,
			},
		},
		{
			name: "DateTime",
			v:    &struct{ V time.Time }{time.Date(2018, time.August, 10, 23, 0, 0, 0, time.UTC)},
			b:    []byte{0x00, 0x98, 0x67, 0xdd, 0xfd, 0x30, 0xd4, 0x01},
		},
		{
			name: "DateTimeZero",
			v:    &struct{ V time.Time }{time.Time{}},
			b:    []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		},
		{
			name: "[]uint32==nil",
			v:    &struct{ V []uint32 }{},
			b: []byte{
				// length
				0xff, 0xff, 0xff, 0xff,
			},
		},
		{
			name: "[]uint32{}",
			v:    &struct{ V []uint32 }{[]uint32{}},
			b: []byte{
				// length
				0x00, 0x00, 0x00, 0x00,
			},
		},
		{
			name: "[]uint32{1,2,3}",
			v:    &struct{ V []uint32 }{[]uint32{1, 2, 3}},
			b: []byte{
				// length
				0x03, 0x00, 0x00, 0x00,
				// values
				0x01, 0x00, 0x00, 0x00,
				0x02, 0x00, 0x00, 0x00,
				0x03, 0x00, 0x00, 0x00,
			},
		},
		{
			name: "[]*A",
			v: &struct{ V []*A }{
				[]*A{
					&A{1},
					&A{2},
					&A{3},
				},
			},
			b: []byte{
				// length
				0x03, 0x00, 0x00, 0x00,
				// values
				0x01, 0x00, 0x00, 0x00,
				0x02, 0x00, 0x00, 0x00,
				0x03, 0x00, 0x00, 0x00,
			},
		},
		{
			name: "&B.A",
			v: &B{
				A: &A{
					V: 0x1234,
				},
			},
			b: []byte{
				// B.A.N
				0x34, 0x12, 0x00, 0x00,
				// B.A.S == nil
				0xff, 0xff, 0xff, 0xff,
			},
		},
		{
			name: "&B.[]A",
			v: &B{
				A: &A{V: 0x7890},
				S: []*A{
					&A{V: 0x1234},
					&A{V: 0x4567},
				},
			},
			b: []byte{
				// B.A.N
				0x90, 0x78, 0x00, 0x00,
				// len(B.A.S)
				0x02, 0x00, 0x00, 0x00,
				// B.A.S[0]
				0x34, 0x12, 0x00, 0x00,
				// B.A.S[1]
				0x67, 0x45, 0x00, 0x00,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if reflect.ValueOf(tt.v).Kind() != reflect.Ptr {
				t.Fatalf("%T is not a pointer", tt.v)
			}

			t.Run("decode", func(t *testing.T) {
				// create a new instance of the same type as tt.v
				// v then contains a pointer to the new instance
				typ := reflect.ValueOf(tt.v).Type()
				v := reflect.New(typ.Elem())

				if _, err := Decode(tt.b, v.Interface()); err != nil {
					t.Fatal(err)
				}

				if got, want := v.Interface(), tt.v; !reflect.DeepEqual(got, want) {
					t.Fatalf("got %#v, want %#v", got, want)
				}
			})
			t.Run("encode", func(t *testing.T) {
				b, err := Encode(tt.v)
				if err != nil {
					t.Fatal(err)
				}
				if got, want := b, tt.b; !bytes.Equal(got, want) {
					t.Fatalf("\ngot  %#v\nwant %#v", got, want)
				}
			})
		})
	}
}
