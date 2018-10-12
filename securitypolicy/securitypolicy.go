// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

// Package securitypolicy implements the encryption, decryption, signing,
// and signature verifying algorithms for Security Policy profiles as
// defined in Part 7 of the OPC-UA specifications (version 1.04)
package securitypolicy

import (
	"crypto/rsa"
	"errors"
)

// EncryptionAlgorithm wraps the functions used to return the various
// methods required to implement the symmetric and asymmetric algorithms
// Function variables were used instead of an interface to make better use
// of policies which implement the same algorithms in different combinations
//
// EncryptionAlgorithm should always be instantiated through calls to
// SecurityPolicy.Symmetric() and SecurityPolicy.Asymmetric() to ensure
// correct behavior.
// The zero value of this struct will use SecurityPolicy#None although
// using in this manner is discouraged for readability
type EncryptionAlgorithm struct {
	blockSize       func() int
	minPadding      func() int
	encrypt         func(cleartext []byte) (ciphertext []byte, err error)
	decrypt         func(ciphertext []byte) (cleartext []byte, err error)
	signature       func(message []byte) (signature []byte, err error)
	verifySignature func(message, signature []byte) error
	encryptionURI   string
	signatureURI    string
}

// Asymmetric returns the EncryptionAlgorithm struct seeded with the required public
// and private RSA keys to fully implement.
// For Security Policy "None", both keys are ignored and may be nil
func Asymmetric(policyURI string, localKey *rsa.PrivateKey, remoteKey *rsa.PublicKey) (*EncryptionAlgorithm, error) {
	policy, ok := supportedPolicies[policyURI]

	if !ok {
		return nil, errors.New("unknown security policy")
	}

	if policy.asymmetricInitFunc == nil {
		return newNoneAsymmetric(localKey, remoteKey)
	}

	return policy.asymmetricInitFunc(localKey, remoteKey)
}

// Symmetric returns the EncryptionAlgorithm struct seeded with the client and server nonces
// negotiated from the OpenSecureChannel service (encrypted by the Asymmetric algorithms)
// For Security Policy "None", both nonces are ignored and may be nil
func Symmetric(policyURI string, localNonce []byte, remoteNonce []byte) (*EncryptionAlgorithm, error) {
	policy, ok := supportedPolicies[policyURI]

	if !ok {
		return nil, errors.New("unknown security policy")
	}

	if policy.symmetricInitFunc == nil {
		return newNoneSymmetric(localNonce, remoteNonce)
	}

	return policy.symmetricInitFunc(localNonce, remoteNonce)
}

// BlockSize returns the underlying encryption algorithm's blocksize.
// Used to calculate the padding required to make the cleartext an
// even multiple of the blocksize
func (e *EncryptionAlgorithm) BlockSize() int {
	if e.blockSize == nil {
		e.blockSize = blockSizeNone
	}

	return e.blockSize()
}

// MinPadding returns the underlying encryption algorithm's minimum padding.
// Used to calculate the maximum plaintext blocksize that can be fed into
// the encryption algorithm.
func (e *EncryptionAlgorithm) MinPadding() int {
	if e.minPadding == nil {
		e.minPadding = minPaddingNone
	}

	return e.minPadding()
}

// Encrypt encrypts the input cleartext based on the algorithms and keys passed in
func (e *EncryptionAlgorithm) Encrypt(cleartext []byte) (ciphertext []byte, err error) {
	if e.encrypt == nil {
		e.encrypt = encryptNone
	}

	return e.encrypt(cleartext)
}

// Decrypt decrypts the input ciphertext based on the algorithms and keys passed in
func (e *EncryptionAlgorithm) Decrypt(ciphertext []byte) (cleartext []byte, err error) {
	if e.decrypt == nil {
		e.decrypt = decryptNone
	}

	return e.decrypt(ciphertext)
}

// Signature returns the cryptographic signature of message
func (e *EncryptionAlgorithm) Signature(message []byte) (signature []byte, err error) {
	if e.signature == nil {
		e.signature = signatureNone
	}

	return e.signature(message)
}

// VerifySignature validates that 'signature' is the correct cryptographic signature
// of 'message' or returns an error.
// A return value of nil means the signature is valid
func (e *EncryptionAlgorithm) VerifySignature(message, signature []byte) error {
	if e.verifySignature == nil {
		e.verifySignature = verifySignatureNone
	}

	return e.verifySignature(message, signature)
}

// EncryptionURI returns the URI for the encryption algorithm as defined
// by the OPC-UA profiles in Part 7
func (e *EncryptionAlgorithm) EncryptionURI() string {
	return e.encryptionURI
}

// SignatureURI returns the URI for the signature algorithm as defined
// by the OPC-UA profiles in Part 7
func (e *EncryptionAlgorithm) SignatureURI() string {
	return e.signatureURI
}
