// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package securitypolicy

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"fmt"
	"os"
)

func decryptRsaOAEPSha1(privKey *rsa.PrivateKey) func([]byte) []byte {
	return func(src []byte) []byte {

		// crypto/rand.Reader is a good source of entropy for blinding the RSA
		// operation.
		rng := rand.Reader

		plaintext, err := rsa.DecryptOAEP(sha1.New(), rng, privKey, src, nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error from decryption: %s\n", err)
		}

		return plaintext
		// Remember that encryption only provides confidentiality. The
		// ciphertext should be signed before authenticity is assumed and, even
		// then, consider that messages might be reordered.
	}
}

func encryptRsaOAEPSha1(pubKey *rsa.PublicKey) func([]byte) []byte {
	return func(src []byte) []byte {
		// crypto/rand.Reader is a good source of entropy for randomizing the
		// encryption function.
		rng := rand.Reader

		ciphertext, err := rsa.EncryptOAEP(sha1.New(), rng, pubKey, src, nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error from encryption: %s\n", err)
		}

		return ciphertext
		// Since encryption is a randomized function, ciphertext will be
		// different each time.
	}
}
