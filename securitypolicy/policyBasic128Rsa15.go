// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package securitypolicy

import (
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"os"
)

/*
OLD SecurityPolicy – Basic128Rsa15" Profile (DEPRECATED IN 1.04)
http://opcfoundation.org/UA/SecurityPolicy#Basic128Rsa15

Name 	Opt. 	 Description 	 From Profile
	Security Certificate Validation 		A certificate will be validated as specified in Part 4. This includes among others structure and signature examination. Allowing for some validation errors to be suppressed by administration directive.
	Security Basic 128Rsa15 		A suite of algorithms that uses RSA15 as Key-Wrap-algorithm and 128-Bit for encryption algorithms.
-> SymmetricSignatureAlgorithm – HmacSha1 – (http://www.w3.org/2000/09/xmldsig#hmac-sha1).
-> SymmetricEncryptionAlgorithm – Aes128 – (http://www.w3.org/2001/04/xmlenc#aes128-cbc).
-> AsymmetricSignatureAlgorithm – RsaSha1 – (http://www.w3.org/2000/09/xmldsig#rsa-sha1).
-> AsymmetricKeyWrapAlgorithm – KwRsa15 – (http://www.w3.org/2001/04/xmlenc#rsa-1_5).
-> AsymmetricEncryptionAlgorithm – Rsa15 – (http://www.w3.org/2001/04/xmlenc#rsa-1_5).
-> KeyDerivationAlgorithm – PSha1 – (http://docs.oasis-open.org/ws-sx/ws-secureconversation/200512/dk/p_sha1).
-> DerivedSignatureKeyLength – 128.
-> MinAsymmetricKeyLength – 1024
-> MaxAsymmetricKeyLength – 2048
-> CertificateSignatureAlgorithm – Sha1

If a certificate or any certificate in the chain is not signed with a hash that is Sha1 or stronger then the certificate shall be rejected.
	Security Encryption Required 		Encryption is required using the algorithms provided in the security algorithm suite.
	Security Signing Required 		Signing is required using the algorithms provided in the security algorithm suite.


*/

func newBasic128Rsa15Symmetric(localNonce []byte, remoteNonce []byte) *EncryptionAlgorithm {
	e := new(EncryptionAlgorithm)

	var (
		signatureKeyLength  = 16
		encryptionKeyLength = 16
		encryptionBlockSize = blockSizeAES()
	)

	localKeys := generateKeys(hmacSha1(remoteNonce), localNonce, signatureKeyLength, encryptionKeyLength, encryptionBlockSize)
	remoteKeys := generateKeys(hmacSha1(localNonce), remoteNonce, signatureKeyLength, encryptionKeyLength, encryptionBlockSize)

	e.blockSize = blockSizeAES
	e.encrypt = encryptAES(128, remoteKeys.iv, remoteKeys.encryption) // AES128
	e.decrypt = decryptAES(128, localKeys.iv, localKeys.encryption)   // AES128
	e.signature = hmacSha1(remoteKeys.signing)                        // HMAC-SHA1
	e.verifySignature = verifyHmacSha1(localKeys.signing)             // HMAC-SHA1

	return e
}

func newBasic128Rsa15Asymmetric(localKey *rsa.PrivateKey, remoteKey *rsa.PublicKey) *EncryptionAlgorithm {
	e := new(EncryptionAlgorithm)

	e.blockSize = blockSizeNone
	e.encrypt = encryptPKCS1v15(remoteKey)            // RSA-SHA15+KWRSA15
	e.decrypt = decryptPKCS1v15(localKey)             // RSA-SHA15+KWRSA15
	e.signature = signRsaPkc15Sha1(localKey)          // RSA-SHA1
	e.verifySignature = verifyRsaPkc15Sha1(remoteKey) // RSA-SHA1

	return e
}

func decryptPKCS1v15(privKey *rsa.PrivateKey) func([]byte) []byte {
	return func(src []byte) []byte {

		// crypto/rand.Reader is a good source of entropy for blinding the RSA
		// operation.
		rng := rand.Reader

		plaintext, err := rsa.DecryptPKCS1v15(rng, privKey, src)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error from decryption: %s\n", err)
		}

		return plaintext
		// Remember that encryption only provides confidentiality. The
		// ciphertext should be signed before authenticity is assumed and, even
		// then, consider that messages might be reordered.
	}
}

func encryptPKCS1v15(pubKey *rsa.PublicKey) func([]byte) []byte {
	return func(src []byte) []byte {
		// crypto/rand.Reader is a good source of entropy for randomizing the
		// encryption function.
		rng := rand.Reader

		ciphertext, err := rsa.EncryptPKCS1v15(rng, pubKey, src)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error from encryption: %s\n", err)
		}

		return ciphertext
		// Since encryption is a randomized function, ciphertext will be
		// different each time.
	}
}
