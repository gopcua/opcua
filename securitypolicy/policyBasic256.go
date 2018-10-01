// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package securitypolicy

import "crypto/rsa"

/*
"OLD SecurityPolicy – Basic256" Profile
http://opcfoundation.org/UA/SecurityPolicy#Basic256

Name 	Opt. 	 Description 	 From Profile
	Security Certificate Validation 		A certificate will be validated as specified in Part 4. This includes among others structure and signature examination. Allowing for some validation errors to be suppressed by administration directive.
	Security Basic 256 		A suite of algorithms that are for 256-Bit encryption, algorithms include:
-> SymmetricSignatureAlgorithm – HmacSha1 – (http://www.w3.org/2000/09/xmldsig#hmac-sha1).
-> SymmetricEncryptionAlgorithm – Aes256_CBC – (http://www.w3.org/2001/04/xmlenc#aes256-cbc).
-> AsymmetricSignatureAlgorithm – RsaSha1 – (http://www.w3.org/2000/09/xmldsig#rsa-sha1).
-> AsymmetricKeyWrapAlgorithm – KwRsaOaep – (http://www.w3.org/2001/04/xmlenc#rsa-oaep-mgf1p).
-> AsymmetricEncryptionAlgorithm – RsaOaep – (http://www.w3.org/2001/04/xmlenc#rsa-oaep).
-> KeyDerivationAlgorithm – PSha1 – (http://docs.oasis-open.org/ws-sx/ws-secureconversation/200512/dk/p_sha1).
-> DerivedSignatureKeyLength – 192.
-> MinAsymmetricKeyLength – 1024
-> MaxAsymmetricKeyLength – 2048
-> CertificateSignatureAlgorithm – Sha1 [deprecated] or Sha256 [recommended]

If a certificate or any certificate in the chain is not signed with a hash that is Sha1 or stronger then the certificate shall be rejected.
Both Sha1 and Sha256 shall be supported. However, it is recommended to use Sha256 since Sha1 is considered not secure anymore.
	Security Encryption Required 		Encryption is required using the algorithms provided in the security algorithm suite.
	Security Signing Required 		Signing is required using the algorithms provided in the security algorithm suite.
*/

func newBasic256Symmetric(localNonce []byte, remoteNonce []byte) *EncryptionAlgorithm {
	e := new(EncryptionAlgorithm)

	var (
		signatureKeyLength  = 24
		encryptionKeyLength = 32
		encryptionBlockSize = blockSizeAES()
	)

	localKeys := generateKeys(hmacSha1(remoteNonce), localNonce, signatureKeyLength, encryptionKeyLength, encryptionBlockSize)
	remoteKeys := generateKeys(hmacSha1(localNonce), remoteNonce, signatureKeyLength, encryptionKeyLength, encryptionBlockSize)

	e.blockSize = blockSizeAES
	e.encrypt = encryptAES(256, remoteKeys.iv, remoteKeys.encryption) // AES256-CBC
	e.decrypt = decryptAES(256, localKeys.iv, localKeys.encryption)   // AES256-CBC
	e.signature = hmacSha1(remoteKeys.signing)                        // HMAC-SHA1
	e.verifySignature = verifyHmacSha1(localKeys.signing)             // HMAC-SHA1

	return e
}

func newBasic256Asymmetric(localKey *rsa.PrivateKey, remoteKey *rsa.PublicKey) *EncryptionAlgorithm {
	e := new(EncryptionAlgorithm)

	e.blockSize = blockSizeNone
	e.encrypt = encryptRsaOAEPSha1(remoteKey)         // RSA-OAEP
	e.decrypt = decryptRsaOAEPSha1(localKey)          // RSA-OAEP
	e.signature = signRsaPkc15Sha1(localKey)          // RSA-SHA1
	e.verifySignature = verifyRsaPkc15Sha1(remoteKey) // RSA-SHA1

	return e
}
