// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package opcua

import (
	"crypto/rsa"
	"crypto/x509"
	"math/rand"
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
		LocaleIDs:          []string{"en-us"},
		UserIdentityToken:  &ua.AnonymousIdentityToken{PolicyID: "open62541-anonymous-policy"},
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
func SecurityFromEndpoint(ep *ua.EndpointDescription) Option {
	return func(c *uasc.Config, sc *uasc.SessionConfig) {
		c.SecurityPolicyURI = ep.SecurityPolicyURI
		c.SecurityMode = ep.SecurityMode
		c.RemoteCertificate = ep.ServerCertificate
		c.Thumbprint = securitypolicy.Thumbprint(ep.ServerCertificate)

		// Find the PolicyID from the endpoint
		var userToken *ua.AnonymousIdentityToken
		for _, t := range ep.UserIdentityTokens {
			// todo(dh): Allow more than anonymous authentication
			if t.TokenType == ua.UserTokenTypeAnonymous {
				userToken = &ua.AnonymousIdentityToken{PolicyID: t.PolicyID}
				break
			}
		}

		if userToken == nil {
			userToken = &ua.AnonymousIdentityToken{PolicyID: "Anonymous"}
		}

		sc.UserIdentityToken = userToken
	}
}
