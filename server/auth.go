// Copyright 2018-2026 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package server

import (
	"crypto/subtle"
	"crypto/x509"
	"errors"
	"log"
	"time"

	"github.com/gopcua/opcua/ua"
)

// UserNameAuthenticator validates the credentials of a UserNameIdentityToken
// after the server has decrypted the password. Return nil to accept the user.
type UserNameAuthenticator func(username, password string) error

// X509Authenticator validates the certificate of an X509IdentityToken. It is
// only called after the client has proven possession of the certificate's
// private key via the UserTokenSignature. Return nil to accept the user.
type X509Authenticator func(cert *x509.Certificate) error

// Identity is the user identity a session was activated with.
type Identity struct {
	// TokenType is the type of user identity token used to activate the session.
	TokenType ua.UserTokenType
	// UserName is set for UserTokenTypeUserName.
	UserName string
	// Certificate is set for UserTokenTypeCertificate.
	Certificate *x509.Certificate
}

// Name returns a human readable name for the identity.
func (id *Identity) Name() string {
	if id == nil {
		return ""
	}
	switch id.TokenType {
	case ua.UserTokenTypeUserName:
		return id.UserName
	case ua.UserTokenTypeCertificate:
		if id.Certificate != nil {
			return id.Certificate.Subject.String()
		}
	case ua.UserTokenTypeAnonymous:
		return "anonymous"
	}
	return ""
}

// UserNameAuth enables the UserName authentication mode and validates
// credentials with fn. Like all non-anonymous modes it is only offered on
// endpoints with a security policy other than None, so the password is always
// transmitted encrypted.
func UserNameAuth(fn UserNameAuthenticator) Option {
	return func(s *serverConfig) {
		s.userNameAuth = fn
		EnableAuthMode(ua.UserTokenTypeUserName)(s)
	}
}

// X509Auth enables the Certificate authentication mode and validates user
// certificates with fn.
func X509Auth(fn X509Authenticator) Option {
	return func(s *serverConfig) {
		s.x509Auth = fn
		EnableAuthMode(ua.UserTokenTypeCertificate)(s)
	}
}

// ErrAccessDenied is returned by the built-in authenticators when the
// credentials are not accepted.
var ErrAccessDenied = errors.New("access denied")

// warnAuthConfig logs auth modes which are enabled but cannot succeed.
func (s *Server) warnAuthConfig() {
	for _, a := range s.cfg.enabledAuth {
		switch {
		case a.tokenType == ua.UserTokenTypeUserName && s.cfg.userNameAuth == nil:
			log.Printf("server: UserName auth mode enabled without UserNameAuth(); all username logins will be rejected")
		case a.tokenType == ua.UserTokenTypeCertificate && s.cfg.x509Auth == nil:
			log.Printf("server: Certificate auth mode enabled without X509Auth(); all certificate logins will be rejected")
		}
	}
}

// StaticUsers returns a UserNameAuthenticator that accepts the given
// username/password pairs. Passwords are compared in constant time.
func StaticUsers(users map[string]string) UserNameAuthenticator {
	m := make(map[string]string, len(users))
	for k, v := range users {
		m[k] = v
	}
	return func(username, password string) error {
		want, ok := m[username]
		if !ok {
			// compare anyway so unknown users take the same time
			want = "\x00invalid"
		}
		if subtle.ConstantTimeCompare([]byte(want), []byte(password)) != 1 || !ok {
			return ErrAccessDenied
		}
		return nil
	}
}

// TrustedUserCerts returns an X509Authenticator that accepts user
// certificates which chain up to a certificate in roots and are currently
// valid. A self-signed user certificate is trusted by adding it to roots.
func TrustedUserCerts(roots *x509.CertPool) X509Authenticator {
	return func(cert *x509.Certificate) error {
		if cert == nil {
			return ErrAccessDenied
		}
		now := time.Now()
		if now.Before(cert.NotBefore) || now.After(cert.NotAfter) {
			return ErrAccessDenied
		}
		_, err := cert.Verify(x509.VerifyOptions{
			Roots:       roots,
			CurrentTime: now,
			KeyUsages:   []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
		})
		if err != nil {
			return ErrAccessDenied
		}
		return nil
	}
}
