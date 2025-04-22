//go:build integration

package uatest

import (
	"context"
	"testing"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/ua"
	"github.com/stretchr/testify/require"
)

// TestRead performs an integration test to read values
// from an OPC/UA server.
func TestRead(t *testing.T) {
	tests := []struct {
		id *ua.NodeID
		v  interface{}
	}{
		{ua.NewStringNodeID(2, "ro_bool"), true},
		{ua.NewStringNodeID(2, "rw_bool"), true},
		{ua.NewStringNodeID(2, "ro_int32"), int32(5)},
		{ua.NewStringNodeID(2, "rw_int32"), int32(5)},
		{ua.NewStringNodeID(2, "array_int32"), []int32{1, 2, 3}},
		{ua.NewStringNodeID(2, "2d_array_int32"), [][]int32{{1}, {2}, {3}}},
	}

	ctx := context.Background()

	srv := NewPythonServer("rw_server.py")
	defer srv.Close()

	c, err := opcua.NewClient(srv.Endpoint, srv.Opts...)
	require.NoError(t, err, "NewClient failed")

	err = c.Connect(ctx)
	require.NoError(t, err, "Connect failed")
	defer c.Close(ctx)

	for _, tt := range tests {
		t.Run(tt.id.String(), func(t *testing.T) {
			t.Run("Read", func(t *testing.T) {
				testRead(t, ctx, c, tt.v, tt.id)
			})
			t.Run("RegisteredRead", func(t *testing.T) {
				testRegisteredRead(t, ctx, c, tt.v, tt.id)
			})
		})
	}
}

func testRead(t *testing.T, ctx context.Context, c *opcua.Client, v interface{}, id *ua.NodeID) {
	t.Helper()

	resp, err := c.Read(ctx, &ua.ReadRequest{
		NodesToRead: []*ua.ReadValueID{
			&ua.ReadValueID{NodeID: id},
		},
		TimestampsToReturn: ua.TimestampsToReturnBoth,
	})
	require.NoError(t, err, "Read failed")
	require.Equal(t, ua.StatusOK, resp.Results[0].Status, "Status not OK")
	require.Equal(t, v, resp.Results[0].Value.Value(), "Results[0].Value not equal")
}

func testRegisteredRead(t *testing.T, ctx context.Context, c *opcua.Client, v interface{}, id *ua.NodeID) {
	t.Helper()

	resp, err := c.RegisterNodes(ctx, &ua.RegisterNodesRequest{
		NodesToRegister: []*ua.NodeID{id},
	})
	require.NoError(t, err, "RegisterNodes failed")

	testRead(t, ctx, c, v, resp.RegisteredNodeIDs[0])
	testRead(t, ctx, c, v, resp.RegisteredNodeIDs[0])
	testRead(t, ctx, c, v, resp.RegisteredNodeIDs[0])
	testRead(t, ctx, c, v, resp.RegisteredNodeIDs[0])
	testRead(t, ctx, c, v, resp.RegisteredNodeIDs[0])

	_, err = c.UnregisterNodes(ctx, &ua.UnregisterNodesRequest{
		NodesToUnregister: []*ua.NodeID{id},
	})
	require.NoError(t, err, "UnregisterNodes failed")
}
