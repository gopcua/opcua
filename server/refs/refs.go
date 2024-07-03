package refs

import (
	"github.com/gopcua/opcua/id"
	"github.com/gopcua/opcua/server/attrs"
	"github.com/gopcua/opcua/ua"
)

// HasSubtype returns a HasSubtype reference.
func HasSubtype(typeID *ua.ExpandedNodeID) *ua.ReferenceDescription {
	return &ua.ReferenceDescription{
		ReferenceTypeID: ua.NewNumericNodeID(0, id.HasSubtype),
		TypeDefinition:  typeID,
		IsForward:       true,
	}
}

// HasSubtype returns a HasSubtype reference.
func Organizes(nid *ua.NodeID, browseName, displayName string, typeID *ua.ExpandedNodeID) *ua.ReferenceDescription {
	return &ua.ReferenceDescription{
		ReferenceTypeID: ua.NewNumericNodeID(0, id.Organizes),
		NodeID:          &ua.ExpandedNodeID{NodeID: nid},
		BrowseName:      attrs.BrowseName(browseName),
		DisplayName:     attrs.DisplayName(displayName, ""),
		TypeDefinition:  typeID,
		IsForward:       true,
	}
}
