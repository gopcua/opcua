// Copyright 2018-2020 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uasc

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/binary"
	"encoding/pem"
	"testing"
	"time"

	uatest "github.com/gopcua/opcua/tests/python"
	"github.com/gopcua/opcua/ua"
	"github.com/gopcua/opcua/uapolicy"
	"github.com/stretchr/testify/require"
)

// When a UserTokenPolicy advertises an empty SecurityPolicyURI, the client's
// encrypted UserName token construction must fall back to the SecureChannel's
// negotiated SecurityPolicyURI. This test pins that contract end-to-end: the
// encrypted password produced with policyURI="" on a Basic256Sha256 channel
// matches the one produced with policyURI=Basic256Sha256, and the returned
// EncryptionAlgorithm URI is the channel-policy algorithm URI.
func TestEncryptUserPasswordFallsBackToChannelPolicy(t *testing.T) {
	certPEM, keyPEM, err := uatest.GenerateCert("localhost", 2048, 24*time.Hour)
	require.NoError(t, err)

	block, _ := pem.Decode(keyPEM)
	require.NotNil(t, block, "decode private key PEM")
	localKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	require.NoError(t, err)

	certBlock, _ := pem.Decode(certPEM)
	require.NotNil(t, certBlock, "decode certificate PEM")
	remoteCertDER := certBlock.Bytes

	const (
		channelPolicy = ua.SecurityPolicyURIBasic256Sha256
		password      = "s3cret"
	)
	nonce := make([]byte, 32)
	for i := range nonce {
		nonce[i] = byte(i + 1)
	}

	sc := &SecureChannel{
		cfg: &Config{
			SecurityPolicyURI: channelPolicy,
			SecurityMode:      ua.MessageSecurityModeSignAndEncrypt,
			LocalKey:          localKey,
		},
	}

	// Path A: explicit channel policy URI on the user-token policy.
	explicitCipher, explicitAlg, err := sc.EncryptUserPassword(channelPolicy, password, remoteCertDER, nonce)
	require.NoError(t, err, "EncryptUserPassword with explicit policy URI")

	// Path B: empty user-token policy URI (legacy node-opcua-style
	// usernamePassword_0 inherit-channel-policy semantics).
	emptyCipher, emptyAlg, err := sc.EncryptUserPassword("", password, remoteCertDER, nonce)
	require.NoError(t, err, "EncryptUserPassword with empty policy URI must fall back to channel policy")

	require.Equal(t, explicitAlg, emptyAlg,
		"empty policy URI must yield the same EncryptionAlgorithm URI as the channel policy")
	require.Equal(t, "http://www.w3.org/2001/04/xmlenc#rsa-oaep", emptyAlg,
		"Basic256Sha256 channel must produce RSA-OAEP-SHA1 EncryptionAlgorithm URI on the wire")

	// Both ciphertexts must be the same length (key-size block) and
	// must decrypt to the same plaintext under the server private key.
	require.Equal(t, len(explicitCipher), len(emptyCipher),
		"ciphertext length must match between explicit and inherit-from-channel paths")

	plainExplicit := decryptUserPassword(t, channelPolicy, localKey, explicitCipher)
	plainEmpty := decryptUserPassword(t, channelPolicy, localKey, emptyCipher)
	require.Equal(t, plainExplicit, plainEmpty,
		"decrypted secret payload must match between explicit and inherit-from-channel paths")
	require.Equal(t, password, plainEmpty,
		"decrypted password must equal the input plaintext")
}

// TestEncryptUserPasswordEmptyOnNoneChannelStaysNone pins that an
// empty user-token policy on a SecurityPolicyURINone channel still
// yields the cleartext-password / empty-algorithm form (the spec's
// degenerate case). This is the regression guard against a too-eager
// fallback accidentally encrypting on a None channel.
func TestEncryptUserPasswordEmptyOnNoneChannelStaysNone(t *testing.T) {
	sc := &SecureChannel{
		cfg: &Config{
			SecurityPolicyURI: ua.SecurityPolicyURINone,
			SecurityMode:      ua.MessageSecurityModeNone,
		},
	}

	password := "s3cret"
	cipher, alg, err := sc.EncryptUserPassword("", password, nil, nil)
	require.NoError(t, err)
	require.Equal(t, "", alg, "None channel must yield empty EncryptionAlgorithm")
	require.Equal(t, []byte(password), cipher, "None channel must send cleartext password bytes")
}

// decryptUserPassword inverts EncryptUserPassword for assertion
// purposes: parse [4-byte LE length][password][nonce], unpadded.
func decryptUserPassword(t *testing.T, policyURI string, serverKey *rsa.PrivateKey, cipher []byte) string {
	t.Helper()

	enc, err := uapolicy.Asymmetric(policyURI, serverKey, &serverKey.PublicKey)
	require.NoError(t, err)
	plain, err := enc.Decrypt(cipher)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(plain), 4, "plaintext must include the 4-byte length prefix")

	length := binary.LittleEndian.Uint32(plain[:4])
	// length includes the trailing nonce; strip it off.
	require.LessOrEqual(t, int(length)+4, len(plain), "encoded length must fit in plaintext")
	// Caller knows nonce length is 32; subtract.
	const nonceLen = 32
	pwLen := int(length) - nonceLen
	require.GreaterOrEqual(t, pwLen, 0, "decoded password length must be non-negative")
	return string(plain[4 : 4+pwLen])
}
