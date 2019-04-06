package opcua

import (
	"math/rand"
	"time"

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
		LocaleIDs:         []string{"en-us"},
		UserIdentityToken: &ua.AnonymousIdentityToken{PolicyID: "open62541-anonymous-policy"},
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
