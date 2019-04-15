// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package opcua

import (
	"crypto/rsa"
	"crypto/x509"
	"log"
	"math/rand"
	"reflect"
	"time"

	"github.com/gopcua/opcua/securitypolicy"
	"github.com/gopcua/opcua/ua"
	"github.com/gopcua/opcua/uasc"
)

// DefaultClientConfig returns the default configuration for a client
// to establish a secure channel and a session.
func DefaultClientConfig() *uasc.Config {
	return &uasc.Config{
		SecurityPolicyURI: uasc.SecurityPolicyNone,
		SecurityMode:      ua.MessageSecurityModeNone,
		Lifetime:          uint32(time.Hour / time.Millisecond),
	}
}

func DefaultSessionConfig() *uasc.SessionConfig {
	return &uasc.SessionConfig{
		SessionTimeout: 0xffff,
		ClientDescription: &ua.ApplicationDescription{
			ApplicationURI:  "urn:gopcua:client",
			ProductURI:      "urn:gopcua",
			ApplicationName: &ua.LocalizedText{Text: "gopcua - OPC UA implementation in Go"},
			ApplicationType: ua.ApplicationTypeClient,
		},
		LocaleIDs: []string{"en-us"},
		//UserIdentityToken:  &ua.AnonymousIdentityToken{PolicyID: "open62541-anonymous-policy"},
		UserTokenSignature: &ua.SignatureData{},
	}
}

// Option is an option function type to modify the configuration.
type Option func(*uasc.Config, *uasc.SessionConfig)

// ApplicationURI sets the application uri in the session configuration.
func ApplicationURI(s string) Option {
	return func(c *uasc.Config, sc *uasc.SessionConfig) {
		sc.ClientDescription.ApplicationURI = s
	}
}

// ApplicationName sets the application name in the session configuration.
func ApplicationName(s string) Option {
	return func(c *uasc.Config, sc *uasc.SessionConfig) {
		sc.ClientDescription.ApplicationName = &ua.LocalizedText{Text: s}
	}
}

// Lifetime sets the lifetime of the secure channel in milliseconds.
func Lifetime(d time.Duration) Option {
	return func(c *uasc.Config, sc *uasc.SessionConfig) {
		c.Lifetime = uint32(d / time.Millisecond)
	}
}

// Locales sets the locales in the session configuration.
func Locales(locale ...string) Option {
	return func(c *uasc.Config, sc *uasc.SessionConfig) {
		sc.LocaleIDs = locale
	}
}

// RandomRequestID assigns a random initial request id.
func RandomRequestID() Option {
	return func(c *uasc.Config, sc *uasc.SessionConfig) {
		c.RequestID = uint32(rand.Int31())
	}
}

// ProductURI sets the product uri in the session configuration.
func ProductURI(s string) Option {
	return func(c *uasc.Config, sc *uasc.SessionConfig) {
		sc.ClientDescription.ProductURI = s
	}
}

// SecurityMode sets the security mode for the secure channel.
func SecurityMode(m ua.MessageSecurityMode) Option {
	return func(c *uasc.Config, sc *uasc.SessionConfig) {
		c.SecurityMode = m
	}
}

// SecurityPolicy sets the security policy uri for the secure channel.
func SecurityPolicy(s string) Option {
	return func(c *uasc.Config, sc *uasc.SessionConfig) {
		c.SecurityPolicyURI = s
	}
}

// SessionTimeout sets the timeout in the session configuration.
func SessionTimeout(seconds float64) Option {
	return func(c *uasc.Config, sc *uasc.SessionConfig) {
		sc.SessionTimeout = seconds
	}
}

// PrivateKey sets the RSA private key in the secure channel configuration.
func PrivateKey(key *rsa.PrivateKey) Option {
	return func(c *uasc.Config, sc *uasc.SessionConfig) {
		c.LocalKey = key
	}
}

// Certificate sets the client X509 certificate in the secure channel configuration
// and also detects and sets the ApplicationURI from the URI within the certificate
func Certificate(cert []byte) Option {
	return func(c *uasc.Config, sc *uasc.SessionConfig) {
		c.Certificate = cert

		// Extract the application URI from the certificate.
		var appURI string
		x509cert, err := x509.ParseCertificate(cert)
		if err == nil && len(x509cert.URIs) > 0 {
			appURI = x509cert.URIs[0].String()
		}

		sc.ClientDescription.ApplicationURI = appURI
	}
}

// SecurityFromEndpoint sets the server-related security parameters from
// a chosen endpoint (received from GetEndpoints())
func SecurityFromEndpoint(ep *ua.EndpointDescription, authType ua.UserTokenType) Option {
	return func(c *uasc.Config, sc *uasc.SessionConfig) {
		c.SecurityPolicyURI = ep.SecurityPolicyURI
		c.SecurityMode = ep.SecurityMode
		c.RemoteCertificate = ep.ServerCertificate
		c.Thumbprint = securitypolicy.Thumbprint(ep.ServerCertificate)

		for _, t := range ep.UserIdentityTokens {

			if t.TokenType == authType {
				if sc.UserIdentityToken == nil {

					switch authType {
					case ua.UserTokenTypeAnonymous:
						sc.UserIdentityToken = &ua.AnonymousIdentityToken{}
					case ua.UserTokenTypeUserName:
						sc.UserIdentityToken = &ua.UserNameIdentityToken{}
					case ua.UserTokenTypeCertificate:
						sc.UserIdentityToken = &ua.X509IdentityToken{}
					case ua.UserTokenTypeIssuedToken:
						sc.UserIdentityToken = &ua.IssuedIdentityToken{}
					}
				}
				// todo: this feels wrong; should this be an interface with a .SetPolicyID() method?
				reflect.ValueOf(sc.UserIdentityToken).Elem().FieldByName("PolicyID").SetString(t.PolicyID)
				sc.AuthPolicyURI = t.SecurityPolicyURI

				break
			}
		}

		if sc.UserIdentityToken == nil {
			sc.UserIdentityToken = &ua.AnonymousIdentityToken{PolicyID: "Anonymous"}
			sc.AuthPolicyURI = uasc.SecurityPolicyNone
		}

	}
}

// AuthPolicyID sets the policy ID of the user identity token
// Note: This should only be called if you know the exact policy ID the server is expecting.
// Most callers should use SecurityFromEndpoint as it automatically finds the policyID
func AuthPolicyID(policy string) Option {
	return func(c *uasc.Config, sc *uasc.SessionConfig) {
		if sc.UserIdentityToken == nil {
			log.Printf("policy ID needs to be set after the policy type is chosen, no changes made.  Call SecurityFromEndpoint() or an AuthXXX() option first")
			return
		}

		reflect.ValueOf(sc.UserIdentityToken).Elem().FieldByName("PolicyID").SetString(policy)
	}
}

// AuthAnonymous sets the client's authentication X509 certificate
// Note: PolicyID still needs to be set outside of this method, typically through
// the SecurityFromEndpoint() Option
func AuthAnonymous() Option {
	return func(c *uasc.Config, sc *uasc.SessionConfig) {
		if sc.UserIdentityToken == nil {
			sc.UserIdentityToken = &ua.AnonymousIdentityToken{}
		}

		_, ok := sc.UserIdentityToken.(*ua.AnonymousIdentityToken)
		if !ok {
			log.Printf("non-anonymous authentication already configured, ignoring")
			return
		}

	}
}

// AuthUsername sets the client's authentication username and password
// Note: PolicyID still needs to be set outside of this method, typically through
// the SecurityFromEndpoint() Option
func AuthUsername(user, pass string) Option {
	return func(c *uasc.Config, sc *uasc.SessionConfig) {
		if sc.UserIdentityToken == nil {
			sc.UserIdentityToken = &ua.UserNameIdentityToken{}
		}

		t, ok := sc.UserIdentityToken.(*ua.UserNameIdentityToken)
		if !ok {
			log.Printf("non-username authentication already configured, ignoring")
			return
		}

		t.UserName = user
		sc.AuthPassword = pass
	}
}

// AuthCertificate sets the client's authentication X509 certificate
// Note: PolicyID still needs to be set outside of this method, typically through
// the SecurityFromEndpoint() Option
func AuthCertificate(cert []byte) Option {
	return func(c *uasc.Config, sc *uasc.SessionConfig) {
		if sc.UserIdentityToken == nil {
			sc.UserIdentityToken = &ua.X509IdentityToken{}
		}

		t, ok := sc.UserIdentityToken.(*ua.X509IdentityToken)
		if !ok {
			log.Printf("non-certificate authentication already configured, ignoring")
			return
		}

		t.CertificateData = cert
	}
}

// AuthIssuedToken sets the client's authentication data based on an externally-issued token
// Note: PolicyID still needs to be set outside of this method, typically through
// the SecurityFromEndpoint() Option
func AuthIssuedToken(tokenData []byte) Option {
	return func(c *uasc.Config, sc *uasc.SessionConfig) {
		if sc.UserIdentityToken == nil {
			sc.UserIdentityToken = &ua.IssuedIdentityToken{}
		}

		t, ok := sc.UserIdentityToken.(*ua.IssuedIdentityToken)
		if !ok {
			log.Printf("non-issued token authentication already configured, ignoring")
			return
		}

		// todo : not correct; need to read spec
		t.TokenData = tokenData
	}
}
