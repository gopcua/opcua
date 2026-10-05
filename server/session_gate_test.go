package server

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/gopcua/opcua/id"
	"github.com/gopcua/opcua/ua"
)

func TestCheckSession(t *testing.T) {
	s := New()
	read := uint16(id.ReadRequest_Encoding_DefaultBinary)
	getEndpoints := uint16(id.GetEndpointsRequest_Encoding_DefaultBinary)

	readReq := func(tok *ua.NodeID) ua.Request {
		return &ua.ReadRequest{RequestHeader: &ua.RequestHeader{AuthenticationToken: tok}}
	}

	// discovery works without a session
	require.Equal(t, ua.StatusOK, s.checkSession(getEndpoints, &ua.GetEndpointsRequest{RequestHeader: &ua.RequestHeader{}}))

	// no / unknown session
	require.Equal(t, ua.StatusBadSessionIDInvalid, s.checkSession(read, &ua.ReadRequest{}))
	require.Equal(t, ua.StatusBadSessionIDInvalid, s.checkSession(read, readReq(ua.NewTwoByteNodeID(0))))

	// created but not activated
	sess := s.sb.NewSession()
	require.Equal(t, ua.StatusBadSessionNotActivated, s.checkSession(read, readReq(sess.AuthTokenID)))

	// activated
	sess.activate(&Identity{TokenType: ua.UserTokenTypeAnonymous})
	require.Equal(t, ua.StatusOK, s.checkSession(read, readReq(sess.AuthTokenID)))
	require.Equal(t, ua.UserTokenTypeAnonymous, sess.Identity().TokenType)

	// closed
	require.NoError(t, s.sb.Close(sess.AuthTokenID))
	require.Equal(t, ua.StatusBadSessionIDInvalid, s.checkSession(read, readReq(sess.AuthTokenID)))
}
