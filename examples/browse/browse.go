// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"text/tabwriter"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/debug"
	"github.com/gopcua/opcua/id"
	"github.com/gopcua/opcua/ua"
	"github.com/pkg/errors"
)

type NodeDef struct {
	NodeID      *ua.NodeID
	NodeClass   ua.NodeClass
	BrowseName  string
	Description string
	AccessLevel ua.AccessLevelType
	Path        string
	DataType    string
	Writable    bool
}

func (n NodeDef) String() string {
	return fmt.Sprintf("%s\t%s\t%s\t%v\t%s", n.NodeID, n.Path, n.DataType, n.Writable, n.Description)
}

func join(a, b string) string {
	if a == "" {
		return b
	}
	return a + "." + b
}

func browse(n *opcua.Node, path string, level int) ([]NodeDef, error) {
	// fmt.Printf("node:%s path:%q level:%d\n", n, path, level)
	if level > 10 {
		return nil, nil
	}

	attrs, err := n.Attributes(ua.AttributeIDNodeClass, ua.AttributeIDBrowseName, ua.AttributeIDDescription, ua.AttributeIDAccessLevel, ua.AttributeIDDataType)
	if err != nil {
		return nil, err
	}

	var def = NodeDef{
		NodeID: n.ID,
	}

	switch err := attrs[0].Status; err {
	case ua.StatusOK:
		def.NodeClass = ua.NodeClass(attrs[0].Value.Int())
	default:
		return nil, err
	}

	switch err := attrs[1].Status; err {
	case ua.StatusOK:
		def.BrowseName = attrs[1].Value.String()
	default:
		return nil, err
	}

	switch err := attrs[2].Status; err {
	case ua.StatusOK:
		def.Description = attrs[2].Value.String()
	case ua.StatusBadAttributeIDInvalid:
		// ignore
	default:
		return nil, err
	}

	switch err := attrs[3].Status; err {
	case ua.StatusOK:
		def.AccessLevel = ua.AccessLevelType(attrs[3].Value.Int())
		def.Writable = def.AccessLevel&ua.AccessLevelTypeCurrentWrite == ua.AccessLevelTypeCurrentWrite
	case ua.StatusBadAttributeIDInvalid:
		// ignore
	default:
		return nil, err
	}

	switch err := attrs[4].Status; err {
	case ua.StatusOK:
		switch v := attrs[4].Value.NodeID().IntID(); v {
		case id.DateTime:
			def.DataType = "DateTime"
		case id.Boolean:
			def.DataType = "Boolean"
		case id.SByte:
			def.DataType = "SByte"
		case id.Int16:
			def.DataType = "Int16"
		case id.Int32:
			def.DataType = "Int32"
		case id.Byte:
			def.DataType = "Byte"
		case id.UInt16:
			def.DataType = "UInt16"
		case id.UInt32:
			def.DataType = "UInt32"
		case id.UtcTime:
			def.DataType = "UtcTime"
		case id.String:
			def.DataType = "String"
		case id.Float:
			def.DataType = "Float"
		default:
			def.DataType = attrs[4].Value.NodeID().String()
		}
	case ua.StatusBadAttributeIDInvalid:
		// ignore
	default:
		return nil, err
	}

	def.Path = join(path, def.BrowseName)

	var nodes []NodeDef
	if def.NodeClass == ua.NodeClassVariable {
		nodes = append(nodes, def)
	}

	browseChildren := func(refType uint32) error {
		refs, err := n.ReferencedNodes(refType, ua.BrowseDirectionForward, ua.NodeClassAll, true)
		if err != nil {
			return errors.Wrapf(err, "References: %d", refType)
		}
		// fmt.Printf("found %d child refs\n", len(refs))
		for _, rn := range refs {
			children, err := browse(rn, def.Path, level+1)
			if err != nil {
				return errors.Wrapf(err, "browse children")
			}
			nodes = append(nodes, children...)
		}
		return nil
	}

	if err := browseChildren(id.HasComponent); err != nil {
		return nil, err
	}
	if err := browseChildren(id.Organizes); err != nil {
		return nil, err
	}
	return nodes, nil
}

func main() {
	endpoint := flag.String("endpoint", "opc.tcp://localhost:4840", "OPC UA Endpoint URL")
	nodeID := flag.String("node", "", "node id for the root node")
	flag.BoolVar(&debug.Enable, "debug", false, "enable debug logging")
	flag.Parse()
	log.SetFlags(0)

	ctx := context.Background()

	c := opcua.NewClient(*endpoint)
	if err := c.Connect(ctx); err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	id, err := ua.ParseNodeID(*nodeID)
	if err != nil {
		log.Fatalf("invalid node id: %s", err)
	}

	nodeList, err := browse(c.Node(id), "", 0)
	if err != nil {
		log.Fatal(err)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "node_id\tpath\ttype\twritable\tdescription")
	for _, s := range nodeList {
		fmt.Fprintln(w, s)
	}
	w.Flush()
}
