// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package securitypolicy

import "crypto/rsa"

/*
 * "SecurityPolicy [A] - Aes128-Sha256-RsaOaep" Profile
	  http://opcfoundation.org/UA/SecurityPolicy#Aes128_Sha256_RsaOaep


  Name 	Opt. 	 Description 	 From Profile
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

func newAes128Sha256RsaOaepSymmetric(localNonce []byte, remoteNonce []byte) *EncryptionAlgorithm {
	e := new(EncryptionAlgorithm)

	var (
		signatureKeyLength  = 32
		encryptionKeyLength = 16
		encryptionBlockSize = blockSizeAES()
	)

	localKeys := generateKeys(hmacSha256(remoteNonce), localNonce, signatureKeyLength, encryptionKeyLength, encryptionBlockSize)
	remoteKeys := generateKeys(hmacSha256(localNonce), remoteNonce, signatureKeyLength, encryptionKeyLength, encryptionBlockSize)

	e.blockSize = blockSizeAES
	e.encrypt = encryptAES(128, remoteKeys.iv, remoteKeys.encryption) // AES128-CBC
	e.decrypt = decryptAES(128, localKeys.iv, localKeys.encryption)   // AES128-CBC
	e.signature = hmacSha256(remoteKeys.signing)                      // HMAC-SHA2-256
	e.verifySignature = verifyHmacSha256(localKeys.signing)           // HMAC-SHA2-256

	return e
}

func newAes128Sha256RsaOaepAsymmetric(localKey *rsa.PrivateKey, remoteKey *rsa.PublicKey) *EncryptionAlgorithm {
	e := new(EncryptionAlgorithm)

	e.blockSize = blockSizeNone
	e.encrypt = encryptRsaOAEPSha1(remoteKey)           // RSA-OAEP-SHA1
	e.decrypt = decryptRsaOAEPSha1(localKey)            // RSA-OAEP-SHA1
	e.signature = signRsaPkc15Sha256(localKey)          // RSA-PKCS15-SHA2-256
	e.verifySignature = verifyRsaPkc15Sha256(remoteKey) // RSA-PKCS15-SHA2-256

	return e
}
