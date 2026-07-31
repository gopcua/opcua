//go:build integration
// +build integration

package uatest2

import (
	"testing"
	"time"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/ua"
	"github.com/stretchr/testify/require"
)

// TestWriteToUnknownNamespace_Part4_Write verifies that a Write whose target
// NodeId references a namespace index that does not exist on the server returns
// an operation-level Bad status, and never terminates the server.
//
// OPC UA Part 4 v1.05 §5.10.4 (Write): "This Service is used to write values to
// one or more Attributes of one or more Nodes." Each NodesToWrite entry yields
// its own operation-level StatusCode in Results; a target that does not exist in
// the server AddressSpace is reported as Bad_NodeIdUnknown ("The node id refers
// to a node that does not exist in the server address space."). Bad_NodeNotInView
// is a View/Browse concept (§5.8) and does not apply to the Write Service.
//
// Cross-impl: open62541 (editNode -> UA_STATUSCODE_BADNODEIDUNKNOWN) and
// UA-.NETStandard (MasterNodeManager.Write -> BadNodeIdUnknown) both return
// Bad_NodeIdUnknown for this case.
//
// Regression for #841: AttributeService.Write omitted `continue` after a failed
// namespace lookup and dereferenced a nil NameSpace, panicking the server.
func TestWriteToUnknownNamespace_Part4_Write(t *testing.T) {
	ctx := t.Context()

	srv := startServer(ctx)
	defer srv.Close(ctx)

	time.Sleep(2 * time.Second)

	c, err := opcua.NewClient("opc.tcp://localhost:4840", opcua.SecurityMode(ua.MessageSecurityModeNone))
	require.NoError(t, err, "NewClient failed")

	err = c.Connect(ctx)
	require.NoError(t, err, "Connect failed")
	defer c.Close(ctx)

	// namespace 132 does not exist on the test server.
	testWrite(t, ctx, c, ua.StatusBadNodeIDUnknown, &ua.WriteRequest{
		NodesToWrite: []*ua.WriteValue{
			{
				NodeID:      ua.NewNumericNodeID(132, 101),
				AttributeID: ua.AttributeIDValue,
				Value: &ua.DataValue{
					EncodingMask: ua.DataValueValue,
					Value:        ua.MustVariant(int32(42)),
				},
			},
		},
	})
}
