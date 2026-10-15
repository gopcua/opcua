// Copyright 2018-2020 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uapolicy

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/gopcua/opcua/ua"
)

// A receiver decrypts every ciphertext block on its own (OPC UA Part 6, 6.7.4) and
// expects each full block to carry the RSA-OAEP maximum of k - 2*hLen - 2 bytes
// (RFC 8017, 7.1.1). Blocks that carry less make a strict receiver, such as the
// OPC Foundation .NET stack, misread the OpenSecureChannel request and drop the
// connection.
func TestRSAOAEPEncryptFillsBlocks(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	for _, hash := range []crypto.Hash{crypto.SHA1, crypto.SHA256} {
		t.Run(hash.String(), func(t *testing.T) {
			maxBlock := key.Size() - 2*hash.Size() - 2
			payload := make([]byte, 2*maxBlock)
			_, err := rand.Read(payload)
			require.NoError(t, err)

			enc := &RSAOAEP{Hash: hash, PublicKey: &key.PublicKey}
			ciphertext, err := enc.Encrypt(payload)
			require.NoError(t, err)
			require.Equal(t, 2*key.Size(), len(ciphertext), "a payload of two full blocks must encrypt to two blocks")

			for start := 0; start < len(ciphertext); start += key.Size() {
				block, err := rsa.DecryptOAEP(hash.New(), rand.Reader, key, ciphertext[start:start+key.Size()], nil)
				require.NoError(t, err)
				require.Equal(t, maxBlock, len(block))
			}
		})
	}
}

// The asymmetric plaintext block size pads messages to block boundaries, so it must
// match what one RSA-OAEP block carries for the policy's hash.
func TestAsymmetricPlaintextBlockSize(t *testing.T) {
	local, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	remote, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	for uri, hashSize := range map[string]int{
		ua.SecurityPolicyURIBasic256Sha256:      20, // RSA-OAEP-SHA1
		ua.SecurityPolicyURIAes128Sha256RsaOaep: 20, // RSA-OAEP-SHA1
		ua.SecurityPolicyURIAes256Sha256RsaPss:  32, // RSA-OAEP-SHA2-256
	} {
		alg, err := Asymmetric(uri, local, &remote.PublicKey)
		require.NoError(t, err, uri)
		require.Equal(t, remote.Size()-2*hashSize-2, alg.PlaintextBlockSize(), uri)
	}
}
