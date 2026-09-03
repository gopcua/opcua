package server

import (
	"testing"

	"github.com/gopcua/opcua/ua"
	"github.com/stretchr/testify/require"
)

// TestRegisterNodesIdentity checks that RegisterNodes echoes the requested
// node ids back unchanged (the spec-legal identity behaviour) instead of
// failing with BadServiceUnsupported, and that UnregisterNodes succeeds.
func TestRegisterNodesIdentity(t *testing.T) {
	vs := &ViewService{srv: New()}

	ids := []*ua.NodeID{
		ua.NewNumericNodeID(1, 1001),
		ua.NewStringNodeID(2, "temperature"),
	}

	resp, err := vs.RegisterNodes(nil, &ua.RegisterNodesRequest{
		RequestHeader:   &ua.RequestHeader{},
		NodesToRegister: ids,
	}, 0)
	require.NoError(t, err)

	rn, ok := resp.(*ua.RegisterNodesResponse)
	require.True(t, ok, "expected *ua.RegisterNodesResponse, got %T", resp)
	require.Equal(t, ua.StatusOK, rn.ResponseHeader.ServiceResult)
	require.Equal(t, ids, rn.RegisteredNodeIDs, "RegisterNodes must return the ids unchanged")

	uresp, err := vs.UnregisterNodes(nil, &ua.UnregisterNodesRequest{
		RequestHeader:     &ua.RequestHeader{},
		NodesToUnregister: ids,
	}, 0)
	require.NoError(t, err)

	un, ok := uresp.(*ua.UnregisterNodesResponse)
	require.True(t, ok, "expected *ua.UnregisterNodesResponse, got %T", uresp)
	require.Equal(t, ua.StatusOK, un.ResponseHeader.ServiceResult)
}
