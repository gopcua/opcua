//go:build integration

package uatest

import (
	"context"
	"testing"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/ua"
	"github.com/stretchr/testify/require"
)

// TestWrite performs an integration test to first write
// and then read values from an OPC/UA server.
func TestWrite(t *testing.T) {
	tests := []struct {
		id     *ua.NodeID
		v      interface{}
		status ua.StatusCode
	}{
		// happy flows
		{ua.NewStringNodeID(2, "rw_bool"), false, ua.StatusOK},
		{ua.NewStringNodeID(2, "rw_int32"), int32(9), ua.StatusOK},

		// error flows
		{ua.NewStringNodeID(2, "ro_bool"), false, ua.StatusBadUserAccessDenied},
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
			testWrite(t, ctx, c, tt.status, &ua.WriteRequest{
				NodesToWrite: []*ua.WriteValue{
					&ua.WriteValue{
						NodeID:      tt.id,
						AttributeID: ua.AttributeIDValue,
						Value: &ua.DataValue{
							EncodingMask: ua.DataValueValue,
							Value:        ua.MustVariant(tt.v),
						},
					},
				},
			})

			// skip read tests if the write is expected to fail
			if tt.status != ua.StatusOK {
				return
			}

			testRead(t, ctx, c, tt.v, tt.id)
		})
	}
}

func testWrite(t *testing.T, ctx context.Context, c *opcua.Client, status ua.StatusCode, req *ua.WriteRequest) {
	t.Helper()

	resp, err := c.Write(ctx, req)
	require.NoError(t, err, "Write failed")
	require.Equal(t, status, resp.Results[0], "Write result not equal")
}
