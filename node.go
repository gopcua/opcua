package opcua

import (
	"time"

	"github.com/gopcua/opcua/ua"
)

// Node is a high-level object to interact with a node in the
// address space. It provides common convenience functions to
// access and manipulate the common attributes of a node.
type Node struct {
	// ID is the node id of the node.
	ID *ua.NodeID

	c *Client
}

// NodeClass returns the node class attribute.
func (a *Node) NodeClass() (ua.NodeClass, error) {
	v, err := a.Attribute(ua.IntegerIDNodeClass)
	if err != nil {
		return 0, err
	}
	return ua.NodeClass(v.Int()), nil
}

// BrowseName returns the browse name of the node.
func (a *Node) BrowseName() (*ua.QualifiedName, error) {
	v, err := a.Attribute(ua.IntegerIDBrowseName)
	if err != nil {
		return nil, err
	}
	return v.Value.(*ua.QualifiedName), nil
}

// DisplayName returns the display name of the node.
func (a *Node) DisplayName() (*ua.LocalizedText, error) {
	v, err := a.Attribute(ua.IntegerIDDisplayName)
	if err != nil {
		return nil, err
	}
	return v.Value.(*ua.LocalizedText), nil
}

// Value returns the value of the node.
func (a *Node) Value() (*ua.Variant, error) {
	return a.Attribute(ua.IntegerIDValue)
}

// Attribute returns the attribute of the node. with the given id.
func (a *Node) Attribute(attrID uint32) (*ua.Variant, error) {
	rv := &ua.ReadValueID{NodeID: a.ID, AttributeID: attrID}
	req := &ua.ReadRequest{NodesToRead: []*ua.ReadValueID{rv}}
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
func (a *Node) References(refs *ua.NodeID) (*ua.BrowseResponse, error) {
	desc := &ua.BrowseDescription{
		NodeID:          a.ID,
		BrowseDirection: ua.BrowseDirectionBoth,
		ReferenceTypeID: refs,
		IncludeSubtypes: true,
		NodeClassMask:   uint32(ua.NodeClassAll),
		ResultMask:      uint32(ua.BrowseResultMaskAll),
	}

	req := &ua.BrowseRequest{
		View: &ua.ViewDescription{
			ViewID:    ua.NewTwoByteNodeID(0),
			Timestamp: time.Now(),
		},
		RequestedMaxReferencesPerNode: 1000,
		NodesToBrowse:                 []*ua.BrowseDescription{desc},
	}

	return a.c.Browse(req)
	// implement browse_next
}
