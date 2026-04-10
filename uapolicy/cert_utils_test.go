package uapolicy

import (
	"crypto/sha1"
	"encoding/pem"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

// loadDERChain reads a PEM file and returns the concatenated DER bytes
// of all certificates, simulating what an OPC-UA server sends when it
// returns a certificate chain.
func loadDERChain(t *testing.T, path string) []byte {
	t.Helper()

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var chain []byte
	for {
		block, rest := pem.Decode(data)
		if block == nil {
			break
		}
		chain = append(chain, block.Bytes...)
		data = rest
	}
	require.NotEmpty(t, chain, "no PEM blocks found in %s", path)
	return chain
}

func TestParseCertificate_Chain(t *testing.T) {
	chain := loadDERChain(t, "testdata/certs/cert_with_chain.pem")

	cert, err := ParseCertificate(chain)
	require.NoError(t, err)
	require.NotNil(t, cert)
	require.Contains(t, cert.Subject.CommonName, "OPCUA Server")
	require.False(t, cert.IsCA, "expected leaf certificate, not CA")
}

func TestThumbprint_Chain(t *testing.T) {
	chain := loadDERChain(t, "testdata/certs/cert_with_chain.pem")

	// Thumbprint of the chain should equal the thumbprint of just the leaf cert.
	cert, err := ParseCertificate(chain)
	require.NoError(t, err)

	expected := sha1.Sum(cert.Raw)
	got := Thumbprint(chain)
	require.Equal(t, expected[:], got, "thumbprint should match the leaf certificate only")

	// It should NOT equal a hash of the entire chain bytes.
	fullHash := sha1.Sum(chain)
	require.NotEqual(t, fullHash[:], got, "thumbprint must not hash the entire chain")
}

func TestPublicKey_Chain(t *testing.T) {
	chain := loadDERChain(t, "testdata/certs/cert_with_chain.pem")

	key, err := PublicKey(chain)
	require.NoError(t, err)
	require.NotNil(t, key)
	require.Equal(t, 2048, key.Size()*8, "expected 2048-bit RSA key")
}
