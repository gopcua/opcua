// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package securitypolicy

import "crypto/rsa"

var supportedPolicies = map[string]policyInitFuncs{
	"http://opcfoundation.org/UA/SecurityPolicy#None": {
		asymmetricInitFunc: newNoneAsymmetric,
		symmetricInitFunc:  newNoneSymmetric,
	},
	"http://opcfoundation.org/UA/SecurityPolicy#Basic128Rsa15": { // Obsolete in OPC-UA 1.04
		asymmetricInitFunc: newBasic128Rsa15Asymmetric,
		symmetricInitFunc:  newBasic128Rsa15Symmetric,
	},
	"http://opcfoundation.org/UA/SecurityPolicy#Basic256": { // Obsolete in OPC-UA 1.04
		asymmetricInitFunc: newBasic256Asymmetric,
		symmetricInitFunc:  newBasic256Symmetric,
	},
	"http://opcfoundation.org/UA/SecurityPolicy#Basic256Sha256": {
		asymmetricInitFunc: newBasic256Rsa256Asymmetric,
		symmetricInitFunc:  newBasic256Rsa256Symmetric,
	},
	"http://opcfoundation.org/UA/SecurityPolicy#Aes128_Sha256_RsaOaep": {
		asymmetricInitFunc: newAes128Sha256RsaOaepAsymmetric,
		symmetricInitFunc:  newAes128Sha256RsaOaepSymmetric,
	},
	"http://opcfoundation.org/UA/SecurityPolicy#Aes256_Sha256_RsaPss": {
		asymmetricInitFunc: newAes256Sha256RsaPssAsymmetric,
		symmetricInitFunc:  newAes256Sha256RsaPssSymmetric,
	},
	// http://opcfoundation.org/UA/SecurityPolicy#PubSub_Aes128_CTR
	// http://opcfoundation.org/UA/SecurityPolicy#PubSub_Aes256_CTR
}

// SupportedPolicies returns all supported Security Policies
// (and therefore, valid inputs to Asymmetric(...) and Symmetric(...))
func SupportedPolicies() []string {
	p := make([]string, len(supportedPolicies))

	i := 0
	for k := range supportedPolicies {
		p[i] = k
		i++
	}

	return p
}

type policyInitFuncs struct {
	asymmetricInitFunc func(localKey *rsa.PrivateKey, remoteKey *rsa.PublicKey) (*EncryptionAlgorithm, error)
	symmetricInitFunc  func(localNonce []byte, remoteNonce []byte) (*EncryptionAlgorithm, error)
}
