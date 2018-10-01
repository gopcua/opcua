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

func signRsaPkc15Sha256(privKey *rsa.PrivateKey) func([]byte) []byte {
	rng := rand.Reader

	return func(msg []byte) []byte {
		hashed := sha256.Sum256(msg)

		// TODO: Add error handling
		signature, _ := rsa.SignPKCS1v15(rng, privKey, crypto.SHA256, hashed[:])

		return signature
	}
}

func verifyRsaPkc15Sha256(pubKey *rsa.PublicKey) func([]byte, []byte) error {

	return func(msg, signature []byte) error {
		hashed := sha256.Sum256(msg)

		err := rsa.VerifyPKCS1v15(pubKey, crypto.SHA256, hashed[:], signature)

		return err
	}
}
