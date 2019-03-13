package opcua

import (
	"time"

	uad "github.com/gopcua/opcua/datatypes"
	uas "github.com/gopcua/opcua/services"
)

// Node is a high-level object to interact with a node in the
// address space. It provides common convenience functions to
// access and manipulate the common attributes of a node.
type Node struct {
	// ID is the node id of the node.
	ID *uad.NodeID

	c *Client
}

// NodeClass returns the node class attribute.
func (a *Node) NodeClass() (uas.NodeClass, error) {
	v, err := a.Attribute(uad.IntegerIDNodeClass)
	if err != nil {
		return 0, err
	}
	return uas.NodeClass(v.Int()), nil
}

// BrowseName returns the browse name of the node.
func (a *Node) BrowseName() (*uad.QualifiedName, error) {
	v, err := a.Attribute(uad.IntegerIDBrowseName)
	if err != nil {
		return nil, err
	}
	return v.Value.(*uad.QualifiedName), nil
}

// DisplayName returns the display name of the node.
func (a *Node) DisplayName() (*uad.LocalizedText, error) {
	v, err := a.Attribute(uad.IntegerIDDisplayName)
	if err != nil {
		return nil, err
	}
	return v.Value.(*uad.LocalizedText), nil
}

// Value returns the value of the node.
func (a *Node) Value() (*uad.Variant, error) {
	return a.Attribute(uad.IntegerIDValue)
}

// Attribute returns the attribute of the node. with the given id.
func (a *Node) Attribute(attrID uint32) (*uad.Variant, error) {
	rv := &uad.ReadValueID{NodeID: a.ID, AttributeID: attrID, DataEncoding: &uad.QualifiedName{}}
	req := &uas.ReadRequest{NodesToRead: []*uad.ReadValueID{rv}}
	res, err := a.c.Read(req)
	if err != nil {
		return nil, err
	}
	if len(res.Results) == 0 {
		return nil, nil
	}
	return res.Results[0].Value, nil
}

// References retrns all references for the node.
// todo(fs): this is not complete since it only returns the
// todo(fs): top-level reference at this point.
func (a *Node) References(refs *uad.NodeID) (*uas.BrowseResponse, error) {
	desc := &uas.BrowseDescription{
		NodeID:          a.ID,
		Direction:       uas.BrowseDirectionBoth,
		ReferenceTypeID: refs,
		IncludeSubtypes: true,
		NodeClassMask:   uas.NodeClassAll,
		ResultMask:      uas.BrowseResultMaskAll,
	}

	req := &uas.BrowseRequest{
		View: &uas.ViewDescription{
			ViewID:    uad.NewTwoByteNodeID(0),
			Timestamp: time.Now(),
		},
		RequestedMaxReferencesPerNode: 1000,
		NodesToBrowse:                 []*uas.BrowseDescription{desc},
	}

	return a.c.Browse(req)
	// implement browse_next
}
