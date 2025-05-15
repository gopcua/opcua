// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package server

import (
	"crypto/rsa"
	"crypto/x509"
	"fmt"
	"log"
	"log/slog"
	"strings"
	"time"

	"github.com/gopcua/opcua/ua"
	"github.com/gopcua/opcua/uapolicy"
	"github.com/gopcua/opcua/uasc"
)

// Option is an option function type to modify the configuration.
type Option func(*serverConfig)

// PrivateKey sets the RSA private key in the secure channel configuration.
func PrivateKey(key *rsa.PrivateKey) Option {
	return func(cfg *serverConfig) {
		cfg.privateKey = key
	}
}

// EndPointHostName adds an additional endpoint to the server based on the host name
func EndPoint(host string, port int) Option {
	return func(cfg *serverConfig) {
		if cfg.endpoints == nil {
			cfg.endpoints = make([]string, 0)
		}
		ep := fmt.Sprintf("opc.tcp://%s:%d", host, port)
		cfg.endpoints = append(cfg.endpoints, ep)
	}
}

// Certificate sets the client X509 certificate in the secure channel configuration
// and also detects and sets the ApplicationURI from the URI within the certificate
func Certificate(cert []byte) Option {
	return func(cfg *serverConfig) {
		cfg.certificate = cert

		// Extract the application URI from the certificate.
		var appURI string
		x509cert, err := x509.ParseCertificate(cert)
		if err == nil && len(x509cert.URIs) > 0 {
			appURI = x509cert.URIs[0].String()
		}

		cfg.applicationURI = appURI
	}
}

// EnableSecurity registers a new endpoint security mode to the server.
// This will also register the security policy against each enabled auth mode
func EnableSecurity(secPolicy string, secMode ua.MessageSecurityMode) Option {
	return func(cfg *serverConfig) {
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
			log.Printf("error adding endpoint to config, %s is unsupported", secPolicy)
			return
		}

		for _, sec := range cfg.enabledSec {
			if sec.secPolicy == secPolicy && sec.secMode == secMode {
				// todo(fs): logging here feels wrong. Ideas?
				slog.Warn("security policy already exists, skipping")
				return
			}
		}

		sec := security{
			secPolicy: secPolicy,
			secMode:   secMode,
		}

		cfg.enabledSec = append(cfg.enabledSec, sec)
	}
}

// EnableAuthMode registers a new user authentication mode to the server.
// All AuthModes except Anonymous require encryption by default, so EnableSecurity()
// must also be called with at least one non-"None" SecurityPolicy
func EnableAuthMode(tokenType ua.UserTokenType) Option {
	return func(cfg *serverConfig) {

		for _, a := range cfg.enabledAuth {
			if a.tokenType == tokenType {
				// todo(fs): logging here feels wrong. Ideas?
				slog.Warn("auth mode already registered, skipping")
				return
			}
		}

		a := authMode{
			tokenType: tokenType,
		}

		cfg.enabledAuth = append(cfg.enabledAuth, a)
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
	return func(cfg *serverConfig) {
		cfg.applicationName = name
	}
}

func ManufacturerName(name string) Option {
	return func(cfg *serverConfig) {
		cfg.manufacturerName = name
	}
}

func ProductName(name string) Option {
	return func(cfg *serverConfig) {
		cfg.productName = name
	}
}

func SoftwareVersion(name string) Option {
	return func(cfg *serverConfig) {
		cfg.softwareVersion = name
	}
}

// SetLoggerHandler sets the slog.Handler for the server.
func SetLoggerHandler(h slog.Handler) Option {
	return func(cfg *serverConfig) {
		cfg.loghandler = h
	}
}
