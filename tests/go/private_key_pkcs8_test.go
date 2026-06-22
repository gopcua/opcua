//go:build integration
// +build integration

// Copyright 2018-2026 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uatest2

// Capstone regression test for PR #871 (PKCS#8 private key support).
//
// This test pins gopcua's acceptance of PKCS#8-encoded RSA private keys loaded
// from a file via opcua.PrivateKeyFile, exercised through a real SignAndEncrypt
// (Basic256Sha256) handshake plus one symmetric request against the in-process
// gopcua server.
//
// What this pins: PKCS#8 RSA keys (PEM "PRIVATE KEY" and raw DER) loaded from
// disk complete a full OPN + session handshake and a symmetric GetEndpoints.
// On main (pre-PR), loadPrivateKey rejected PKCS#8 PEM (requires block.Type
// "RSA PRIVATE KEY") and called x509.ParsePKCS1PrivateKey on the DER bytes
// (which fails on PKCS#8 input), so the test fails at config (ApplyConfig,
// before Connect). On the PR branch the same rows are green.
//
// Cross-implementation note (NOT a universal invariant): gopcua eagerly rejects
// non-RSA keys at load time (parsePKCS8RSAPrivateKey asserts *rsa.PrivateKey).
// This matches UA-.NETStandard (the OPC Foundation reference, RSA-only by
// construction) and is stricter than open62541, which defers the RSA check to
// security-policy-use time. The test therefore pins gopcua's .NET-aligned
// choice, not a cross-implementation rule.
//
// What this does NOT verify: cross-stack interop. Both peers here are gopcua,
// so a regression that broke PKCS#8 only against an open62541 or .NET peer
// would not be caught. A hardware or reference-server interop run is the
// gold-standard gate for that (out of scope here; mirrors the limitation noted
// in TestSecurityModeRoundtrip_Part6_674). There is no OPC UA spec clause
// governing local private-key encoding (Part 6 covers the on-wire X.509
// certificate, not how the application stores its key), so this test is
// behavior-named without a _Part6_§X suffix per the gopcua testing convention.

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/server"
	"github.com/gopcua/opcua/ua"
)

func TestPrivateKeyFilePKCS8Handshake(t *testing.T) {
	// Shared in-process server: the server config is identical for both rows,
	// and the only variable under test is the client's key file, which the
	// server never sees. A per-row server would be redundant (unlike
	// TestSecurityModeRoundtrip_Part6_674, where each row varies the
	// server-side security mode).
	// Single source of truth for the server port: the const is declared above
	// server.New so the same value flows into EndPoint and the client address.
	const port = 48680
	srvCert, srvKey := genSelfSignedCert(t, "urn:gopcua:conformance:server")
	s := server.New(
		server.EnableSecurity("Basic256Sha256", ua.MessageSecurityModeSignAndEncrypt),
		server.EnableAuthMode(ua.UserTokenTypeAnonymous),
		server.EndPoint("localhost", port),
		server.PrivateKey(srvKey),
		server.Certificate(srvCert),
	)
	require.NoError(t, s.Start(context.Background()))
	defer s.Close()

	addr := fmt.Sprintf("opc.tcp://localhost:%d", port)

	// Discovery: poll until the listener is ready, bounded by a 15s context,
	// then locate the Basic256Sha256/SignAndEncrypt endpoint the way a real
	// client does so the connection uses the server-provided certificate chain.
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Discovery: poll until the listener is ready AND the target
	// Basic256Sha256/SignAndEncrypt endpoint is present, bounded by a 15s
	// context. Breaking on a nil error alone is insufficient: GetEndpoints can
	// succeed with an empty or partial endpoint list during startup, so the
	// endpoint search happens inside the loop and the loop only breaks when the
	// target endpoint is actually found.
	var ep *ua.EndpointDescription
	var lastErr error
	for {
		eps, err := opcua.GetEndpoints(ctx, addr)
		if err == nil {
			for _, e := range eps {
				if e.SecurityMode == ua.MessageSecurityModeSignAndEncrypt &&
					e.SecurityPolicyURI == ua.SecurityPolicyURIBasic256Sha256 {
					ep = e
					break
				}
			}
		} else {
			lastErr = err
		}
		if ep != nil {
			break
		}
		select {
		case <-ctx.Done():
			if lastErr != nil {
				require.NoError(t, lastErr, "discovery (server never became ready)")
			}
			require.NotNil(t, ep, "discovery timed out before target endpoint was available")
		case <-time.After(50 * time.Millisecond):
		}
	}

	tests := []struct {
		name   string
		format string
		// write writes the PKCS#8-encoded cliKey to a temp file and returns
		// the path. The structural self-assertions that anchor the red-on-main
		// guarantee live inside write, so they cannot be silently dropped
		// without dropping the row's setup.
		write func(t *testing.T, cliKey interface{}) string
	}{
		{
			name:   "PEM",
			format: "PEM",
			write: func(t *testing.T, cliKey interface{}) string {
				der, err := x509.MarshalPKCS8PrivateKey(cliKey)
				require.NoError(t, err, "marshal PKCS#8")
				keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})
				// Structural guarantee: pin block.Type to "PRIVATE KEY" so
				// the PR's switch routes through parsePKCS8RSAPrivateKey
				// (the PKCS#8 PEM branch under test). A flip to
				// "RSA PRIVATE KEY" would route to the PKCS#1 case, where
				// ParsePKCS1PrivateKey errors on the PKCS#8 bytes and the
				// row would fail without exercising the branch under test.
				// Red-on-main is guaranteed independently by the bytes
				// being PKCS#8: main rejects "PRIVATE KEY" at the type
				// gate, and even a PKCS#1-typed PKCS#8 block would fail at
				// ParsePKCS1PrivateKey. The bytes are independently
				// guaranteed PKCS#8 by x509.MarshalPKCS8PrivateKey above.
				block, _ := pem.Decode(keyPEM)
				require.NotNil(t, block, "decoded PEM")
				require.Equal(t, "PRIVATE KEY", block.Type)
				keyPath := filepath.Join(t.TempDir(), "key.pem")
				require.NoError(t, os.WriteFile(keyPath, keyPEM, 0600))
				return keyPath
			},
		},
		{
			name:   "DER",
			format: "DER",
			write: func(t *testing.T, cliKey interface{}) string {
				der, err := x509.MarshalPKCS8PrivateKey(cliKey)
				require.NoError(t, err, "marshal PKCS#8")
				// Structural red guarantee: main's loadPrivateKey calls
				// x509.ParsePKCS1PrivateKey on the raw .der bytes, which
				// fails on PKCS#8 input (this IS the red-on-main invariant).
				// The reciprocal NoError on ParsePKCS8PrivateKey confirms the
				// bytes are well-formed PKCS#8. A wrong Marshal producing
				// PKCS#1 DER would fail the ParsePKCS1PrivateKey-errors
				// assertion. This couples to Go's x509 stdlib, correctly:
				// main's red DER path calls the identical ParsePKCS1PrivateKey,
				// so if a future Go release made it lenient on PKCS#8, the
				// assertion and main's red behavior break together (the
				// assertion failing is the canary that the red invariant is
				// gone).
				_, errPKCS1 := x509.ParsePKCS1PrivateKey(der)
				require.Error(t, errPKCS1, "PKCS#1 parse must fail on PKCS#8 input")
				_, err = x509.ParsePKCS8PrivateKey(der)
				require.NoError(t, err, "PKCS#8 parse must succeed")
				keyPath := filepath.Join(t.TempDir(), "key.der")
				require.NoError(t, os.WriteFile(keyPath, der, 0600))
				return keyPath
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cliCert, cliKey := genSelfSignedCert(t, "urn:gopcua:conformance:client")
			keyPath := tt.write(t, cliKey)

			opts := []opcua.Option{
				opcua.SecurityFromEndpoint(ep, ua.UserTokenTypeAnonymous),
				// Certificate is supplied in-memory (not via CertificateFile):
				// the PR does not change cert loading, and using a file here
				// would couple pre-existing cert-loading correctness into the
				// red signal. Only the key uses the file path, isolating the
				// PR's single variable.
				opcua.Certificate(cliCert),
				opcua.PrivateKeyFile(keyPath),
			}

			c, err := opcua.NewClient(addr, opts...)
			require.NoError(t, err, "NewClient with PKCS#8 %s key", tt.format)
			require.NoError(t, c.Connect(ctx), "handshake with PKCS#8 %s key", tt.format)
			defer c.Close(ctx)

			// One symmetric request over the established channel proves the
			// PKCS#8 key is usable for ongoing sign+encrypt traffic, not just
			// the asymmetric OPN.
			_, err = c.GetEndpoints(ctx)
			require.NoError(t, err, "symmetric request with PKCS#8 %s key", tt.format)
		})
	}
}
