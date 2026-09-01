// Copyright 2018-2020 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uapolicy

import (
	"crypto/rsa"
)

/*
"SecurityPolicy – None" Profile
http://opcfoundation.org/UA/SecurityPolicy#None

Security None CreateSession ActivateSession
When SecurityPolicy=None, the CreateSession and ActivateSession service allow for a NULL/empty signature and do not require Application Certificates or a Nonce.
(OPT) Security None CreateSession ActivateSession 1.0

	The Client can connect to Servers that require a certificate being passed on Session establishment.
	The Client in this case will first try without a certificate and if this fails present a certificate.

SymmetricSignatureAlgorithm_None 		This algorithm does not apply.
SymmetricEncryptionAlgorithm_None 		This algorithm does not apply.
AsymmetricSignatureAlgorithm_None 		This algorithm does not apply.
AsymmetricEncryptionAlgorithm_None 		This algorithm does not apply.
KeyDerivationAlgorithm_None 		This algorithm does not apply.
SecurityPolicy_None_Limits 		DerivedSignatureKeyLength: 0
*/
func newNoneAsymmetric(*rsa.PrivateKey, *rsa.PublicKey) (*EncryptionAlgorithm, error) {
	return &EncryptionAlgorithm{
		blockSize:             NoneBlockSize,
		plainttextBlockSize:   NoneBlockSize - NoneMinPadding,
		encrypt:               &None{},
		decrypt:               &None{},
		signature:             &None{},
		verifySignature:       &None{},
		signatureLength:       0,
		remoteSignatureLength: 0,
		// A 0-length ClientNonce is allowed for SecurityPolicy#None, but some
		// embedded servers close the connection when they receive one. A 1-byte
		// nonce matches UaExpert and is ignored by None-policy crypto. See #894.
		nonceLength: 1,
	}, nil
}

func newNoneSymmetric([]byte, []byte) (*EncryptionAlgorithm, error) {
	return &EncryptionAlgorithm{
		blockSize:             NoneBlockSize,
		plainttextBlockSize:   NoneBlockSize - NoneMinPadding,
		encrypt:               &None{},
		decrypt:               &None{},
		signature:             &None{},
		verifySignature:       &None{},
		signatureLength:       0,
		remoteSignatureLength: 0,
	}, nil
}
