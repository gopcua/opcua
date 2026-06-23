//go:build integration
// +build integration

// Copyright 2018-2026 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uatest2

// Conformance regression tests.
//
// Convention: tests in this file are added only when a real regression
// occurred, and are named after the OPC UA specification clause they pin
// (behavior first, clause as suffix). Each test quotes the clause verbatim
// so the expected behavior is reviewable against the test body without
// opening the specification.

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"math/big"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/server"
	"github.com/gopcua/opcua/ua"
)

func genSelfSignedCert(t *testing.T, appURI string) ([]byte, *rsa.PrivateKey) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	uri, err := url.Parse(appURI)
	require.NoError(t, err)
	tmpl := x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "gopcua conformance test"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageDataEncipherment | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{"localhost"},
		URIs:                  []*url.URL{uri},
	}
	der, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, &key.PublicKey, key)
	require.NoError(t, err)
	return der, key
}

// TestSecurityModeRoundtrip_Part6_674 pins OPC UA Part 6 v1.05 §6.7.4:
//
//	"The OpenSecureChannel Messages are signed and encrypted if the
//	 SecurityMode is not None (even if the SecurityMode is Sign)."
//
// The load-bearing invariant is broader than §6.7.4's literal "mode != None"
// text: an OpenSecureChannel (OPN) message always uses the asymmetric crypto
// path regardless of the channel's currently-negotiated SecurityMode. This is
// open62541's "messageType == OPN" rule, and it is what lets the server
// decrypt the incoming OPN while its channel mode still reads None (the OPN
// is decrypted before the requested mode is known). The test pins decryption
// in exactly that None-window, which §6.7.4's literal text alone does not
// ground.
//
// What this verifies: all three modes (None, Sign, SignAndEncrypt) complete a
// full handshake plus one symmetric request. That catches the #817 break as a
// regression, because the bug encrypted symmetric Sign-mode traffic the peer
// expected signed, which dropped the chunk and hung the connection.
//
// What this does NOT verify: wire-level sign-only or true cross-stack behavior.
// Both peers here are gopcua and derive encrypt-vs-sign from the same code, so a
// regression that re-encrypted symmetric traffic on both sides could still pass.
// An open62541 or OPC Foundation .NET interop test is the gold-standard gate for
// that.
//
// Regressions pinned:
//   - The server never decrypted the incoming asymmetric OPN, because its
//     channel config starts with SecurityMode None (#815).
//   - The fix for #815 persistently overrode the channel SecurityMode, which
//     encrypted symmetric messages in Sign mode and broke Sign entirely (#817).
//   - Removing the #817 override without the asymmetric carve-out re-broke the
//     None-window OPN decryption from #815 (#853).
func TestSecurityModeRoundtrip_Part6_674(t *testing.T) {
	tests := []struct {
		name   string
		policy string
		mode   ua.MessageSecurityMode
		port   int
	}{
		{"None", "None", ua.MessageSecurityModeNone, 48671},
		{"Sign", "Basic256Sha256", ua.MessageSecurityModeSign, 48672},
		{"SignAndEncrypt", "Basic256Sha256", ua.MessageSecurityModeSignAndEncrypt, 48673},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := t.Context()
			srvCert, srvKey := genSelfSignedCert(t, "urn:gopcua:conformance:server")
			cliCert, cliKey := genSelfSignedCert(t, "urn:gopcua:conformance:client")

			s := server.New(
				ctx,
				server.EnableSecurity("None", ua.MessageSecurityModeNone),
				server.EnableSecurity("Basic256Sha256", ua.MessageSecurityModeSign),
				server.EnableSecurity("Basic256Sha256", ua.MessageSecurityModeSignAndEncrypt),
				server.EnableAuthMode(ua.UserTokenTypeAnonymous),
				server.EndPoint("localhost", tt.port),
				server.PrivateKey(srvKey),
				server.Certificate(srvCert),
			)
			require.NoError(t, s.Start(ctx))
			defer s.Close(ctx)

			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			addr := fmt.Sprintf("opc.tcp://localhost:%d", tt.port)

			// Discover the endpoint the way a real client does, so the
			// connection uses the server-provided certificate (chain).
			// Poll until the listener accepts a discovery request rather than
			// sleeping a fixed interval, so the test waits exactly as long as
			// the server needs and no longer, bounded by ctx.
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

			var ep *ua.EndpointDescription
			for _, e := range eps {
				if e.SecurityMode == tt.mode &&
					(tt.mode == ua.MessageSecurityModeNone || e.SecurityPolicyURI == ua.SecurityPolicyURIBasic256Sha256) {
					ep = e
					break
				}
			}
			require.NotNil(t, ep, "no endpoint for policy=%s mode=%s", tt.policy, tt.mode)

			opts := []opcua.Option{opcua.SecurityFromEndpoint(ep, ua.UserTokenTypeAnonymous)}
			if tt.mode != ua.MessageSecurityModeNone {
				opts = append(opts, opcua.Certificate(cliCert), opcua.PrivateKey(cliKey))
			}

			c, err := opcua.NewClient(addr, opts...)
			require.NoError(t, err)
			require.NoError(t, c.Connect(ctx), "handshake (OPN + session) in mode %s", tt.mode)
			defer c.Close(ctx)

			// One request over the established symmetric channel proves the
			// per-mode symmetric security (sign-only vs sign+encrypt) is
			// what the peer expects.
			_, err = c.GetEndpoints(ctx)
			require.NoError(t, err, "symmetric request in mode %s", tt.mode)
		})
	}
}
