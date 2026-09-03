package uasc

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"math"
	"math/big"
	"testing"
	"time"

	"github.com/gopcua/opcua/id"
	"github.com/gopcua/opcua/ua"
	"github.com/gopcua/opcua/uacp"
	"github.com/gopcua/opcua/uapolicy"
	"github.com/stretchr/testify/require"
)

func genThumbprintTestCert(t *testing.T, cn string) ([]byte, *rsa.PrivateKey) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	tmpl := x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: cn},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageDataEncipherment | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, &key.PublicKey, key)
	require.NoError(t, err)
	return der, key
}

// TestServerReadChunkDerivesReceiverThumbprint_Part6_672 verifies that when a
// server SecureChannel receives an asymmetric OpenSecureChannel under a real
// security policy, it derives cfg.Thumbprint from the peer (client) certificate
// in the inbound chunk. That thumbprint is what the server then emits as the
// ReceiverCertificateThumbprint in its OPN response.
//
// OPC UA Part 6 v1.05 §6.7.2: the asymmetric header's ReceiverCertificateThumbprint
// is the SHA1 thumbprint of the receiver's certificate. The receiver of the
// server's OPN response is the client, so the server must derive it from the
// client certificate it received. open62541 and UA-.NETStandard both do this on
// the OPN receive path, and a strict client (Prosys, UaExpert) rejects an empty
// thumbprint with BadCertificateInvalid.
func TestServerReadChunkDerivesReceiverThumbprint_Part6_672(t *testing.T) {
	cliCertDER, cliKey := genThumbprintTestCert(t, "gopcua thumbprint test client")
	srvCertDER, srvKey := genThumbprintTestCert(t, "gopcua thumbprint test server")
	srvCert, err := x509.ParseCertificate(srvCertDER)
	require.NoError(t, err)
	srvPub := srvCert.PublicKey.(*rsa.PublicKey)

	serverTCP, clientTCP := newTestTCPConnPair(t)
	defer serverTCP.Close()
	defer clientTCP.Close()

	srvConn, err := uacp.NewConn(serverTCP, &uacp.Acknowledge{
		ReceiveBufSize: 65535,
		SendBufSize:    65535,
		MaxMessageSize: 2 * 1024 * 1024,
		MaxChunkCount:  512,
	})
	require.NoError(t, err)

	cfg := &Config{
		SecurityPolicyURI: ua.SecurityPolicyURIBasic256Sha256,
		SecurityMode:      ua.MessageSecurityModeSignAndEncrypt,
		LocalKey:          srvKey,
		Certificate:       srvCertDER,
	}
	s, err := NewServerSecureChannel("opc.tcp://localhost", srvConn, cfg, make(chan error, 1), 1, 1, 1)
	require.NoError(t, err)

	// Build a plaintext asymmetric OPN chunk that carries the client certificate
	// as the SenderCertificate, the way a client opens an encrypted channel.
	m := &Message{
		MessageHeader: &MessageHeader{
			Header:                   NewHeader(MessageTypeOpenSecureChannel, ChunkTypeFinal, 0),
			AsymmetricSecurityHeader: NewAsymmetricSecurityHeader(ua.SecurityPolicyURIBasic256Sha256, cliCertDER, uapolicy.Thumbprint(srvCertDER)),
			SequenceHeader:           NewSequenceHeader(1, 1),
		},
		TypeID: ua.NewFourByteExpandedNodeID(0, id.OpenSecureChannelRequest_Encoding_DefaultBinary),
		Service: &ua.OpenSecureChannelRequest{
			RequestHeader: &ua.RequestHeader{
				AuthenticationToken: ua.NewTwoByteNodeID(0),
				Timestamp:           time.Now(),
				AdditionalHeader:    ua.NewExtensionObject(nil),
			},
			RequestType:       ua.SecurityTokenRequestTypeIssue,
			SecurityMode:      ua.MessageSecurityModeSignAndEncrypt,
			ClientNonce:       []byte{},
			RequestedLifetime: 3600000,
		},
	}
	chunks, err := m.EncodeChunks(math.MaxUint32)
	require.NoError(t, err)
	require.Len(t, chunks, 1)

	// Sign with the client key and encrypt to the server key, the way a client
	// emits an asymmetric OPN, so the server's readChunk can decrypt it.
	cliAlgo, err := uapolicy.Asymmetric(ua.SecurityPolicyURIBasic256Sha256, cliKey, srvPub)
	require.NoError(t, err)
	enc := &channelInstance{
		sc:   &SecureChannel{cfg: &Config{SecurityMode: ua.MessageSecurityModeSignAndEncrypt, SecurityPolicyURI: ua.SecurityPolicyURIBasic256Sha256}},
		algo: cliAlgo,
	}
	cipher, err := enc.signAndEncrypt(m, chunks[0])
	require.NoError(t, err)

	go func() { _, _ = clientTCP.Write(cipher) }()

	_, err = s.readChunk()
	require.NoError(t, err)

	require.NotEmpty(t, s.cfg.Thumbprint, "server must derive the receiver thumbprint from the inbound OPN")
	require.Equal(t, uapolicy.Thumbprint(cliCertDER), s.cfg.Thumbprint, "thumbprint must be SHA1 of the received client certificate")
}
