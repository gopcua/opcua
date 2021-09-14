// Copyright 2018-2020 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uasc

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/binary"

	"github.com/gopcua/opcua/ua"
	"github.com/gopcua/opcua/uapolicy"
)

// NewSessionSignature issues a new signature for the client to send on the next ActivateSessionRequest
func (s *SecureChannel) NewSessionSignature(cert, nonce []byte) ([]byte, string, error) {
	if s.cfg.SecurityMode == ua.MessageSecurityModeNone {
		return nil, "", nil
	}

	remoteX509Cert, err := x509.ParseCertificate(cert)
	if err != nil {
		return nil, "", err
	}
	remoteKey := remoteX509Cert.PublicKey.(*rsa.PublicKey)

	enc, err := uapolicy.Asymmetric(s.cfg.SecurityPolicyURI, s.cfg.LocalKey, remoteKey)
	if err != nil {
		return nil, "", err
	}

	sig, err := enc.Signature(append(cert, nonce...))
	if err != nil {
		return nil, "", err
	}
	sigAlg := enc.SignatureURI()

	return sig, sigAlg, nil
}

// VerifySessionSignature checks the integrity of a Create/Activate Session response's signature
func (s *SecureChannel) VerifySessionSignature(cert, nonce, signature []byte) error {
	if s.cfg.SecurityMode == ua.MessageSecurityModeNone {
		return nil
	}

	remoteX509Cert, err := x509.ParseCertificate(cert)
	if err != nil {
		return err
	}
	remoteKey := remoteX509Cert.PublicKey.(*rsa.PublicKey)

	enc, err := uapolicy.Asymmetric(s.cfg.SecurityPolicyURI, s.cfg.LocalKey, remoteKey)
	if err != nil {
		return err
	}
	err = enc.VerifySignature(append(s.cfg.Certificate, nonce...), signature)
	if err != nil {
		return err
	}

	return nil
}

// EncryptUserPassword issues a new signature for the client to send in ActivateSessionRequest
func (s *SecureChannel) EncryptUserPassword(policyURI, password string, cert, nonce []byte) ([]byte, string, error) {
	// If the User ID Token's policy was null, then default to the secure channel's policy
	if policyURI == "" {
		policyURI = s.cfg.SecurityPolicyURI
	}

	if policyURI == ua.SecurityPolicyURINone {
		return []byte(password), "", nil
	}

	remoteX509Cert, err := x509.ParseCertificate(cert)
	if err != nil {
		return nil, "", err
	}
	remoteKey := remoteX509Cert.PublicKey.(*rsa.PublicKey)

	enc, err := uapolicy.Asymmetric(policyURI, s.cfg.LocalKey, remoteKey)
	if err != nil {
		return nil, "", err
	}

	l := len(password) + len(nonce)
	secret := make([]byte, 4)
	binary.LittleEndian.PutUint32(secret, uint32(l))
	secret = append(secret, []byte(password)...)
	secret = append(secret, nonce...)
	pass, err := enc.Encrypt(secret)
	if err != nil {
		return nil, "", err
	}
	passAlg := enc.EncryptionURI()

	return pass, passAlg, nil
}

// NewUserTokenSignature issues a new signature for the client to send in ActivateSessionRequest
func (s *SecureChannel) NewUserTokenSignature(policyURI string, cert, nonce []byte) ([]byte, string, error) {
	if policyURI == ua.SecurityPolicyURINone {
		return nil, "", nil
	}

	remoteX509Cert, err := x509.ParseCertificate(cert)
	if err != nil {
		return nil, "", err
	}
	remoteKey := remoteX509Cert.PublicKey.(*rsa.PublicKey)

	enc, err := uapolicy.Asymmetric(policyURI, s.cfg.LocalKey, remoteKey)
	if err != nil {
		return nil, "", err
	}

	sig, err := enc.Signature(append(cert, nonce...))
	if err != nil {
		return nil, "", err
	}
	sigAlg := enc.SignatureURI()

	return sig, sigAlg, nil
}
