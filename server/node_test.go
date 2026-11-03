package server

import (
	"testing"

	"github.com/gopcua/opcua/id"
	"github.com/gopcua/opcua/ua"
	"github.com/stretchr/testify/require"
)

// TestDescriptionIsLocalizedText checks that nodes built by NewVariableNode
// and NewFolderNode store a LocalizedText in the Description attribute.
// Previously they stored a uint32 NodeClass, so reading the attribute through
// Node.Description() panicked on the type assertion.
func TestDescriptionIsLocalizedText(t *testing.T) {
	v := NewVariableNode(ua.NewNumericNodeID(1, 100), "temperature", int32(7))
	require.Equal(t, "temperature", v.Description().Text)

	f := NewFolderNode(ua.NewNumericNodeID(1, 101), "sensors")
	require.Equal(t, "sensors", f.Description().Text)
}

// TestDataTypeSurvivesClientWrite checks that DataType() tolerates a value
// written by a client. The Write service stores client-supplied values
// without validation, and the DataType attribute is spec-typed NodeId, so a
// write stores a *ua.NodeID where DataType() previously asserted
// *ua.ExpandedNodeID — panicking the server on the next Browse that resolved
// a reference to the node.
func TestDataTypeSurvivesClientWrite(t *testing.T) {
	n := NewVariableNode(ua.NewNumericNodeID(1, 102), "count", int32(0))

	// what AttributeService.Write does with a client WriteValue for the
	// DataType attribute. The returned status is ignored: SetAttribute
	// stores the value regardless.
	_ = n.SetAttribute(ua.AttributeIDDataType, DataValueFromValue(ua.NewNumericNodeID(0, id.Int32)))

	dt := n.DataType()
	require.NotNil(t, dt)
	require.NotNil(t, dt.NodeID)
	require.Equal(t, uint32(id.Int32), dt.NodeID.IntID())
}
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
