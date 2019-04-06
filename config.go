package opcua

import (
	"github.com/gopcua/opcua/ua"
	"github.com/gopcua/opcua/uasc"
)

// DefaultClientConfig returns the default configuration for a client
// to establish a secure channel and a session.
func DefaultClientConfig() *uasc.Config {
	return &uasc.Config{
		SecurityPolicyURI: uasc.SecurityPolicyNone,
		SecurityMode:      ua.MessageSecurityModeNone,
		Session: &uasc.SessionConfig{
			SessionTimeout: 0xffff,
			ClientDescription: &ua.ApplicationDescription{
				ApplicationURI:  "urn:gopcua:client",
				ProductURI:      "urn:gopcua",
				ApplicationName: &ua.LocalizedText{Text: "gopcua - OPC UA implementation in Go"},
				ApplicationType: ua.ApplicationTypeClient,
			},
			LocaleIDs:         []string{"en-us"},
			UserIdentityToken: &ua.AnonymousIdentityToken{PolicyID: "open62541-anonymous-policy"},
		},
	}
}

// Option is an option function type to modify the configuration.
type Option func(*uasc.Config)

// ApplicationURI sets the application uri in the session configuration.
func ApplicationURI(s string) Option {
	return func(c *uasc.Config) {
		c.Session.ClientDescription.ApplicationURI = s
	}
}

// ApplicationName sets the application name in the session configuration.
func ApplicationName(s string) Option {
	return func(c *uasc.Config) {
		c.Session.ClientDescription.ApplicationName = &ua.LocalizedText{Text: s}
	}
}

// Locales sets the locales in the session configuration.
func Locales(locale ...string) Option {
	return func(c *uasc.Config) {
		c.Session.LocaleIDs = locale
	}
}

// ProductURI sets the product uri in the session configuration.
func ProductURI(s string) Option {
	return func(c *uasc.Config) {
		c.Session.ClientDescription.ProductURI = s
	}
}

// SecurityMode sets the security mode for the secure channel.
func SecurityMode(m ua.MessageSecurityMode) Option {
	return func(c *uasc.Config) {
		c.SecurityMode = m
	}
}

// SecurityPolicy sets the security policy uri for the secure channel.
func SecurityPolicy(s string) Option {
	return func(c *uasc.Config) {
		c.SecurityPolicyURI = s
	}
}

// SessionTimeout sets the timeout in the session configuration.
func SessionTimeout(seconds float64) Option {
	return func(c *uasc.Config) {
		c.Session.SessionTimeout = seconds
	}
}
