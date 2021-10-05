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
func TestRead_UnknowNodeID(t *testing.T) {

	unknownNodeId := ua.NewStringNodeID(2, "ComplexZero")

	srv := NewServer("read_unknow_node_id_server.py")
	defer srv.Close()

	c := opcua.NewClient(srv.Endpoint, srv.Opts...)
	if err := c.Connect(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	resp, err := c.Read(&ua.ReadRequest{NodesToRead: []*ua.ReadValueID{{NodeID: unknownNodeId}}})
	if err != nil {
		t.Error(err)
	}
	if resp.Results[0].Status == ua.StatusOK {
		t.Errorf("got StatusOK for a node we don't know")
	}

	resp, err = c.Read(&ua.ReadRequest{NodesToRead: []*ua.ReadValueID{
		{NodeID: ua.NewNumericNodeID(0, id.Server_ServerStatus_State), AttributeID: ua.AttributeIDValue}}})
	if err != nil {
		t.Error(err)
	}
}
