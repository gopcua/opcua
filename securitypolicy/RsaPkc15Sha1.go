// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package securitypolicy

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
)

func signRsaPkc15Sha1(privKey *rsa.PrivateKey) func([]byte) []byte {
	rng := rand.Reader

	return func(msg []byte) []byte {
		hashed := sha1.Sum(msg)

		// TODO: Add error handling
		signature, _ := rsa.SignPKCS1v15(rng, privKey, crypto.SHA1, hashed[:])

		return signature
	}
}

func verifyRsaPkc15Sha1(pubKey *rsa.PublicKey) func([]byte, []byte) error {

	return func(msg, signature []byte) error {
		hashed := sha1.Sum(msg)

		err := rsa.VerifyPKCS1v15(pubKey, crypto.SHA1, hashed[:], signature)

		return err
	}
}
