// Copyright 2018-2026 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package server

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/gopcua/opcua/ua"
	"github.com/gopcua/opcua/uacp"
	"github.com/gopcua/opcua/uasc"
)

// newTestSecureChannel returns an unopened security policy None channel. It
// is enough for handlers that only look at the channel's security config.
func newTestSecureChannel(t *testing.T) *uasc.SecureChannel {
	t.Helper()
	// ActivateSession never touches the connection
	sc, err := uasc.NewServerSecureChannel("opc.tcp://localhost:4840", &uacp.Conn{}, &uasc.Config{
		SecurityPolicyURI: ua.SecurityPolicyURINone,
		SecurityMode:      ua.MessageSecurityModeNone,
	}, make(chan error, 1), 1, 1, 1)
	require.NoError(t, err)
	return sc
}

// TestActivateSessionConcurrent runs ActivateSession concurrently on the same
// session, as the server does since activation is handled off the message
// loop. Run with -race.
func TestActivateSessionConcurrent(t *testing.T) {
	s := New(EnableAuthMode(ua.UserTokenTypeAnonymous))
	svc := &SessionService{srv: s}
	sess := s.sb.NewSession()

	sc := newTestSecureChannel(t)
	req := func() *ua.ActivateSessionRequest {
		return &ua.ActivateSessionRequest{
			RequestHeader:     &ua.RequestHeader{AuthenticationToken: sess.AuthTokenID},
			ClientSignature:   &ua.SignatureData{},
			UserIdentityToken: ua.NewExtensionObject(&ua.AnonymousIdentityToken{PolicyID: "anonymous_none"}),
		}
	}

	var wg sync.WaitGroup
	for range 16 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := svc.ActivateSession(sc, req(), 1)
			require.NoError(t, err)
			require.NotEmpty(t, resp.(*ua.ActivateSessionResponse).ServerNonce)
			_ = s.checkSession(0, &ua.ReadRequest{RequestHeader: &ua.RequestHeader{AuthenticationToken: sess.AuthTokenID}})
		}()
	}
	wg.Wait()
	require.True(t, sess.Activated())
	require.Equal(t, ua.UserTokenTypeAnonymous, sess.Identity().TokenType)
}
