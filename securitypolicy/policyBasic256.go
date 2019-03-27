// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package securitypolicy

import (
	"crypto"
	"crypto/aes"
	"crypto/rsa"
	"errors"
	"fmt"

	// Force compilation of required hashing algorithms, although we don't directly use the packages
	_ "crypto/sha1"
	_ "crypto/sha256"
)

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

func newBasic256Symmetric(localNonce []byte, remoteNonce []byte) (*EncryptionAlgorithm, error) {
	e := new(EncryptionAlgorithm)

	var (
		signatureKeyLength  = 24
		encryptionKeyLength = 32
		encryptionBlockSize = blockSizeAES()
	)

	localKeys := generateKeys(computeHmac(crypto.SHA1, localNonce), remoteNonce, signatureKeyLength, encryptionKeyLength, encryptionBlockSize)
	remoteKeys := generateKeys(computeHmac(crypto.SHA1, remoteNonce), localNonce, signatureKeyLength, encryptionKeyLength, encryptionBlockSize)

	e.blockSize = aes.BlockSize
	e.plainttextBlockSize = aes.BlockSize - minPaddingAES()
	e.encrypt = encryptAES(256, remoteKeys.iv, remoteKeys.encryption) // AES256-CBC
	e.decrypt = decryptAES(256, localKeys.iv, localKeys.encryption)   // AES256-CBC
	e.signature = computeHmac(crypto.SHA1, remoteKeys.signing)        // HMAC-SHA1
	e.verifySignature = verifyHmac(crypto.SHA1, localKeys.signing)    // HMAC-SHA1
	e.signatureLength = 160 / 8
	e.encryptionURI = "http://www.w3.org/2001/04/xmlenc#aes256-cbc"
	e.signatureURI = "http://www.w3.org/2000/09/xmldsig#hmac-sha1"

	return e, nil
}

func newBasic256Asymmetric(localKey *rsa.PrivateKey, remoteKey *rsa.PublicKey) (*EncryptionAlgorithm, error) {
	const (
		minAsymmetricKeyLength = 128 // 1024 bits
		maxAsymmetricKeyLength = 256 // 2048 bits
		nonceLength            = 32
	)

	if localKey != nil && (localKey.PublicKey.Size() < minAsymmetricKeyLength || localKey.PublicKey.Size() > maxAsymmetricKeyLength) {
		msg := fmt.Sprintf("local key size should be %d-%d bytes, got %d bytes", minAsymmetricKeyLength, maxAsymmetricKeyLength, localKey.PublicKey.Size())
		return nil, errors.New(msg)
	}

	if remoteKey != nil && (remoteKey.Size() < minAsymmetricKeyLength || remoteKey.Size() > maxAsymmetricKeyLength) {
		msg := fmt.Sprintf("remote key size should be %d-%d bytes, got %d bytes", minAsymmetricKeyLength, maxAsymmetricKeyLength, remoteKey.Size())
		return nil, errors.New(msg)
	}

	e := new(EncryptionAlgorithm)

	e.blockSize = remoteKey.Size()
	e.plainttextBlockSize = remoteKey.Size() - minPaddingRsaOAEP(crypto.SHA1)
	e.encrypt = encryptRsaOAEP(crypto.SHA1, remoteKey)         // RSA-OAEP
	e.decrypt = decryptRsaOAEP(crypto.SHA1, localKey)          // RSA-OAEP
	e.signature = signPKCS1v15(crypto.SHA1, localKey)          // RSA-SHA1
	e.verifySignature = verifyPKCS1v15(crypto.SHA1, remoteKey) // RSA-SHA1
	e.nonceLength = nonceLength
	e.signatureLength = localKey.PublicKey.Size()
	e.encryptionURI = "http://www.w3.org/2001/04/xmlenc#rsa-oaep"
	e.signatureURI = "http://www.w3.org/2000/09/xmldsig#rsa-sha1"

	return e, nil
}
