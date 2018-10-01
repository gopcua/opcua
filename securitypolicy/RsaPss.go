// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package securitypolicy

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
)

func signRsaPssSha256(privKey *rsa.PrivateKey) func([]byte) []byte {
	rng := rand.Reader

	return func(msg []byte) []byte {
		hashed := sha256.Sum256(msg)

		// TODO: Add error handling
		signature, _ := rsa.SignPSS(rng, privKey, crypto.SHA256, hashed[:], nil)

		return signature
	}
}

func verifyRsaPssSha256(pubKey *rsa.PublicKey) func([]byte, []byte) error {

	return func(msg, signature []byte) error {
		hashed := sha256.Sum256(msg)

		err := rsa.VerifyPSS(pubKey, crypto.SHA256, hashed[:], signature, nil)

		return err
	}
}
