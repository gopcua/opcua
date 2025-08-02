// Copyright 2018-2020 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package ua

import (
	"math"
	"testing"

	"github.com/gopcua/opcua/errors"
	"github.com/stretchr/testify/require"
)

func TestExpandedNodeID(t *testing.T) {
	cases := []CodecTestCase{
		{
			Name:   "Without optional fields",
			Struct: NewExpandedNodeID(NewTwoByteNodeID(0xff), "", 0),
			Bytes: []byte{
				0x00, 0xff,
			},
		},
		{
			Name:   "With NamespaceURI",
			Struct: NewExpandedNodeID(NewTwoByteNodeID(0xff), "foobar", 0),
			Bytes: []byte{
				0x80, 0xff, 0x06, 0x00, 0x00, 0x00, 0x66, 0x6f,
				0x6f, 0x62, 0x61, 0x72,
			},
		},
		{
			Name:   "With ServerIndex",
			Struct: NewExpandedNodeID(NewTwoByteNodeID(0xff), "", 32768),
			Bytes: []byte{ // With ServerIndex
				0x40, 0xff, 0x00, 0x80, 0x00, 0x00,
			},
		},
		{
			Name:   "With NamespaceURI and ServerIndex",
			Struct: NewExpandedNodeID(NewTwoByteNodeID(0xff), "foobar", 32768),
			Bytes: []byte{
				0xc0, 0xff, 0x06, 0x00, 0x00, 0x00, 0x66, 0x6f,
				0x6f, 0x62, 0x61, 0x72, 0x00, 0x80, 0x00, 0x00,
			},
		},
	}
	RunCodecTest(t, cases)
}

func TestParseExpandedNodeID(t *testing.T) {
	cases := []struct {
		s   string
		ns  []string
		n   *ExpandedNodeID
		err error
	}{
		// happy flows (same as for ParseNodeID)
		{s: "", n: NewTwoByteExpandedNodeID(0)},
		{s: "ns=0;i=1", n: NewTwoByteExpandedNodeID(1)},
		{s: "i=1", n: NewTwoByteExpandedNodeID(1)},
		{s: "i=2253", n: NewFourByteExpandedNodeID(0, 2253)},
		{s: "ns=1;i=2", n: NewFourByteExpandedNodeID(1, 2)},
		{s: "ns=256;i=2", n: NewNumericExpandedNodeID(256, 2)},
		{s: "ns=1;i=65536", n: NewNumericExpandedNodeID(1, 65536)},
		{s: "ns=65535;i=65536", n: NewNumericExpandedNodeID(65535, 65536)},
		{s: "ns=2;i=4294967295", n: NewNumericExpandedNodeID(2, math.MaxUint32)},
		{s: "ns=1;g=5eac051c-c313-43d7-b790-24aa2c3cfd37", n: NewGUIDExpandedNodeID(1, "5eac051c-c313-43d7-b790-24aa2c3cfd37")},
		{s: "ns=1;b=YWJj", n: NewByteStringExpandedNodeID(1, []byte{'a', 'b', 'c'})},
		{s: "ns=1;s=a", n: NewStringExpandedNodeID(1, "a")},
		{s: "ns=1;a", n: NewStringExpandedNodeID(1, "a")},
		{s: "ns=1;s=foo;bar;", n: NewStringExpandedNodeID(1, "foo;bar;")},

		// from https://github.com/Azure-Samples/iot-edge-opc-plc
		{s: "ns=5;s=Special_\"!§$%&/()=?`´\\\\+~*\\'#_-:.;,<>|@^°€µ{[]}", n: NewStringExpandedNodeID(5, "Special_\"!§$%&/()=?`´\\\\+~*\\'#_-:.;,<>|@^°€µ{[]}")},

		// error flows (same as ParseNodeID)
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

		// nsu happy flows
		{s: "nsu=abc;i=2", ns: []string{"", "abc"}, n: NewExpandedNodeID(NewFourByteNodeID(1, 2), "abc", 0)},
		{s: "nsu=abc;i=65536", ns: []string{"", "abc"}, n: NewExpandedNodeID(NewNumericNodeID(1, 65536), "abc", 0)},
		{s: "nsu=abc;b=YWJj", ns: []string{"", "abc"}, n: NewExpandedNodeID(NewByteStringNodeID(1, []byte{'a', 'b', 'c'}), "abc", 0)},
		{s: "nsu=abc;a", ns: []string{"", "abc"}, n: NewExpandedNodeID(NewStringNodeID(1, "a"), "abc", 0)},
		{s: "nsu=abc;s=a", ns: []string{"", "abc"}, n: NewExpandedNodeID(NewStringNodeID(1, "a"), "abc", 0)},

		// nsu error flows
		{s: "nsu=abc;i=2253", ns: []string{}, err: errors.New("namespace uri nsu=abc not found in the server NamespaceArray []string{}")},
		{s: "nsu=abc;i=2253", ns: []string{"", "def", "xyz"}, err: errors.New(`namespace uri nsu=abc not found in the server NamespaceArray []string{"", "def", "xyz"}`)},
	}

	for _, c := range cases {
		t.Run(c.s, func(t *testing.T) {
			n, err := ParseExpandedNodeID(c.s, c.ns)
			require.Equal(t, c.err, err, "Errors not equal")
			require.Equal(t, c.n, n, "ExpandedNodeID not equal")
		})
	}
}

func TestExpandedNodeID_String(t *testing.T) {
	cases := []struct {
		name string
		n    ExpandedNodeID
		sRep string
	}{
		{name: "svr=1;nsu=urn:abc:123;i=3344", n: ExpandedNodeID{ServerIndex: 1, NamespaceURI: "urn:abc:123", NodeID: NewNumericNodeID(1, 3344)}, sRep: "svr=1;nsu=urn:abc:123;i=3344"},
		{name: "nsu=urn:abc:123;i=3344, ns=1", n: ExpandedNodeID{ServerIndex: 0, NamespaceURI: "urn:abc:123", NodeID: NewNumericNodeID(1, 3344)}, sRep: "nsu=urn:abc:123;i=3344"},
		{name: "nsu=urn:abc:123;i=3344, ns=0", n: ExpandedNodeID{ServerIndex: 0, NamespaceURI: "urn:abc:123", NodeID: NewNumericNodeID(0, 3344)}, sRep: "nsu=urn:abc:123;i=3344"},
		{name: "ns=1;i=3344", n: ExpandedNodeID{ServerIndex: 0, NamespaceURI: "", NodeID: NewNumericNodeID(1, 3344)}, sRep: "ns=1;i=3344"},
		{name: "i=4, TwoByteNodeId", n: ExpandedNodeID{ServerIndex: 0, NamespaceURI: "", NodeID: NewTwoByteNodeID(uint8(4))}, sRep: "i=4"},
		{name: "ns=2;i=5432, FourByteNodeId", n: ExpandedNodeID{ServerIndex: 0, NamespaceURI: "", NodeID: NewFourByteNodeID(2, 5432)}, sRep: "ns=2;i=5432"},
		{name: "i=3344", n: ExpandedNodeID{ServerIndex: 0, NamespaceURI: "", NodeID: NewNumericNodeID(0, 3344)}, sRep: "i=3344"},
		{name: "guid identifier", n: ExpandedNodeID{ServerIndex: 0, NamespaceURI: "", NodeID: NewGUIDNodeID(3, NewGUID("09087e75-8e5e-499b-954f-f2a9603db28a").String())}, sRep: "ns=3;g=09087E75-8E5E-499B-954F-F2A9603DB28A"},
		{name: "string identifier", n: ExpandedNodeID{ServerIndex: 0, NamespaceURI: "", NodeID: NewStringNodeID(5, "mystring")}, sRep: "ns=5;s=mystring"},
		{name: "opaque identifier", n: ExpandedNodeID{ServerIndex: 0, NamespaceURI: "", NodeID: NewByteStringNodeID(5, []byte{1, 2, 3, 4})}, sRep: "ns=5;b=AQIDBA=="},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			str := c.n.String()
			require.Equal(t, c.sRep, str)
		})
	}
}
