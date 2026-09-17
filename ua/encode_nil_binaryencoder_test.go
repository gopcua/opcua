// Copyright 2018-2020 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package ua

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestEncodeNilBinaryEncoderPointer verifies that Encode methods on
// the builtin BinaryEncoder types are nil-receiver safe and produce
// the type's OPC UA null encoding (Part 6 §5.2.2), following the
// pattern ExtensionObject.Encode established. Without the guards,
// encoding a struct with a nil BinaryEncoder pointer field panics
// (SIGSEGV observed in NodeID.Encode when a server response carried a
// nil *NodeID field). ExpandedNodeID is deliberately unchanged: its
// pre-existing nil-receiver contract returns an error, never panics.
func TestEncodeNilBinaryEncoderPointer(t *testing.T) {
	t.Run("top-level nil *NodeID", func(t *testing.T) {
		got, err := Encode((*NodeID)(nil))
		require.NoError(t, err)

		want, err := Encode(NewTwoByteNodeID(0))
		require.NoError(t, err)
		require.Equal(t, want, got, "nil *NodeID must encode as the null NodeId")
	})

	t.Run("top-level nil *ExpandedNodeID", func(t *testing.T) {
		// Pre-existing contract preserved: a nil *ExpandedNodeID is an
		// error, not a panic and not a silent null.
		_, err := Encode((*ExpandedNodeID)(nil))
		require.EqualError(t, err, "opcua: e was nil")
	})

	t.Run("top-level nil *LocalizedText", func(t *testing.T) {
		got, err := Encode((*LocalizedText)(nil))
		require.NoError(t, err)

		want, err := Encode(&LocalizedText{})
		require.NoError(t, err)
		require.Equal(t, want, got, "nil *LocalizedText must encode as the null LocalizedText")
	})

	t.Run("top-level nil *DataValue", func(t *testing.T) {
		got, err := Encode((*DataValue)(nil))
		require.NoError(t, err)
		require.Equal(t, []byte{0x00}, got, "nil *DataValue must encode as the null DataValue")
	})

	t.Run("top-level nil *GUID", func(t *testing.T) {
		got, err := Encode((*GUID)(nil))
		require.NoError(t, err)
		require.Equal(t, make([]byte, 16), got, "nil *GUID must encode as the null GUID (16 zero bytes)")
	})

	t.Run("top-level nil *Variant", func(t *testing.T) {
		got, err := Encode((*Variant)(nil))
		require.NoError(t, err)
		require.Equal(t, []byte{0x00}, got, "nil *Variant must encode as the null Variant")
	})

	t.Run("top-level nil *DiagnosticInfo", func(t *testing.T) {
		got, err := Encode((*DiagnosticInfo)(nil))
		require.NoError(t, err)
		require.Equal(t, []byte{0x00}, got, "nil *DiagnosticInfo must encode as the null DiagnosticInfo")
	})

	t.Run("nil BinaryEncoder struct fields", func(t *testing.T) {
		// Mirrors the field layout that triggered the panic: a struct
		// whose BinaryEncoder pointer fields are nil (e.g. a masked-out
		// ReferenceDescription built by a server). The *ExpandedNodeID
		// field is non-nil on both sides because its nil-receiver
		// contract is an error by design.
		type sample struct {
			ReferenceTypeID *NodeID
			IsForward       bool
			NodeID          *ExpandedNodeID
			DisplayName     *LocalizedText
		}

		got, err := Encode(&sample{NodeID: NewTwoByteExpandedNodeID(0)})
		require.NoError(t, err)

		want, err := Encode(&sample{
			ReferenceTypeID: NewTwoByteNodeID(0),
			NodeID:          NewTwoByteExpandedNodeID(0),
			DisplayName:     &LocalizedText{},
		})
		require.NoError(t, err)
		require.Equal(t, want, got, "nil fields must encode identically to explicit null values")
	})

	t.Run("nil *ExtensionObject keeps its null-extension-object encoding", func(t *testing.T) {
		// Regression pin: the null extension object is TypeID i=0 plus
		// the empty encoding mask. A struct with a nil *ExtensionObject
		// field (e.g. a nil AdditionalHeader in a RequestHeader) must
		// keep encoding this way.
		got, err := Encode((*ExtensionObject)(nil))
		require.NoError(t, err)
		require.Equal(t, []byte{0x00, 0x00, 0x00}, got, "nil *ExtensionObject must encode as the null extension object")
	})
}
