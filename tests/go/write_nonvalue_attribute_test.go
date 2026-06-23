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

// TestWriteNonValueAttribute_Part4_Write verifies that writing a non-Value
// attribute (here Description) of a writable node returns Good, not a Bad
// status.
//
// OPC UA Part 4 v1.05 §5.10.4 (Write): "This Service is used to write values to
// one or more Attributes of one or more Nodes." Each NodesToWrite entry yields
// its own operation-level StatusCode; a permitted write returns Good. Writability
// of a non-Value attribute is governed by the WriteMask attribute (Part 3 §5.2.7);
// AccessLevel governs the Value attribute only.
//
// Cross-impl: open62541 (copyAttributeIntoNode) and UA-.NETStandard (NodeState
// attribute writes) both store the attribute and return Good.
//
// Regression for #843: Node.SetAttribute stored a non-Value attribute but then
// returned Bad_NodeAttributesInvalid, which NodeNameSpace.SetAttribute surfaced
// to the client as Bad_AttributeIdInvalid.
func TestWriteNonValueAttribute_Part4_Write(t *testing.T) {
	ctx := context.Background()

	srv := startServer()
	defer srv.Close()

	time.Sleep(2 * time.Second)

	c, err := opcua.NewClient("opc.tcp://localhost:4840", opcua.SecurityMode(ua.MessageSecurityModeNone))
	require.NoError(t, err, "NewClient failed")

	err = c.Connect(ctx)
	require.NoError(t, err, "Connect failed")
	defer c.Close(ctx)

	// Description is a non-Value attribute on a writable node (ns=1).
	testWrite(t, ctx, c, ua.StatusOK, &ua.WriteRequest{
		NodesToWrite: []*ua.WriteValue{
			{
				NodeID:      ua.NewStringNodeID(1, "ReadWriteVariable"),
				AttributeID: ua.AttributeIDDescription,
				Value: &ua.DataValue{
					EncodingMask: ua.DataValueValue,
					Value:        ua.MustVariant(ua.NewLocalizedText("set via write")),
				},
			},
		},
	})
}
