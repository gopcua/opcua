//go:build integration
// +build integration

package uatest

import (
	"context"
	"testing"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/id"
	"github.com/gopcua/opcua/ua"
)

// TestRead performs an integration test to read values
// from an OPC/UA server.
func TestReadUnknowNodeID(t *testing.T) {
	ctx := context.Background()

	srv := NewServer("read_unknow_node_id_server.py")
	defer srv.Close()

	c, err := opcua.NewClient(srv.Endpoint, srv.Opts...)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Connect(ctx); err != nil {
		t.Fatal(err)
	}
	defer c.Close(ctx)

	// read node with unknown extension object
	// This should be OK
	nodeWithUnknownType := ua.NewStringNodeID(2, "IntValZero")
	//unknownType := ua.NewStringNodeID(2, "IntValType")

	// read should fail b/c node uses an unknown extension object
	resp, err := c.Read(ctx, &ua.ReadRequest{
		NodesToRead: []*ua.ReadValueID{
			{NodeID: nodeWithUnknownType},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	// expect data type unknown
	if got, want := resp.Results[0].Status, ua.StatusBadDataTypeIDUnknown; got != want {
		t.Errorf("got status %v want %v for a node with an unknown type", got, want)
	}

	// register the data type from the type
	err = c.RegisterExtensionObjectFromServer(ctx, nodeWithUnknownType)
	if err != nil {
		t.Fatal(err)
	}

	resp, err = c.Read(ctx, &ua.ReadRequest{
		NodesToRead: []*ua.ReadValueID{
			{NodeID: nodeWithUnknownType},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	// should work now
	if got, want := resp.Results[0].Status, ua.StatusOK; got != want {
		t.Errorf("got status %v want %v", got, want)
	}

	// check that the connection is still usable by reading another node.
	_, err = c.Read(ctx, &ua.ReadRequest{
		NodesToRead: []*ua.ReadValueID{
			{
				NodeID: ua.NewNumericNodeID(0, id.Server_ServerStatus_State),
			},
		},
	})
	if err != nil {
		t.Error(err)
	}
}
