// Copyright 2018-2019 opcua authors. All rights reserved.
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

func minPaddingRsaPKCS1v15() int {
	return 11
}

func decryptPKCS1v15(privKey *rsa.PrivateKey) func([]byte) ([]byte, error) {
	rng := rand.Reader

	return func(src []byte) ([]byte, error) {
		var plaintext []byte

		blockSize := privKey.PublicKey.Size()
		srcRemaining := len(src)
		start := 0

		for srcRemaining > 0 {
			end := start + blockSize
			if end > len(src) {
				end = len(src)
			}

			p, err := rsa.DecryptPKCS1v15(rng, privKey, src[start:end])
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

func encryptPKCS1v15(pubKey *rsa.PublicKey) func([]byte) ([]byte, error) {
	rng := rand.Reader

	return func(src []byte) ([]byte, error) {
		var ciphertext []byte

		maxBlock := pubKey.Size() - minPaddingRsaPKCS1v15()
		srcRemaining := len(src)
		start := 0
		for srcRemaining > 0 {
			end := start + maxBlock
			if end > len(src) {
				end = len(src)
			}

			c, err := rsa.EncryptPKCS1v15(rng, pubKey, src[start:end])
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

func signPKCS1v15(hash crypto.Hash, privKey *rsa.PrivateKey) func([]byte) ([]byte, error) {
	rng := rand.Reader

	return func(msg []byte) ([]byte, error) {
		h := hash.New()
		h.Write(msg)
		hashed := h.Sum(nil)

		signature, err := rsa.SignPKCS1v15(rng, privKey, hash, hashed[:])
		if err != nil {
			return nil, err
		}
		return signature, err
	}
}

func verifyPKCS1v15(hash crypto.Hash, pubKey *rsa.PublicKey) func([]byte, []byte) error {
	return func(msg, signature []byte) error {
		h := hash.New()
		h.Write(msg)
		hashed := h.Sum(nil)

		err := rsa.VerifyPKCS1v15(pubKey, hash, hashed[:], signature)

		return err
	}
}
