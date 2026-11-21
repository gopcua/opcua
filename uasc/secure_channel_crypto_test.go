package uasc

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/binary"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/gopcua/opcua/ua"
	"github.com/gopcua/opcua/uapolicy"
)

var rsaUserTokenPolicies = []string{
	ua.SecurityPolicyURIBasic128Rsa15,
	ua.SecurityPolicyURIBasic256,
	ua.SecurityPolicyURIBasic256Sha256,
	ua.SecurityPolicyURIAes128Sha256RsaOaep,
	ua.SecurityPolicyURIAes256Sha256RsaPss,
}

func genUserTokenTestCert(t *testing.T, cn string) ([]byte, *rsa.PrivateKey) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	tmpl := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: cn},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment | x509.KeyUsageDataEncipherment,
	}
	der, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, &key.PublicKey, key)
	require.NoError(t, err)
	return der, key
}

func TestDecryptUserPasswordRoundtrip(t *testing.T) {
	serverCert, serverKey := genUserTokenTestCert(t, "server")
	_, clientKey := genUserTokenTestCert(t, "client")

	nonce := make([]byte, 32)
	_, err := rand.Read(nonce)
	require.NoError(t, err)

	for _, policy := range rsaUserTokenPolicies {
		t.Run(policy, func(t *testing.T) {
			client := &SecureChannel{cfg: &Config{SecurityPolicyURI: policy, LocalKey: clientKey}}
			server := &SecureChannel{cfg: &Config{SecurityPolicyURI: policy, LocalKey: serverKey}}

			secret, alg, err := client.EncryptUserPassword(policy, "s3cret-pässwörd", serverCert, nonce)
			require.NoError(t, err)

			pass, err := server.DecryptUserPassword(policy, alg, secret, nonce)
			require.NoError(t, err)
			require.Equal(t, "s3cret-pässwörd", pass)

			// empty policy falls back to the channel policy
			pass, err = server.DecryptUserPassword("", alg, secret, nonce)
			require.NoError(t, err)
			require.Equal(t, "s3cret-pässwörd", pass)

			// empty password is valid
			secret2, alg2, err := client.EncryptUserPassword(policy, "", serverCert, nonce)
			require.NoError(t, err)
			pass, err = server.DecryptUserPassword(policy, alg2, secret2, nonce)
			require.NoError(t, err)
			require.Equal(t, "", pass)

			t.Run("wrong nonce", func(t *testing.T) {
				other := append([]byte{}, nonce...)
				other[0] ^= 0xff
				_, err := server.DecryptUserPassword(policy, alg, secret, other)
				require.Error(t, err)
			})
			t.Run("wrong algorithm", func(t *testing.T) {
				_, err := server.DecryptUserPassword(policy, "http://example.com/bogus", secret, nonce)
				require.Error(t, err)
			})
			t.Run("tampered ciphertext", func(t *testing.T) {
				bad := append([]byte{}, secret...)
				bad[len(bad)/2] ^= 0xff
				_, err := server.DecryptUserPassword(policy, alg, bad, nonce)
				require.Error(t, err)
			})
			t.Run("wrong server key", func(t *testing.T) {
				_, otherKey := genUserTokenTestCert(t, "other")
				other := &SecureChannel{cfg: &Config{SecurityPolicyURI: policy, LocalKey: otherKey}}
				_, err := other.DecryptUserPassword(policy, alg, secret, nonce)
				require.Error(t, err)
			})
		})
	}
}

func TestDecryptUserPasswordBadLength(t *testing.T) {
	serverCert, serverKey := genUserTokenTestCert(t, "server")
	policy := ua.SecurityPolicyURIBasic256Sha256
	server := &SecureChannel{cfg: &Config{SecurityPolicyURI: policy, LocalKey: serverKey}}
	nonce := []byte("0123456789abcdef0123456789abcdef")

	cert, err := uapolicy.ParseCertificate(serverCert)
	require.NoError(t, err)
	enc, err := uapolicy.Asymmetric(policy, nil, cert.PublicKey.(*rsa.PublicKey))
	require.NoError(t, err)

	build := func(l uint32, body []byte) []byte {
		b := make([]byte, 4)
		binary.LittleEndian.PutUint32(b, l)
		ct, err := enc.Encrypt(append(b, body...))
		require.NoError(t, err)
		return ct
	}
	body := append([]byte("pw"), nonce...)

	for name, secret := range map[string][]byte{
		"length too large":   build(uint32(len(body)+1), body),
		"length too small":   build(uint32(len(body)-1), body),
		"huge length":        build(0xffffffff, body),
		"shorter than nonce": build(4, []byte("abcd")),
	} {
		t.Run(name, func(t *testing.T) {
			_, err := server.DecryptUserPassword(policy, enc.EncryptionURI(), secret, nonce)
			require.Error(t, err)
		})
	}

	t.Run("policy none", func(t *testing.T) {
		_, err := server.DecryptUserPassword(ua.SecurityPolicyURINone, "", []byte("pw"), nonce)
		require.Error(t, err)
	})
	t.Run("no local key", func(t *testing.T) {
		nokey := &SecureChannel{cfg: &Config{SecurityPolicyURI: policy}}
		_, err := nokey.DecryptUserPassword(policy, enc.EncryptionURI(), build(uint32(len(body)), body), nonce)
		require.Error(t, err)
	})
}

func TestVerifyUserTokenSignatureRoundtrip(t *testing.T) {
	serverCert, serverKey := genUserTokenTestCert(t, "server")
	userCert, userKey := genUserTokenTestCert(t, "user")
	otherCert, _ := genUserTokenTestCert(t, "other user")

	nonce := make([]byte, 32)
	_, err := rand.Read(nonce)
	require.NoError(t, err)

	for _, policy := range rsaUserTokenPolicies {
		t.Run(policy, func(t *testing.T) {
			client := &SecureChannel{cfg: &Config{SecurityPolicyURI: policy, UserKey: userKey}}
			server := &SecureChannel{cfg: &Config{SecurityPolicyURI: policy, LocalKey: serverKey}}

			s, alg, err := client.NewUserTokenSignature(policy, serverCert, nonce)
			require.NoError(t, err)
			sig := &ua.SignatureData{Signature: s, Algorithm: alg}

			require.NoError(t, server.VerifyUserTokenSignature(policy, userCert, serverCert, nonce, sig))
			require.NoError(t, server.VerifyUserTokenSignature("", userCert, serverCert, nonce, sig))

			t.Run("wrong nonce", func(t *testing.T) {
				other := append([]byte{}, nonce...)
				other[0] ^= 0xff
				require.Error(t, server.VerifyUserTokenSignature(policy, userCert, serverCert, other, sig))
			})
			t.Run("other user cert", func(t *testing.T) {
				require.Error(t, server.VerifyUserTokenSignature(policy, otherCert, serverCert, nonce, sig))
			})
			t.Run("tampered signature", func(t *testing.T) {
				bad := &ua.SignatureData{Signature: append([]byte{}, s...), Algorithm: alg}
				bad.Signature[0] ^= 0xff
				require.Error(t, server.VerifyUserTokenSignature(policy, userCert, serverCert, nonce, bad))
			})
			t.Run("wrong algorithm", func(t *testing.T) {
				bad := &ua.SignatureData{Signature: s, Algorithm: "http://example.com/bogus"}
				require.Error(t, server.VerifyUserTokenSignature(policy, userCert, serverCert, nonce, bad))
			})
			t.Run("missing signature", func(t *testing.T) {
				require.Error(t, server.VerifyUserTokenSignature(policy, userCert, serverCert, nonce, nil))
				require.Error(t, server.VerifyUserTokenSignature(policy, userCert, serverCert, nonce, &ua.SignatureData{Algorithm: alg}))
			})
			t.Run("garbage cert", func(t *testing.T) {
				require.Error(t, server.VerifyUserTokenSignature(policy, []byte("nope"), serverCert, nonce, sig))
			})
		})
	}

	t.Run("policy none", func(t *testing.T) {
		server := &SecureChannel{cfg: &Config{SecurityPolicyURI: ua.SecurityPolicyURINone}}
		require.Error(t, server.VerifyUserTokenSignature("", userCert, serverCert, nonce, &ua.SignatureData{Signature: []byte{1}}))
	})
}
