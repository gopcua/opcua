// Copyright 2018-2020 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uapolicy

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"sort"
	"testing"

	"github.com/gopcua/opcua/ua"
	"github.com/stretchr/testify/require"
)

func TestSupportedPolicies(t *testing.T) {
	got := SupportedPolicies()
	var want []string
	for k := range policies {
		want = append(want, k)
	}
	sort.Strings(want)
	require.Equal(t, want, got)
}

func TestGenerateKeysLength(t *testing.T) {
	localNonce := make([]byte, 32)
	remoteNonce := make([]byte, 32)
	_, err := rand.Read(localNonce)
	require.NoError(t, err, "Could not generate local nonce")

	_, err = rand.Read(remoteNonce)
	require.NoError(t, err, "Could not generate remote nonce")

	hmac := &HMAC{Hash: crypto.SHA256, Secret: remoteNonce}
	keys := generateKeys(hmac, localNonce, 32, 32, 16)
	require.Equal(t, 32, len(keys.signing), "Signing Key Invalid Length")
	require.Equal(t, 32, len(keys.encryption), "Encryption Key Invalid Length")
	require.Equal(t, 16, len(keys.iv), "Encryption IV Invalid Length")
}

func TestGenerateKeys(t *testing.T) {
	localNonce := []byte("\xEE\x51\x68\x84\x0E\x07\xF3\x94\x5B\x6D\xB7\x3A\x41\x3E\xC2\x5C")
	remoteNonce := []byte("\x9B\x0F\x5B\xF8\x5E\x32\xFB\x37\x01\x43\x69\xB3\x14\xDE\x7A\xE7")

	localKeys := &derivedKeys{
		signing:    []byte("\xCB\xFB\x77\x42\x44\xB1\x03\xB3\xB5\x2C\x10\x7C\xA3\xAE\x80\xD4"),
		encryption: []byte("\x00\x52\xB6\x82\xB2\x2C\x75\x54\x71\xDB\xF7\xC9\x8F\x88\x39\xFA"),
		iv:         []byte("\xF8\x97\xF4\x13\xCC\xC7\xB8\x19\xE5\x45\xC7\xAE\xC3\x5D\x9D\x77"),
	}

	remoteKeys := &derivedKeys{
		signing:    []byte("\x9E\x0A\xA9\x20\xED\x7E\xC2\x18\x6D\xB8\x19\x95\x8C\xD9\x0F\xA5"),
		encryption: []byte("\x9C\x11\xEA\x7D\xAA\xD8\x7B\xBC\x94\x47\xCB\x1C\x06\xB5\xC6\x4B"),
		iv:         []byte("\x09\xAA\x4F\x50\x15\x4D\x69\xC5\x0B\x3B\x78\x7F\xD8\x54\x36\x45"),
	}

	localHmac := &HMAC{Hash: crypto.SHA1, Secret: localNonce}
	keys := generateKeys(localHmac, remoteNonce, 16, 16, 16)
	require.Equal(t, localKeys.signing, keys.signing, "local signing key generation failed")
	require.Equal(t, localKeys.encryption, keys.encryption, "local encryption key generation failed")
	require.Equal(t, localKeys.iv, keys.iv, "local iv key generation failed")

	remoteHmac := &HMAC{Hash: crypto.SHA1, Secret: remoteNonce}
	keys = generateKeys(remoteHmac, localNonce, 16, 16, 16)
	require.Equal(t, remoteKeys.signing, keys.signing, "remote signing key generation failed")
	require.Equal(t, remoteKeys.encryption, keys.encryption, "remote encryption key generation failed")
	require.Equal(t, remoteKeys.iv, keys.iv, "remote iv key generation failed")
}

// Test all supported encryption algorithms.  Because the majority of the algorithms
// use randomization, the ciphertext will be different on every run even if we used
// the same keys.  This makes testing against known byte slices impossible.
// Therefore, the test simply encrypts the message, decrypts it, then compares the
// results.
func TestEncryptionAlgorithms(t *testing.T) {
	payload := make([]byte, 5000)
	_, err := rand.Read(payload)
	require.NoError(t, err, "could not generate random payload")

	payloadRef := make([]byte, len(payload))
	copy(payloadRef, payload)

	// todo (dh): 2048 happens to be a keysize compatable with all current algorithms.
	// This won't be the case forever and will be too small for future algorithms
	// and the test will need to be able to input keys of varying size
	localKey, err := generatePrivateKey(2048)
	require.NoError(t, err, "Unable to generate local private key")

	remoteKey, err := generatePrivateKey(2048)
	require.NoError(t, err, "Unable to generate remote private key")

	for uri, p := range policies {
		t.Run(uri, func(t *testing.T) {
			localAsymmetric, err := p.asymmetric(localKey, &remoteKey.PublicKey)
			require.NoError(t, err)

			makeNonce := func(n int) []byte {
				t.Helper()
				if n == 0 {
					return nil
				}
				b := make([]byte, n)
				_, err = rand.Read(b)
				require.NoError(t, err, "could not generate nonce")
				return b
			}

			nonceLength := localAsymmetric.NonceLength()
			localNonce, remoteNonce := makeNonce(nonceLength), makeNonce(nonceLength)
			require.False(t, nonceLength == 0 && uri != ua.SecurityPolicyURINone, "client nonce length zero")

			localSymmetric, err := p.symmetric(localNonce, remoteNonce)
			require.NoError(t, err, "failed local Symmetric: %s", err)

			remoteSymmetric, err := p.symmetric(remoteNonce, localNonce)
			require.NoError(t, err, "failed remote Symmetric: %s", err)

			remoteAsymmetric, err := p.asymmetric(remoteKey, &localKey.PublicKey)
			require.NoError(t, err, "failed remote Asymmetric: %s", err)

			// Symmetric Algorithm
			plaintext := make([]byte, len(payload))
			copy(plaintext, payload)

			padSize := len(plaintext) % localSymmetric.BlockSize()
			if padSize > 0 {
				padSize = localSymmetric.BlockSize() - padSize
			}
			paddedPlaintext := make([]byte, len(plaintext)+padSize)
			copy(paddedPlaintext, plaintext)

			symCiphertext, err := localSymmetric.Encrypt(paddedPlaintext)
			require.NoError(t, err, "failed to encrypt Symmetric: %s", err)

			symDeciphered, err := remoteSymmetric.Decrypt(symCiphertext)
			require.NoError(t, err, "failed to decrypt Symmetric: %s", err)

			symDeciphered = symDeciphered[:len(symDeciphered)-padSize] // Trim off padding
			require.Equal(t, plaintext, symDeciphered, "symmetric encryption failed")

			// Modify the plaintext and detect if the decrypted message changes; if it does,
			// our byte slices are referencing the same data and the previous test may have
			// been a false positive
			paddedPlaintext[4] = 0xff ^ paddedPlaintext[4]
			require.Equal(t, payloadRef, symDeciphered, "symmetric input corruption detected")

			symSignature, err := localSymmetric.Signature(paddedPlaintext)
			require.NoError(t, err, "symmetric signature generation failed")

			err = remoteSymmetric.VerifySignature(paddedPlaintext, symSignature)
			require.NoError(t, err, "symmetric signature validation failed")

			// Asymmetric Algorithm
			asymCiphertext, err := localAsymmetric.Encrypt(plaintext)
			require.NoError(t, err, "failed to encrypt Asymmetric: %s", err)

			asymDeciphered, err := remoteAsymmetric.Decrypt(asymCiphertext)
			require.NoError(t, err, "failed to decrypt Asymmetric: %s", err)

			require.Equal(t, plaintext, asymDeciphered, "asymmetric encryption failed")

			paddedPlaintext[4] = 0xff ^ paddedPlaintext[4]
			require.Equal(t, payloadRef, asymDeciphered, "asymmetric input corruption detected")

			asymSignature, err := localAsymmetric.Signature(plaintext)
			require.NoError(t, err, "asymmetric signature generation failed")

			err = remoteAsymmetric.VerifySignature(plaintext, asymSignature)
			require.NoError(t, err, "asymmetric signature validation failed")
		})
	}
}

func TestMissingKey(t *testing.T) {
	payload := make([]byte, 5000)
	_, err := rand.Read(payload)
	require.NoError(t, err, "could not generate random payload")

	payloadRef := make([]byte, len(payload))
	copy(payloadRef, payload)

	key, err := generatePrivateKey(2048)
	require.NoError(t, err, "Unable to generate private key")

	for uri, p := range policies {
		if uri == ua.SecurityPolicyURINone {
			continue
		}

		t.Run(uri, func(t *testing.T) {
			encryptOnly, err := p.asymmetric(nil, &key.PublicKey)
			require.NoError(t, err, "failed to create encrypt-only asymmetric algorithms")

			decryptOnly, err := p.asymmetric(key, nil)
			require.NoError(t, err, "failed to create decrypt-only asymmetric algorithms")

			ciphertext, err := encryptOnly.Encrypt(payload)
			require.NoError(t, err, "failed to encrypt with encrypt-only policy")

			signature, err := decryptOnly.Signature(payload)
			require.NoError(t, err, "decrypt-only algorithm failed to generate signature")

			err = encryptOnly.VerifySignature(payload, signature)
			require.NoError(t, err, "failed to verify signature with encrypt-only algorithm")

			plaintext, err := decryptOnly.Decrypt(ciphertext)
			require.NoError(t, err, "failed to decrypt with decrypt-only algorithm")

			require.Equal(t, payloadRef, plaintext, "decryption failed")

			_, err = encryptOnly.Decrypt(ciphertext)
			require.Error(t, err, "encrypt-only algorithm decrypted block without error - should be impossible")

			_, err = encryptOnly.Signature(payload)
			require.Error(t, err, "encrypt-only algorithm generated a signature without error - should be impossible")

			_, err = decryptOnly.Encrypt(payload)
			require.Error(t, err, "decrypt-only algorithm encrypted block without error - should be impossible")

			err = decryptOnly.VerifySignature(payload, signature)
			require.Error(t, err, "decrypt-only algorithm verified a signature without error - should be impossible")
		})

	}
}

func TestZeroStruct(t *testing.T) {
	ze := &EncryptionAlgorithm{}

	const payload string = "The quick brown fox jumps over the lazy dog."
	plaintext := []byte(payload)

	// Call all the methods and make sure they don't panic due to nil pointers
	require.NotPanics(t, func() { ze.BlockSize() })
	require.NotPanics(t, func() { ze.PlaintextBlockSize() })
	require.NotPanics(t, func() { ze.Encrypt(plaintext) })
	require.NotPanics(t, func() { ze.Decrypt(plaintext) })
	require.NotPanics(t, func() { ze.Signature(plaintext) })
	require.NotPanics(t, func() { ze.VerifySignature(plaintext, plaintext) })
	require.NotPanics(t, func() { ze.NonceLength() })
	require.NotPanics(t, func() { ze.SignatureLength() })
	require.NotPanics(t, func() { ze.EncryptionURI() })
	require.NotPanics(t, func() { ze.SignatureURI() })
}

func generatePrivateKey(bitSize int) (*rsa.PrivateKey, error) {
	// Private Key generation
	privateKey, err := rsa.GenerateKey(rand.Reader, bitSize)
	if err != nil {
		return nil, err
	}

	// Validate Private Key
	err = privateKey.Validate()
	if err != nil {
		return nil, err
	}

	return privateKey, nil
}
