// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uasc

import (
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
)

var symHdrCases = []struct {
	description string
	structured  *SymmetricSecurityHeader
	serialized  []byte
}{
	{
		"normal",
		NewSymmetricSecurityHeader(
			0x11223344,
			[]byte{0xde, 0xad, 0xbe, 0xef},
		),
		[]byte{
			// TokenID
			0x44, 0x33, 0x22, 0x11,
			// dummy Payload
			0xde, 0xad, 0xbe, 0xef,
		},
	}, {
		"no-payload",
		NewSymmetricSecurityHeader(
			0x11223344,
			nil,
		),
		[]byte{
			// TokenID
			0x44, 0x33, 0x22, 0x11,
		},
	},
}

func TestDecodeSymmetricSecurityHeader(t *testing.T) {
	// option to regard []T{} and []T{nil} as equal
	// https://godoc.org/github.com/google/go-cmp/cmp#example-Option--EqualEmpty
	alwaysEqual := cmp.Comparer(func(_, _ interface{}) bool { return true })
	opt := cmp.FilterValues(func(x, y interface{}) bool {
		vx, vy := reflect.ValueOf(x), reflect.ValueOf(y)
		return (vx.IsValid() && vy.IsValid() && vx.Type() == vy.Type()) &&
			(vx.Kind() == reflect.Slice) && (vx.Len() == 0 && vy.Len() == 0)
	}, alwaysEqual)

	for _, c := range symHdrCases {
		got, err := DecodeSymmetricSecurityHeader(c.serialized)
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(got, c.structured, opt); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}

func TestSerializeSymmetricSecurityHeader(t *testing.T) {
	for _, c := range symHdrCases {
		got, err := c.structured.Serialize()
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(got, c.serialized); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}

func TestSymmetricSecurityHeaderLen(t *testing.T) {
	for _, c := range symHdrCases {
		got := c.structured.Len()

		if diff := cmp.Diff(got, len(c.serialized)); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}
