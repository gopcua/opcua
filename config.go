// Copyright 2018-2020 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package opcua

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"strings"
	"time"

	"github.com/gopcua/opcua/errors"
	"github.com/gopcua/opcua/ua"
	"github.com/gopcua/opcua/uacp"
	"github.com/gopcua/opcua/uapolicy"
	"github.com/gopcua/opcua/uasc"
)

// DefaultClientConfig returns the default configuration for a client
// to establish a secure channel.
func DefaultClientConfig() *uasc.Config {
	return &uasc.Config{
		SecurityPolicyURI: ua.SecurityPolicyURINone,
		SecurityMode:      ua.MessageSecurityModeNone,
		Lifetime:          uint32(time.Hour / time.Millisecond),
		RequestTimeout:    10 * time.Second,
		AutoReconnect:     true,
		ReconnectInterval: 5 * time.Second,
	}
}

// DefaultSessionConfig returns the default configuration for a client
// to establish a session.
func DefaultSessionConfig() *uasc.SessionConfig {
	return &uasc.SessionConfig{
		SessionTimeout: 20 * time.Minute,
		ClientDescription: &ua.ApplicationDescription{
			ApplicationURI:  "urn:gopcua:client",
			ProductURI:      "urn:gopcua",
			ApplicationName: ua.NewLocalizedText("gopcua - OPC UA implementation in Go"),
			ApplicationType: ua.ApplicationTypeClient,
		},
		LocaleIDs:          []string{"en-us"},
		UserTokenSignature: &ua.SignatureData{},
	}
}

// Config contains all config options.
type Config struct {
	dialer  *uacp.Dialer
	sechan  *uasc.Config
	session *uasc.SessionConfig
}

// NewDialer creates a uacp.Dialer from the config options
func NewDialer(cfg *Config) *uacp.Dialer {
	if cfg.dialer == nil {
		return &uacp.Dialer{}
	}
	return cfg.dialer
}

// ApplyConfig applies the config options to the default configuration.
// todo(fs): Can we find a better name?
func ApplyConfig(opts ...Option) *Config {
	cfg := &Config{
		sechan:  DefaultClientConfig(),
		session: DefaultSessionConfig(),
	}
	for _, opt := range opts {
		opt(cfg)
	}
	return cfg
}

// Option is an option function type to modify the configuration.
type Option func(*Config)

// ApplicationName sets the application name in the session configuration.
func ApplicationName(s string) Option {
	return func(cfg *Config) {
		cfg.session.ClientDescription.ApplicationName = ua.NewLocalizedText(s)
	}
}

// ApplicationURI sets the application uri in the session configuration.
func ApplicationURI(s string) Option {
	return func(cfg *Config) {
		cfg.session.ClientDescription.ApplicationURI = s
	}
}

// AutoReconnect sets the auto reconnect state of the secure channel.
func AutoReconnect(b bool) Option {
	return func(cfg *Config) {
		cfg.sechan.AutoReconnect = b
	}
}

// ReconnectInterval is interval duration between each reconnection attempt.
func ReconnectInterval(d time.Duration) Option {
	return func(cfg *Config) {
		cfg.sechan.ReconnectInterval = d
	}
}

// Lifetime sets the lifetime of the secure channel in milliseconds.
func Lifetime(d time.Duration) Option {
	return func(cfg *Config) {
		cfg.sechan.Lifetime = uint32(d / time.Millisecond)
	}
}

// Locales sets the locales in the session configuration.
func Locales(locale ...string) Option {
	return func(cfg *Config) {
		cfg.session.LocaleIDs = locale
	}
}

// ProductURI sets the product uri in the session configuration.
func ProductURI(s string) Option {
	return func(cfg *Config) {
		cfg.session.ClientDescription.ProductURI = s
	}
}

// stubbed out for testing
var randomRequestID func() uint32 = nil

// RandomRequestID assigns a random initial request id.
//
// The request id is generated using the 'rand' package and it
// is the caller's responsibility to initialize the random number
// generator properly.
func RandomRequestID() Option {
	return func(cfg *Config) {
		if randomRequestID != nil {
			cfg.sechan.RequestIDSeed = randomRequestID()
		} else {
			cfg.sechan.RequestIDSeed = uint32(rand.Int31())
		}
	}
}

// RemoteCertificate sets the server certificate.
func RemoteCertificate(cert []byte) Option {
	return func(cfg *Config) {
		cfg.sechan.RemoteCertificate = cert
	}
}

// RemoteCertificateFile sets the server certificate from the file
// in PEM or DER encoding.
func RemoteCertificateFile(filename string) Option {
	return func(cfg *Config) {
		cert, err := loadCertificate(filename)
		if err != nil {
			log.Fatal(err)
		}
		cfg.sechan.RemoteCertificate = cert
	}
}

// SecurityMode sets the security mode for the secure channel.
func SecurityMode(m ua.MessageSecurityMode) Option {
	return func(cfg *Config) {
		cfg.sechan.SecurityMode = m
	}
}

// SecurityModeString sets the security mode for the secure channel.
// Valid values are "None", "Sign", and "SignAndEncrypt".
func SecurityModeString(s string) Option {
	return func(cfg *Config) {
		cfg.sechan.SecurityMode = ua.MessageSecurityModeFromString(s)
	}
}

// SecurityPolicy sets the security policy uri for the secure channel.
func SecurityPolicy(s string) Option {
	return func(cfg *Config) {
		cfg.sechan.SecurityPolicyURI = ua.FormatSecurityPolicyURI(s)
	}
}

// SessionName sets the name in the session configuration.
func SessionName(s string) Option {
	return func(cfg *Config) {
		cfg.session.SessionName = s
	}
}

// SessionTimeout sets the timeout in the session configuration.
func SessionTimeout(d time.Duration) Option {
	return func(cfg *Config) {
		cfg.session.SessionTimeout = d
	}
}

// PrivateKey sets the RSA private key in the secure channel configuration.
func PrivateKey(key *rsa.PrivateKey) Option {
	return func(cfg *Config) {
		cfg.sechan.LocalKey = key
	}
}

// PrivateKeyFile sets the RSA private key in the secure channel configuration
// from a PEM or DER encoded file.
func PrivateKeyFile(filename string) Option {
	return func(cfg *Config) {
		if filename == "" {
			return
		}
		key, err := loadPrivateKey(filename)
		if err != nil {
			log.Fatal(err)
		}
		cfg.sechan.LocalKey = key
	}
}

func loadPrivateKey(filename string) (*rsa.PrivateKey, error) {
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, errors.Errorf("Failed to load private key: %s", err)
	}

	derBytes := b
	if strings.HasSuffix(filename, ".pem") {
		block, _ := pem.Decode(b)
		if block == nil || block.Type != "RSA PRIVATE KEY" {
			return nil, errors.Errorf("Failed to decode PEM block with private key")
		}
		derBytes = block.Bytes
	}

	pk, err := x509.ParsePKCS1PrivateKey(derBytes)
	if err != nil {
		return nil, errors.Errorf("Failed to parse private key: %s", err)
	}
	return pk, nil
}

// Certificate sets the client X509 certificate in the secure channel configuration.
// It also detects and sets the ApplicationURI from the URI within the certificate.
func Certificate(cert []byte) Option {
	return func(cfg *Config) {
		setCertificate(cert, cfg)
	}
}

// Certificate sets the client X509 certificate in the secure channel configuration
// from the PEM or DER encoded file. It also detects and sets the ApplicationURI
// from the URI within the certificate.
func CertificateFile(filename string) Option {
	return func(cfg *Config) {
		if filename == "" {
			return
		}

		cert, err := loadCertificate(filename)
		if err != nil {
			log.Fatal(err)
		}
		setCertificate(cert, cfg)
	}
}

func loadCertificate(filename string) ([]byte, error) {
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, errors.Errorf("Failed to load certificate: %s", err)
	}

	if !strings.HasSuffix(filename, ".pem") {
		return b, nil
	}

	block, _ := pem.Decode(b)
	if block == nil || block.Type != "CERTIFICATE" {
		return nil, errors.Errorf("Failed to decode PEM block with certificate")
	}
	return block.Bytes, nil
}

func setCertificate(cert []byte, cfg *Config) {
	cfg.sechan.Certificate = cert

	// Extract the application URI from the certificate.
	x509cert, err := x509.ParseCertificate(cert)
	if err != nil {
		log.Fatalf("Failed to parse certificate: %s", err)
		return
	}
	if len(x509cert.URIs) == 0 {
		return
	}
	appURI := x509cert.URIs[0].String()
	if appURI == "" {
		return
	}
	cfg.session.ClientDescription.ApplicationURI = appURI
}

// SecurityFromEndpoint sets the server-related security parameters from
// a chosen endpoint (received from GetEndpoints())
func SecurityFromEndpoint(ep *ua.EndpointDescription, authType ua.UserTokenType) Option {
	return func(cfg *Config) {
		cfg.sechan.SecurityPolicyURI = ep.SecurityPolicyURI
		cfg.sechan.SecurityMode = ep.SecurityMode
		cfg.sechan.RemoteCertificate = ep.ServerCertificate
		cfg.sechan.Thumbprint = uapolicy.Thumbprint(ep.ServerCertificate)

		for _, t := range ep.UserIdentityTokens {
			if t.TokenType != authType {
				continue
			}

			if cfg.session.UserIdentityToken == nil {
				switch authType {
				case ua.UserTokenTypeAnonymous:
					cfg.session.UserIdentityToken = &ua.AnonymousIdentityToken{}
				case ua.UserTokenTypeUserName:
					cfg.session.UserIdentityToken = &ua.UserNameIdentityToken{}
				case ua.UserTokenTypeCertificate:
					cfg.session.UserIdentityToken = &ua.X509IdentityToken{}
				case ua.UserTokenTypeIssuedToken:
					cfg.session.UserIdentityToken = &ua.IssuedIdentityToken{}
				}
			}

			setPolicyID(cfg.session.UserIdentityToken, t.PolicyID)
			cfg.session.AuthPolicyURI = t.SecurityPolicyURI
			return
		}

		if cfg.session.UserIdentityToken == nil {
			cfg.session.UserIdentityToken = &ua.AnonymousIdentityToken{PolicyID: defaultAnonymousPolicyID}
			cfg.session.AuthPolicyURI = ua.SecurityPolicyURINone
		}
	}
}

func setPolicyID(t interface{}, policy string) {
	switch tok := t.(type) {
	case *ua.AnonymousIdentityToken:
		tok.PolicyID = policy
	case *ua.UserNameIdentityToken:
		tok.PolicyID = policy
	case *ua.X509IdentityToken:
		tok.PolicyID = policy
	case *ua.IssuedIdentityToken:
		tok.PolicyID = policy
	}
}

// AuthPolicyID sets the policy ID of the user identity token
// Note: This should only be called if you know the exact policy ID the server is expecting.
// Most callers should use SecurityFromEndpoint as it automatically finds the policyID
// todo(fs): Should we make 'policy' an option to the other
// todo(fs): AuthXXX methods since this approach requires context
// todo(fs): and ordering?
func AuthPolicyID(policy string) Option {
	return func(cfg *Config) {
		if cfg.session.UserIdentityToken == nil {
			log.Printf("policy ID needs to be set after the policy type is chosen, no changes made.  Call SecurityFromEndpoint() or an AuthXXX() option first")
			return
		}
		setPolicyID(cfg.session.UserIdentityToken, policy)
	}
}

// AuthAnonymous sets the client's authentication X509 certificate
// Note: PolicyID still needs to be set outside of this method, typically through
// the SecurityFromEndpoint() Option
func AuthAnonymous() Option {
	return func(cfg *Config) {
		if cfg.session.UserIdentityToken == nil {
			cfg.session.UserIdentityToken = &ua.AnonymousIdentityToken{}
		}

		_, ok := cfg.session.UserIdentityToken.(*ua.AnonymousIdentityToken)
		if !ok {
			// todo(fs): should we Fatal here?
			log.Printf("non-anonymous authentication already configured, ignoring")
			return
		}
	}
}

// AuthUsername sets the client's authentication username and password
// Note: PolicyID still needs to be set outside of this method, typically through
// the SecurityFromEndpoint() Option
func AuthUsername(user, pass string) Option {
	return func(cfg *Config) {
		if cfg.session.UserIdentityToken == nil {
			cfg.session.UserIdentityToken = &ua.UserNameIdentityToken{}
		}

		t, ok := cfg.session.UserIdentityToken.(*ua.UserNameIdentityToken)
		if !ok {
			// todo(fs): should we Fatal here?
			log.Printf("non-username authentication already configured, ignoring")
			return
		}

		t.UserName = user
		cfg.session.AuthPassword = pass
	}
}

// AuthCertificate sets the client's authentication X509 certificate
// Note: PolicyID still needs to be set outside of this method, typically through
// the SecurityFromEndpoint() Option
func AuthCertificate(cert []byte) Option {
	return func(cfg *Config) {
		if cfg.session.UserIdentityToken == nil {
			cfg.session.UserIdentityToken = &ua.X509IdentityToken{}
		}

		t, ok := cfg.session.UserIdentityToken.(*ua.X509IdentityToken)
		if !ok {
			// todo(fs): should we Fatal here?
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
	return func(cfg *Config) {
		if cfg.session.UserIdentityToken == nil {
			cfg.session.UserIdentityToken = &ua.IssuedIdentityToken{}
		}

		t, ok := cfg.session.UserIdentityToken.(*ua.IssuedIdentityToken)
		if !ok {
			log.Printf("non-issued token authentication already configured, ignoring")
			return
		}

		// todo(dw): not correct; need to read spec
		t.TokenData = tokenData
	}
}

// RequestTimeout sets the timeout for all requests over SecureChannel
func RequestTimeout(t time.Duration) Option {
	return func(cfg *Config) {
		cfg.sechan.RequestTimeout = t
	}
}

// Dialer sets the uacp.Dialer to establish the connection to the server.
func Dialer(d *uacp.Dialer) Option {
	return func(cfg *Config) {
		initDialer(cfg)
		cfg.dialer = d
	}
}

// DialTimeout sets the timeout for establishing the UACP connection.
// Defaults to DefaultDialTimeout. Set to zero for no timeout.
func DialTimeout(d time.Duration) Option {
	return func(cfg *Config) {
		initDialer(cfg)
		cfg.dialer.Dialer.Timeout = d
	}
}

// MaxMessageSize sets the maximum message size for the UACP handshake.
func MaxMessageSize(n uint32) Option {
	return func(cfg *Config) {
		initDialer(cfg)
		cfg.dialer.ClientACK.MaxMessageSize = n
	}
}

// MaxChunkCount sets the maximum chunk count for the UACP handshake.
func MaxChunkCount(n uint32) Option {
	return func(cfg *Config) {
		initDialer(cfg)
		cfg.dialer.ClientACK.MaxChunkCount = n
	}
}

// ReceiveBufferSize sets the receive buffer size for the UACP handshake.
func ReceiveBufferSize(n uint32) Option {
	return func(cfg *Config) {
		initDialer(cfg)
		cfg.dialer.ClientACK.ReceiveBufSize = n
	}
}

// SendBufferSize sets the send buffer size for the UACP handshake.
func SendBufferSize(n uint32) Option {
	return func(cfg *Config) {
		initDialer(cfg)
		cfg.dialer.ClientACK.SendBufSize = n
	}
}

func initDialer(cfg *Config) {
	if cfg.dialer == nil {
		cfg.dialer = &uacp.Dialer{}
	}
	if cfg.dialer.Dialer == nil {
		cfg.dialer.Dialer = &net.Dialer{}
	}
	if cfg.dialer.ClientACK == nil {
		cfg.dialer.ClientACK = uacp.DefaultClientACK
	}
}
