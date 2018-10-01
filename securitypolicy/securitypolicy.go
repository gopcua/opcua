// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package securitypolicy

import (
	"crypto/rsa"
	"errors"
)

/* Security Policy
Name							Description
PolicyUri						The URI assigned to the SecurityPolicy.
SymmetricSignatureAlgorithm		The symmetric signature algorithm to use.
SymmetricEncryptionAlgorithm	The symmetric encryption algorithm to use.
AsymmetricSignatureAlgorithm	The asymmetric signature algorithm to use.
AsymmetricEncryptionAlgorithm	The asymmetric encryption algorithm to use.
MinAsymmetricKeyLength			The minimum length, in bits, for an asymmetric key.
MaxAsymmetricKeyLength			The maximum length, in bits, for an asymmetric key.
KeyDerivationAlgorithm			The key derivation algorithm to use.
DerivedSignatureKeyLength		The length in bits of the derived key used for Message authentication.
CertificateSignatureAlgorithm	The asymmetric signature algorithm used to sign certificates.
SecureChannelNonceLength		The length, in bytes, of the Nonces exchanged when creating a SecureChannel.

*/

// EncryptionAlgorithm wraps the functions used to return the various
// methods required to implement the symmetric and asymmetric algorithms
// Function variables were used instead of an interface to make better use
// of policies which implement the same algorithms in different combinations
//
// EncryptionAlgorithm should always be instantiated through calls to
// SecurityPolicy.Symmetric() and SecurityPolicy.Asymmetric() to ensure
// correct behavior.
// The zero value of this struct will use SecurityPolicy#None
type EncryptionAlgorithm struct {
	blockSize       func() int
	encrypt         func(cleartext []byte) (ciphertext []byte)
	decrypt         func(ciphertext []byte) (cleartext []byte)
	signature       func(message []byte) (signature []byte)
	verifySignature func(message, signature []byte) error
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

// Encrypt encrypts the input cleartext based on the algorithms and keys passed in
func (e *EncryptionAlgorithm) Encrypt(cleartext []byte) (ciphertext []byte) {
	if e.encrypt == nil {
		e.encrypt = encryptNone
	}

	return e.encrypt(cleartext)
}

// Decrypt decrypts the input ciphertext based on the algorithms and keys passed in
func (e *EncryptionAlgorithm) Decrypt(ciphertext []byte) (cleartext []byte) {
	if e.decrypt == nil {
		e.decrypt = decryptNone
	}

	return e.decrypt(ciphertext)
}

// Signature returns the cryptographic signature of message
func (e *EncryptionAlgorithm) Signature(message []byte) (signature []byte) {
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

// SecurityPolicy wraps both the Asymmetric and Symmetric algorithms for a specific
// security policy as defined by the OPC-UA specifications
// These functions need to be instantiated with appropriate security keys which are
// received from various parts of the Secure Channel negotiation.
//
// SecurityPolicy should always be instantiated through a call to New(policyURI)
// to ensure correct behavior.
// The zero value of this struct will use SecurityPolicy#None
type SecurityPolicy struct {
	asymmetric func(localKey *rsa.PrivateKey, remoteKey *rsa.PublicKey) *EncryptionAlgorithm
	symmetric  func(localNonce []byte, remoteNonce []byte) *EncryptionAlgorithm
}

// Asymmetric returns the EncryptionAlgorithm struct seeded with the required public
// and private RSA keys to fully implement.
// For Security Policy "None", both keys are ignored and may be nil
func (s *SecurityPolicy) Asymmetric(localKey *rsa.PrivateKey, remoteKey *rsa.PublicKey) *EncryptionAlgorithm {
	if s.asymmetric == nil {
		s.asymmetric = newNoneAsymmetric
	}

	return s.asymmetric(localKey, remoteKey)
}

// Symmetric returns the EncryptionAlgorithm struct seeded with the client and server nonces
// negotiated from the OpenSecureChannel service (encrypted by the Asymmetric algorithms)
// For Security Policy "None", both nonces are ignored and may be nil
func (s *SecurityPolicy) Symmetric(localNonce []byte, remoteNonce []byte) *EncryptionAlgorithm {
	if s.symmetric == nil {
		s.symmetric = newNoneSymmetric
	}

	return s.symmetric(localNonce, remoteNonce)
}

// New creates a new security policy for encoding/decoding UASC messages
func New(policyURI string) (*SecurityPolicy, error) {
	var p = new(SecurityPolicy)

	policy, ok := supportedPolicies[policyURI]

	if !ok {
		return nil, errors.New("unknown security policy")
	}

	p.asymmetric = policy.asymmetricInitFunc
	p.symmetric = policy.symmetricInitFunc

	return p, nil
}
