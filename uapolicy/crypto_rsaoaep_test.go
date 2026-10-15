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
)

// An RSA-OAEP block carries at most keySize - 2*hashLen - 2 bytes (RFC 8017,
// 7.1.1). Receivers such as the OPC Foundation .NET stack expect each
// ciphertext block to carry that many, so a full block must not be split.
func TestRSAOAEPFullBlock(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	for _, tc := range []struct {
		hash    crypto.Hash
		hashLen int
	}{
		{crypto.SHA1, 20},
		{crypto.SHA256, 32},
	} {
		t.Run(tc.hash.String(), func(t *testing.T) {
			a := &RSAOAEP{Hash: tc.hash, PublicKey: &key.PublicKey, PrivateKey: key}
			plaintext := make([]byte, key.Size()-2*tc.hashLen-2)
			_, err := rand.Read(plaintext)
			require.NoError(t, err)

			ciphertext, err := a.Encrypt(plaintext)
			require.NoError(t, err)
			require.Len(t, ciphertext, key.Size(), "a full block must encrypt to one ciphertext block")

			decrypted, err := a.Decrypt(ciphertext)
			require.NoError(t, err)
			require.Equal(t, plaintext, decrypted)
		})
	}
}

func TestAes256Sha256RsaPssPlaintextBlockSize(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	algo, err := newAes256Sha256RsaPssAsymmetric(key, &key.PublicKey)
	require.NoError(t, err)
	require.Equal(t, key.Size()-2*32-2, algo.PlaintextBlockSize())
}
