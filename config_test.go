package opcua

import (
	"testing"
	"time"

	"github.com/gopcua/opcua/ua"
	"github.com/gopcua/opcua/uasc"

	"github.com/pascaldekloe/goe/verify"
)

func TestOptions(t *testing.T) {
	tests := []struct {
		name string
		opt  Option
		c    *uasc.Config
		sc   *uasc.SessionConfig
	}{
		{
			name: `ApplicationName("a")`,
			opt:  ApplicationName("a"),
			sc: func() *uasc.SessionConfig {
				sc := DefaultSessionConfig()
				sc.ClientDescription.ApplicationName = &ua.LocalizedText{Text: "a"}
				return sc
			}(),
		},
		{
			name: `ApplicationURI("a")`,
			opt:  ApplicationURI("a"),
			sc: func() *uasc.SessionConfig {
				sc := DefaultSessionConfig()
				sc.ClientDescription.ApplicationURI = "a"
				return sc
			}(),
		},
		{
			name: `Lifetime(10ms)`,
			opt:  Lifetime(10 * time.Millisecond),
			c: func() *uasc.Config {
				c := DefaultClientConfig()
				c.Lifetime = 10
				return c
			}(),
		},
		{
			name: `Locales("en-us", "de-de")`,
			opt:  Locales("en-us", "de-de"),
			sc: func() *uasc.SessionConfig {
				sc := DefaultSessionConfig()
				sc.LocaleIDs = []string{"en-us", "de-de"}
				return sc
			}(),
		},
		{
			name: `ProductURI("a")`,
			opt:  ProductURI("a"),
			sc: func() *uasc.SessionConfig {
				sc := DefaultSessionConfig()
				sc.ClientDescription.ProductURI = "a"
				return sc
			}(),
		},
		{
			name: `SecurityPolicy("None")`,
			opt:  SecurityPolicy("None"),
			c: func() *uasc.Config {
				c := DefaultClientConfig()
				c.SecurityPolicyURI = ua.SecurityPolicyURINone
				return c
			}(),
		},
		{
			name: `SecurityMode(Sign)`,
			opt:  SecurityMode(ua.MessageSecurityModeSign),
			c: func() *uasc.Config {
				c := DefaultClientConfig()
				c.SecurityMode = ua.MessageSecurityModeSign
				return c
			}(),
		},
		{
			name: `SecurityModeString("None")`,
			opt:  SecurityModeString("None"),
			c: func() *uasc.Config {
				c := DefaultClientConfig()
				c.SecurityMode = ua.MessageSecurityModeNone
				return c
			}(),
		},
		{
			name: `SecurityModeString("Sign")`,
			opt:  SecurityModeString("Sign"),
			c: func() *uasc.Config {
				c := DefaultClientConfig()
				c.SecurityMode = ua.MessageSecurityModeSign
				return c
			}(),
		},
		{
			name: `SecurityModeString("SignAndEncrypt")`,
			opt:  SecurityModeString("SignAndEncrypt"),
			c: func() *uasc.Config {
				c := DefaultClientConfig()
				c.SecurityMode = ua.MessageSecurityModeSignAndEncrypt
				return c
			}(),
		},
		{
			name: `SecurityPolicy("Basic128Rsa15")`,
			opt:  SecurityPolicy("Basic128Rsa15"),
			c: func() *uasc.Config {
				c := DefaultClientConfig()
				c.SecurityPolicyURI = ua.SecurityPolicyURIBasic128Rsa15
				return c
			}(),
		},
		{
			name: `SecurityPolicy("Basic256")`,
			opt:  SecurityPolicy("Basic256"),
			c: func() *uasc.Config {
				c := DefaultClientConfig()
				c.SecurityPolicyURI = ua.SecurityPolicyURIBasic256
				return c
			}(),
		},
		{
			name: `SecurityPolicy("Basic256Sha256")`,
			opt:  SecurityPolicy("Basic256Sha256"),
			c: func() *uasc.Config {
				c := DefaultClientConfig()
				c.SecurityPolicyURI = ua.SecurityPolicyURIBasic256Sha256
				return c
			}(),
		},
		{
			name: `SecurityPolicy("Aes128_Sha256_RsaOaep")`,
			opt:  SecurityPolicy("Aes128_Sha256_RsaOaep"),
			c: func() *uasc.Config {
				c := DefaultClientConfig()
				c.SecurityPolicyURI = ua.SecurityPolicyURIAes128Sha256RsaOaep
				return c
			}(),
		},
		{
			name: `SecurityPolicy("Aes256_Sha256_RsaPss")`,
			opt:  SecurityPolicy("Aes256_Sha256_RsaPss"),
			c: func() *uasc.Config {
				c := DefaultClientConfig()
				c.SecurityPolicyURI = ua.SecurityPolicyURIAes256Sha256RsaPss
				return c
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ttc := tt.c
			if ttc == nil {
				ttc = DefaultClientConfig()
			}
			ttsc := tt.sc
			if ttsc == nil {
				ttsc = DefaultSessionConfig()
			}

			c, sc := ApplyConfig(tt.opt)
			verify.Values(t, "", c, ttc)
			verify.Values(t, "", sc, ttsc)
		})
	}
}
