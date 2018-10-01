// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package securitypolicy

import (
	"crypto/rand"
	"crypto/rsa"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestSupportedPolicies(t *testing.T) {
	s := SupportedPolicies()

	if len(s) != len(supportedPolicies) {
		t.Errorf("SupportedPolicies() has extra or missing entries")
	}

	for _, policy := range s {
		if _, ok := supportedPolicies[policy]; !ok {
			t.Errorf("SupportedPolicy returned \"%s\" but cannot fetch details\n", policy)
		}
	}

}

func TestGenerateKeys(t *testing.T) {
	localNonce := make([]byte, 32)
	remoteNonce := make([]byte, 32)
	_, err := rand.Read(localNonce)
	if err != nil {
		t.Fatalf("Could not generate local nonce")
	}

	_, err = rand.Read(remoteNonce)
	if err != nil {
		t.Fatalf("Could not generate remote nonce")
	}

	keys := generateKeys(hmacSha256(remoteNonce), localNonce, 32, 32, 16)
	if len(keys.signing) != 32 {
		t.Errorf("Signing Key Invalid Length\n")
	}
	if len(keys.encryption) != 32 {
		t.Errorf("Encryption Key Invalid Length\n")
	}
	if len(keys.iv) != 16 {
		t.Errorf("Encryption IV Invalid Length\n")
	}

}

// Test all supported encryption algorithms.  Because the majority of the algorithms
// use randomization, the ciphertext will be different on every run even if we used
// the same keys.  This makes testing against known byte slices impossible.
// Therefore, the test simply encrypts the message, decrypts it, then compares the
// results.
func TestEncryptionAlgorithms(t *testing.T) {
	const payload string = "The quick brown fox jumps over the lazy dog."

	localNonce := make([]byte, 32)
	remoteNonce := make([]byte, 32)
	_, err := rand.Read(localNonce)
	if err != nil {
		t.Fatalf("could not generate local nonce")
	}

	_, err = rand.Read(remoteNonce)
	if err != nil {
		t.Fatalf("could not generate remote nonce")
	}

	localKey, err := generatePrivateKey(2048)
	if err != nil {
		t.Fatalf("Unable to generate local private key\n")
	}
	remoteKey, err := generatePrivateKey(2048)
	if err != nil {
		t.Fatalf("Unable to generate remote private key\n")
	}

	cases := SupportedPolicies()

	for _, c := range cases {

		localPolicy, err := New(c)
		if err != nil {
			t.Fatalf("failed local New(%s) : %s", c, err)
		}

		remotePolicy, err := New(c)
		if err != nil {
			t.Fatalf("failed remote New(%s) : %s", c, err)
		}

		localSymmetric := localPolicy.Symmetric(localNonce, remoteNonce)
		localAsymmetric := localPolicy.Asymmetric(localKey, &remoteKey.PublicKey)

		remoteSymmetric := remotePolicy.Symmetric(remoteNonce, localNonce)
		remoteAsymmetric := remotePolicy.Asymmetric(remoteKey, &localKey.PublicKey)

		// Symmetric Algorithm
		plaintext := []byte(payload)

		padSize := len(plaintext) % localSymmetric.BlockSize()
		if padSize > 0 {
			padSize = localSymmetric.BlockSize() - padSize
		}
		paddedPlaintext := make([]byte, len(plaintext)+padSize)
		copy(paddedPlaintext, plaintext)

		symCiphertext := localSymmetric.Encrypt(paddedPlaintext)

		symDeciphered := remoteSymmetric.Decrypt(symCiphertext)
		symDeciphered = symDeciphered[:len(symDeciphered)-padSize] // Trim off padding
		if diff := cmp.Diff(symDeciphered, plaintext, nil); diff != "" {
			t.Errorf("Policy: %s\nsymmetric encryption failed:\n%s\n", c, diff)
		}

		paddedPlaintext[4] = 'X'
		if diff := cmp.Diff(symDeciphered, []byte(payload), nil); diff != "" {
			t.Errorf("Policy: %s\nsymmetric input corruption detected:\n%s\n", c, diff)
		}

		symSignature := localSymmetric.Signature(paddedPlaintext)
		err = remoteSymmetric.VerifySignature(paddedPlaintext, symSignature)
		if err != nil {
			t.Errorf("Policy: %s\nsymmetric signature validation failed\n", c)
		}

		// Asymmetric Algorithm
		asymCiphertext := localAsymmetric.Encrypt(plaintext)
		asymDeciphered := remoteAsymmetric.Decrypt(asymCiphertext)
		if diff := cmp.Diff(asymDeciphered, plaintext, nil); diff != "" {
			t.Errorf("Policy: %s\nasymmetric encryption failed:\n%s\n", c, diff)
		}

		plaintext[4] = 'X'
		if diff := cmp.Diff(asymDeciphered, []byte(payload), nil); diff != "" {
			t.Errorf("Policy: %s\nasymmetric input corruption detected:\n%s\n", c, diff)
		}

		asymSignature := localAsymmetric.Signature(plaintext)
		err = remoteAsymmetric.VerifySignature(plaintext, asymSignature)
		if err != nil {
			t.Errorf("Policy: %s\nasymmetric signature validation failed\n", c)
		}

	}

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
