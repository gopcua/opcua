// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package securitypolicy

import (
	"crypto"
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

	keys := generateKeys(computeHmac(crypto.SHA256, remoteNonce), localNonce, 32, 32, 16)
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
	payload := make([]byte, 5000)
	_, err := rand.Read(payload)
	if err != nil {
		t.Fatalf("could not generate random payload")
	}

	payloadRef := make([]byte, len(payload))
	copy(payloadRef, payload)

	localNonce := make([]byte, 32)
	remoteNonce := make([]byte, 32)
	_, err = rand.Read(localNonce)
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

		localSymmetric, err := Symmetric(c, localNonce, remoteNonce)
		if err != nil {
			t.Fatalf("failed local Symmetric New(%s) : %s", c, err)
		}
		localAsymmetric, err := Asymmetric(c, localKey, &remoteKey.PublicKey)
		if err != nil {
			t.Fatalf("failed local Asymmetric New(%s) : %s", c, err)
		}

		remoteSymmetric, err := Symmetric(c, remoteNonce, localNonce)
		if err != nil {
			t.Fatalf("failed remote Symmetric New(%s) : %s", c, err)
		}

		remoteAsymmetric, err := Asymmetric(c, remoteKey, &localKey.PublicKey)
		if err != nil {
			t.Fatalf("failed remote Asymmetric New(%s) : %s", c, err)
		}

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
		if err != nil {
			t.Fatalf("failed to encrypt Symmetric (%s) : %s", c, err)
		}

		symDeciphered, err := remoteSymmetric.Decrypt(symCiphertext)
		if err != nil {
			t.Fatalf("failed to decrypt Symmetric (%s) : %s", c, err)
		}
		symDeciphered = symDeciphered[:len(symDeciphered)-padSize] // Trim off padding
		if diff := cmp.Diff(symDeciphered, plaintext); diff != "" {
			t.Errorf("Policy: %s\nsymmetric encryption failed:\n%s\n", c, diff)
		}

		// Modify the plaintext and detect if the decrypted message changes; if it does,
		// our byte slices are referencing the same data and the previous test may have
		// been a false positive
		paddedPlaintext[4] = 0xff ^ paddedPlaintext[4]
		if diff := cmp.Diff(symDeciphered, payloadRef); diff != "" {
			t.Errorf("Policy: %s\nsymmetric input corruption detected:\n%s\n", c, diff)
		}

		symSignature, err := localSymmetric.Signature(paddedPlaintext)
		if err != nil {
			t.Errorf("Policy: %s\nsymmetric signature generation failed\n", c)
		}

		err = remoteSymmetric.VerifySignature(paddedPlaintext, symSignature)
		if err != nil {
			t.Errorf("Policy: %s\nsymmetric signature validation failed\n", c)
		}

		// Asymmetric Algorithm
		asymCiphertext, err := localAsymmetric.Encrypt(plaintext)
		if err != nil {
			t.Fatalf("failed to encrypt Asymmetric (%s) : %s", c, err)
		}
		asymDeciphered, err := remoteAsymmetric.Decrypt(asymCiphertext)
		if err != nil {
			t.Fatalf("failed to decrypt Asymmetric (%s) : %s", c, err)
		}
		if diff := cmp.Diff(asymDeciphered, plaintext); diff != "" {
			t.Errorf("Policy: %s\nasymmetric encryption failed:\n%s\n", c, diff)
		}

		paddedPlaintext[4] = 0xff ^ paddedPlaintext[4]
		if diff := cmp.Diff(asymDeciphered, payloadRef); diff != "" {
			t.Errorf("Policy: %s\nasymmetric input corruption detected:\n%s\n", c, diff)
		}

		asymSignature, err := localAsymmetric.Signature(plaintext)
		if err != nil {
			t.Errorf("Policy: %s\nasymmetric signature generation failed\n", c)
		}

		err = remoteAsymmetric.VerifySignature(plaintext, asymSignature)
		if err != nil {
			t.Errorf("Policy: %s\nasymmetric signature validation failed\n", c)
		}

	}

}

func TestZeroStruct(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Error("panicked while checking zero value of structs", r)
		}
	}()

	ze := &EncryptionAlgorithm{}

	const payload string = "The quick brown fox jumps over the lazy dog."
	plaintext := []byte(payload)

	// Call all the methods and make sure they don't panic due to nil pointers
	_ = ze.BlockSize()
	_ = ze.MinPadding()
	_, _ = ze.Encrypt(plaintext)
	_, _ = ze.Decrypt(plaintext)
	_, _ = ze.Signature(plaintext)
	_ = ze.VerifySignature(plaintext, plaintext)
	_ = ze.EncryptionURI()
	_ = ze.SignatureURI()

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
