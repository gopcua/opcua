// Copyright 2018-2020 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package ua

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

// decodeNestingRecNode is a registered type that contains itself, the shape
// a user-defined type takes when the type it describes has children
// (examples/udt). The type registry has no unregister, so the ids below are
// names no other test uses.
type decodeNestingRecNode struct {
	Children []*decodeNestingRecNode
}

var decodeNestingRecNodeID = MustParseNodeID("ns=4;s=gopcua.decode_nesting_test.rec_node")

// A registered type that contains itself sends the generic decoder around
// a cycle with no limit of its own. Decoding must refuse to go deeper than
// maxDecodeNesting.
func TestDecodeRejectsARegisteredRecursiveTypePastTheLimit(t *testing.T) {
	smallStack(t, 64<<20)

	RegisterExtensionObject(decodeNestingRecNodeID, new(decodeNestingRecNode))

	t.Run("within the limit", func(t *testing.T) {
		// a chain of three nodes, the shallow shape real data takes
		root := &decodeNestingRecNode{}
		n := root
		for i := 0; i < 3; i++ {
			n.Children = []*decodeNestingRecNode{{}}
			n = n.Children[0]
		}
		b, err := NewExtensionObject(root).Encode()
		require.NoError(t, err)

		got := new(ExtensionObject)
		_, err = got.Decode(b)
		require.NoError(t, err)
		require.Equal(t, root, got.Value)
	})

	t.Run("past the limit", func(t *testing.T) {
		// A level of the chain is a one-element array on the wire, four
		// bytes per level, and the innermost node ends the chain with a
		// null array. The body is written by hand because encoding a
		// value this deep grows the encoder's stack the same way.
		body := append(bytes.Repeat([]byte{1, 0, 0, 0}, 1<<20), 0xff, 0xff, 0xff, 0xff)

		// the extension object that carries it: type id, encoding mask,
		// body length, body
		msg := NewBuffer(nil)
		msg.WriteStruct(&ExpandedNodeID{NodeID: decodeNestingRecNodeID})
		msg.WriteByte(ExtensionObjectBinary)
		msg.WriteUint32(uint32(len(body)))
		msg.Write(body)
		require.NoError(t, msg.Error())

		_, err := new(ExtensionObject).Decode(msg.Bytes())
		require.ErrorIs(t, err, StatusBadEncodingLimitsExceeded)
	})
}

// decodeNestingStation is a registered type with the nesting a legal message
// carries: a slice of structs, a fixed-size array of structs, and structs
// inside those.
type decodeNestingStation struct {
	Name    string
	Modules []decodeNestingModule
	Sensors [2]decodeNestingReading
}

type decodeNestingModule struct {
	ID       int32
	Readings []decodeNestingReading
}

type decodeNestingReading struct {
	Label string
	Value float64
}

var decodeNestingStationID = MustParseNodeID("ns=4;s=gopcua.decode_nesting_test.station")

// The nesting levels a decoder has to accept include the ones inside a
// registered type: an array of structs is legal data, however many elements
// it holds, and the decode must keep accepting it.
func TestDecodeAcceptsARegisteredStructWithAnArrayOfStructs(t *testing.T) {
	RegisterExtensionObject(decodeNestingStationID, new(decodeNestingStation))

	st := &decodeNestingStation{
		Name: "station",
		Modules: []decodeNestingModule{
			{ID: 1, Readings: []decodeNestingReading{
				{Label: "in", Value: 1.5},
				{Label: "out", Value: 2.5},
			}},
			{ID: 2, Readings: []decodeNestingReading{
				{Label: "spare", Value: 3.5},
			}},
		},
		Sensors: [2]decodeNestingReading{
			{Label: "left", Value: 8},
			{Label: "right", Value: 9},
		},
	}
	b, err := NewExtensionObject(st).Encode()
	require.NoError(t, err)

	got := new(ExtensionObject)
	_, err = got.Decode(b)
	require.NoError(t, err)
	require.Equal(t, st, got.Value)
}
