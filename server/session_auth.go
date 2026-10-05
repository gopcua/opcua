// Copyright 2018-2026 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package server

import (
	"crypto/x509"

	"github.com/gopcua/opcua/ua"
	"github.com/gopcua/opcua/uasc"
)

// authenticate validates the user identity token of an ActivateSessionRequest
// against the enabled authentication modes.
//
// https://reference.opcfoundation.org/Core/Part4/v105/docs/5.6.3
// https://reference.opcfoundation.org/Core/Part4/v105/docs/7.41
func (s *SessionService) authenticate(sc *uasc.SecureChannel, sess *session, req *ua.ActivateSessionRequest) (*Identity, ua.StatusCode) {
	var tok any
	if req.UserIdentityToken != nil {
		tok = req.UserIdentityToken.Value
	}

	switch t := tok.(type) {
	case nil:
		// A null token is treated as an anonymous token (Part 4, 5.6.3).
		return s.authAnonymous("")
	case *ua.AnonymousIdentityToken:
		return s.authAnonymous(t.PolicyID)
	case *ua.UserNameIdentityToken:
		return s.authUserName(sc, sess, t)
	case *ua.X509IdentityToken:
		return s.authX509(sc, sess, t, req.UserTokenSignature)
	case *ua.IssuedIdentityToken:
		s.warn("activate session: issued identity tokens are not supported")
		return nil, ua.StatusBadIdentityTokenRejected
	default:
		s.warn("activate session: unknown identity token %T", tok)
		return nil, ua.StatusBadIdentityTokenInvalid
	}
}

func (s *SessionService) authEnabled(tt ua.UserTokenType) bool {
	for _, a := range s.srv.cfg.enabledAuth {
		if a.tokenType == tt {
			return true
		}
	}
	return false
}

// userTokenPolicy returns the advertised policy with the given ID and type.
func (s *SessionService) userTokenPolicy(policyID string, tt ua.UserTokenType) *ua.UserTokenPolicy {
	for _, ep := range s.srv.Endpoints() {
		for _, p := range ep.UserIdentityTokens {
			if p.PolicyID == policyID && p.TokenType == tt {
				return p
			}
		}
	}
	return nil
}

func (s *SessionService) authAnonymous(policyID string) (*Identity, ua.StatusCode) {
	// A server without any configured auth mode keeps the historical
	// behavior of accepting anonymous sessions.
	if len(s.srv.cfg.enabledAuth) > 0 && !s.authEnabled(ua.UserTokenTypeAnonymous) {
		s.warn("activate session: anonymous authentication is not enabled")
		return nil, ua.StatusBadIdentityTokenRejected
	}
	return &Identity{TokenType: ua.UserTokenTypeAnonymous}, ua.StatusOK
}

func (s *SessionService) authUserName(sc *uasc.SecureChannel, sess *session, t *ua.UserNameIdentityToken) (*Identity, ua.StatusCode) {
	if !s.authEnabled(ua.UserTokenTypeUserName) {
		s.warn("activate session: username authentication is not enabled")
		return nil, ua.StatusBadIdentityTokenRejected
	}
	p := s.userTokenPolicy(t.PolicyID, ua.UserTokenTypeUserName)
	if p == nil {
		s.warn("activate session: unknown username token policy %q", t.PolicyID)
		return nil, ua.StatusBadIdentityTokenInvalid
	}
	if s.srv.cfg.userNameAuth == nil {
		s.warn("activate session: username authentication enabled without an authenticator, rejecting user %q", t.UserName)
		return nil, ua.StatusBadIdentityTokenRejected
	}

	pass, err := sc.DecryptUserPassword(p.SecurityPolicyURI, t.EncryptionAlgorithm, t.Password, sess.serverNonce)
	if err != nil {
		s.warn("activate session: invalid password secret for user %q: %s", t.UserName, err)
		return nil, ua.StatusBadIdentityTokenInvalid
	}
	if err := s.srv.cfg.userNameAuth(t.UserName, pass); err != nil {
		s.warn("activate session: access denied for user %q: %s", t.UserName, err)
		return nil, ua.StatusBadUserAccessDenied
	}
	return &Identity{TokenType: ua.UserTokenTypeUserName, UserName: t.UserName}, ua.StatusOK
}

func (s *SessionService) authX509(sc *uasc.SecureChannel, sess *session, t *ua.X509IdentityToken, sig *ua.SignatureData) (*Identity, ua.StatusCode) {
	if !s.authEnabled(ua.UserTokenTypeCertificate) {
		s.warn("activate session: certificate authentication is not enabled")
		return nil, ua.StatusBadIdentityTokenRejected
	}
	p := s.userTokenPolicy(t.PolicyID, ua.UserTokenTypeCertificate)
	if p == nil {
		s.warn("activate session: unknown certificate token policy %q", t.PolicyID)
		return nil, ua.StatusBadIdentityTokenInvalid
	}
	if s.srv.cfg.x509Auth == nil {
		s.warn("activate session: certificate authentication enabled without an authenticator, rejecting")
		return nil, ua.StatusBadIdentityTokenRejected
	}

	cert, err := x509.ParseCertificate(t.CertificateData)
	if err != nil {
		s.warn("activate session: invalid user certificate: %s", err)
		return nil, ua.StatusBadIdentityTokenInvalid
	}
	if err := sc.VerifyUserTokenSignature(p.SecurityPolicyURI, t.CertificateData, s.srv.cfg.certificate, sess.serverNonce, sig); err != nil {
		s.warn("activate session: invalid user token signature for %q: %s", cert.Subject, err)
		return nil, ua.StatusBadUserSignatureInvalid
	}
	if err := s.srv.cfg.x509Auth(cert); err != nil {
		s.warn("activate session: access denied for certificate %q: %s", cert.Subject, err)
		return nil, ua.StatusBadUserAccessDenied
	}
	return &Identity{TokenType: ua.UserTokenTypeCertificate, Certificate: cert}, ua.StatusOK
}

func (s *SessionService) warn(msg string, args ...any) {
	if s.srv.cfg.logger != nil {
		s.srv.cfg.logger.Warn(msg, args...)
	}
}
