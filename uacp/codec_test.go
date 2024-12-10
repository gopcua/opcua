// Copyright 2018-2020 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

// This file is copied to packages ua, uacp and uasc to break an import cycle.

package uacp

import (
	"reflect"
	"testing"

	"github.com/gopcua/opcua/ua"
	"github.com/stretchr/testify/require"
)

// CodecTestCase describes a test case for a encoding and decoding an
// object from bytes.
type CodecTestCase struct {
	Name   string
	Struct interface{}
	Bytes  []byte
}

// RunCodecTest tests encoding, decoding and length calclulation for the given
// object.
func RunCodecTest(t *testing.T, cases []CodecTestCase) {
	t.Helper()

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			t.Run("decode", func(t *testing.T) {
				// create a new instance of the same type as c.Struct
				typ := reflect.ValueOf(c.Struct).Type()
				var v reflect.Value
				switch typ.Kind() {
				case reflect.Ptr:
					v = reflect.New(typ.Elem()) // typ: *struct, v: *struct
				case reflect.Slice:
					v = reflect.New(typ) // typ: []x, v: *[]x
				default:
					require.Fail(t, "%T is not a pointer or a slice", c.Struct)
				}

				_, err := ua.Decode(c.Bytes, v.Interface())
				require.NoError(t, err, "Decode failed")

				// if v is a *[]x we need to dereference it before comparing it.
				if typ.Kind() == reflect.Slice {
					v = v.Elem()
				}
				require.Equal(t, c.Struct, v.Interface(), "Decoded payload not equal")
			})

			t.Run("encode", func(t *testing.T) {
				b, err := ua.Encode(c.Struct)
				require.NoError(t, err, "Encode failed")
				require.Equal(t, c.Bytes, b, "Encoded payload not equal")
			})
		})
	}
}
