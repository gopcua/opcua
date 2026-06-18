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

// loadLeafDER returns the DER bytes of only the first PEM block, i.e. a single
// certificate with no chain appended.
func loadLeafDER(t *testing.T, path string) []byte {
	t.Helper()

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	block, _ := pem.Decode(data)
	require.NotNil(t, block, "no PEM block found in %s", path)
	return block.Bytes
}

// TestParseCertificate_SingleCert pins the backward-compatibility invariant: for
// a single DER certificate (the dominant production input), ParseCertificate
// returns the same certificate the old x509.ParseCertificate did, and Thumbprint
// hashes the same bytes the old raw-input hash did.
func TestParseCertificate_SingleCert(t *testing.T) {
	leaf := loadLeafDER(t, "testdata/certs/cert_with_chain.pem")

	cert, err := ParseCertificate(leaf)
	require.NoError(t, err)
	require.NotNil(t, cert)
	require.Equal(t, leaf, cert.Raw, "single-cert input must round-trip to the same DER")
	require.False(t, cert.IsCA, "leaf is the same whether parsed alone or from a chain")

	// For a single well-formed cert, the leaf-only thumbprint equals hashing the
	// raw input — the exact behavior of the pre-change Thumbprint.
	rawHash := sha1.Sum(leaf)
	require.Equal(t, rawHash[:], Thumbprint(leaf))

	// And the leaf selected from the full chain is identical to this single cert.
	chainLeaf, err := ParseCertificate(loadDERChain(t, "testdata/certs/cert_with_chain.pem"))
	require.NoError(t, err)
	require.Equal(t, cert.Raw, chainLeaf.Raw, "leaf must be the same from a single cert and from the chain")
}

// TestParseCertificate_Malformed covers the un-parseable input path: ParseCertificate
// surfaces the error, and Thumbprint falls back to hashing the raw bytes.
func TestParseCertificate_Malformed(t *testing.T) {
	garbage := []byte("this is not a DER certificate")

	_, err := ParseCertificate(garbage)
	require.Error(t, err)

	rawHash := sha1.Sum(garbage)
	require.Equal(t, rawHash[:], Thumbprint(garbage), "malformed input falls back to a raw-byte thumbprint")
}

// TestParseCertificate_Empty covers empty input: x509.ParseCertificates returns no
// certificates without an error, so ParseCertificate must surface its own sentinel.
func TestParseCertificate_Empty(t *testing.T) {
	_, err := ParseCertificate(nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "no certificates found")
}
