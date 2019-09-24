// +build integration

package uatest

import (
	"context"
	"testing"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/ua"
	"github.com/pascaldekloe/goe/verify"
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
	}

	srv := NewServer("rw_server.py")
	defer srv.Close()

	c := opcua.NewClient(srv.Endpoint, srv.Opts...)
	if err := c.Connect(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	for _, tt := range tests {
		t.Run(tt.id.String(), func(t *testing.T) {
			testRead(t, c, tt.v, &ua.ReadRequest{
				NodesToRead: []*ua.ReadValueID{
					&ua.ReadValueID{NodeID: tt.id},
				},
				TimestampsToReturn: ua.TimestampsToReturnBoth,
			})
		})
	}
}

func testRead(t *testing.T, c *opcua.Client, v interface{}, req *ua.ReadRequest) {
	t.Helper()

	resp, err := c.Read(req)
	if err != nil {
		t.Fatalf("Read failed: %s", err)
	}
	if resp.Results[0].Status != ua.StatusOK {
		t.Fatalf("Status not OK: %v", resp.Results[0].Status)
	}
	if got, want := resp.Results[0].Value.Value(), v; !verify.Values(t, "", got, want) {
		t.Fail()
	}
}
