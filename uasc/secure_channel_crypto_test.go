// Copyright 2018-2020 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uasc

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"testing"
	"time"

	"github.com/gopcua/opcua/ua"
	"github.com/gopcua/opcua/uapolicy"
	"github.com/stretchr/testify/require"
)

// ecdsaServerCertificate returns a DER certificate with an ECDSA key, which
// no SecurityPolicy this package implements can use; see uapolicy.ErrNotRSA.
func ecdsaServerCertificate(t *testing.T) []byte {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "ecdsa server"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
	}
	cert, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	require.NoError(t, err)
	return cert
}

// TestSessionCryptoRejectsCertificateWithoutRSAKey checks that the session
// crypto helpers report a server certificate without an RSA key as an error
// instead of panicking on an unchecked key conversion.
func TestSessionCryptoRejectsCertificateWithoutRSAKey(t *testing.T) {
	cert := ecdsaServerCertificate(t)

	localKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	s := &SecureChannel{cfg: &Config{
		SecurityMode:      ua.MessageSecurityModeSignAndEncrypt,
		SecurityPolicyURI: ua.SecurityPolicyURIBasic256Sha256,
		LocalKey:          localKey,
		UserKey:           localKey,
	}}

	nonce := make([]byte, 32)
	signature := make([]byte, 256)

	_, _, err = s.NewSessionSignature(cert, nonce)
	require.ErrorIs(t, err, uapolicy.ErrNotRSA)

	err = s.VerifySessionSignature(cert, nonce, signature)
	require.ErrorIs(t, err, uapolicy.ErrNotRSA)

	_, _, err = s.EncryptUserPassword(ua.SecurityPolicyURIBasic256Sha256, "password", cert, nonce)
	require.ErrorIs(t, err, uapolicy.ErrNotRSA)

	_, _, err = s.NewUserTokenSignature(ua.SecurityPolicyURIBasic256Sha256, cert, nonce)
	require.ErrorIs(t, err, uapolicy.ErrNotRSA)
}
