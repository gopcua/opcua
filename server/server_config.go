// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package server

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"fmt"
	"strings"
	"time"

	"github.com/gopcua/opcua/ua"
	"github.com/gopcua/opcua/ualog"
	"github.com/gopcua/opcua/uapolicy"
	"github.com/gopcua/opcua/uasc"
)

// Option is an option function type to modify the configuration.
type Option func(context.Context, *serverConfig)

// PrivateKey sets the RSA private key in the secure channel configuration.
func PrivateKey(key *rsa.PrivateKey) Option {
	return func(_ context.Context, s *serverConfig) {
		s.privateKey = key
	}
}

// EndPointHostName adds an additional endpoint to the server based on the host name
func EndPoint(host string, port int) Option {
	return func(_ context.Context, s *serverConfig) {
		if s.endpoints == nil {
			s.endpoints = make([]string, 0)
		}
		ep := fmt.Sprintf("opc.tcp://%s:%d", host, port)
		s.endpoints = append(s.endpoints, ep)
	}
}

// Certificate sets the client X509 certificate in the secure channel configuration
// and also detects and sets the ApplicationURI from the URI within the certificate
func Certificate(cert []byte) Option {
	return func(_ context.Context, s *serverConfig) {
		s.certificate = cert

		// Extract the application URI from the certificate.
		var appURI string
		x509cert, err := x509.ParseCertificate(cert)
		if err == nil && len(x509cert.URIs) > 0 {
			appURI = x509cert.URIs[0].String()
		}

		s.applicationURI = appURI
	}
}

// EnableSecurity registers a new endpoint security mode to the server.
// This will also register the security policy against each enabled auth mode
func EnableSecurity(secPolicy string, secMode ua.MessageSecurityMode) Option {
	return func(ctx context.Context, s *serverConfig) {
		if !strings.HasPrefix(secPolicy, "http://opcfoundation.org/UA/SecurityPolicy#") {
			secPolicy = "http://opcfoundation.org/UA/SecurityPolicy#" + secPolicy
		}

		var ok bool
		ss := uapolicy.SupportedPolicies()
		for _, sp := range ss {
			if sp == secPolicy {
				ok = true
				break
			}
		}
		if !ok {
			ualog.Error(ctx, "unable to add endpoint security mode to config",
				ualog.String(ualog.ErrorKey, "unsupported policy"),
				ualog.String("policy", secPolicy),
			)
			return
		}

		for _, sec := range s.enabledSec {
			if sec.secPolicy == secPolicy && sec.secMode == secMode {
				ualog.Warn(ctx, "security policy already exists, skipping")
				return
			}
		}

		sec := security{
			secPolicy: secPolicy,
			secMode:   secMode,
		}

		s.enabledSec = append(s.enabledSec, sec)
	}
}

// EnableAuthMode registers a new user authentication mode to the server.
// All AuthModes except Anonymous require encryption by default, so EnableSecurity()
// must also be called with at least one non-"None" SecurityPolicy
func EnableAuthMode(tokenType ua.UserTokenType) Option {
	return func(ctx context.Context, s *serverConfig) {

		for _, a := range s.enabledAuth {
			if a.tokenType == tokenType {
				ualog.Warn(ctx, "auth mode already registered, skipping",
					ualog.String("mode", tokenType.String()),
				)
				return
			}
		}

		a := authMode{
			tokenType: tokenType,
		}

		s.enabledAuth = append(s.enabledAuth, a)
	}
}

func defaultChannelConfig() *uasc.Config {
	return &uasc.Config{
		SecurityPolicyURI: ua.SecurityPolicyURINone,
		SecurityMode:      ua.MessageSecurityModeNone,
		Lifetime:          uint32(time.Hour / time.Millisecond),
	}
}

func ServerName(name string) Option {
	return func(_ context.Context, s *serverConfig) {
		s.applicationName = name
	}
}

func ManufacturerName(name string) Option {
	return func(_ context.Context, s *serverConfig) {
		s.manufacturerName = name
	}
}

func ProductName(name string) Option {
	return func(_ context.Context, s *serverConfig) {
		s.productName = name
	}
}

func SoftwareVersion(name string) Option {
	return func(_ context.Context, s *serverConfig) {
		s.softwareVersion = name
	}
}
