package uasc

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/gopcua/opcua/id"
	uatest "github.com/gopcua/opcua/tests/python"
	"github.com/gopcua/opcua/ua"
	"github.com/gopcua/opcua/uacp"
	"github.com/gopcua/opcua/uapolicy"
	"github.com/stretchr/testify/require"
)

func TestNewRequestMessage(t *testing.T) {
	fixedTime := func() time.Time { return time.Date(2019, 1, 1, 12, 13, 14, 0, time.UTC) }

	buildSecureChannel := func(sc *SecureChannel, instance *channelInstance) *SecureChannel {
		if instance == nil {
			instance = newChannelInstance(sc)
		}
		sc.activeInstance = instance
		sc.activeInstance.sc = sc
		return sc
	}

	tests := []struct {
		name      string
		sechan    *SecureChannel
		req       ua.Request
		authToken *ua.NodeID
		timeout   time.Duration
		m         *Message
	}{
		{
			name: "first-request",
			sechan: buildSecureChannel(&SecureChannel{
				cfg: &Config{},
				// reqhdr: &ua.RequestHeader{},
				time: fixedTime,
			}, nil),
			req: &ua.ReadRequest{},
			m: &Message{
				MessageHeader: &MessageHeader{
					Header: &Header{
						MessageType: MessageTypeMessage,
						ChunkType:   ChunkTypeFinal,
					},
					SymmetricSecurityHeader: &SymmetricSecurityHeader{},
					SequenceHeader: &SequenceHeader{
						SequenceNumber: 1,
						RequestID:      1,
					},
				},
				TypeID: ua.NewFourByteExpandedNodeID(0, id.ReadRequest_Encoding_DefaultBinary),
				Service: &ua.ReadRequest{
					RequestHeader: &ua.RequestHeader{
						AuthenticationToken: ua.NewTwoByteNodeID(0),
						Timestamp:           fixedTime(),
						RequestHandle:       1,
					},
				},
			},
		},
		{
			name: "subsequent-request",
			sechan: buildSecureChannel(
				&SecureChannel{
					cfg:       &Config{},
					requestID: 555,
					// reqhdr: &ua.RequestHeader{
					// 	RequestHandle: 444,
					// },
					time: fixedTime,
				},
				&channelInstance{
					sequenceNumber: 777,
				},
			),
			req: &ua.ReadRequest{},
			m: &Message{
				MessageHeader: &MessageHeader{
					Header: &Header{
						MessageType: MessageTypeMessage,
						ChunkType:   ChunkTypeFinal,
					},
					SymmetricSecurityHeader: &SymmetricSecurityHeader{},
					SequenceHeader: &SequenceHeader{
						SequenceNumber: 778,
						RequestID:      556,
					},
				},
				TypeID: ua.NewFourByteExpandedNodeID(0, id.ReadRequest_Encoding_DefaultBinary),
				Service: &ua.ReadRequest{
					RequestHeader: &ua.RequestHeader{
						AuthenticationToken: ua.NewTwoByteNodeID(0),
						Timestamp:           fixedTime(),
						RequestHandle:       556,
					},
				},
			},
		},
		{
			name: "counter-rollover",
			sechan: buildSecureChannel(
				&SecureChannel{
					cfg:       &Config{},
					requestID: math.MaxUint32,
					time:      fixedTime,
				},
				&channelInstance{
					sequenceNumber: math.MaxUint32 - 1023,
				}),
			req: &ua.ReadRequest{},
			m: &Message{
				MessageHeader: &MessageHeader{
					Header: &Header{
						MessageType: MessageTypeMessage,
						ChunkType:   ChunkTypeFinal,
					},
					SymmetricSecurityHeader: &SymmetricSecurityHeader{},
					SequenceHeader: &SequenceHeader{
						SequenceNumber: 1,
						RequestID:      1,
					},
				},
				TypeID: ua.NewFourByteExpandedNodeID(0, id.ReadRequest_Encoding_DefaultBinary),
				Service: &ua.ReadRequest{
					RequestHeader: &ua.RequestHeader{
						AuthenticationToken: ua.NewTwoByteNodeID(0),
						Timestamp:           fixedTime(),
						RequestHandle:       1,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, err := tt.sechan.activeInstance.newRequestMessage(tt.req, tt.sechan.nextRequestID(), tt.authToken, tt.timeout)
			require.NoError(t, err)
			require.Equal(t, tt.m, m)
		})
	}
}

func TestSignAndEncryptVerifyAndDecrypt(t *testing.T) {
	buildSecPolicy := func(bits int, uri string) *uapolicy.EncryptionAlgorithm {
		t.Helper()

		certPEM, keyPEM, err := uatest.GenerateCert("localhost", bits, 24*time.Hour)
		require.NoError(t, err)

		block, _ := pem.Decode(keyPEM)
		pk, err := x509.ParsePKCS1PrivateKey(block.Bytes)
		require.NoError(t, err)

		certblock, _ := pem.Decode(certPEM)
		remoteX509Cert, err := x509.ParseCertificate(certblock.Bytes)
		require.NoError(t, err)

		remoteKey := remoteX509Cert.PublicKey.(*rsa.PublicKey)
		alg, _ := uapolicy.Asymmetric(uri, pk, remoteKey)
		return alg
	}

	getConfig := func(uri string) *Config {
		t.Helper()

		if uri == ua.SecurityPolicyURINone {
			return &Config{SecurityMode: ua.MessageSecurityModeNone}
		}
		return &Config{SecurityMode: ua.MessageSecurityModeSignAndEncrypt}
	}

	tests := []struct {
		name string
		c    *channelInstance
		m    *Message
		b    []byte
	}{}

	for _, uri := range ua.SecurityPolicyURIs {
		for i, keyLength := range []int{2048, 4096} {
			if i == 1 && (uri == ua.SecurityPolicyURIBasic128Rsa15 || uri == ua.SecurityPolicyURIBasic256) {
				continue
			}
			tests = append(tests, struct {
				name string
				c    *channelInstance
				m    *Message
				b    []byte
			}{fmt.Sprintf("encrypt/decrypt: bits: %d uri: %s", keyLength, uri),
				&channelInstance{
					sc:   &SecureChannel{cfg: getConfig(uri)},
					algo: buildSecPolicy(keyLength, uri),
				},
				&Message{
					MessageHeader: &MessageHeader{
						Header: &Header{
							MessageType: MessageTypeOpenSecureChannel,
							ChunkType:   ChunkTypeFinal,
						},
						AsymmetricSecurityHeader: &AsymmetricSecurityHeader{
							SecurityPolicyURI: "http://gopcua.example/OPCUA/SecurityPolicy#Foo",
						},
						SequenceHeader: &SequenceHeader{
							SequenceNumber: 1,
							RequestID:      1,
						},
					},
				},
				[]byte{ // OpenSecureChannelRequest
					// Message Header
					// MessageType: OPN
					0x4f, 0x50, 0x4e,
					// Chunk Type: Final
					0x46,
					// MessageSize: 131
					0x8E, 0x00, 0x00, 0x00,
					// SecureChannelID: 0
					0x00, 0x00, 0x00, 0x00,
					// AsymmetricSecurityHeader
					// SecurityPolicyURILength
					0x2e, 0x00, 0x00, 0x00,
					// SecurityPolicyURI
					0x68, 0x74, 0x74, 0x70, 0x3a, 0x2f, 0x2f, 0x67,
					0x6f, 0x70, 0x63, 0x75, 0x61, 0x2e, 0x65, 0x78,
					0x61, 0x6d, 0x70, 0x6c, 0x65, 0x2f, 0x4f, 0x50,
					0x43, 0x55, 0x41, 0x2f, 0x53, 0x65, 0x63, 0x75,
					0x72, 0x69, 0x74, 0x79, 0x50, 0x6f, 0x6c, 0x69,
					0x63, 0x79, 0x23, 0x46, 0x6f, 0x6f,
					// SenderCertificate
					0xff, 0xff, 0xff, 0xff,
					// ReceiverCertificateThumbprint
					0xff, 0xff, 0xff, 0xff,
					// Sequence Header
					// SequenceNumber
					0x01, 0x00, 0x00, 0x00,
					// RequestID
					0x01, 0x00, 0x00, 0x00,
					// TypeID
					0x01, 0x00, 0xbe, 0x01,

					// RequestHeader
					// - AuthenticationToken
					0x00, 0x00,
					// - Timestamp
					0x00, 0x98, 0x67, 0xdd, 0xfd, 0x30, 0xd4, 0x01,
					// - RequestHandle
					0x01, 0x00, 0x00, 0x00,
					// - ReturnDiagnostics
					0xff, 0x03, 0x00, 0x00,
					// - AuditEntry
					0xff, 0xff, 0xff, 0xff,
					// - TimeoutHint
					0x00, 0x00, 0x00, 0x00,
					// - AdditionalHeader
					//   - TypeID
					0x00, 0x00,
					//   - EncodingMask
					0x00,
					// ClientProtocolVersion
					0x00, 0x00, 0x00, 0x00,
					// SecurityTokenRequestType
					0x00, 0x00, 0x00, 0x00,
					// MessageSecurityMode
					0x01, 0x00, 0x00, 0x00,
					// ClientNonce
					0xff, 0xff, 0xff, 0xff,
					// RequestedLifetime
					0x80, 0x8d, 0x5b, 0x00,
				}})
		}
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cipher, err := tt.c.signAndEncrypt(tt.m, tt.b)
			require.NoError(t, err, "error: message encrypt")

			m := new(MessageChunk)
			_, err = m.Decode(cipher)
			require.NoError(t, err, "error: message decode")

			plain, err := tt.c.verifyAndDecrypt(m, cipher)
			require.NoError(t, err, "error: message decrypt")

			headerLength := 12 + m.AsymmetricSecurityHeader.Len()
			require.Equal(t, tt.b[headerLength:], plain, "header not equal")
		})
	}
}

func TestNewSecureChannel(t *testing.T) {
	t.Run("no connection", func(t *testing.T) {
		_, err := NewSecureChannel("", nil, nil, nil)
		require.ErrorContains(t, err, "no connection")
	})
	t.Run("no error channel", func(t *testing.T) {
		_, err := NewSecureChannel("", &uacp.Conn{}, nil, nil)
		require.ErrorContains(t, err, "no secure channel config")
	})
	t.Run("no config", func(t *testing.T) {
		_, err := NewSecureChannel("", &uacp.Conn{}, nil, make(chan error))
		require.ErrorContains(t, err, "no secure channel config")
	})
	t.Run("uri none, mode not none", func(t *testing.T) {
		cfg := &Config{SecurityPolicyURI: ua.SecurityPolicyURINone, SecurityMode: ua.MessageSecurityModeSign}
		_, err := NewSecureChannel("", &uacp.Conn{}, cfg, make(chan error))
		require.ErrorContains(t, err, "invalid channel config: Security policy 'http://opcfoundation.org/UA/SecurityPolicy#None' cannot be used with 'MessageSecurityModeSign'")
	})
	t.Run("uri not none, mode none", func(t *testing.T) {
		cfg := &Config{SecurityPolicyURI: ua.SecurityPolicyURIBasic256, SecurityMode: ua.MessageSecurityModeNone}
		_, err := NewSecureChannel("", &uacp.Conn{}, cfg, make(chan error))
		require.ErrorContains(t, err, "invalid channel config: Security policy 'http://opcfoundation.org/UA/SecurityPolicy#Basic256' can only be used with 'MessageSecurityModeSign' or 'MessageSecurityModeSignAndEncrypt'")
	})
	t.Run("uri not none, security policy not none, mode invalid", func(t *testing.T) {
		cfg := &Config{SecurityPolicyURI: ua.SecurityPolicyURIBasic256, SecurityMode: ua.MessageSecurityModeInvalid}
		_, err := NewSecureChannel("", &uacp.Conn{}, cfg, make(chan error))
		require.ErrorContains(t, err, "invalid channel config: Security policy 'http://opcfoundation.org/UA/SecurityPolicy#Basic256' can only be used with 'MessageSecurityModeSign' or 'MessageSecurityModeSignAndEncrypt'")
	})
	t.Run("uri not none, local key missing", func(t *testing.T) {
		cfg := &Config{SecurityPolicyURI: ua.SecurityPolicyURIBasic256, SecurityMode: ua.MessageSecurityModeSign}
		_, err := NewSecureChannel("", &uacp.Conn{}, cfg, make(chan error))
		require.ErrorContains(t, err, "invalid channel config: Security policy 'http://opcfoundation.org/UA/SecurityPolicy#Basic256' requires a private key")
	})
}
