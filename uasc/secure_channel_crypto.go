// Copyright 2018-2020 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uasc

import (
	"crypto/rsa"
	"crypto/subtle"
	"encoding/binary"

	"github.com/gopcua/opcua/errors"
	"github.com/gopcua/opcua/ua"
	"github.com/gopcua/opcua/uapolicy"
)

// NewSessionSignature issues a new signature for the client to send on the next ActivateSessionRequest
func (s *SecureChannel) NewSessionSignature(cert, nonce []byte) ([]byte, string, error) {
	if s.cfg.SecurityMode == ua.MessageSecurityModeNone {
		return nil, "", nil
	}

	remoteX509Cert, err := uapolicy.ParseCertificate(cert)
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

	remoteX509Cert, err := uapolicy.ParseCertificate(cert)
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

	remoteX509Cert, err := uapolicy.ParseCertificate(cert)
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

// DecryptUserPassword is the server-side counterpart of EncryptUserPassword.
// It decrypts the legacy encrypted secret of a UserNameIdentityToken with the
// channel's local private key and verifies that it ends with nonce (the last
// server nonce issued to the session).
//
// The policyURI is the SecurityPolicyURI of the UserTokenPolicy; if empty the
// channel's policy is used. encAlg is the token's EncryptionAlgorithm and must
// match the policy. Unencrypted passwords (policy None) are rejected.
//
// https://reference.opcfoundation.org/Core/Part4/v105/docs/7.41.2.3
func (s *SecureChannel) DecryptUserPassword(policyURI, encAlg string, secret, nonce []byte) (string, error) {
	if policyURI == "" {
		policyURI = s.cfg.SecurityPolicyURI
	}
	if policyURI == ua.SecurityPolicyURINone {
		return "", errors.New("unencrypted user password not allowed")
	}
	if s.cfg.LocalKey == nil {
		return "", errors.New("no local private key")
	}

	enc, err := uapolicy.Asymmetric(policyURI, s.cfg.LocalKey, nil)
	if err != nil {
		return "", err
	}
	if encAlg != enc.EncryptionURI() {
		return "", errors.Errorf("unexpected encryption algorithm %q, want %q", encAlg, enc.EncryptionURI())
	}

	plain, err := enc.Decrypt(secret)
	if err != nil {
		return "", errors.Errorf("decrypt user password: %s", err)
	}
	if len(plain) < 4 {
		return "", errors.New("user password secret too short")
	}
	l := binary.LittleEndian.Uint32(plain[:4])
	body := plain[4:]
	if uint64(l) != uint64(len(body)) || len(body) < len(nonce) {
		return "", errors.New("invalid user password secret length")
	}
	pass, gotNonce := body[:len(body)-len(nonce)], body[len(body)-len(nonce):]
	if len(nonce) == 0 || subtle.ConstantTimeCompare(gotNonce, nonce) != 1 {
		return "", errors.New("user password nonce mismatch")
	}
	return string(pass), nil
}

// VerifyUserTokenSignature is the server-side counterpart of
// NewUserTokenSignature. It checks that sig is a signature over
// (serverCert || nonce) made with the private key of userCert (DER).
//
// The policyURI is the SecurityPolicyURI of the UserTokenPolicy; if empty the
// channel's policy is used.
//
// https://reference.opcfoundation.org/Core/Part4/v105/docs/5.6.3
func (s *SecureChannel) VerifyUserTokenSignature(policyURI string, userCert, serverCert, nonce []byte, sig *ua.SignatureData) error {
	if policyURI == "" {
		policyURI = s.cfg.SecurityPolicyURI
	}
	if policyURI == ua.SecurityPolicyURINone {
		return errors.New("user token signature requires a security policy")
	}
	if sig == nil || len(sig.Signature) == 0 {
		return errors.New("missing user token signature")
	}

	cert, err := uapolicy.ParseCertificate(userCert)
	if err != nil {
		return err
	}
	userKey, ok := cert.PublicKey.(*rsa.PublicKey)
	if !ok {
		return errors.New("user certificate does not contain an RSA public key")
	}

	enc, err := uapolicy.Asymmetric(policyURI, nil, userKey)
	if err != nil {
		return err
	}
	if sig.Algorithm != enc.SignatureURI() {
		return errors.Errorf("unexpected signature algorithm %q, want %q", sig.Algorithm, enc.SignatureURI())
	}

	msg := make([]byte, 0, len(serverCert)+len(nonce))
	msg = append(msg, serverCert...)
	msg = append(msg, nonce...)
	return enc.VerifySignature(msg, sig.Signature)
}

// NewUserTokenSignature issues a new signature for the client to send in ActivateSessionRequest
// The security policy for the SecureChannel is used if policyURI value is null or empty
// https://reference.opcfoundation.org/Core/Part4/v104/docs/7.37
func (s *SecureChannel) NewUserTokenSignature(policyURI string, cert, nonce []byte) ([]byte, string, error) {
	if policyURI == "" {
		policyURI = s.cfg.SecurityPolicyURI
	}

	if policyURI == ua.SecurityPolicyURINone {
		return nil, "", nil
	}

	remoteX509Cert, err := uapolicy.ParseCertificate(cert)
	if err != nil {
		return nil, "", err
	}
	remoteKey := remoteX509Cert.PublicKey.(*rsa.PublicKey)

	enc, err := uapolicy.Asymmetric(policyURI, s.cfg.UserKey, remoteKey)
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
