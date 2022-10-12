// Copyright 2018-2020 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package ua

import (
	"encoding/base64"
	"encoding/json"
	"math"
	"reflect"
	"testing"

	"github.com/gopcua/opcua/errors"
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
			Name:   "Numeric max.Uint32",
			Struct: NewNumericNodeID(10, math.MaxUint32),
			Bytes: []byte{
				// mask
				0x02,
				// namespace
				0x0a, 0x00,
				// id
				0xff, 0xff, 0xff, 0xff,
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
			Struct: NewGUIDNodeID(4660, "72962B91-FA75-4AE6-8D28-B404DC7DAF63"),
			Bytes: []byte{
				// mask
				0x04,
				// namespace
				0x34, 0x12,
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
		{s: "i=1", n: NewTwoByteNodeID(1)},
		{s: "i=2253", n: NewFourByteNodeID(0, 2253)},
		{s: "ns=1;i=2", n: NewFourByteNodeID(1, 2)},
		{s: "ns=256;i=2", n: NewNumericNodeID(256, 2)},
		{s: "ns=1;i=65536", n: NewNumericNodeID(1, 65536)},
		{s: "ns=65535;i=65536", n: NewNumericNodeID(65535, 65536)},
		{s: "ns=2;i=4294967295", n: NewNumericNodeID(2, math.MaxUint32)},
		{s: "ns=1;g=5eac051c-c313-43d7-b790-24aa2c3cfd37", n: NewGUIDNodeID(1, "5eac051c-c313-43d7-b790-24aa2c3cfd37")},
		{s: "ns=1;b=YWJj", n: NewByteStringNodeID(1, []byte{'a', 'b', 'c'})},
		{s: "ns=1;s=a", n: NewStringNodeID(1, "a")},
		{s: "ns=1;a", n: NewStringNodeID(1, "a")},
		{s: "ns=1;s=foo;bar;", n: NewStringNodeID(1, "foo;bar;")},

		// from https://github.com/Azure-Samples/iot-edge-opc-plc
		{s: "ns=5;s=Special_\"!§$%&/()=?`´\\\\+~*\\'#_-:.;,<>|@^°€µ{[]}", n: NewStringNodeID(5, "Special_\"!§$%&/()=?`´\\\\+~*\\'#_-:.;,<>|@^°€µ{[]}")},

		// error flows
		{s: "abc=0;i=2", err: errors.New("invalid node id: abc=0;i=2")},
		{s: "ns=0;i=1;s=2", err: errors.New("invalid numeric id: ns=0;i=1;s=2")},
		{s: "ns=0", err: errors.New("invalid node id: ns=0")},
		{s: "nsu=abc;i=1", err: errors.New("namespace urls require a server NamespaceArray")},
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
			if got, want := err, c.err; !errors.Equal(got, want) {
				t.Fatalf("got error %v want %v", got, want)
			}
			if got, want := n, c.n; !reflect.DeepEqual(got, want) {
				t.Fatalf("\ngot  %#v\nwant %#v", got, want)
			}
		})
	}
}

func TestStringID(t *testing.T) {
	cases := []struct {
		name string
		s    string
		n    *NodeID
	}{
		{name: "basic", s: "i=1", n: NewTwoByteNodeID(1)},
		{name: "basic guid", s: "ns=1;g=5EAC051C-C313-43D7-B790-24AA2C3CFD37", n: NewGUIDNodeID(1, "5EAC051C-C313-43D7-B790-24AA2C3CFD37")},
		{name: "lower case guid", s: "ns=1;g=5EAC051C-C313-43D7-B790-24AA2C3CFD37", n: NewGUIDNodeID(1, "5eac051c-c313-43d7-b790-24aa2c3cfd37")},
		{name: "zero guid", s: "ns=1;g=00000000-0000-0000-0000-000000000000", n: NewGUIDNodeID(1, "00000000-0000-0000-0000-000000000000")},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got, want := c.n.String(), c.s; got != want {
				t.Fatalf("got %s want %s", got, want)
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
			if got, want := err, tt.err; !errors.Equal(got, want) {
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
			if got, want := err, tt.err; !errors.Equal(got, want) {
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
			if got, want := err, tt.err; !errors.Equal(got, want) {
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

func TestNodeIDJSON(t *testing.T) {
	t.Run("value", func(t *testing.T) {
		n, err := ParseNodeID(`ns=4;s=abc`)
		if err != nil {
			t.Fatal(err)
		}
		b, err := json.Marshal(n)
		if err != nil {
			t.Fatal(err)
		}
		if got, want := string(b), `"ns=4;s=abc"`; got != want {
			t.Fatalf("got %s want %s", got, want)
		}
		var nn NodeID
		if err := json.Unmarshal(b, &nn); err != nil {
			t.Fatal(err)
		}
		if got, want := nn.String(), n.String(); got != want {
			t.Fatalf("got %s want %s", got, want)
		}
	})

	t.Run("nil", func(t *testing.T) {
		var n *NodeID
		b, err := json.Marshal(n)
		if err != nil {
			t.Fatal(err)
		}
		if got, want := string(b), "null"; got != want {
			t.Fatalf("got %s want %s", got, want)
		}
	})

	type X struct{ N *NodeID }
	t.Run("struct", func(t *testing.T) {
		x := X{NewStringNodeID(4, "abc")}
		b, err := json.Marshal(x)
		if err != nil {
			t.Fatal(err)
		}
		if got, want := string(b), `{"N":"ns=4;s=abc"}`; got != want {
			t.Fatalf("got %s want %s", got, want)
		}
	})

	t.Run("nil struct", func(t *testing.T) {
		var x X
		b, err := json.Marshal(x)
		if err != nil {
			t.Fatal(err)
		}
		if got, want := string(b), `{"N":null}`; got != want {
			t.Fatalf("got %s want %s", got, want)
		}
		var xx X
		if err := json.Unmarshal(b, &xx); err != nil {
			t.Fatal(err)
		}
		if got, want := xx, x; !reflect.DeepEqual(got, want) {
			t.Fatalf("got %s want %s", got, want)
		}
	})
}

func TestNodeIDToString(t *testing.T) {
	tests := []struct {
		s    string
		want string
	}{
		{"i=123", "i=123"},
		{"s=123", "s=123"},
		{"b=MTIz", "b=MTIz"},
		{"g=F11BD41E-2DF5-41D3-8CE2-A37D22D1E469", "g=F11BD41E-2DF5-41D3-8CE2-A37D22D1E469"},

		{"ns=0;i=123", "i=123"},
		{"ns=0;s=123", "s=123"},
		{"ns=0;b=MTIz", "b=MTIz"},
		{"ns=0;g=F11BD41E-2DF5-41D3-8CE2-A37D22D1E469", "g=F11BD41E-2DF5-41D3-8CE2-A37D22D1E469"},

		{"ns=1;i=123", "ns=1;i=123"},
		{"ns=1;s=123", "ns=1;s=123"},
		{"ns=1;b=MTIz", "ns=1;b=MTIz"},
		{"ns=1;g=F11BD41E-2DF5-41D3-8CE2-A37D22D1E469", "ns=1;g=F11BD41E-2DF5-41D3-8CE2-A37D22D1E469"},
	}

	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			n, err := ParseNodeID(tt.s)
			if err != nil {
				t.Fatal(err)
			}
			if got, want := n.String(), tt.want; got != want {
				t.Fatalf("got %s want %s", got, want)
			}
		})
	}
}
