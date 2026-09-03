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
