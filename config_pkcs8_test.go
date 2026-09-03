// Copyright 2018-2026 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package opcua

// Fast unit coverage for PR #871 (PKCS#8 private key support) branches that
// the integration capstone (tests/go/private_key_pkcs8_test.go) does not pin
// in isolation and that lack fast unit coverage in TestOptions. Each row names
// the branch it pins.
//
// Row A pins parseRSAPrivateKeyDER -> parsePKCS8RSAPrivateKey type-assert
// rejection for a non-RSA (EC) PKCS#8 key delivered as raw DER (no PEM
// header), the path a .der EC key takes. The joined PKCS#1/PKCS#8 error still
// carries the "Private key is not an RSA key" message.
//
// Row B pins the default branch of the loadPrivateKey PEM switch
// ("unsupported type %q") for a SEC1 EC key whose PEM block Type is
// "EC PRIVATE KEY". This branch had zero coverage before this row (the
// existing TestOptions rows only exercise the PKCS#8 "PRIVATE KEY" case).
// The content-sniff loader routes by block.Type, so this row uses a .pem file
// to pin the default branch directly.
//
// Row C pins suffix-independence: a PKCS#8 RSA key written as PEM to a .key
// file (openssl genpkey default) loads cleanly. With extension-based
// dispatch this failed (the .key suffix skipped PEM decoding and the loader
// handed PEM-armored text to the DER parser). This row is the permanent pin
// for the content-sniff change.

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadPrivateKeyPKCS8Coverage(t *testing.T) {
	// Shared RSA key for the success row.
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	rsaPKCS8DER, err := x509.MarshalPKCS8PrivateKey(rsaKey)
	require.NoError(t, err)
	rsaPKCS8PEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: rsaPKCS8DER})

	// Shared EC key for the rejection rows.
	ecKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	ecPKCS8DER, err := x509.MarshalPKCS8PrivateKey(ecKey)
	require.NoError(t, err)
	ecSEC1DER, err := x509.MarshalECPrivateKey(ecKey)
	require.NoError(t, err)
	ecSEC1PEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: ecSEC1DER})

	tests := []struct {
		name      string
		write     func(t *testing.T) string
		wantError string // non-empty => expect an error containing this substring
	}{
		{
			// Row A: DER-EC rejection. Pins parseRSAPrivateKeyDER ->
			// parsePKCS8RSAPrivateKey type-assert branch for a non-RSA PKCS#8
			// key delivered as raw DER. The PKCS#1/PKCS#8 errors are joined,
			// but the "not an RSA key" message is preserved.
			name: "DER EC key rejected as not-RSA",
			write: func(t *testing.T) string {
				p := filepath.Join(t.TempDir(), "ec.der")
				require.NoError(t, os.WriteFile(p, ecPKCS8DER, 0600))
				return p
			},
			wantError: "Private key is not an RSA key",
		},
		{
			// Row B: default-branch unsupported PEM type. Pins the default
			// branch of the loadPrivateKey switch for a SEC1 EC key whose PEM
			// block Type is "EC PRIVATE KEY". This branch was previously
			// untested; the content-sniff loader routes by block.Type.
			name: "PEM EC PRIVATE KEY rejected as unsupported type",
			write: func(t *testing.T) string {
				p := filepath.Join(t.TempDir(), "ec.pem")
				require.NoError(t, os.WriteFile(p, ecSEC1PEM, 0600))
				return p
			},
			wantError: "unsupported type",
		},
		{
			// Row C: suffix-independence. Pins the content-sniff change: a
			// PKCS#8 RSA key as PEM in a .key file loads cleanly. With
			// extension-based dispatch this failed (the .key suffix skipped
			// PEM decoding). This is the permanent pin for the suffix fix.
			name: "PKCS#8 RSA PEM in .key file loads",
			write: func(t *testing.T) string {
				p := filepath.Join(t.TempDir(), "key.key")
				require.NoError(t, os.WriteFile(p, rsaPKCS8PEM, 0600))
				return p
			},
			wantError: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keyPath := tt.write(t)
			cfg, err := ApplyConfig(PrivateKeyFile(keyPath))
			if tt.wantError != "" {
				require.Error(t, err)
				require.True(t,
					strings.Contains(err.Error(), tt.wantError),
					"error %q does not contain %q", err.Error(), tt.wantError,
				)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, cfg, "config must be non-nil on success")
			require.NotNil(t, cfg.sechan.LocalKey, "LocalKey must be set on success")
			require.Equal(t, rsaKey.N, cfg.sechan.LocalKey.N, "loaded key must match generated key")
		})
	}
}

// TestLoadCertificateContentSniff pins the certificate side of the content-sniff
// change, which the key-side rows above do not cover. Two behaviors:
//
//   - Suffix-independence: a PEM certificate in a .crt file (openssl's default
//     PEM output suffix) loads. With extension-based dispatch a .crt file
//     skipped PEM decoding and the raw armored text was returned as "DER".
//
//   - Fullchain preservation: a PEM fullchain (leaf + intermediate, the common
//     fullchain.pem layout) loads the whole chain, not just the first block.
//     loadCertificate concatenates every CERTIFICATE block so the full chain is
//     sent on the wire, matching the raw-DER path.
func TestLoadCertificateContentSniff(t *testing.T) {
	// A second self-signed cert to stand in for an intermediate.
	secondDER := genSelfSignedCertDER(t)

	t.Run("PEM cert in .crt file loads", func(t *testing.T) {
		p := filepath.Join(t.TempDir(), "cert.crt")
		require.NoError(t, os.WriteFile(p, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER}), 0600))
		cfg, err := ApplyConfig(CertificateFile(p))
		require.NoError(t, err)
		require.Equal(t, certDER, cfg.sechan.Certificate, "loaded cert DER must match the PEM-decoded bytes")
	})

	t.Run("PEM fullchain loads the whole chain", func(t *testing.T) {
		chainPEM := bytes.NewBuffer(nil)
		chainPEM.Write(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER}))
		chainPEM.Write(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: secondDER}))
		p := filepath.Join(t.TempDir(), "fullchain.pem")
		require.NoError(t, os.WriteFile(p, chainPEM.Bytes(), 0600))
		cfg, err := ApplyConfig(CertificateFile(p))
		require.NoError(t, err)
		// The whole chain (leaf + intermediate) must be on the wire, not just
		// the first block. Concatenated DER: certDER || secondDER.
		want := append(append([]byte{}, certDER...), secondDER...)
		require.Equal(t, want, cfg.sechan.Certificate, "fullchain must load both certs, not just the leaf")
	})
}

// genSelfSignedCertDER returns a fresh self-signed certificate as raw DER, for
// use as a second chain entry in tests.
func genSelfSignedCertDER(t *testing.T) []byte {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	tmpl := &x509.Certificate{SerialNumber: big.NewInt(1), IsCA: false}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	require.NoError(t, err)
	return der
}
