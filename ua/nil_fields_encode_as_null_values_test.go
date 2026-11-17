// Copyright 2018-2020 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package ua

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestNilFieldsEncodeAsNullValues verifies that a ReferenceDescription
// whose pointer fields are all nil encodes byte-identically to one whose
// fields carry their null values (Part 6, 5.2.2.10 and 5.2.2.13), so that
// each field frames the rest of the message correctly.
func TestNilFieldsEncodeAsNullValues(t *testing.T) {
	t.Run("nil ReferenceDescription fields frame each other", func(t *testing.T) {
		allNil, err := Encode(&ReferenceDescription{})
		require.NoError(t, err)

		nullValues, err := Encode(&ReferenceDescription{
			ReferenceTypeID: NewTwoByteNodeID(0),
			NodeID:          NewTwoByteExpandedNodeID(0),
			BrowseName:      &QualifiedName{},
			DisplayName:     &LocalizedText{},
			TypeDefinition:  NewTwoByteExpandedNodeID(0),
		})
		require.NoError(t, err)

		want := []byte{
			0x00, 0x00, // ReferenceTypeID: null NodeID
			0x00,       // IsForward: false
			0x00, 0x00, // NodeID: null ExpandedNodeID
			0x00, 0x00, 0xff, 0xff, 0xff, 0xff, // BrowseName: null QualifiedName
			0x00,                   // DisplayName: null LocalizedText
			0x00, 0x00, 0x00, 0x00, // NodeClass: 0
			0x00, 0x00, // TypeDefinition: null ExpandedNodeID
		}
		require.Equal(t, want, allNil,
			"a ReferenceDescription with all fields nil must encode to the null-value message")
		require.Equal(t, nullValues, allNil,
			"nil fields must encode identically to explicit null values")
	})
}
