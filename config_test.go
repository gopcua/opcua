package opcua

import (
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gopcua/opcua/ua"
	"github.com/gopcua/opcua/uacp"
	"github.com/gopcua/opcua/uapolicy"
	"github.com/gopcua/opcua/uasc"
	"github.com/stretchr/testify/require"
)

// test certificate generated with
// go run ~/sdk/gotip/src/crypto/tls/generate_cert.go -rsa-bits 1024 -host localhost
// expires Jun  5 20:10:13 2020 GMT
var (
	certPEM = []byte(`-----BEGIN CERTIFICATE-----
MIIB9TCCAV6gAwIBAgIRAJkygYaTfLZ9tOwtJvxdP7EwDQYJKoZIhvcNAQELBQAw
EjEQMA4GA1UEChMHQWNtZSBDbzAeFw0xOTA2MDYyMDEwMTNaFw0yMDA2MDUyMDEw
MTNaMBIxEDAOBgNVBAoTB0FjbWUgQ28wgZ8wDQYJKoZIhvcNAQEBBQADgY0AMIGJ
AoGBAJ8cfel4q4jcXrmGxiXmPrJzg0aZxKdWlPE23fr9KwpYkrQZ5ykzs6sGuuXE
OYtqINQNBP/5VXCinnDOZppI4QHlbUrWfKoGgJU2wQZuAQ7+Pz4l96EM5DnIBArb
liSp5s2LZiVLgw6v9tS6yu/Ci5QyfuyMz4JLg25Vt1KHFCD9AgMBAAGjSzBJMA4G
A1UdDwEB/wQEAwIFoDATBgNVHSUEDDAKBggrBgEFBQcDATAMBgNVHRMBAf8EAjAA
MBQGA1UdEQQNMAuCCWxvY2FsaG9zdDANBgkqhkiG9w0BAQsFAAOBgQAPuVO5vkF9
LfQ1JCrXC52CdKm8Gs+bYaDLQa6re4HaNPHAuEJaeMAHJ/4PHSsg6ghZ1MmBj1pc
GY1Q+sfu64IRjFdhnbL97a6GL+MgEVIvT9cl/DDcXtNZIl28Xk4KwAp3/lB1XrgK
cdqKnNkOBU19ulD8SOKzAPch5ydHPFfXCw==
-----END CERTIFICATE-----
`)
	certDER = derBytes(certPEM)

	keyPKCS1PEM = []byte(`-----BEGIN RSA PRIVATE KEY-----
MIICXQIBAAKBgQCfHH3peKuI3F65hsYl5j6yc4NGmcSnVpTxNt36/SsKWJK0Gecp
M7OrBrrlxDmLaiDUDQT/+VVwop5wzmaaSOEB5W1K1nyqBoCVNsEGbgEO/j8+Jfeh
DOQ5yAQK25YkqebNi2YlS4MOr/bUusrvwouUMn7sjM+CS4NuVbdShxQg/QIDAQAB
AoGAfDdmJUtLv4ErgnOxZg0mjXKY3hlV6b4ycU6AZl4Xp/AWN/yw9v2iGrcaWh6j
PjAQiPvRF3W0Okb6ot7AQ1r6LaGWoRRlAhapJ0ZYw/TrlXULgXuvDjmlU6yfiSQQ
VBBdqJkqDr5El6a85MZlprxYz1OPudZyUGIPcr/wwj+a7pECQQDFVDS5G54T+x/m
RHUkGV+M2osHHBBaRKmiKlblg2U0Ep2P2eYmldcnt5xQxxrQedxkDJAuwR7hTQH0
o/Nnqh9fAkEAzmtUtCJ4Yp+6IBKFffGj86PfGHNSh9DARZ3fQPBDVFtRT9qVCwMA
rU9vJ/n1jvtYCYhzCFCuSSFTEIBglPiJIwJAQSdvjRsgU8qcGsTJxNSX5wMV2pAa
miOHuyKttHRxCwOGgMPaqSzacKPAei9znBhQe7xmMvnS/2MU3TjxGm5ikwJBALXl
YNvnsDwAUsymZZoJEJfHJPXv0Z869eOi7bPUxRAV9D4w+LueZr9SSzpoCtp3ZCnq
YqvGJP7ubbsR1YoQxQ8CQQCyCrltDYji5+KdxMOsDt0v7bCQWkQ3+pik09faK51Y
4upIBnmHPbJ80DfFIj/93JXna5JQpnIZGn/hitRixBWU
-----END RSA PRIVATE KEY-----
`)
	keyPKCS1DER = derBytes(keyPKCS1PEM)
	cert        = x509Cert(certPEM, keyPKCS1PEM)

	keyPKCS8PEM = []byte(`-----BEGIN PRIVATE KEY-----
MIICeAIBADANBgkqhkiG9w0BAQEFAASCAmIwggJeAgEAAoGBANUocwaux45f6JhU
QHygnO/UfitJFEFd/CeztncyAQtW1Sja19s8w/Xf5YZSSnsp96N7Op1SFE+0dAKc
dZT5KF6H0bAlcx1YMQWRaRc17523Ft3fkzikPL1dRrUprrnz2MhhyqlL1UiB3CEg
7m1yly3ut1feE+zuGTnvP7ETiuY5AgMBAAECgYEAjE/oB8odSjcP4NX07RS8uZJi
yxN75dt8FJZT0fp0fYZXImGMHaDOTZdoexbIOHLTtCV13AEfpaffhaiALeQlEYA4
3eo+YEbFNtRs3BtxLoDc82McH70RFKjXudtsXDdq6eea0I34GvpWHEKfRJSdmEB9
jna0pI5XV4cXJJ+qBdECQQD9psF5qnGnXjU7f4SA3nNo67pT1sz/TJLsMd3pxqmS
w6VR25Y+LcU/iJEeBzPzCgH5wpuWcAHZiEVeYee1hwbPAkEA1yG1tVMESvv8yN71
qec2BOWKnDA9EN0qjIreBQbB9F+vnalJpJ2b4h1/rRkTi0OzmXzlLTP07+TIPC6K
WnyEdwJAWROBqF9h8FvWJ+HdP4BfWT5HPgAWF6XlhsrwWpOoo2DPotKRjZ53QZuN
EtWGudgO344nI4qMK79+VOne/FHB4wJBALLzbIY3VyPUtrKUnH9HP+0Uz5canUFQ
59rejM5bj6zqh1e7gPG41PljFlhzuokmuNfdR3mxdXaztUgyYo3gdAMCQQC/TMqc
94ZVfT5WHoLDd86J0G4Eatl/xoA+NdHs9WRxN11RrAhH/9tdK5jV9xeMa9tncHwY
qVM50NTCCBbetzVG
-----END PRIVATE KEY-----
`)
	keyPKCS8DER = derBytes(keyPKCS8PEM)
	keyPKCS8    = rsaKeyFromPKCS8PEM(keyPKCS8PEM)

	keyPKCS8ECPem = []byte(`-----BEGIN PRIVATE KEY-----
MIGHAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBG0wawIBAQQgiM2hA0sCCaNxhwPy
E0GxKISgpTDWaL/h8FmM/PuprqihRANCAAQHtEMlgfUrDdt24KJRQihb5PsjOVB+
17YDvxN/otim+q8xEacXbb7MlY6N1Vx5bbzFUDw00jVHOfWjbppru98b
-----END PRIVATE KEY-----
`)

	endpoints = []*ua.EndpointDescription{
		// anonymous auth
		{
			SecurityPolicyURI: "a", // random value for testing
			SecurityMode:      5,   // random value for testing
			ServerCertificate: certDER,
			UserIdentityTokens: []*ua.UserTokenPolicy{
				{
					TokenType:         ua.UserTokenTypeAnonymous,
					SecurityPolicyURI: "b", // random value for testing
				},
			},
		},
		// username auth
		{
			SecurityPolicyURI: "a", // random value for testing
			SecurityMode:      5,   // random value for testing
			ServerCertificate: certDER,
			UserIdentityTokens: []*ua.UserTokenPolicy{
				{
					TokenType:         ua.UserTokenTypeUserName,
					SecurityPolicyURI: "b", // random value for testing
				},
			},
		},
		// x509 cert auth
		{
			SecurityPolicyURI: "a", // random value for testing
			SecurityMode:      5,   // random value for testing
			ServerCertificate: certDER,
			UserIdentityTokens: []*ua.UserTokenPolicy{
				{
					TokenType:         ua.UserTokenTypeCertificate,
					SecurityPolicyURI: "b", // random value for testing
				},
			},
		},
		// issued token auth
		{
			SecurityPolicyURI: "a", // random value for testing
			SecurityMode:      5,   // random value for testing
			ServerCertificate: certDER,
			UserIdentityTokens: []*ua.UserTokenPolicy{
				{
					TokenType:         ua.UserTokenTypeIssuedToken,
					SecurityPolicyURI: "b", // random value for testing
				},
			},
		},
		// no user security policy uri
		{
			SecurityPolicyURI: "c", // random value for testing
			SecurityMode:      5,   // random value for testing
			ServerCertificate: certDER,
			UserIdentityTokens: []*ua.UserTokenPolicy{
				{
					TokenType:         ua.UserTokenTypeIssuedToken,
					SecurityPolicyURI: "", // empty
				},
			},
		},
	}
)

func derBytes(b []byte) []byte {
	block, _ := pem.Decode(b)
	return block.Bytes
}

func x509Cert(c, k []byte) tls.Certificate {
	cert, err := tls.X509KeyPair(c, k)
	if err != nil {
		panic(err)
	}
	return cert
}

// rsaKeyFromPKCS8PEM parses a PKCS#8 PEM-encoded RSA private key.
// It is used to produce the expected *rsa.PrivateKey value in tests.
func rsaKeyFromPKCS8PEM(pemBytes []byte) *rsa.PrivateKey {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		panic("rsaKeyFromPKCS8PEM: failed to decode PEM block")
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		panic("rsaKeyFromPKCS8PEM: " + err.Error())
	}
	pk, ok := key.(*rsa.PrivateKey)
	if !ok {
		panic("rsaKeyFromPKCS8PEM: key is not an RSA key")
	}
	return pk
}

func TestOptions(t *testing.T) {
	randomRequestID = func() uint32 { return 125 }
	defer func() { randomRequestID = nil }()

	d := t.TempDir()

	var (
		certDERFile     = filepath.Join(d, "cert.der")
		certPEMFile     = filepath.Join(d, "cert.pem")
		keyDERFile      = filepath.Join(d, "key.der")
		keyPEMFile      = filepath.Join(d, "key.pem")
		keyPKCS8PEMFile = filepath.Join(d, "key-pkcs8.pem")
		keyPKCS8DERFile = filepath.Join(d, "key-pkcs8.der")
		keyPKCS8ECFile  = filepath.Join(d, "key-pkcs8-ec.pem")
	)

	// the error message for "file not found" is platform dependent.
	notFoundError := func(msg, name string) error {
		switch runtime.GOOS {
		case "windows":
			return fmt.Errorf("opcua: Failed to load %s: open %s: The system cannot find the file specified.", msg, name)
		default:
			return fmt.Errorf("opcua: Failed to load %s: open %s: no such file or directory", msg, name)
		}
	}

	err := os.WriteFile(certDERFile, certDER, 0644)
	require.NoError(t, err, "WriteFile(certDERFile) failed")

	err = os.WriteFile(certPEMFile, certPEM, 0644)
	require.NoError(t, err, "WriteFile(certPEMFile) failed")

	err = os.WriteFile(keyDERFile, keyPKCS1DER, 0644)
	require.NoError(t, err, "WriteFile(keyDERFile) failed")

	err = os.WriteFile(keyPEMFile, keyPKCS1PEM, 0644)
	require.NoError(t, err, "WriteFile(keyPEMFile) failed")

	err = os.WriteFile(keyPKCS8PEMFile, keyPKCS8PEM, 0644)
	require.NoError(t, err, "WriteFile(keyPKCS8PEMFile) failed")

	err = os.WriteFile(keyPKCS8DERFile, keyPKCS8DER, 0644)
	require.NoError(t, err, "WriteFile(keyPKCS8DERFile) failed")

	err = os.WriteFile(keyPKCS8ECFile, keyPKCS8ECPem, 0644)
	require.NoError(t, err, "WriteFile(keyPKCS8ECFile) failed")

	connStateCh := make(chan ConnState)
	connStateFunc := func(ConnState) {}

	tests := []struct {
		name string
		opt  Option
		cfg  *Config
		err  error
	}{
		{
			name: `ApplicationName("a")`,
			opt:  ApplicationName("a"),
			cfg: &Config{
				session: func() *uasc.SessionConfig {
					sc := DefaultSessionConfig()
					sc.ClientDescription.ApplicationName = ua.NewLocalizedText("a")
					return sc
				}(),
			},
		},
		{
			name: `ApplicationURI("a")`,
			opt:  ApplicationURI("a"),
			cfg: &Config{
				session: func() *uasc.SessionConfig {
					sc := DefaultSessionConfig()
					sc.ClientDescription.ApplicationURI = "a"
					return sc
				}(),
			},
		},
		{
			name: `AuthAnonymous()`,
			opt:  AuthAnonymous(),
			cfg: &Config{
				session: func() *uasc.SessionConfig {
					sc := DefaultSessionConfig()
					sc.UserIdentityToken = &ua.AnonymousIdentityToken{}
					return sc
				}(),
			},
		},
		{
			name: `AuthCertificate()`,
			opt:  AuthCertificate(certDER),
			cfg: &Config{
				session: func() *uasc.SessionConfig {
					sc := DefaultSessionConfig()
					sc.UserIdentityToken = &ua.X509IdentityToken{
						CertificateData: certDER,
					}
					return sc
				}(),
			},
		},
		{
			name: `AuthPrivateKey()`,
			opt:  AuthPrivateKey(cert.PrivateKey.(*rsa.PrivateKey)),
			cfg: &Config{
				sechan: func() *uasc.Config {
					c := DefaultClientConfig()
					c.UserKey = cert.PrivateKey.(*rsa.PrivateKey)
					return c
				}(),
			},
		},
		{
			name: `AuthIssuedToken()`,
			opt:  AuthIssuedToken([]byte("a")),
			cfg: &Config{
				session: func() *uasc.SessionConfig {
					sc := DefaultSessionConfig()
					sc.UserIdentityToken = &ua.IssuedIdentityToken{
						TokenData: []byte("a"),
					}
					return sc
				}(),
			},
		},
		{
			name: `AuthUsername()`,
			opt:  AuthUsername("user", "pass"),
			cfg: &Config{
				session: func() *uasc.SessionConfig {
					sc := DefaultSessionConfig()
					sc.UserIdentityToken = &ua.UserNameIdentityToken{
						UserName: "user",
					}
					sc.AuthPassword = "pass"
					return sc
				}(),
			},
		},
		{
			name: `AutoReconnect()`,
			opt:  AutoReconnect(true),
			cfg: &Config{
				sechan: func() *uasc.Config {
					c := DefaultClientConfig()
					c.AutoReconnect = true
					return c
				}(),
			},
		},
		{
			name: `Certificate`,
			opt:  Certificate(certDER),
			cfg: &Config{
				sechan: func() *uasc.Config {
					c := DefaultClientConfig()
					c.Certificate = certDER
					// todo(fs): test with cert that has a URI
					// sc.ClientDescription.ApplicationURI = ...
					return c
				}(),
			},
		},
		{
			name: `CertificateFile("cert.der")`,
			opt:  CertificateFile(certDERFile),
			cfg: &Config{
				sechan: func() *uasc.Config {
					c := DefaultClientConfig()
					c.Certificate = certDER
					// todo(fs): test with cert that has a URI
					// sc.ClientDescription.ApplicationURI = ...
					return c
				}(),
			},
		},
		{
			name: `CertificateFile("cert.pem")`,
			opt:  CertificateFile(certPEMFile),
			cfg: &Config{
				sechan: func() *uasc.Config {
					c := DefaultClientConfig()
					c.Certificate = certDER
					// todo(fs): test with cert that has a URI
					// sc.ClientDescription.ApplicationURI = ...
					return c
				}(),
			},
		},
		{
			name: `CertificateFile() error`,
			opt:  CertificateFile("x"),
			cfg:  &Config{},
			err:  notFoundError("certificate", "x"),
		},
		{
			name: `ConnStateCh`,
			opt:  StateChangedCh(connStateCh),
			cfg: &Config{
				stateCh: connStateCh,
			},
		},
		{
			name: `ConnStateCh`,
			opt:  StateChangedFunc(connStateFunc),
			cfg: &Config{
				stateFunc: connStateFunc,
			},
		},
		{
			name: `Lifetime(10ms)`,
			opt:  Lifetime(10 * time.Millisecond),
			cfg: &Config{
				sechan: func() *uasc.Config {
					c := DefaultClientConfig()
					c.Lifetime = 10
					return c
				}(),
			},
		},
		{
			name: `Locales("en-us", "de-de")`,
			opt:  Locales("en-us", "de-de"),
			cfg: &Config{
				session: func() *uasc.SessionConfig {
					sc := DefaultSessionConfig()
					sc.LocaleIDs = []string{"en-us", "de-de"}
					return sc
				}(),
			},
		},
		{
			name: `PrivateKey()`,
			opt:  PrivateKey(cert.PrivateKey.(*rsa.PrivateKey)),
			cfg: &Config{
				sechan: func() *uasc.Config {
					c := DefaultClientConfig()
					c.LocalKey = cert.PrivateKey.(*rsa.PrivateKey)
					return c
				}(),
			},
		},
		{
			name: `PrivateKeyFile("key.der")`,
			opt:  PrivateKeyFile(keyDERFile),
			cfg: &Config{
				sechan: func() *uasc.Config {
					c := DefaultClientConfig()
					c.LocalKey = cert.PrivateKey.(*rsa.PrivateKey)
					return c
				}(),
			},
		},
		{
			name: `PrivateKeyFile("key.pem")`,
			opt:  PrivateKeyFile(keyPEMFile),
			cfg: &Config{
				sechan: func() *uasc.Config {
					c := DefaultClientConfig()
					c.LocalKey = cert.PrivateKey.(*rsa.PrivateKey)
					return c
				}(),
			},
		},
		{
			name: `PrivateKeyFile("key-pkcs8.pem")`,
			opt:  PrivateKeyFile(keyPKCS8PEMFile),
			cfg: &Config{
				sechan: func() *uasc.Config {
					c := DefaultClientConfig()
					c.LocalKey = keyPKCS8
					return c
				}(),
			},
		},
		{
			name: `PrivateKeyFile("key-pkcs8.der")`,
			opt:  PrivateKeyFile(keyPKCS8DERFile),
			cfg: &Config{
				sechan: func() *uasc.Config {
					c := DefaultClientConfig()
					c.LocalKey = keyPKCS8
					return c
				}(),
			},
		},
		{
			name: `PrivateKeyFile("key-pkcs8-ec.pem") error`,
			opt:  PrivateKeyFile(keyPKCS8ECFile),
			cfg:  &Config{},
			err:  fmt.Errorf("opcua: Private key is not an RSA key"),
		},
		{
			name: `PrivateKeyFile() error`,
			opt:  PrivateKeyFile("x"),
			cfg:  &Config{},
			err:  notFoundError("private key", "x"),
		},
		{
			name: `ProductURI("a")`,
			opt:  ProductURI("a"),
			cfg: &Config{
				session: func() *uasc.SessionConfig {
					sc := DefaultSessionConfig()
					sc.ClientDescription.ProductURI = "a"
					return sc
				}(),
			},
		},
		{
			name: `RandomRequestID()`,
			opt:  RandomRequestID(),
			cfg: &Config{
				sechan: func() *uasc.Config {
					c := DefaultClientConfig()
					c.RequestIDSeed = 125
					return c
				}(),
			},
		},
		{
			name: `ReconnectInterval()`,
			opt:  ReconnectInterval(5 * time.Second),
			cfg: &Config{
				sechan: func() *uasc.Config {
					c := DefaultClientConfig()
					c.ReconnectInterval = 5 * time.Second
					return c
				}(),
			},
		},
		{
			name: `RemoteCertificate`,
			opt:  RemoteCertificate(certDER),
			cfg: &Config{
				sechan: func() *uasc.Config {
					c := DefaultClientConfig()
					c.RemoteCertificate = certDER
					return c
				}(),
			},
		},
		{
			name: `RemoteCertificateFile("cert.der")`,
			opt:  RemoteCertificateFile(certDERFile),
			cfg: &Config{
				sechan: func() *uasc.Config {
					c := DefaultClientConfig()
					c.RemoteCertificate = certDER
					return c
				}(),
			},
		},
		{
			name: `RemoteCertificateFile("cert.pem")`,
			opt:  RemoteCertificateFile(certPEMFile),
			cfg: &Config{
				sechan: func() *uasc.Config {
					c := DefaultClientConfig()
					c.RemoteCertificate = certDER
					return c
				}(),
			},
		},
		{
			name: `RemoteCertificateFile() error`,
			opt:  RemoteCertificateFile("x"),
			cfg:  &Config{},
			err:  notFoundError("certificate", "x"),
		},
		{
			name: `RequestTimeout(5s)`,
			opt:  RequestTimeout(5 * time.Second),
			cfg: &Config{
				sechan: func() *uasc.Config {
					c := DefaultClientConfig()
					c.RequestTimeout = 5 * time.Second
					return c
				}(),
			},
		},
		{
			name: `SecurityFromEndpoint(no-match)`,
			opt: SecurityFromEndpoint(&ua.EndpointDescription{
				SecurityPolicyURI: "a",
				SecurityMode:      5,
				ServerCertificate: certDER,
			}, 0),
			cfg: &Config{
				sechan: func() *uasc.Config {
					c := DefaultClientConfig()
					c.SecurityPolicyURI = "a"
					c.SecurityMode = 5
					c.RemoteCertificate = certDER
					c.Thumbprint = uapolicy.Thumbprint(certDER)
					return c
				}(),
				session: func() *uasc.SessionConfig {
					sc := DefaultSessionConfig()
					sc.UserIdentityToken = &ua.AnonymousIdentityToken{
						PolicyID: defaultAnonymousPolicyID,
					}
					sc.AuthPolicyURI = ua.SecurityPolicyURINone
					return sc
				}(),
			},
		},
		{
			name: `SecurityFromEndpoint(anonymous)`,
			opt:  SecurityFromEndpoint(endpoints[0], ua.UserTokenTypeAnonymous),
			cfg: &Config{
				sechan: func() *uasc.Config {
					c := DefaultClientConfig()
					c.SecurityPolicyURI = "a"
					c.SecurityMode = 5
					c.RemoteCertificate = certDER
					c.Thumbprint = uapolicy.Thumbprint(certDER)
					return c
				}(),
				session: func() *uasc.SessionConfig {
					sc := DefaultSessionConfig()
					sc.UserIdentityToken = &ua.AnonymousIdentityToken{}
					sc.AuthPolicyURI = "b"
					return sc
				}(),
			},
		},
		{
			name: `SecurityFromEndpoint(username)`,
			opt:  SecurityFromEndpoint(endpoints[1], ua.UserTokenTypeUserName),
			cfg: &Config{
				sechan: func() *uasc.Config {
					c := DefaultClientConfig()
					c.SecurityPolicyURI = "a"
					c.SecurityMode = 5
					c.RemoteCertificate = certDER
					c.Thumbprint = uapolicy.Thumbprint(certDER)
					return c
				}(),
				session: func() *uasc.SessionConfig {
					sc := DefaultSessionConfig()
					sc.UserIdentityToken = &ua.UserNameIdentityToken{}
					sc.AuthPolicyURI = "b"
					return sc
				}(),
			},
		},
		{
			name: `SecurityFromEndpoint(certificate)`,
			opt:  SecurityFromEndpoint(endpoints[2], ua.UserTokenTypeCertificate),
			cfg: &Config{
				sechan: func() *uasc.Config {
					c := DefaultClientConfig()
					c.SecurityPolicyURI = "a"
					c.SecurityMode = 5
					c.RemoteCertificate = certDER
					c.Thumbprint = uapolicy.Thumbprint(certDER)
					return c
				}(),
				session: func() *uasc.SessionConfig {
					sc := DefaultSessionConfig()
					sc.UserIdentityToken = &ua.X509IdentityToken{}
					sc.AuthPolicyURI = "b"
					return sc
				}(),
			},
		},
		{
			name: `SecurityFromEndpoint(token)`,
			opt:  SecurityFromEndpoint(endpoints[3], ua.UserTokenTypeIssuedToken),
			cfg: &Config{
				sechan: func() *uasc.Config {
					c := DefaultClientConfig()
					c.SecurityPolicyURI = "a"
					c.SecurityMode = 5
					c.RemoteCertificate = certDER
					c.Thumbprint = uapolicy.Thumbprint(certDER)
					return c
				}(),
				session: func() *uasc.SessionConfig {
					sc := DefaultSessionConfig()
					sc.UserIdentityToken = &ua.IssuedIdentityToken{}
					sc.AuthPolicyURI = "b"
					return sc
				}(),
			},
		},
		{
			name: `SecurityFromEndpoint(no-user-auth-policy-uri)`,
			opt:  SecurityFromEndpoint(endpoints[4], ua.UserTokenTypeIssuedToken),
			cfg: &Config{
				sechan: func() *uasc.Config {
					c := DefaultClientConfig()
					c.SecurityPolicyURI = "c"
					c.SecurityMode = 5
					c.RemoteCertificate = certDER
					c.Thumbprint = uapolicy.Thumbprint(certDER)
					return c
				}(),
				session: func() *uasc.SessionConfig {
					sc := DefaultSessionConfig()
					sc.UserIdentityToken = &ua.IssuedIdentityToken{}
					sc.AuthPolicyURI = "c"
					return sc
				}(),
			},
		},
		{
			name: `SecurityPolicy("None")`,
			opt:  SecurityPolicy("None"),
			cfg: &Config{
				sechan: func() *uasc.Config {
					c := DefaultClientConfig()
					c.SecurityPolicyURI = ua.SecurityPolicyURINone
					return c
				}(),
			},
		},
		{
			name: `SecurityMode(Sign)`,
			opt:  SecurityMode(ua.MessageSecurityModeSign),
			cfg: &Config{
				sechan: func() *uasc.Config {
					c := DefaultClientConfig()
					c.SecurityMode = ua.MessageSecurityModeSign
					return c
				}(),
			},
		},
		{
			name: `SecurityModeString("bad")`,
			opt:  SecurityModeString("bad"),
			cfg: &Config{
				sechan: func() *uasc.Config {
					c := DefaultClientConfig()
					c.SecurityMode = ua.MessageSecurityModeInvalid
					return c
				}(),
			},
		},
		{
			name: `SecurityModeString("None")`,
			opt:  SecurityModeString("None"),
			cfg: &Config{
				sechan: func() *uasc.Config {
					c := DefaultClientConfig()
					c.SecurityMode = ua.MessageSecurityModeNone
					return c
				}(),
			},
		},
		{
			name: `SecurityModeString("Sign")`,
			opt:  SecurityModeString("Sign"),
			cfg: &Config{
				sechan: func() *uasc.Config {
					c := DefaultClientConfig()
					c.SecurityMode = ua.MessageSecurityModeSign
					return c
				}(),
			},
		},
		{
			name: `SecurityModeString("SignAndEncrypt")`,
			opt:  SecurityModeString("SignAndEncrypt"),
			cfg: &Config{
				sechan: func() *uasc.Config {
					c := DefaultClientConfig()
					c.SecurityMode = ua.MessageSecurityModeSignAndEncrypt
					return c
				}(),
			},
		},
		{
			name: `SecurityPolicy("Basic128Rsa15")`,
			opt:  SecurityPolicy("Basic128Rsa15"),
			cfg: &Config{
				sechan: func() *uasc.Config {
					c := DefaultClientConfig()
					c.SecurityPolicyURI = ua.SecurityPolicyURIBasic128Rsa15
					return c
				}(),
			},
		},
		{
			name: `SecurityPolicy("Basic256")`,
			opt:  SecurityPolicy("Basic256"),
			cfg: &Config{
				sechan: func() *uasc.Config {
					c := DefaultClientConfig()
					c.SecurityPolicyURI = ua.SecurityPolicyURIBasic256
					return c
				}(),
			},
		},
		{
			name: `SecurityPolicy("Basic256Sha256")`,
			opt:  SecurityPolicy("Basic256Sha256"),
			cfg: &Config{
				sechan: func() *uasc.Config {
					c := DefaultClientConfig()
					c.SecurityPolicyURI = ua.SecurityPolicyURIBasic256Sha256
					return c
				}(),
			},
		},
		{
			name: `SecurityPolicy("Aes128_Sha256_RsaOaep")`,
			opt:  SecurityPolicy("Aes128_Sha256_RsaOaep"),
			cfg: &Config{
				sechan: func() *uasc.Config {
					c := DefaultClientConfig()
					c.SecurityPolicyURI = ua.SecurityPolicyURIAes128Sha256RsaOaep
					return c
				}(),
			},
		},
		{
			name: `SecurityPolicy("Aes256_Sha256_RsaPss")`,
			opt:  SecurityPolicy("Aes256_Sha256_RsaPss"),
			cfg: &Config{
				sechan: func() *uasc.Config {
					c := DefaultClientConfig()
					c.SecurityPolicyURI = ua.SecurityPolicyURIAes256Sha256RsaPss
					return c
				}(),
			},
		},
		{
			name: `SessionName()`,
			opt:  SessionName("a"),
			cfg: &Config{
				session: func() *uasc.SessionConfig {
					sc := DefaultSessionConfig()
					sc.SessionName = "a"
					return sc
				}(),
			},
		},
		{
			name: `SessionTimeout(5s)`,
			opt:  SessionTimeout(5 * time.Second),
			cfg: &Config{
				session: func() *uasc.SessionConfig {
					sc := DefaultSessionConfig()
					sc.SessionTimeout = 5 * time.Second
					return sc
				}(),
			},
		},
		{
			name: `Dialer()`,
			opt: Dialer(&uacp.Dialer{
				Dialer: &net.Dialer{Timeout: 3 * time.Second},
				ClientACK: &uacp.Acknowledge{
					MaxMessageSize: 1,
					MaxChunkCount:  2,
					SendBufSize:    3,
					ReceiveBufSize: 4,
				},
			}),
			cfg: &Config{
				dialer: &uacp.Dialer{
					Dialer: &net.Dialer{Timeout: 3 * time.Second},
					ClientACK: &uacp.Acknowledge{
						MaxMessageSize: 1,
						MaxChunkCount:  2,
						SendBufSize:    3,
						ReceiveBufSize: 4,
					},
				},
			},
		},
		{
			name: `DialTimeout(5s)`,
			opt:  DialTimeout(5 * time.Second),
			cfg: &Config{
				dialer: &uacp.Dialer{
					Dialer:    &net.Dialer{Timeout: 5 * time.Second},
					ClientACK: uacp.DefaultClientACK,
				},
			},
		},
		{
			name: `MaxMessageSize()`,
			opt:  MaxMessageSize(5),
			cfg: &Config{
				dialer: func() *uacp.Dialer {
					d := DefaultDialer()
					d.ClientACK.MaxMessageSize = 5
					return d
				}(),
			},
		},
		{
			name: `MaxChunkCount()`,
			opt:  MaxChunkCount(5),
			cfg: &Config{
				dialer: func() *uacp.Dialer {
					d := DefaultDialer()
					d.ClientACK.MaxChunkCount = 5
					return d
				}(),
			},
		},
		{
			name: `ReceiveBufferSize()`,
			opt:  ReceiveBufferSize(5),
			cfg: &Config{
				dialer: func() *uacp.Dialer {
					d := DefaultDialer()
					d.ClientACK.ReceiveBufSize = 5
					return d
				}(),
			},
		},
		{
			name: `SendBufferSize()`,
			opt:  SendBufferSize(5),
			cfg: &Config{
				dialer: func() *uacp.Dialer {
					d := DefaultDialer()
					d.ClientACK.SendBufSize = 5
					return d
				}(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.cfg.dialer == nil {
				tt.cfg.dialer = DefaultDialer()
			}
			if tt.cfg.sechan == nil {
				tt.cfg.sechan = DefaultClientConfig()
			}
			if tt.cfg.session == nil {
				tt.cfg.session = DefaultSessionConfig()
			}

			errstr := func(err error) string {
				if err != nil {
					return err.Error()
				}
				return ""
			}

			cfg, err := ApplyConfig(tt.opt)
			if got, want := errstr(err), errstr(tt.err); got != "" || want != "" {
				require.Equal(t, want, got, "got error %q want %q", got, want)
				return
			}
			if tt.cfg.stateFunc != nil { // Functions are not comparable, so we check manually and unset
				require.NotNil(t, cfg.stateFunc)
				tt.cfg.stateFunc = nil
				cfg.stateFunc = nil
			} else {
				require.Nil(t, cfg.stateFunc)
			}
			require.Equal(t, tt.cfg, cfg)
		})
	}
}
