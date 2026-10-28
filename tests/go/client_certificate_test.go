//go:build integration
// +build integration

// Copyright 2018-2026 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uatest2

import (
	"context"
	"crypto/rsa"
	"errors"
	"fmt"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/server"
	"github.com/gopcua/opcua/ua"
	"github.com/gopcua/opcua/uacp"
)

// certRecorder is a verifier that records every certificate it is asked about
// and refuses all of them while refuse is set.
type certRecorder struct {
	mu     sync.Mutex
	seen   [][]byte
	refuse bool
}

func (r *certRecorder) verify(cert []byte) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.seen = append(r.seen, cert)
	if r.refuse {
		return errors.New("not in the trusted list")
	}
	return nil
}

func (r *certRecorder) asked() [][]byte {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.seen)
}

func (r *certRecorder) setRefuse(v bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.refuse = v
}

type clientIdentity struct {
	cert []byte
	key  *rsa.PrivateKey
}

// startCertServer starts a server on port with the given policy and verifier.
func startCertServer(t *testing.T, port int, policy string, mode ua.MessageSecurityMode, verify func([]byte) error) *server.Server {
	t.Helper()
	srvCert, srvKey := genSelfSignedCert(t, "urn:gopcua:certtest:server")
	opts := []server.Option{
		server.EnableSecurity(policy, mode),
		server.EnableAuthMode(ua.UserTokenTypeAnonymous),
		server.EndPoint("localhost", port),
		server.PrivateKey(srvKey),
		server.Certificate(srvCert),
	}
	if verify != nil {
		opts = append(opts, server.ClientCertificateVerifier(verify))
	}
	s := server.New(opts...)
	require.NoError(t, s.Start(context.Background()))
	t.Cleanup(func() { s.Close() })
	return s
}

// connectCertClient discovers the endpoint for policy and mode, presenting
// cl's certificate if any, and returns the client or the error Connect gave.
func connectCertClient(t *testing.T, port int, policy string, mode ua.MessageSecurityMode, cl *clientIdentity) (*opcua.Client, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	addr := fmt.Sprintf("opc.tcp://localhost:%d", port)

	// poll until the listener accepts a discovery request, bounded by ctx
	var (
		eps []*ua.EndpointDescription
		err error
	)
	for {
		eps, err = opcua.GetEndpoints(ctx, addr)
		if err == nil {
			break
		}
		select {
		case <-ctx.Done():
			require.NoError(t, err, "discovery (server never became ready)")
		case <-time.After(50 * time.Millisecond):
		}
	}
	ep, err := opcua.SelectEndpoint(eps, policy, mode)
	require.NoError(t, err)

	opts := []opcua.Option{
		opcua.SecurityFromEndpoint(ep, ua.UserTokenTypeAnonymous),
		opcua.RequestTimeout(3 * time.Second),
		opcua.AutoReconnect(false),
	}
	if cl != nil {
		opts = append(opts, opcua.Certificate(cl.cert), opcua.PrivateKey(cl.key))
	}
	c, err := opcua.NewClient(addr, opts...)
	require.NoError(t, err)
	if err := c.Connect(ctx); err != nil {
		return nil, err
	}
	t.Cleanup(func() { c.Close(context.Background()) })
	return c, nil
}

// TestClientCertificateVerifierAtOpenSecureChannel checks that on a
// SignAndEncrypt channel a refused certificate is answered with an ERR message
// carrying Bad_CertificateUntrusted and holds no session, and that an accepted
// one opens a session Sessions() lists.
func TestClientCertificateVerifierAtOpenSecureChannel(t *testing.T) {
	const port = 48690
	cliCert, cliKey := genSelfSignedCert(t, "urn:gopcua:certtest:client")
	cl := &clientIdentity{cliCert, cliKey}
	rec := &certRecorder{refuse: true}
	s := startCertServer(t, port, "Basic256Sha256", ua.MessageSecurityModeSignAndEncrypt, rec.verify)

	_, err := connectCertClient(t, port, "Basic256Sha256", ua.MessageSecurityModeSignAndEncrypt, cl)
	require.Error(t, err, "a refused certificate opened a channel")
	require.True(t, errors.Is(err, ua.StatusBadCertificateUntrusted), "refused with %v", err)
	var uacpErr *uacp.Error
	require.True(t, errors.As(err, &uacpErr), "want an ERR message at the channel, got %T: %v", err, err)
	require.Equal(t, [][]byte{cliCert}, rec.asked(), "verifier asked once, about the client certificate")
	require.Empty(t, s.Sessions(), "a refused client holds no session")

	rec.setRefuse(false)
	before := time.Now()
	c, err := connectCertClient(t, port, "Basic256Sha256", ua.MessageSecurityModeSignAndEncrypt, cl)
	require.NoError(t, err, "an accepted certificate connects")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err = c.GetEndpoints(ctx)
	require.NoError(t, err, "symmetric request over the accepted channel")

	sessions := s.Sessions()
	require.Len(t, sessions, 1)
	require.NotNil(t, sessions[0].ID)
	require.Equal(t, cliCert, sessions[0].ClientCertificate)
	require.False(t, sessions[0].Created.Before(before), "created %v before connect at %v", sessions[0].Created, before)
	require.False(t, sessions[0].Created.After(time.Now()))

	require.NoError(t, c.Close(ctx))
	require.Empty(t, s.Sessions(), "a closed session is gone")
}

// TestClientCertificateVerifierAtCreateSession checks that under policy None the
// verifier is asked once, at CreateSession, about the ClientCertificate of the
// request (empty when the client has none), and that a refusal is a ServiceFault
// with Bad_CertificateUntrusted and no session.
func TestClientCertificateVerifierAtCreateSession(t *testing.T) {
	const port = 48691
	cliCert, cliKey := genSelfSignedCert(t, "urn:gopcua:certtest:client")
	cl := &clientIdentity{cliCert, cliKey}
	rec := &certRecorder{refuse: true}
	s := startCertServer(t, port, "None", ua.MessageSecurityModeNone, rec.verify)

	_, err := connectCertClient(t, port, "None", ua.MessageSecurityModeNone, cl)
	require.Error(t, err, "a refused certificate created a session")
	require.True(t, errors.Is(err, ua.StatusBadCertificateUntrusted), "refused with %v", err)
	var uacpErr *uacp.Error
	require.False(t, errors.As(err, &uacpErr), "want a service fault, not an ERR message: %v", err)
	require.Equal(t, [][]byte{cliCert}, rec.asked())
	require.Empty(t, s.Sessions())

	// a client without a certificate is asked about an empty one
	_, err = connectCertClient(t, port, "None", ua.MessageSecurityModeNone, nil)
	require.True(t, errors.Is(err, ua.StatusBadCertificateUntrusted), "refused with %v", err)
	asked := rec.asked()
	require.Len(t, asked, 2)
	require.Empty(t, asked[1])

	rec.setRefuse(false)
	_, err = connectCertClient(t, port, "None", ua.MessageSecurityModeNone, cl)
	require.NoError(t, err)
	sessions := s.Sessions()
	require.Len(t, sessions, 1)
	require.Equal(t, cliCert, sessions[0].ClientCertificate)
}

// TestNoClientCertificateVerifierAcceptsEveryCertificate checks the default:
// without a verifier every certificate is accepted.
func TestNoClientCertificateVerifierAcceptsEveryCertificate(t *testing.T) {
	const port = 48692
	cliCert, cliKey := genSelfSignedCert(t, "urn:gopcua:certtest:client")
	s := startCertServer(t, port, "Basic256Sha256", ua.MessageSecurityModeSignAndEncrypt, nil)
	_, err := connectCertClient(t, port, "Basic256Sha256", ua.MessageSecurityModeSignAndEncrypt, &clientIdentity{cliCert, cliKey})
	require.NoError(t, err)
	require.Len(t, s.Sessions(), 1)
}
