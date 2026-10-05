//go:build integration
// +build integration

// Copyright 2018-2026 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uatest2

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/id"
	"github.com/gopcua/opcua/server"
	"github.com/gopcua/opcua/ua"
)

const authTestPort = 48680

var authTestPolicies = []string{
	"None",
	"Basic128Rsa15",
	"Basic256",
	"Basic256Sha256",
	"Aes128_Sha256_RsaOaep",
	"Aes256_Sha256_RsaPss",
}

type authTestEnv struct {
	addr              string
	endpoints         []*ua.EndpointDescription
	cliCert           []byte
	cliKey            *rsa.PrivateKey
	trustedUserCert   []byte
	trustedUserKey    *rsa.PrivateKey
	untrustedUserCert []byte
	untrustedUserKey  *rsa.PrivateKey
}

func startAuthServer(t *testing.T, port int, anonymous bool) *authTestEnv {
	t.Helper()
	srvCert, srvKey := genSelfSignedCert(t, "urn:gopcua:auth:server")
	cliCert, cliKey := genSelfSignedCert(t, "urn:gopcua:auth:client")
	userCert, userKey := genSelfSignedCert(t, "urn:gopcua:auth:user")
	badCert, badKey := genSelfSignedCert(t, "urn:gopcua:auth:untrusted")

	trusted, err := x509.ParseCertificate(userCert)
	require.NoError(t, err)
	pool := x509.NewCertPool()
	pool.AddCert(trusted)

	opts := []server.Option{
		server.EndPoint("localhost", port),
		server.PrivateKey(srvKey),
		server.Certificate(srvCert),
		server.UserNameAuth(server.StaticUsers(map[string]string{"alice": "s3cret"})),
		server.X509Auth(server.TrustedUserCerts(pool)),
	}
	if anonymous {
		opts = append(opts, server.EnableAuthMode(ua.UserTokenTypeAnonymous))
	}
	for _, p := range authTestPolicies {
		if p == "None" {
			opts = append(opts, server.EnableSecurity(p, ua.MessageSecurityModeNone))
			continue
		}
		opts = append(opts,
			server.EnableSecurity(p, ua.MessageSecurityModeSign),
			server.EnableSecurity(p, ua.MessageSecurityModeSignAndEncrypt),
		)
	}

	s := server.New(opts...)
	ns := server.NewNodeNameSpace(s, "AuthTest")
	s.AddNamespace(ns)
	root, _ := s.Namespace(0)
	root.Objects().AddRef(ns.Objects(), id.HasComponent, true)
	n := ns.AddNewVariableStringNode("value", int32(42))
	ns.Objects().AddRef(n, id.HasComponent, true)

	require.NoError(t, s.Start(context.Background()))
	t.Cleanup(func() { s.Close() })

	addr := fmt.Sprintf("opc.tcp://localhost:%d", port)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	var eps []*ua.EndpointDescription
	for {
		eps, err = opcua.GetEndpoints(ctx, addr)
		if err == nil {
			break
		}
		select {
		case <-ctx.Done():
			require.NoError(t, err, "server never became ready")
		case <-time.After(50 * time.Millisecond):
		}
	}

	return &authTestEnv{
		addr:              addr,
		endpoints:         eps,
		cliCert:           cliCert,
		cliKey:            cliKey,
		trustedUserCert:   userCert,
		trustedUserKey:    userKey,
		untrustedUserCert: badCert,
		untrustedUserKey:  badKey,
	}
}

func (e *authTestEnv) endpoint(t *testing.T, policy string, mode ua.MessageSecurityMode) *ua.EndpointDescription {
	t.Helper()
	uri := ua.FormatSecurityPolicyURI(policy)
	for _, ep := range e.endpoints {
		if ep.SecurityPolicyURI == uri && ep.SecurityMode == mode {
			return ep
		}
	}
	t.Fatalf("no endpoint for %s/%s", policy, mode)
	return nil
}

// connect connects with the given auth options and reads a value to prove
// the session is usable.
func (e *authTestEnv) connect(t *testing.T, ep *ua.EndpointDescription, tokenType ua.UserTokenType, auth ...opcua.Option) error {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	opts := append([]opcua.Option{}, auth...)
	opts = append(opts, opcua.SecurityFromEndpoint(ep, tokenType))
	if ep.SecurityMode != ua.MessageSecurityModeNone {
		opts = append(opts, opcua.Certificate(e.cliCert), opcua.PrivateKey(e.cliKey))
	}
	c, err := opcua.NewClient(e.addr, opts...)
	require.NoError(t, err)
	if err := c.Connect(ctx); err != nil {
		return err
	}
	defer c.Close(ctx)

	resp, err := c.Read(ctx, &ua.ReadRequest{
		NodesToRead: []*ua.ReadValueID{{NodeID: ua.NewStringNodeID(1, "value"), AttributeID: ua.AttributeIDValue}},
	})
	if err != nil {
		return err
	}
	require.Equal(t, ua.StatusOK, resp.Results[0].Status)
	require.Equal(t, int32(42), resp.Results[0].Value.Value())
	return nil
}

func requireStatus(t *testing.T, want ua.StatusCode, err error) {
	t.Helper()
	require.Error(t, err)
	var got ua.StatusCode
	require.True(t, errors.As(err, &got), "expected a status code, got %T: %v", err, err)
	require.Equal(t, want, got, "got %v", err)
}

// TestUserNameAndX509Auth exercises UserName and X509 user authentication
// over every security policy and mode the server offers.
//
// OPC UA Part 4 v1.05 §5.6.3 (ActivateSession): the server validates the
// userIdentityToken and returns Bad_UserAccessDenied / Bad_IdentityTokenInvalid /
// Bad_IdentityTokenRejected / Bad_UserSignatureInvalid on failure.
func TestUserNameAndX509Auth(t *testing.T) {
	env := startAuthServer(t, authTestPort, false)

	type mode struct {
		policy string
		mode   ua.MessageSecurityMode
	}
	var modes []mode
	for _, p := range authTestPolicies {
		if p == "None" {
			modes = append(modes, mode{p, ua.MessageSecurityModeNone})
			continue
		}
		modes = append(modes, mode{p, ua.MessageSecurityModeSign}, mode{p, ua.MessageSecurityModeSignAndEncrypt})
	}

	for _, m := range modes {
		t.Run(m.policy+"/"+m.mode.String(), func(t *testing.T) {
			ep := env.endpoint(t, m.policy, m.mode)

			t.Run("username ok", func(t *testing.T) {
				require.NoError(t, env.connect(t, ep, ua.UserTokenTypeUserName, opcua.AuthUsername("alice", "s3cret")))
			})
			t.Run("username wrong password", func(t *testing.T) {
				requireStatus(t, ua.StatusBadUserAccessDenied,
					env.connect(t, ep, ua.UserTokenTypeUserName, opcua.AuthUsername("alice", "wrong")))
			})
			t.Run("username unknown user", func(t *testing.T) {
				requireStatus(t, ua.StatusBadUserAccessDenied,
					env.connect(t, ep, ua.UserTokenTypeUserName, opcua.AuthUsername("mallory", "s3cret")))
			})
			t.Run("certificate ok", func(t *testing.T) {
				require.NoError(t, env.connect(t, ep, ua.UserTokenTypeCertificate,
					opcua.AuthCertificate(env.trustedUserCert), opcua.AuthPrivateKey(env.trustedUserKey)))
			})
			t.Run("certificate untrusted", func(t *testing.T) {
				requireStatus(t, ua.StatusBadUserAccessDenied,
					env.connect(t, ep, ua.UserTokenTypeCertificate,
						opcua.AuthCertificate(env.untrustedUserCert), opcua.AuthPrivateKey(env.untrustedUserKey)))
			})
			t.Run("certificate wrong private key", func(t *testing.T) {
				requireStatus(t, ua.StatusBadUserSignatureInvalid,
					env.connect(t, ep, ua.UserTokenTypeCertificate,
						opcua.AuthCertificate(env.trustedUserCert), opcua.AuthPrivateKey(env.untrustedUserKey)))
			})
			t.Run("anonymous not enabled", func(t *testing.T) {
				requireStatus(t, ua.StatusBadIdentityTokenRejected,
					env.connect(t, ep, ua.UserTokenTypeAnonymous))
			})
		})
	}
}

// TestAnonymousAuthStillWorks makes sure enabling user auth does not break
// anonymous access when it is explicitly enabled.
func TestAnonymousAuthStillWorks(t *testing.T) {
	env := startAuthServer(t, authTestPort+1, true)
	ep := env.endpoint(t, "None", ua.MessageSecurityModeNone)
	require.NoError(t, env.connect(t, ep, ua.UserTokenTypeAnonymous))
	require.NoError(t, env.connect(t, ep, ua.UserTokenTypeUserName, opcua.AuthUsername("alice", "s3cret")))
}
