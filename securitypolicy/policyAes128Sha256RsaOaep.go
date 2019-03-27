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
 * 	"SecurityPolicy [A] - Aes128-Sha256-RsaOaep" Profile
 	http://opcfoundation.org/UA/SecurityPolicy#Aes128_Sha256_RsaOaep

  	Name	Opt.	Description		From Profile
	Security Certificate Validation 		A certificate will be validated as specified in Part 4. This includes among others structure and signature examination. Allowing for some validation errors to be suppressed by administration directive.
	Security Encryption Required 		Encryption is required using the algorithms provided in the security algorithm suite.
	Security Signing Required 		Signing is required using the algorithms provided in the security algorithm suite.
	SymmetricSignatureAlgorithm_HMAC-SHA2-256 		A keyed hash used for message authentication which is defined in https://tools.ietf.org/html/rfc2104. The hash algorithm is SHA2 with 256 bits and described in https://tools.ietf.org/html/rfc4634
	SymmetricEncryptionAlgorithm_AES128-CBC 		The AES encryption algorithm which is defined in http://nvlpubs.nist.gov/nistpubs/FIPS/NIST.FIPS.197.pdf.
Multiple blocks encrypted using the CBC mode described in http://nvlpubs.nist.gov/nistpubs/Legacy/SP/nistspecialpublication800-38a.pdf.
The key size is 128 bits. The block size is 16 bytes.
The URI is http://www.w3.org/2001/04/xmlenc#aes128-cbc.
	AsymmetricSignatureAlgorithm_RSA-PKCS15-SHA2-256 		The RSA signature algorithm which is defined in https://tools.ietf.org/html/rfc3447.
The RSASSA-PKCS1-v1_5 scheme is used.
The hash algorithm is SHA2 with 256bits and is described in https://tools.ietf.org/html/rfc6234.
The URI is http://www.w3.org/2001/04/xmldsig-more#rsa-sha256.
	AsymmetricEncryptionAlgorithm_RSA-OAEP-SHA1 		The RSA encryption algorithm which is defined in https://tools.ietf.org/html/rfc3447.
The RSAES-OAEP scheme is used.
The hash algorithm is SHA1 and is described in https://tools.ietf.org/html/rfc6234.
The mask generation algorithm also uses SHA1.
The URI is http://www.w3.org/2001/04/xmlenc#rsa-oaep.
No known exploits exist when using SHA1 with RSAES-OAEP, however, SHA1 was broken in 2017 so use of this algorithm is not recommended.
	KeyDerivationAlgorithm_P-SHA2-256 		The P_SHA256 pseudo-random function defined in https://tools.ietf.org/html/rfc5246.
The URI is http://docs.oasis-open.org/ws-sx/ws-secureconversation/200512/dk/p_sha256.
	CertificateSignatureAlgorithm_RSA-PKCS15-SHA2-256 		The RSA signature algorithm which is defined in https://tools.ietf.org/html/rfc3447.
The RSASSA-PKCS1-v1_5 scheme is used.
The hash algorithm is SHA2 with 256bits and is described in https://tools.ietf.org/html/rfc6234.
The SHA2 algorithm with 384 or 512 bits may be used instead of SHA2 with 256 bits.
The URI is http://www.w3.org/2001/04/xmldsig-more#rsa-sha256.
	Aes128-Sha256-RsaOaep_Limits 		-> DerivedSignatureKeyLength: 256 bits
-> MinAsymmetricKeyLength: 2048 bits
-> MaxAsymmetricKeyLength: 4096 bits
-> SecureChannelNonceLength: 32 bytes
*/

func newAes128Sha256RsaOaepSymmetric(localNonce []byte, remoteNonce []byte) (*EncryptionAlgorithm, error) {
	e := new(EncryptionAlgorithm)

	var (
		signatureKeyLength  = 32
		encryptionKeyLength = 16
		encryptionBlockSize = blockSizeAES()
	)

	localKeys := generateKeys(computeHmac(crypto.SHA256, localNonce), remoteNonce, signatureKeyLength, encryptionKeyLength, encryptionBlockSize)
	remoteKeys := generateKeys(computeHmac(crypto.SHA256, remoteNonce), localNonce, signatureKeyLength, encryptionKeyLength, encryptionBlockSize)

	e.blockSize = aes.BlockSize
	e.plainttextBlockSize = aes.BlockSize - minPaddingAES()
	e.encrypt = encryptAES(128, remoteKeys.iv, remoteKeys.encryption) // AES128-CBC
	e.decrypt = decryptAES(128, localKeys.iv, localKeys.encryption)   // AES128-CBC
	e.signature = computeHmac(crypto.SHA256, remoteKeys.signing)      // HMAC-SHA2-256
	e.verifySignature = verifyHmac(crypto.SHA256, localKeys.signing)  // HMAC-SHA2-256
	e.signatureLength = 256 / 8
	e.encryptionURI = "http://www.w3.org/2001/04/xmlenc#aes128-cbc"
	e.signatureURI = "http://www.w3.org/2000/09/xmldsig#hmac-sha256"

	return e, nil
}

func newAes128Sha256RsaOaepAsymmetric(localKey *rsa.PrivateKey, remoteKey *rsa.PublicKey) (*EncryptionAlgorithm, error) {
	const (
		minAsymmetricKeyLength = 256 // 2048 bits
		maxAsymmetricKeyLength = 512 // 4096 bits
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
	e.encrypt = encryptRsaOAEP(crypto.SHA1, remoteKey)           // RSA-OAEP-SHA1
	e.decrypt = decryptRsaOAEP(crypto.SHA1, localKey)            // RSA-OAEP-SHA1
	e.signature = signPKCS1v15(crypto.SHA256, localKey)          // RSA-PKCS15-SHA2-256
	e.verifySignature = verifyPKCS1v15(crypto.SHA256, remoteKey) // RSA-PKCS15-SHA2-256
	e.nonceLength = nonceLength
	e.signatureLength = localKey.PublicKey.Size()
	e.encryptionURI = "http://opcfoundation.org/ua/security/rsa-oaep-sha1"
	e.signatureURI = "http://www.w3.org/2001/04/xmldsig-more#rsa-sha256"

	return e, nil
}
