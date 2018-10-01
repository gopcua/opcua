// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package securitypolicy

import "crypto/rsa"

/*
"SecurityPolicy - Aes256-Sha256-RsaPss" Profile
http://opcfoundation.org/UA/SecurityPolicy#Aes256_Sha256_RsaPss

 	Security Certificate Validation 		A certificate will be validated as specified in Part 4. This includes among others structure and signature examination. Allowing for some validation errors to be suppressed by administration directive.
	Security Encryption Required 		Encryption is required using the algorithms provided in the security algorithm suite.
	Security Signing Required 		Signing is required using the algorithms provided in the security algorithm suite.
	SymmetricSignatureAlgorithm_HMAC-SHA2-256 		A keyed hash used for message authentication which is defined in https://tools.ietf.org/html/rfc2104.
The hash algorithm is SHA2 with 256 bits and described in https://tools.ietf.org/html/rfc4634

	SymmetricEncryptionAlgorithm_AES256-CBC 		The AES encryption algorithm which is defined in http://nvlpubs.nist.gov/nistpubs/FIPS/NIST.FIPS.197.pdf.
Multiple blocks encrypted using the CBC mode described in http://nvlpubs.nist.gov/nistpubs/Legacy/SP/nistspecialpublication800-38a.pdf.
The key size is 256 bits. The block size is 16 bytes.
The URI is http://www.w3.org/2001/04/xmlenc#aes256-cbc.
	AsymmetricSignatureAlgorithm_RSA-PSS-SHA2-256 		The RSA signature algorithm which is defined in https://tools.ietf.org/html/rfc3447.
The RSASSA-PSS scheme is used.
The hash algorithm is SHA2 with 256bits and is described in https://tools.ietf.org/html/rfc6234.
The mask generation algorithm also uses SHA2 with 256 bits.
The salt length is 32 bytes.
The URI is http://opcfoundation.org/UA/security/rsa-pss-sha2-256.
	AsymmetricEncryptionAlgorithm_RSA-OAEP-SHA2-256 		The RSA encryption algorithm which is defined in https://tools.ietf.org/html/rfc3447.
The RSAES-OAEP scheme is used.
The hash algorithm is SHA2 with 256 bits and is described in https://tools.ietf.org/html/rfc6234.
The mask generation algorithm also uses SHA2 with 256 bits.
The URI is http://opcfoundation.org/UA/security/rsa-oaep-sha2-256.
	KeyDerivationAlgorithm_P-SHA2-256 		The P_SHA256 pseudo-random function defined in https://tools.ietf.org/html/rfc5246.
The URI is http://docs.oasis-open.org/ws-sx/ws-secureconversation/200512/dk/p_sha256.
	CertificateSignatureAlgorithm_RSA-PKCS15-SHA2-256 		The RSA signature algorithm which is defined in https://tools.ietf.org/html/rfc3447.
The RSASSA-PKCS1-v1_5 scheme is used.
The hash algorithm is SHA2 with 256bits and is described in https://tools.ietf.org/html/rfc6234.
The SHA2 algorithm with 384 or 512 bits may be used instead of SHA2 with 256 bits.
The URI is http://www.w3.org/2001/04/xmldsig-more#rsa-sha256.
	Aes256-Sha256-RsaPss_Limits 		-> DerivedSignatureKeyLength: 256 bits
-> MinAsymmetricKeyLength: 2048 bits
-> MaxAsymmetricKeyLength: 4096 bits
-> SecureChannelNonceLength: 32 bytes


*/

func newAes256Sha256RsaPssSymmetric(localNonce []byte, remoteNonce []byte) *EncryptionAlgorithm {
	e := new(EncryptionAlgorithm)

	var (
		signatureKeyLength  = 32
		encryptionKeyLength = 32
		encryptionBlockSize = blockSizeAES()
	)

	localKeys := generateKeys(hmacSha256(remoteNonce), localNonce, signatureKeyLength, encryptionKeyLength, encryptionBlockSize)
	remoteKeys := generateKeys(hmacSha256(localNonce), remoteNonce, signatureKeyLength, encryptionKeyLength, encryptionBlockSize)

	e.blockSize = blockSizeAES
	e.encrypt = encryptAES(256, remoteKeys.iv, remoteKeys.encryption) // AES256-CBC
	e.decrypt = decryptAES(256, localKeys.iv, localKeys.encryption)   // AES256-CBC
	e.signature = hmacSha256(remoteKeys.signing)                      // HMAC-SHA2-256
	e.verifySignature = verifyHmacSha256(localKeys.signing)           // HMAC-SHA2-256

	return e
}

func newAes256Sha256RsaPssAsymmetric(localKey *rsa.PrivateKey, remoteKey *rsa.PublicKey) *EncryptionAlgorithm {
	e := new(EncryptionAlgorithm)

	e.blockSize = blockSizeNone
	e.encrypt = encryptRsaOAEPSha1(remoteKey)         // RSA-OAEP-SHA1
	e.decrypt = decryptRsaOAEPSha1(localKey)          // RSA-OAEP-SHA1
	e.signature = signRsaPssSha256(localKey)          // RSA-PSS-SHA2-256
	e.verifySignature = verifyRsaPssSha256(remoteKey) // RSA-PSS-SHA2-256

	return e
}
