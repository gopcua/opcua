// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package ua

import (
	"encoding/base64"
	"errors"
	"reflect"
	"testing"
)

func TestNodeID(t *testing.T) {
	cases := []CodecTestCase{
		{
			Name:   "TwoByte",
			Struct: NewTwoByteNodeID(0xff),
			Bytes: []byte{
				// mask
				0x00,
				// id
				0xff,
			},
		},
		{
			Name:   "FourByte",
			Struct: NewFourByteNodeID(0, 0xcafe),
			Bytes: []byte{
				// mask
				0x01,
				// namespace
				0x00,
				// id
				0xfe, 0xca,
			},
		},
		{
			Name:   "Numeric",
			Struct: NewNumericNodeID(10, 0xdeadbeef),
			Bytes: []byte{
				// mask
				0x02,
				// namespace
				0x0a, 0x00,
				// id
				0xef, 0xbe, 0xad, 0xde,
			},
		},
		{
			Name:   "String",
			Struct: NewStringNodeID(255, "foobar"),
			Bytes: []byte{
				// mask
				0x03,
				// namespace
				0xff, 0x00,
				// length
				0x06, 0x00, 0x00, 0x00,
				// value
				0x66, 0x6f, 0x6f, 0x62, 0x61, 0x72,
			},
		},
		{
			Name:   "GUID",
			Struct: NewGUIDNodeID(4660, "AAAABBBB-CCDD-EEFF-0101-0123456789AB"),
			Bytes: []byte{
				// mask
				0x04,
				// namespace
				0x34, 0x12,
				// id
				0xbb, 0xbb, 0xaa, 0xaa, 0xdd, 0xcc, 0xff, 0xee,
				0xab, 0x89, 0x67, 0x45, 0x23, 0x01, 0x01, 0x01,
			},
		},
		{
			Name:   "Opaque",
			Struct: NewByteStringNodeID(32768, []byte{0xde, 0xad, 0xbe, 0xef}),
			Bytes: []byte{
				// mask
				0x05,
				// namespace
				0x00, 0x80,
				// length
				0x04, 0x00, 0x00, 0x00,
				// value
				0xde, 0xad, 0xbe, 0xef,
			},
		},
	}
	RunCodecTest(t, cases)
}

func BenchmarkReflectDecode(b *testing.B) {
	data := []byte{
		// mask
		0x05,
		// namespace
		0x00, 0x80,
		// length
		0x04, 0x00, 0x00, 0x00,
		// value
		0xde, 0xad, 0xbe, 0xef,
	}

	for i := 0; i < b.N; i++ {
		v := new(NodeID)
		if _, err := Decode(data, v); err != nil {
			b.Fatal(err)
		}
	}
}

func TestParseNodeID(t *testing.T) {
	cases := []struct {
		s   string
		n   *NodeID
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
		{s: "ns=1;b=YWJj", n: NewByteStringNodeID(1, []byte{'a', 'b', 'c'})},
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
			if got, want := n, c.n; !reflect.DeepEqual(got, want) {
				t.Fatalf("\ngot  %#v\nwant %#v", got, want)
			}
		})
	}
}

func TestSetIntID(t *testing.T) {
	tests := []struct {
		name string
		n    *NodeID
		v    uint32
		err  error
	}{
		// happy flows
		{
			name: "TwoByte",
			n:    NewTwoByteNodeID(1),
			v:    2,
		},
		{
			name: "FourByte",
			n:    NewFourByteNodeID(0, 1),
			v:    2,
		},
		{
			name: "Numeric",
			n:    NewNumericNodeID(0, 1),
			v:    2,
		},

		// error flows
		{
			name: "TwoByte.tooBig",
			n:    NewTwoByteNodeID(1),
			v:    256,
			err:  errors.New("out of range [0..255]: 256"),
		},
		{
			name: "FourByte.tooBig",
			n:    NewFourByteNodeID(0, 1),
			v:    65536,
			err:  errors.New("out of range [0..65535]: 65536"),
		},
		{
			name: "String.incompatible",
			n:    NewStringNodeID(0, "a"),
			v:    1,
			err:  errors.New("incompatible node id type"),
		},
		{
			name: "GUID.incompatible",
			n:    NewGUIDNodeID(0, "a"),
			v:    1,
			err:  errors.New("incompatible node id type"),
		},
		{
			name: "Opaque.incompatible",
			n:    NewByteStringNodeID(0, []byte{0x01}),
			v:    1,
			err:  errors.New("incompatible node id type"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := tt.n.IntID()

			// sanity check
			if before, after := v, tt.v; before == after {
				t.Fatalf("before == after: %d == %d", before, after)
			}

			err := tt.n.SetIntID(tt.v)
			if got, want := err, tt.err; !reflect.DeepEqual(got, want) {
				t.Fatalf("got error %v want %v", got, want)
			}
			// if the test should fail and the error was correct
			// we need to stop here.
			if tt.err != nil {
				return
			}
			if got, want := tt.n.IntID(), tt.v; got != want {
				t.Fatalf("got value %d want %d", got, want)
			}
		})
	}
}

func TestSetStringID(t *testing.T) {
	tests := []struct {
		name string
		n    *NodeID
		v    string
		err  error
	}{
		// happy flows
		{
			name: "String",
			n:    NewStringNodeID(0, "a"),
			v:    "b",
		},
		{
			name: "GUID",
			n:    NewGUIDNodeID(0, "AAAABBBB-CCDD-EEFF-0101-0123456789AB"),
			v:    "AAAABBBB-CCDD-EEFF-0101-012345678900",
		},
		{
			name: "Opaque",
			n:    NewByteStringNodeID(0, []byte{'a'}),
			v:    "Yg==",
		},

		// error flows
		{
			name: "TwoByte.incompatible",
			n:    NewTwoByteNodeID(1),
			v:    "a",
			err:  errors.New("incompatible node id type"),
		},
		{
			name: "FourByte.incompatible",
			n:    NewFourByteNodeID(0, 1),
			v:    "a",
			err:  errors.New("incompatible node id type"),
		},
		{
			name: "Numeric.incompatible",
			n:    NewNumericNodeID(0, 1),
			v:    "a",
			err:  errors.New("incompatible node id type"),
		},
		{
			name: "Opaque.badBase64",
			n:    NewByteStringNodeID(0, []byte{'a'}),
			v:    "%",
			err:  base64.CorruptInputError(0),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := tt.n.StringID()

			// sanity check
			if before, after := v, tt.v; before == after {
				t.Fatalf("before == after: %s == %s", before, after)
			}

			err := tt.n.SetStringID(tt.v)
			if got, want := err, tt.err; !reflect.DeepEqual(got, want) {
				t.Fatalf("got error %q (%T) want %q (%T)", got, got, want, want)
			}
			// if the test should fail and the error was correct
			// we need to stop here.
			if tt.err != nil {
				return
			}
			if got, want := tt.n.StringID(), tt.v; got != want {
				t.Fatalf("got value %s want %s", got, want)
			}
		})
	}
}

func TestSetNamespace(t *testing.T) {
	tests := []struct {
		name string
		n    *NodeID
		v    uint16
		err  error
	}{
		// happy flows
		{
			name: "TwoByte",
			n:    NewTwoByteNodeID(1),
			v:    0,
		},
		{
			name: "FourByte",
			n:    NewFourByteNodeID(0, 1),
			v:    1,
		},
		{
			name: "Numeric",
			n:    NewNumericNodeID(0, 1),
			v:    1,
		},

		// error flows
		{
			name: "TwoByte.invalid",
			n:    NewTwoByteNodeID(1),
			v:    1,
			err:  errors.New("out of range [0..0]: 1"),
		},
		{
			name: "FourByte.tooBig",
			n:    NewFourByteNodeID(0, 1),
			v:    256,
			err:  errors.New("out of range [0..255]: 256"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.n.SetNamespace(tt.v)
			if got, want := err, tt.err; !reflect.DeepEqual(got, want) {
				t.Fatalf("got error %v want %v", got, want)
			}
			// if the test should fail and the error was correct
			// we need to stop here.
			if tt.err != nil {
				return
			}
			if got, want := tt.n.Namespace(), tt.v; got != want {
				t.Fatalf("got value %d want %d", got, want)
			}
		})
	}
}
