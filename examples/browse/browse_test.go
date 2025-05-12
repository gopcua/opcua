package main

import (
	"context"
	"testing"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/id"
	"github.com/gopcua/opcua/internal/ualog"
	"github.com/gopcua/opcua/server"
	"github.com/gopcua/opcua/ua"
)

func TestBrowse(t *testing.T) {
	ctx := context.Background()

	// start the server
	s := server.New(
		server.EndPoint("localhost", 4840),
	)
	populateServer(s)
	if err := s.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	// prepare the client
	c, err := opcua.NewClient("opc.tcp://localhost:4840")
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Connect(ctx); err != nil {
		t.Fatal(err)
	}
	defer c.Close(ctx)

	// browse the nodes
	nodeList, err := browse(
		ctx,
		c.Node(ua.MustParseNodeID("i=84")),
		"",
		maxDepth-3, // faster test with reduced depth
	)
	if err != nil {
		t.Fatal(err)
	}

	// ensure that the TestVar1 node was found
	found := false
	for _, n := range nodeList {
		if n.BrowseName == "TestVar1" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("TestVar1 not found in nodeList: %v", nodeList)
	}
}

func populateServer(s *server.Server) {
	// When the server is created, it will automatically create namespace 0 and populate it with
	// the core opc ua nodes.

	// add the namespaces to the server, and add a reference to them (otherwise browsing will not find it).
	// here we are choosing to add the namespaces to the root/object folder
	// to do this we first need to get the root namespace object folder so we
	// get the object node
	root_ns, _ := s.Namespace(0)
	root_obj_node := root_ns.Objects()

	// Now we'll add a node namespace.
	nodeNS := server.NewNodeNameSpace(s, "NodeNamespace")
	ualog.Info("Node Namespace added", "index", nodeNS.ID())

	// add the reference for this namespace's root object folder to the server's root object folder
	// but you can add a reference to whatever node(s) you need
	nns_obj := nodeNS.Objects()
	root_obj_node.AddRef(nns_obj, id.HasComponent, true)

	// Create some nodes for it.  Here we are usin gthe AddNewVariableNode utility function to create a new variable node
	// with an integer node ID that is automatically assigned. (ns=<namespace id>,s=<auto assigned>)
	// be sure to add the reference to the node somewhere if desired, or clients won't be able to browse it.
	var1 := nodeNS.AddNewVariableNode("TestVar1", float32(123.45))
	nns_obj.AddRef(var1, id.HasComponent, true)
}
