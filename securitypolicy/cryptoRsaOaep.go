// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package securitypolicy

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"

	// Force compilation of required hashing algorithms, although we don't directly use the packages
	_ "crypto/sha1"
	_ "crypto/sha256"
)

func minPaddingRsaOAEP(hash crypto.Hash) int {
	// messageLen = (keyLenBits / 8) - 2*(hashLenBits / 8) - 2
	// paddingLen = keyLen - messageLen
	//            = 2*hashLenBytes + 2
	var hLen int
	switch hash {
	case crypto.SHA1:
		hLen = 20
	case crypto.SHA256:
		hLen = 64
	}

	return (2 * hLen) + 2
}

func decryptRsaOAEP(hash crypto.Hash, privKey *rsa.PrivateKey) func([]byte) ([]byte, error) {
	rng := rand.Reader

	return func(src []byte) ([]byte, error) {
		var plaintext []byte

		blockSize := keySize(&privKey.PublicKey)
		srcRemaining := len(src)
		start := 0

		for srcRemaining > 0 {
			end := start + blockSize
			if end > len(src) {
				end = len(src)
			}

			p, err := rsa.DecryptOAEP(hash.New(), rng, privKey, src[start:end], nil)
			if err != nil {
				return nil, err
			}

			plaintext = append(plaintext, p...)
			start = end
			srcRemaining = len(src) - start
		}

		return plaintext, nil
	}
}

func encryptRsaOAEP(hash crypto.Hash, pubKey *rsa.PublicKey) func([]byte) ([]byte, error) {
	rng := rand.Reader

	return func(src []byte) ([]byte, error) {
		var ciphertext []byte

		maxBlock := keySize(pubKey) - minPaddingRsaOAEP(hash)
		srcRemaining := len(src)
		start := 0
		for srcRemaining > 0 {
			end := start + maxBlock
			if end > len(src) {
				end = len(src)
			}

			c, err := rsa.EncryptOAEP(hash.New(), rng, pubKey, src[start:end], nil)
			if err != nil {
				return nil, err
			}

			ciphertext = append(ciphertext, c...)
			start = end
			srcRemaining = len(src) - start
		}

		return ciphertext, nil
	}
}
