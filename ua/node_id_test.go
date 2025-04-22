// Copyright 2018-2020 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package ua

import (
	"encoding/base64"
	"encoding/json"
	"math"
	"testing"

	"github.com/gopcua/opcua/errors"
	"github.com/stretchr/testify/require"
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
			require.Equal(t, c.err, err, "Error not equal")
			require.Equal(t, c.n, n, "Parsed NodeID not equal")
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
			require.Equal(t, c.s, c.n.String())
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
			require.NotEqual(t, tt.v, v, "before == after: %d == %d", v, tt.v)

			err := tt.n.SetIntID(tt.v)
			require.Equal(t, tt.err, err)

			// if the test should fail and the error was correct
			// we need to stop here.
			if tt.err != nil {
				return
			}
			require.Equal(t, tt.v, tt.n.IntID(), "IntID not equal")
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
			require.NotEqual(t, tt.v, v, "before == after: %s == %s", v, tt.v)

			err := tt.n.SetStringID(tt.v)
			require.Equal(t, tt.err, err)

			// if the test should fail and the error was correct
			// we need to stop here.
			if tt.err != nil {
				return
			}
			require.Equal(t, tt.v, tt.n.StringID())
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
			require.Equal(t, tt.err, err)

			// if the test should fail and the error was correct
			// we need to stop here.
			if tt.err != nil {
				return
			}
			require.Equal(t, tt.v, tt.n.Namespace())
		})
	}
}

func TestNodeIDJSON(t *testing.T) {
	t.Run("value", func(t *testing.T) {
		n, err := ParseNodeID(`ns=4;s=abc`)
		require.NoError(t, err)

		b, err := json.Marshal(n)
		require.NoError(t, err)
		require.Equal(t, `"ns=4;s=abc"`, string(b))

		var nn NodeID
		err = json.Unmarshal(b, &nn)
		require.NoError(t, err)
		require.Equal(t, n.String(), nn.String(), "NodeIDs not equal")
	})

	t.Run("nil", func(t *testing.T) {
		var n *NodeID
		b, err := json.Marshal(n)
		require.NoError(t, err)
		require.Equal(t, "null", string(b))
	})

	type X struct{ N *NodeID }
	t.Run("struct", func(t *testing.T) {
		x := X{NewStringNodeID(4, "abc")}
		b, err := json.Marshal(x)
		require.NoError(t, err)
		require.Equal(t, `{"N":"ns=4;s=abc"}`, string(b))
	})

	t.Run("nil struct", func(t *testing.T) {
		var x X
		b, err := json.Marshal(x)
		require.NoError(t, err)
		require.Equal(t, `{"N":null}`, string(b))

		var xx X
		err = json.Unmarshal(b, &xx)
		require.NoError(t, err)
		require.Equal(t, x, xx)
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
			require.NoError(t, err)
			require.Equal(t, tt.want, n.String())
		})
	}
}

func TestNewNodeIDFromExpandedNodeID(t *testing.T) {
	type args struct {
		id *ExpandedNodeID
	}
	tests := []struct {
		name string
		args args
		want *NodeID
	}{
		{
			name: "NewExpandedNodeID",
			args: args{
				id: NewExpandedNodeID(NewTwoByteNodeID(42), "someUri", 42),
			},
			want: NewTwoByteNodeID(42),
		},
		{
			name: "NewTwoByteExpandedNodeID",
			args: args{
				id: NewTwoByteExpandedNodeID(42),
			},
			want: NewTwoByteNodeID(42),
		},
		{
			name: "NewFourByteExpandedNodeID",
			args: args{
				NewFourByteExpandedNodeID(42, 24),
			},
			want: NewFourByteNodeID(42, 24),
		},
		{
			name: "NewNumericExpandedNodeID",
			args: args{
				NewNumericExpandedNodeID(42, 24),
			},
			want: NewNumericNodeID(42, 24),
		},
		{
			name: "NewStringExpandedNodeID",
			args: args{
				NewStringExpandedNodeID(42, "42"),
			},
			want: NewStringNodeID(42, "42"),
		},
		{
			name: "NewGUIDExpandedNodeID",
			args: args{
				NewGUIDExpandedNodeID(42, "42"),
			},
			want: NewGUIDNodeID(42, "42"),
		},
		{
			name: "NewByteStringExpandedNodeID",
			args: args{
				NewByteStringExpandedNodeID(42, []byte{0xAF, 0xFE}),
			},
			want: NewByteStringNodeID(42, []byte{0xAF, 0xFE}),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, NewNodeIDFromExpandedNodeID(tt.args.id))
		})
	}
}
