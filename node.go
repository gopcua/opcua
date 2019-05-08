// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

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

func (n *Node) String() string {
	return n.ID.String()
}

// NodeClass returns the node class attribute.
func (n *Node) NodeClass() (ua.NodeClass, error) {
	v, err := n.Attribute(ua.AttributeIDNodeClass)
	if err != nil {
		return 0, err
	}
	return ua.NodeClass(v.Int()), nil
}

// BrowseName returns the browse name of the node.
func (n *Node) BrowseName() (*ua.QualifiedName, error) {
	v, err := n.Attribute(ua.AttributeIDBrowseName)
	if err != nil {
		return nil, err
	}
	return v.Value.(*ua.QualifiedName), nil
}

// DisplayName returns the display name of the node.
func (n *Node) DisplayName() (*ua.LocalizedText, error) {
	v, err := n.Attribute(ua.AttributeIDDisplayName)
	if err != nil {
		return nil, err
	}
	return v.Value.(*ua.LocalizedText), nil
}

// Value returns the value of the node.
func (n *Node) Value() (*ua.Variant, error) {
	return n.Attribute(ua.AttributeIDValue)
}

// Attribute returns the attribute of the node. with the given id.
func (n *Node) Attribute(attrID ua.AttributeID) (*ua.Variant, error) {
	rv := &ua.ReadValueID{NodeID: n.ID, AttributeID: attrID}
	req := &ua.ReadRequest{NodesToRead: []*ua.ReadValueID{rv}}
	res, err := n.c.Read(req)
	if err != nil {
		return nil, err
	}
	if len(res.Results) == 0 {
		// #188: we return StatusBadUnexpectedError because it is unclear, under what
		// circumstances the server would return no error and no results in the response
		return nil, ua.StatusBadUnexpectedError
	}
	value := res.Results[0].Value
	if res.Results[0].Status != ua.StatusOK {
		return value, res.Results[0].Status
	}
	return value, nil
}

// References retrns all references for the node.
// todo(fs): this is not complete since it only returns the
// todo(fs): top-level reference at this point.
func (n *Node) References(refs *ua.NodeID) (*ua.BrowseResponse, error) {
	desc := &ua.BrowseDescription{
		NodeID:          n.ID,
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

	return n.c.Browse(req)
	// implement browse_next
}
