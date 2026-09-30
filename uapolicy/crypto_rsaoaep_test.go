// Copyright 2018-2020 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uapolicy

import (
	"crypto"
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/require"
)

// RSA-OAEP consumes 2*hashLen+2 bytes of each block, where hashLen is the digest (output)
// size of the hash: 20 bytes for SHA-1, 32 bytes for SHA-256.
func TestRSAOAEPMinPadding(t *testing.T) {
	require.Equal(t, 2*crypto.SHA1.Size()+2, RSAOAEPMinPaddingSHA1)
	require.Equal(t, 2*crypto.SHA256.Size()+2, RSAOAEPMinPaddingSHA256)
}

// Encrypt and Decrypt use the same block size, so a round trip does not find a wrong
// block size. Full blocks of plaintext must encrypt to the same number of RSA blocks.
func TestAes256Sha256RsaPssAsymmetricBlocks(t *testing.T) {
	localKey, err := generatePrivateKey(2048)
	require.NoError(t, err)
	remoteKey, err := generatePrivateKey(2048)
	require.NoError(t, err)

	enc, err := newAes256Sha256RsaPssAsymmetric(localKey, &remoteKey.PublicKey)
	require.NoError(t, err)

	// 2048-bit key, OAEP-SHA256: 256 - (2*32 + 2) = 190 bytes of plaintext per block.
	require.Equal(t, 190, enc.PlaintextBlockSize())
	require.Equal(t, 256, enc.BlockSize())

	plaintext := make([]byte, 2*enc.PlaintextBlockSize())
	_, err = rand.Read(plaintext)
	require.NoError(t, err)

	ciphertext, err := enc.Encrypt(plaintext)
	require.NoError(t, err)
	require.Len(t, ciphertext, 2*enc.BlockSize(), "two blocks of plaintext must encrypt to two RSA blocks")

	remote, err := newAes256Sha256RsaPssAsymmetric(remoteKey, &localKey.PublicKey)
	require.NoError(t, err)
	decrypted, err := remote.Decrypt(ciphertext)
	require.NoError(t, err)
	require.Equal(t, plaintext, decrypted)
}
