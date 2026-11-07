package server

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/gopcua/opcua/ua"
)

func genAuthTestCert(t *testing.T, cn string, notBefore, notAfter time.Time) *x509.Certificate {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	tmpl := x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: cn},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	der, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, &key.PublicKey, key)
	require.NoError(t, err)
	cert, err := x509.ParseCertificate(der)
	require.NoError(t, err)
	return cert
}

func TestStaticUsers(t *testing.T) {
	auth := StaticUsers(map[string]string{"alice": "s3cret", "bob": ""})

	require.NoError(t, auth("alice", "s3cret"))
	require.NoError(t, auth("bob", ""))
	require.ErrorIs(t, auth("alice", "wrong"), ErrAccessDenied)
	require.ErrorIs(t, auth("alice", ""), ErrAccessDenied)
	require.ErrorIs(t, auth("mallory", "s3cret"), ErrAccessDenied)
	require.ErrorIs(t, auth("mallory", "\x00invalid"), ErrAccessDenied)
	require.ErrorIs(t, auth("", ""), ErrAccessDenied)
}

func TestStaticUsersCopiesMap(t *testing.T) {
	m := map[string]string{"alice": "s3cret"}
	auth := StaticUsers(m)
	m["alice"] = "changed"
	require.NoError(t, auth("alice", "s3cret"))
}

func TestTrustedUserCerts(t *testing.T) {
	now := time.Now()
	trusted := genAuthTestCert(t, "trusted", now.Add(-time.Hour), now.Add(time.Hour))
	untrusted := genAuthTestCert(t, "untrusted", now.Add(-time.Hour), now.Add(time.Hour))
	expired := genAuthTestCert(t, "expired", now.Add(-2*time.Hour), now.Add(-time.Hour))

	pool := x509.NewCertPool()
	pool.AddCert(trusted)
	pool.AddCert(expired)
	auth := TrustedUserCerts(pool)

	require.NoError(t, auth(trusted))
	require.ErrorIs(t, auth(untrusted), ErrAccessDenied)
	require.ErrorIs(t, auth(expired), ErrAccessDenied)
	require.ErrorIs(t, auth(nil), ErrAccessDenied)
}

func TestAuthOptionsEnableModes(t *testing.T) {
	s := New(
		UserNameAuth(StaticUsers(nil)),
		X509Auth(TrustedUserCerts(x509.NewCertPool())),
		EnableAuthMode(ua.UserTokenTypeUserName), // duplicate is ignored
	)
	var modes []ua.UserTokenType
	for _, a := range s.cfg.enabledAuth {
		modes = append(modes, a.tokenType)
	}
	require.Equal(t, []ua.UserTokenType{ua.UserTokenTypeUserName, ua.UserTokenTypeCertificate}, modes)
	require.NotNil(t, s.cfg.userNameAuth)
	require.NotNil(t, s.cfg.x509Auth)
}

func TestIdentityName(t *testing.T) {
	var nilID *Identity
	require.Equal(t, "", nilID.Name())
	require.Equal(t, "anonymous", (&Identity{TokenType: ua.UserTokenTypeAnonymous}).Name())
	require.Equal(t, "alice", (&Identity{TokenType: ua.UserTokenTypeUserName, UserName: "alice"}).Name())
	cert := genAuthTestCert(t, "user1", time.Now().Add(-time.Hour), time.Now().Add(time.Hour))
	require.Equal(t, "CN=user1", (&Identity{TokenType: ua.UserTokenTypeCertificate, Certificate: cert}).Name())
}
