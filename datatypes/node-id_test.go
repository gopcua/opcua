// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package datatypes

import (
	"errors"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/wmnsk/gopcua/utils/codectest"
)

func TestNodeID(t *testing.T) {
	cases := []codectest.Case{
		{
			Name:   "TwoByte",
			Struct: NewTwoByteNodeID(0xff),
			Bytes:  []byte{0x00, 0xff},
		},
		{
			Name:   "FourByte",
			Struct: NewFourByteNodeID(0, 0xcafe),
			Bytes:  []byte{0x01, 0x00, 0xfe, 0xca},
		},
		{
			Name:   "Numeric",
			Struct: NewNumericNodeID(10, 0xdeadbeef),
			Bytes:  []byte{0x02, 0x0a, 0x00, 0xef, 0xbe, 0xad, 0xde},
		},
		{
			Name:   "String",
			Struct: NewStringNodeID(255, "foobar"),
			Bytes: []byte{
				0x03, 0xff, 0x00, 0x06, 0x00, 0x00, 0x00, 0x66,
				0x6f, 0x6f, 0x62, 0x61, 0x72,
			},
		},
		{
			Name:   "GUID",
			Struct: NewGUIDNodeID(4660, "AAAABBBB-CCDD-EEFF-0101-0123456789AB"),
			Bytes: []byte{
				0x04, 0x34, 0x12,
				0xbb, 0xbb, 0xaa, 0xaa, 0xdd, 0xcc, 0xff, 0xee,
				0xab, 0x89, 0x67, 0x45, 0x23, 0x01, 0x01, 0x01,
			},
		},
		{
			Name:   "Opaque",
			Struct: NewOpaqueNodeID(32768, []byte{0xde, 0xad, 0xbe, 0xef}),
			Bytes: []byte{
				0x05, 0x00, 0x80, 0x04, 0x00, 0x00, 0x00, 0xde,
				0xad, 0xbe, 0xef,
			},
		},
	}
	codectest.Run(t, cases, func(b []byte) (codectest.S, error) {
		return DecodeNodeID(b)
	})
}

func TestParseNodeID(t *testing.T) {
	cases := []struct {
		s   string
		n   NodeID
		err error
	}{
		// happy flows
		{s: "", n: NewTwoByteNodeID(0)},
		{s: "ns=0;i=1", n: NewTwoByteNodeID(1)},
		{s: "ns=1;i=2", n: NewFourByteNodeID(1, 2)},
		{s: "ns=256;i=2", n: NewNumericNodeID(256, 2)},
		{s: "ns=1;i=65536", n: NewNumericNodeID(1, 65536)},
		{s: "ns=65535;i=65536", n: NewNumericNodeID(65535, 65536)},
		{s: "ns=1;g=5eac051c-c313-43d7-b790-24aa2c3cfd37", n: NewGUIDNodeID(1, "5eac051c-c313-43d7-b790-24aa2c3cfd37")},
		{s: "ns=1;b=YWJj", n: NewOpaqueNodeID(1, []byte{'a', 'b', 'c'})},
		{s: "ns=1;s=a", n: NewStringNodeID(1, "a")},
		{s: "ns=1;a", n: NewStringNodeID(1, "a")},

		// error flows
		{s: "i=1", err: errors.New("invalid node id: i=1")},
		{s: "nsu=abc;i=1", err: errors.New("namespace urls are not supported: nsu=abc;i=1")},
		{s: "ns=65536;i=1", err: errors.New("namespace id out of range (0..65535): ns=65536;i=1")},
		{s: "ns=abc;i=1", err: errors.New("invalid namespace id: ns=abc;i=1")},
		{s: "ns=1;i=abc", err: errors.New("invalid numeric id: ns=1;i=abc")},
		{s: "ns=1;i=4294967296", err: errors.New("numeric id out of range (0..2^32-1): ns=1;i=4294967296")},
		{s: "ns=1;g=x", err: errors.New("invalid guid node id: ns=1;g=x")},
		{s: "ns=1;b=aW52YWxp%ZA==", err: errors.New("invalid opaque node id: ns=1;b=aW52YWxp%ZA==")},
	}

	for _, c := range cases {
		t.Run(c.s, func(t *testing.T) {
			n, err := ParseNodeID(c.s)
			if got, want := err, c.err; !reflect.DeepEqual(got, want) {
				t.Fatalf("got error %v want %v", got, want)
			}
			if got, want := n, c.n; !cmp.Equal(got, want) {
				t.Fatal(cmp.Diff(got, want))
			}
		})
	}
}
