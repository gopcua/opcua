// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uapolicy

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"sort"
	"testing"

	"github.com/pascaldekloe/goe/verify"

	"github.com/gopcua/opcua/ua"
)

func TestSupportedPolicies(t *testing.T) {
	got := SupportedPolicies()
	var want []string
	for k := range policies {
		want = append(want, k)
	}
	sort.Strings(want)
	verify.Values(t, "", got, want)
}

func TestGenerateKeysLength(t *testing.T) {
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

	hmac := &HMAC{Hash: crypto.SHA256, Secret: remoteNonce}
	keys := generateKeys(hmac, localNonce, 32, 32, 16)
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
	if got, want := keys.signing, localKeys.signing; !bytes.Equal(got, want) {
		t.Errorf("local signing key generation failed:\ngot %#v want %#v\n", got, want)
	}
	if got, want := keys.encryption, localKeys.encryption; !bytes.Equal(got, want) {
		t.Errorf("local encryption key generation failed:\ngot %#v want %#v\n", got, want)
	}
	if got, want := keys.iv, localKeys.iv; !bytes.Equal(got, want) {
		t.Errorf("local iv key generation failed:\ngot %#v want %#v\n", got, want)
	}

	remoteHmac := &HMAC{Hash: crypto.SHA1, Secret: remoteNonce}
	keys = generateKeys(remoteHmac, localNonce, 16, 16, 16)
	if got, want := keys.signing, remoteKeys.signing; !bytes.Equal(got, want) {
		t.Errorf("remote signing key generation failed:\ngot %#v want %#v\n", got, want)
	}
	if got, want := keys.encryption, remoteKeys.encryption; !bytes.Equal(got, want) {
		t.Errorf("remote encryption key generation failed:\ngot %#v want %#v\n", got, want)
	}
	if got, want := keys.iv, remoteKeys.iv; !bytes.Equal(got, want) {
		t.Errorf("remote iv key generation failed:\ngot %#v want %#v\n", got, want)
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

	// todo (dh): 2048 happens to be a keysize compatable with all current algorithms.
	// This won't be the case forever and will be too small for future algorithms
	// and the test will need to be able to input keys of varying size
	localKey, err := generatePrivateKey(2048)
	if err != nil {
		t.Fatalf("Unable to generate local private key\n")
	}
	remoteKey, err := generatePrivateKey(2048)
	if err != nil {
		t.Fatalf("Unable to generate remote private key\n")
	}

	for uri, p := range policies {
		t.Run(uri, func(t *testing.T) {
			localAsymmetric, err := p.asymmetric(localKey, &remoteKey.PublicKey)
			if err != nil {
				t.Fatal(err)
			}

			makeNonce := func(n int) []byte {
				t.Helper()
				if n == 0 {
					return nil
				}
				b := make([]byte, n)
				if _, err = rand.Read(b); err != nil {
					t.Fatalf("could not generate nonce")
				}
				return b
			}

			nonceLength := localAsymmetric.NonceLength()
			localNonce, remoteNonce := makeNonce(nonceLength), makeNonce(nonceLength)
			if nonceLength == 0 && uri != ua.SecurityPolicyURINone {
				t.Fatalf("client nonce length zero")
			}

			localSymmetric, err := p.symmetric(localNonce, remoteNonce)
			if err != nil {
				t.Fatalf("failed local Symmetric: %s", err)
			}

			remoteSymmetric, err := p.symmetric(remoteNonce, localNonce)
			if err != nil {
				t.Fatalf("failed remote Symmetric: %s", err)
			}

			remoteAsymmetric, err := p.asymmetric(remoteKey, &localKey.PublicKey)
			if err != nil {
				t.Fatalf("failed remote Asymmetric: %s", err)
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
				t.Fatalf("failed to encrypt Symmetric: %s", err)
			}

			symDeciphered, err := remoteSymmetric.Decrypt(symCiphertext)
			if err != nil {
				t.Fatalf("failed to decrypt Symmetric: %s", err)
			}
			symDeciphered = symDeciphered[:len(symDeciphered)-padSize] // Trim off padding
			if got, want := symDeciphered, plaintext; !bytes.Equal(got, want) {
				t.Errorf("symmetric encryption failed:\ngot %#v want %#v\n", got, want)
			}

			// Modify the plaintext and detect if the decrypted message changes; if it does,
			// our byte slices are referencing the same data and the previous test may have
			// been a false positive
			paddedPlaintext[4] = 0xff ^ paddedPlaintext[4]
			if got, want := symDeciphered, payloadRef; !bytes.Equal(got, want) {
				t.Errorf("symmetric input corruption detected:\ngot %#v want %#v\n", got, want)
			}

			symSignature, err := localSymmetric.Signature(paddedPlaintext)
			if err != nil {
				t.Errorf("symmetric signature generation failed")
			}

			err = remoteSymmetric.VerifySignature(paddedPlaintext, symSignature)
			if err != nil {
				t.Errorf("symmetric signature validation failed")
			}

			// Asymmetric Algorithm
			asymCiphertext, err := localAsymmetric.Encrypt(plaintext)
			if err != nil {
				t.Fatalf("failed to encrypt Asymmetric: %s", err)
			}
			asymDeciphered, err := remoteAsymmetric.Decrypt(asymCiphertext)
			if err != nil {
				t.Fatalf("failed to decrypt Asymmetric: %s", err)
			}
			if got, want := asymDeciphered, plaintext; !bytes.Equal(got, want) {
				t.Errorf("asymmetric encryption failed:\ngot %#v want %#v\n", got, want)
			}

			paddedPlaintext[4] = 0xff ^ paddedPlaintext[4]
			if got, want := asymDeciphered, payloadRef; !bytes.Equal(got, want) {
				t.Errorf("asymmetric input corruption detected:\ngot %#v want %#v\n", got, want)
			}

			asymSignature, err := localAsymmetric.Signature(plaintext)
			if err != nil {
				t.Errorf("asymmetric signature generation failed\n")
			}

			err = remoteAsymmetric.VerifySignature(plaintext, asymSignature)
			if err != nil {
				t.Errorf("asymmetric signature validation failed\n")
			}
		})
	}
}

func TestMissingKey(t *testing.T) {
	payload := make([]byte, 5000)
	_, err := rand.Read(payload)
	if err != nil {
		t.Fatalf("could not generate random payload")
	}

	payloadRef := make([]byte, len(payload))
	copy(payloadRef, payload)

	key, err := generatePrivateKey(2048)
	if err != nil {
		t.Fatalf("Unable to generate private key\n")
	}

	for uri, p := range policies {
		if uri == ua.SecurityPolicyURINone {
			continue
		}

		t.Run(uri, func(t *testing.T) {
			encryptOnly, err := p.asymmetric(nil, &key.PublicKey)
			if err != nil {
				t.Fatalf("failed to create encrypt-only asymmetric algorithms: %s", err)
			}

			decryptOnly, err := p.asymmetric(key, nil)
			if err != nil {
				t.Fatalf("failed to create decrypt-only asymmetric algorithms: %s", err)
			}

			ciphertext, err := encryptOnly.Encrypt(payload)
			if err != nil {
				t.Fatalf("failed to encrypt with encrypt-only policy: %s", err)
			}

			signature, err := decryptOnly.Signature(payload)
			if err != nil {
				t.Fatalf("decrypt-only algorithm failed to generate signature: %s", err)
			}

			err = encryptOnly.VerifySignature(payload, signature)
			if err != nil {
				t.Fatalf("failed to verify signature with encrypt-only algorithm: %s", err)
			}

			plaintext, err := decryptOnly.Decrypt(ciphertext)
			if err != nil {
				t.Fatalf("failed to decrypt with decrypt-only algorithm: %s", err)
			}

			if got, want := plaintext, payloadRef; !bytes.Equal(got, want) {
				t.Errorf("decryption failed:\ngot %#v want %#v\n", got, want)
			}

			_, err = encryptOnly.Decrypt(ciphertext)
			if err == nil {
				t.Fatal("encrypt-only algorithm decrypted block without error - should be impossible")
			}

			_, err = encryptOnly.Signature(payload)
			if err == nil {
				t.Fatalf("encrypt-only algorithm generated a signature without error - should be impossible")
			}

			_, err = decryptOnly.Encrypt(payload)
			if err == nil {
				t.Fatal("decrypt-only algorithm encrypted block without error - should be impossible")
			}

			err = decryptOnly.VerifySignature(payload, signature)
			if err == nil {
				t.Fatalf("decrypt-only algorithm verified a signature without error - should be impossible")
			}

		})

	}
}

func TestZeroStruct(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Error("panicked while checking zero value of struct", r)
		}
	}()

	ze := &EncryptionAlgorithm{}

	const payload string = "The quick brown fox jumps over the lazy dog."
	plaintext := []byte(payload)

	// Call all the methods and make sure they don't panic due to nil pointers
	_ = ze.BlockSize()
	_ = ze.PlaintextBlockSize()
	_, _ = ze.Encrypt(plaintext)
	_, _ = ze.Decrypt(plaintext)
	_, _ = ze.Signature(plaintext)
	_ = ze.VerifySignature(plaintext, plaintext)
	_ = ze.NonceLength()
	_ = ze.SignatureLength()
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
