// Copyright 2018-2020 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package server

import (
	"github.com/gopcua/opcua/id"
	"github.com/gopcua/opcua/ua"
)

var (
	ObjectsFolder = ua.NewNumericNodeID(0, id.ObjectsFolder)
	RootFolder    = ua.NewNumericNodeID(0, id.RootFolder)
)

// These are all the functions a namespace needs in order to provide nodes into the server
type NameSpace interface {
	// Name of the namespace.  Per the standard it should be an URI.
	Name() string

	// This function should create a new node
	AddNode(n *Node) *Node

	// This function should lookup and return the node indicated by the Node ID
	Node(id *ua.NodeID) *Node

	// This function should return the base Objects node that contains other nodes
	Objects() *Node

	// This function should return the root node
	Root() *Node

	// This is the function to list all available nodes to the client that is browsing.
	// The BrowseDescription has the root node of the browse and what kind of nodes the
	// client is looking for.  The Browse Result should have the list of matching nodes.
	Browse(req *ua.BrowseDescription) *ua.BrowseResult

	// ID and SetID are the namespace ID number of this namespace.  When you add it to the server
	// with srv.AddNamespace(xxx) it will set these for you.
	ID() uint16
	SetID(uint16)

	// These are the functions for reading and writing arbitrary attributes.  The most common
	// is the value attribute, but many clients also read the datatype and description attributes.
	// as well as attributes related to array bounds
	Attribute(*ua.NodeID, ua.AttributeID) *ua.DataValue
	SetAttribute(*ua.NodeID, ua.AttributeID, *ua.DataValue) ua.StatusCode
}
