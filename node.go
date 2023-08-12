// Copyright 2018-2020 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package opcua

import (
	"context"
	"strings"

	"github.com/gopcua/opcua/id"
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
func (n *Node) NodeClass(ctx context.Context) (ua.NodeClass, error) {
	v, err := n.Attribute(ctx, ua.AttributeIDNodeClass)
	if err != nil {
		return 0, err
	}
	return ua.NodeClass(v.Int()), nil
}

// BrowseName returns the browse name of the node.
func (n *Node) BrowseName(ctx context.Context) (*ua.QualifiedName, error) {
	v, err := n.Attribute(ctx, ua.AttributeIDBrowseName)
	if err != nil {
		return nil, err
	}
	return v.Value().(*ua.QualifiedName), nil
}

// Description returns the description of the node.
func (n *Node) Description(ctx context.Context) (*ua.LocalizedText, error) {
	v, err := n.Attribute(ctx, ua.AttributeIDDescription)
	if err != nil {
		return nil, err
	}
	return v.Value().(*ua.LocalizedText), nil
}

// DisplayName returns the display name of the node.
func (n *Node) DisplayName(ctx context.Context) (*ua.LocalizedText, error) {
	v, err := n.Attribute(ctx, ua.AttributeIDDisplayName)
	if err != nil {
		return nil, err
	}
	return v.Value().(*ua.LocalizedText), nil
}

// AccessLevel returns the access level of the node.
// The returned value is a mask where multiple values can be
// set, e.g. read and write.
func (n *Node) AccessLevel(ctx context.Context) (ua.AccessLevelType, error) {
	v, err := n.Attribute(ctx, ua.AttributeIDAccessLevel)
	if err != nil {
		return 0, err
	}
	return ua.AccessLevelType(v.Value().(uint8)), nil
}

// HasAccessLevel returns true if all bits from mask are
// set in the access level mask of the node.
func (n *Node) HasAccessLevel(ctx context.Context, mask ua.AccessLevelType) (bool, error) {
	v, err := n.AccessLevel(ctx)
	if err != nil {
		return false, err
	}
	return (v & mask) == mask, nil
}

// UserAccessLevel returns the access level of the node.
func (n *Node) UserAccessLevel(ctx context.Context) (ua.AccessLevelType, error) {
	v, err := n.Attribute(ctx, ua.AttributeIDUserAccessLevel)
	if err != nil {
		return 0, err
	}
	return ua.AccessLevelType(v.Value().(uint8)), nil
}

// HasUserAccessLevel returns true if all bits from mask are
// set in the user access level mask of the node.
func (n *Node) HasUserAccessLevel(ctx context.Context, mask ua.AccessLevelType) (bool, error) {
	v, err := n.UserAccessLevel(ctx)
	if err != nil {
		return false, err
	}
	return (v & mask) == mask, nil
}

// Value returns the value of the node.
func (n *Node) Value(ctx context.Context) (*ua.Variant, error) {
	return n.Attribute(ctx, ua.AttributeIDValue)
}

// Attribute returns the attribute of the node. with the given id.
func (n *Node) Attribute(ctx context.Context, attrID ua.AttributeID) (*ua.Variant, error) {
	rv := &ua.ReadValueID{NodeID: n.ID, AttributeID: attrID}
	req := &ua.ReadRequest{NodesToRead: []*ua.ReadValueID{rv}}
	res, err := n.c.Read(ctx, req)
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

// Attributes returns the given node attributes.
func (n *Node) Attributes(ctx context.Context, attrID ...ua.AttributeID) ([]*ua.DataValue, error) {
	req := &ua.ReadRequest{}
	for _, id := range attrID {
		rv := &ua.ReadValueID{NodeID: n.ID, AttributeID: id}
		req.NodesToRead = append(req.NodesToRead, rv)
	}
	res, err := n.c.Read(ctx, req)
	if err != nil {
		return nil, err
	}
	return res.Results, nil
}

// Children returns the child nodes which match the node class mask.
func (n *Node) Children(ctx context.Context, refs uint32, mask ua.NodeClass) ([]*Node, error) {
	if refs == 0 {
		refs = id.HierarchicalReferences
	}
	return n.ReferencedNodes(ctx, refs, ua.BrowseDirectionForward, mask, true)
}

// ReferencedNodes returns the nodes referenced by this node.
func (n *Node) ReferencedNodes(ctx context.Context, refs uint32, dir ua.BrowseDirection, mask ua.NodeClass, includeSubtypes bool) ([]*Node, error) {
	if refs == 0 {
		refs = id.References
	}
	var nodes []*Node
	res, err := n.References(ctx, refs, dir, mask, includeSubtypes)
	if err != nil {
		return nil, err
	}
	for _, r := range res {
		nodes = append(nodes, n.c.Node(r.NodeID.NodeID))
	}
	return nodes, nil
}

// References returns all references for the node.
//
// todo(fs): this is not complete since it only returns the
// todo(fs): top-level reference at this point.
func (n *Node) References(ctx context.Context, refType uint32, dir ua.BrowseDirection, mask ua.NodeClass, includeSubtypes bool) ([]*ua.ReferenceDescription, error) {
	if refType == 0 {
		refType = id.References
	}
	if mask == 0 {
		mask = ua.NodeClassAll
	}

	desc := &ua.BrowseDescription{
		NodeID:          n.ID,
		BrowseDirection: dir,
		ReferenceTypeID: ua.NewNumericNodeID(0, refType),
		IncludeSubtypes: includeSubtypes,
		NodeClassMask:   uint32(mask),
		ResultMask:      uint32(ua.BrowseResultMaskAll),
	}

	req := &ua.BrowseRequest{
		View: &ua.ViewDescription{
			ViewID: ua.NewTwoByteNodeID(0),
		},
		RequestedMaxReferencesPerNode: 0,
		NodesToBrowse:                 []*ua.BrowseDescription{desc},
	}

	resp, err := n.c.Browse(ctx, req)
	if err != nil {
		return nil, err
	}
	return n.browseNext(ctx, resp.Results)
}

func (n *Node) browseNext(ctx context.Context, results []*ua.BrowseResult) ([]*ua.ReferenceDescription, error) {
	refs := results[0].References
	for len(results[0].ContinuationPoint) > 0 {
		req := &ua.BrowseNextRequest{
			ContinuationPoints:        [][]byte{results[0].ContinuationPoint},
			ReleaseContinuationPoints: false,
		}
		resp, err := n.c.BrowseNext(ctx, req)
		if err != nil {
			return nil, err
		}
		results = resp.Results
		refs = append(refs, results[0].References...)
	}
	return refs, nil
}

// TranslateBrowsePathsToNodeIDs translates an array of browseName segments to NodeIDs.
func (n *Node) TranslateBrowsePathsToNodeIDs(ctx context.Context, pathNames []*ua.QualifiedName) (*ua.NodeID, error) {
	req := ua.TranslateBrowsePathsToNodeIDsRequest{
		BrowsePaths: []*ua.BrowsePath{
			{
				StartingNode: n.ID,
				RelativePath: &ua.RelativePath{
					Elements: []*ua.RelativePathElement{},
				},
			},
		}}

	for _, name := range pathNames {
		req.BrowsePaths[0].RelativePath.Elements = append(req.BrowsePaths[0].RelativePath.Elements,
			&ua.RelativePathElement{ReferenceTypeID: ua.NewTwoByteNodeID(id.HierarchicalReferences),
				IsInverse:       false,
				IncludeSubtypes: true,
				TargetName:      name,
			},
		)
	}

	var nodeID *ua.NodeID
	err := n.c.Send(ctx, &req, func(i interface{}) error {
		if resp, ok := i.(*ua.TranslateBrowsePathsToNodeIDsResponse); ok {
			if len(resp.Results) == 0 {
				return ua.StatusBadUnexpectedError
			}

			if resp.Results[0].StatusCode != ua.StatusOK {
				return resp.Results[0].StatusCode
			}

			if len(resp.Results[0].Targets) == 0 {
				return ua.StatusBadUnexpectedError
			}
			nodeID = resp.Results[0].Targets[0].TargetID.NodeID
			return nil
		}
		return ua.StatusBadUnexpectedError
	})
	return nodeID, err
}

// TranslateBrowsePathInNamespaceToNodeID translates a browseName to a NodeID within the same namespace.
func (n *Node) TranslateBrowsePathInNamespaceToNodeID(ctx context.Context, ns uint16, browsePath string) (*ua.NodeID, error) {
	segments := strings.Split(browsePath, ".")
	var names []*ua.QualifiedName
	for _, segment := range segments {
		qn := &ua.QualifiedName{NamespaceIndex: ns, Name: segment}
		names = append(names, qn)
	}
	return n.TranslateBrowsePathsToNodeIDs(ctx, names)
}
