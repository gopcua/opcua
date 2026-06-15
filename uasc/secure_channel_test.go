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
			return &Config{SecurityMode: ua.MessageSecurityModeNone, SecurityPolicyURI: ua.SecurityPolicyURINone}
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
					// MessageSize: 142
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

// TestVerifyAndDecrypt_Part6_674_AsymmetricOPNDecryptedInNoneWindow pins the
// derived invariant: an OpenSecureChannel message always uses the asymmetric
// crypto path regardless of the channel's currently-negotiated SecurityMode.
// open62541 encodes the same rule via the messageType == OPN disjunct in its
// receive-side decrypt decision (src/ua_securechannel_crypto.c:440), which
// forces OPN decryption even while the channel mode still reads None.
//
// OPC UA Part 6 v1.05 §6.7.4: "The OpenSecureChannel Messages are signed and
// encrypted if the SecurityMode is not None (even if the SecurityMode is Sign)."
//
// When verifyAndDecrypt is handed SecurityMode==None together with a real
// SecurityPolicyURI and an asymmetric header, it must DECRYPT the OPN chunk, not
// return it raw. readChunk (secure_channel.go:500) promotes SecurityMode to
// SignAndEncrypt before calling verifyAndDecrypt, so this exact state is not
// reached today; the guard defends the invariant against a future readChunk
// change that has not yet promoted the mode. Without the guard verifyAndDecrypt
// returns m.Data raw whenever SecurityMode==None, regardless of the asymmetric
// header or policy.
func TestVerifyAndDecrypt_Part6_674_AsymmetricOPNDecryptedInNoneWindow(t *testing.T) {
	const uri = ua.SecurityPolicyURIBasic256Sha256

	certPEM, keyPEM, err := uatest.GenerateCert("localhost", 2048, 24*time.Hour)
	require.NoError(t, err)

	block, _ := pem.Decode(keyPEM)
	pk, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	require.NoError(t, err)

	certblock, _ := pem.Decode(certPEM)
	remoteX509Cert, err := x509.ParseCertificate(certblock.Bytes)
	require.NoError(t, err)

	remoteKey := remoteX509Cert.PublicKey.(*rsa.PublicKey)
	algo, err := uapolicy.Asymmetric(uri, pk, remoteKey)
	require.NoError(t, err)

	m := &Message{
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
	}

	plaintext := []byte{ // OpenSecureChannelRequest
		// Message Header
		// MessageType: OPN
		0x4f, 0x50, 0x4e,
		// Chunk Type: Final
		0x46,
		// MessageSize: 142
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
	}

	headerLength := 12 + m.AsymmetricSecurityHeader.Len()
	expectedBody := append([]byte(nil), plaintext[headerLength:]...)

	// Produce a real signed+encrypted asymmetric OPN chunk, the way the peer
	// sent it on the wire.
	enc := &channelInstance{
		sc:   &SecureChannel{cfg: &Config{SecurityMode: ua.MessageSecurityModeSignAndEncrypt, SecurityPolicyURI: uri}},
		algo: algo,
	}
	cipher, err := enc.signAndEncrypt(m, plaintext)
	require.NoError(t, err, "error: message encrypt")

	chunk := new(MessageChunk)
	_, err = chunk.Decode(cipher)
	require.NoError(t, err, "error: message decode")

	// SecurityMode==None with a real SecurityPolicyURI and an asymmetric
	// header. The asymmetric OPN must still be decrypted, not returned raw.
	dec := &channelInstance{
		sc:   &SecureChannel{cfg: &Config{SecurityMode: ua.MessageSecurityModeNone, SecurityPolicyURI: uri}},
		algo: algo,
	}

	plain, err := dec.verifyAndDecrypt(chunk, cipher)
	require.NoError(t, err, "error: message decrypt")

	require.Equal(t, expectedBody, plain,
		"asymmetric OPN under a real policy must be decrypted in the None window, not returned raw")
}

// TestVerifyAndDecrypt_Part6_674_NonePolicyAsymmetricReturnsRaw asserts that
// when verifyAndDecrypt is handed SecurityMode==None together with the explicit
// #None SecurityPolicyURI and an asymmetric header, it returns the chunk's raw
// Data unchanged rather than attempting decryption.
//
// OPC UA Part 6 v1.05 §6.7.4: "The OpenSecureChannel Messages are not signed or
// encrypted if the SecurityMode is None."
//
// Regression guard: without the SecurityPolicyURI == ua.SecurityPolicyURINone
// clause, a #None-policy asymmetric chunk would reach c.algo.Decrypt with a nil
// algo and panic the receive loop. The clause is what forces the raw-return path
// here; TestVerifyAndDecrypt_Part6_674_AsymmetricOPNDecryptedInNoneWindow cannot,
// since on a real policy that clause is false.
func TestVerifyAndDecrypt_Part6_674_NonePolicyAsymmetricReturnsRaw(t *testing.T) {
	chunk := &MessageChunk{
		MessageHeader: &MessageHeader{
			Header: &Header{
				MessageType: MessageTypeOpenSecureChannel,
				ChunkType:   ChunkTypeFinal,
			},
			AsymmetricSecurityHeader: &AsymmetricSecurityHeader{
				SecurityPolicyURI: ua.SecurityPolicyURINone,
			},
			SequenceHeader: &SequenceHeader{
				SequenceNumber: 1,
				RequestID:      1,
			},
		},
		Data: []byte{0xde, 0xad, 0xbe, 0xef},
	}

	// SecurityMode==None with the explicit #None policy and an asymmetric
	// header. The chunk must be returned raw, not handed to the crypto path.
	dec := &channelInstance{
		sc: &SecureChannel{cfg: &Config{
			SecurityMode:      ua.MessageSecurityModeNone,
			SecurityPolicyURI: ua.SecurityPolicyURINone,
		}},
	}

	plain, err := dec.verifyAndDecrypt(chunk, nil)
	require.NoError(t, err, "error: message decrypt")

	require.Equal(t, chunk.Data, plain,
		"None-policy asymmetric chunk must be returned raw, not decrypted")
}

// TestVerifyAndDecrypt_Part6_674_NoneModeSymmetricReturnsRaw pins the realistic
// unsecured steady state: a None-mode, #None-policy symmetric message returns
// raw. It does not pin the !isAsymmetric disjunct, because the #None policy
// satisfies the guard's first disjunct and short-circuits before !isAsymmetric
// is evaluated. TestVerifyAndDecrypt_Part6_674_RealPolicyNoneModeSymmetricReturnsRaw
// is the test that forces !isAsymmetric.
//
// OPC UA Part 6 v1.05 §6.7.4 secures the OpenSecureChannel handshake under a
// real policy, but a None-mode channel's steady-state symmetric messages carry
// no crypto. A symmetric chunk on such a channel must return m.Data raw and
// never reach c.algo.Decrypt; here algo is nil, so the guard must return before
// any algo dereference.
func TestVerifyAndDecrypt_Part6_674_NoneModeSymmetricReturnsRaw(t *testing.T) {
	chunk := &MessageChunk{
		MessageHeader: &MessageHeader{
			Header: &Header{
				MessageType: MessageTypeMessage,
				ChunkType:   ChunkTypeFinal,
			},
			SymmetricSecurityHeader: &SymmetricSecurityHeader{
				TokenID: 1,
			},
			SequenceHeader: &SequenceHeader{
				SequenceNumber: 1,
				RequestID:      1,
			},
		},
		Data: []byte{0xde, 0xad, 0xbe, 0xef},
	}

	// SecurityMode==None with the #None policy and a symmetric header. The
	// chunk must be returned raw, not handed to the crypto path.
	dec := &channelInstance{
		sc: &SecureChannel{cfg: &Config{
			SecurityMode:      ua.MessageSecurityModeNone,
			SecurityPolicyURI: ua.SecurityPolicyURINone,
		}},
	}

	plain, err := dec.verifyAndDecrypt(chunk, nil)
	require.NoError(t, err, "error: message decrypt")

	require.Equal(t, chunk.Data, plain,
		"None-mode symmetric chunk must be returned raw, not decrypted")
}

// TestVerifyAndDecrypt_Part6_674_RealPolicyNoneModeSymmetricReturnsRaw forces
// the guard's !isAsymmetric disjunct. With SecurityMode==None and a real
// SecurityPolicyURI (not #None), the first disjunct (policy == #None) is false,
// so only !isAsymmetric can return the chunk raw. A symmetric chunk in this
// state must return m.Data unchanged and never reach c.algo.Decrypt; here algo
// is nil, so the guard must return before any algo dereference.
//
// This is the only config that pins !isAsymmetric: every other current test
// satisfies the first disjunct (#None policy) and short-circuits before
// !isAsymmetric is evaluated.
//
// OPC UA Part 6 v1.05 §6.7.4 secures the OpenSecureChannel handshake under a
// real policy, but a None-mode channel's steady-state symmetric traffic carries
// no crypto.
func TestVerifyAndDecrypt_Part6_674_RealPolicyNoneModeSymmetricReturnsRaw(t *testing.T) {
	chunk := &MessageChunk{
		MessageHeader: &MessageHeader{
			Header: &Header{
				MessageType: MessageTypeMessage,
				ChunkType:   ChunkTypeFinal,
			},
			SymmetricSecurityHeader: &SymmetricSecurityHeader{
				TokenID: 1,
			},
			SequenceHeader: &SequenceHeader{
				SequenceNumber: 1,
				RequestID:      1,
			},
		},
		Data: []byte{0xde, 0xad, 0xbe, 0xef},
	}

	// SecurityMode==None with a REAL policy and a symmetric header. The first
	// disjunct (policy == #None) is false, so !isAsymmetric is what returns the
	// chunk raw.
	dec := &channelInstance{
		sc: &SecureChannel{cfg: &Config{
			SecurityMode:      ua.MessageSecurityModeNone,
			SecurityPolicyURI: ua.SecurityPolicyURIBasic256Sha256,
		}},
	}

	plain, err := dec.verifyAndDecrypt(chunk, nil)
	require.NoError(t, err, "error: message decrypt")

	require.Equal(t, chunk.Data, plain,
		"real-policy None-mode symmetric chunk must be returned raw, not decrypted")
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
