// Copyright 2018-2020 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package server

import (
	"testing"

	"github.com/gopcua/opcua/id"
	"github.com/gopcua/opcua/ua"
	"github.com/stretchr/testify/require"
)

func TestNamespaceAttributeDataTypeReturnsNodeID(t *testing.T) {
	ns := NewNameSpace("test")
	nodeID := ua.NewStringNodeID(1, "x")

	n := NewNode(
		nodeID,
		Attributes{
			ua.AttributeIDBrowseName:  DataValueFromValue(&ua.QualifiedName{Name: "x"}),
			ua.AttributeIDDisplayName: DataValueFromValue(&ua.LocalizedText{Text: "x"}),
			ua.AttributeIDNodeClass:   DataValueFromValue(uint32(ua.NodeClassVariable)),
			ua.AttributeIDDataType:    DataValueFromValue(ua.NewNumericExpandedNodeID(0, id.Double)),
		},
		nil,
		nil,
	)

	ns.AddNode(n)

	dv := ns.Attribute(nodeID, ua.AttributeIDDataType)

	require.Equal(t, ua.StatusOK, dv.Status)
	require.NotNil(t, dv.Value.NodeID())
	require.Equal(t, ua.TypeIDNodeID, dv.Value.Type())
	require.Equal(t, uint32(id.Double), dv.Value.NodeID().IntID())
}

func TestNamespaceAttributeMissingNodeClassReturnsBadAttributeIDInvalid(t *testing.T) {
	ns := NewNameSpace("test")
	nodeID := ua.NewStringNodeID(1, "x")

	n := NewNode(
		nodeID,
		Attributes{
			ua.AttributeIDBrowseName:  DataValueFromValue(&ua.QualifiedName{Name: "x"}),
			ua.AttributeIDDisplayName: DataValueFromValue(&ua.LocalizedText{Text: "x"}),
		},
		nil,
		nil,
	)

	ns.AddNode(n)

	require.NotPanics(t, func() {
		dv := ns.Attribute(nodeID, ua.AttributeIDNodeClass)
		require.Equal(t, ua.StatusBadAttributeIDInvalid, dv.Status)
	})
}
