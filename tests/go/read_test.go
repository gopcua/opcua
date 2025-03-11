//go:build integration
// +build integration

package uatest2

import (
	"context"
	"testing"
	"time"

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
		{ua.NewStringNodeID(1, "ro_bool"), true},
		{ua.NewStringNodeID(1, "rw_bool"), true},
		{ua.NewStringNodeID(1, "ro_int32"), int32(5)},
		{ua.NewStringNodeID(1, "rw_int32"), int32(5)},
		// TODO: not implemented in server yet.
		//{ua.NewStringNodeID(2, "array_int32"), []int32{1, 2, 3}},
		//{ua.NewStringNodeID(2, "2d_array_int32"), [][]int32{{1}, {2}, {3}}},
	}

	ctx := context.Background()

	srv := startServer()
	defer srv.Close()

	time.Sleep(2 * time.Second)

	c, err := opcua.NewClient("opc.tcp://localhost:4840", opcua.SecurityMode(ua.MessageSecurityModeNone))
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
				t.Skip("Not implemented in server")
				testRegisteredRead(t, ctx, c, tt.v, tt.id)
			})
		})
	}
}

func testReadPerm(t *testing.T, ctx context.Context, c *opcua.Client, v interface{}, id *ua.NodeID, resultCode ua.StatusCode) {
	t.Helper()

	resp, err := c.Read(ctx, &ua.ReadRequest{
		NodesToRead: []*ua.ReadValueID{
			{NodeID: id},
		},
		TimestampsToReturn: ua.TimestampsToReturnBoth,
	})

	require.NoError(t, err, "Read failed")
	require.Equal(t, resultCode, resp.Results[0].Status, "Status not OK")
	if resultCode == ua.StatusOK {
		require.Equal(t, v, resp.Results[0].Value.Value(), "Results[0].Value not equal")
	}
}

func testRead(t *testing.T, ctx context.Context, c *opcua.Client, v interface{}, id *ua.NodeID) {
	t.Helper()

	resp, err := c.Read(ctx, &ua.ReadRequest{
		NodesToRead: []*ua.ReadValueID{
			{NodeID: id},
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

func TestReadPerms(t *testing.T) {
	tests := []struct {
		id  *ua.NodeID
		err ua.StatusCode
		v   any
	}{
		{ua.NewStringNodeID(1, "NoPermVariable"), ua.StatusOK, int32(742)},
		{ua.NewStringNodeID(1, "ReadWriteVariable"), ua.StatusOK, 12.34},
		{ua.NewStringNodeID(1, "ReadOnlyVariable"), ua.StatusOK, 9.87},
		{ua.NewStringNodeID(1, "NoAccessVariable"), ua.StatusBadUserAccessDenied, nil},
	}

	ctx := context.Background()

	srv := startServer()
	defer srv.Close()

	time.Sleep(2 * time.Second)

	c, err := opcua.NewClient("opc.tcp://localhost:4840", opcua.SecurityMode(ua.MessageSecurityModeNone))
	require.NoError(t, err, "NewClient failed")

	err = c.Connect(ctx)
	require.NoError(t, err, "Connect failed")
	defer c.Close(ctx)

	for _, tt := range tests {
		t.Run(tt.id.String(), func(t *testing.T) {
			t.Run("Read", func(t *testing.T) {
				testReadPerm(t, ctx, c, tt.v, tt.id, tt.err)
			})
			t.Run("RegisteredRead", func(t *testing.T) {
				t.Skip("Not implemented in server")
				testRegisteredRead(t, ctx, c, tt.v, tt.id)
			})
		})
	}
}
