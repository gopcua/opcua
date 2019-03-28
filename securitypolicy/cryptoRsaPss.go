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

func signRsaPss(hash crypto.Hash, privKey *rsa.PrivateKey) func([]byte) ([]byte, error) {
	rng := rand.Reader

	return func(msg []byte) ([]byte, error) {
		h := hash.New()
		h.Write(msg)
		hashed := h.Sum(nil)

		signature, err := rsa.SignPSS(rng, privKey, hash, hashed[:], &rsa.PSSOptions{SaltLength: rsa.PSSSaltLengthEqualsHash})
		if err != nil {
			return nil, err
		}

		return signature, nil
	}
}

func verifyRsaPss(hash crypto.Hash, pubKey *rsa.PublicKey) func([]byte, []byte) error {
	return func(msg, signature []byte) error {
		h := hash.New()
		h.Write(msg)
		hashed := h.Sum(nil)

		err := rsa.VerifyPSS(pubKey, hash, hashed[:], signature, nil)

		return err
	}
}
