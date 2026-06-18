package uasc

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/binary"
	"encoding/pem"
	"fmt"
	"io"
	"math"
	"net"
	"strings"
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
// SecurityPolicyURI and an asymmetric header, it must decrypt the OPN chunk, not
// return it raw. readChunk no longer promotes the mode (the #817 override is no
// longer present), so SecurityMode==None + real policy + asymmetric header is the
// live path the server takes while decrypting an incoming OPN during the
// handshake. The guard exercised here is that path's production mechanism: it
// preserves the #815 fix without mutating the negotiated mode. Without it,
// verifyAndDecrypt returns m.Data raw whenever SecurityMode==None, regardless of
// the asymmetric header or policy, which re-introduces #815.
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

// TestVerifyAndDecrypt_Part6_674_SignModeSymmetricRoundtrip covers Sign mode
// (MessageSecurityModeSign) on symmetric traffic: the message is signed but not
// encrypted. signAndEncrypt's Sign branch and verifyAndDecrypt's matching verify
// path are otherwise unexercised by unit tests, since every non-None policy in
// TestSignAndEncryptVerifyAndDecrypt runs SignAndEncrypt. This is the regime #817
// broke.
//
// OPC UA Part 6 v1.05 §6.7.4: when the SecurityMode is Sign, Messages are signed
// but not encrypted.
//
// The test asserts both that the roundtrip succeeds and that the body bytes
// (between the header and the appended signature) come back as the original
// plaintext, confirming Sign mode does not encrypt symmetric traffic.
func TestVerifyAndDecrypt_Part6_674_SignModeSymmetricRoundtrip(t *testing.T) {
	const uri = ua.SecurityPolicyURIBasic256Sha256

	// Equal nonces make the signing and verifying HMAC keys identical, so a
	// single channelInstance can sign a message and verify its own signature.
	nonce := make([]byte, 32)
	for i := range nonce {
		nonce[i] = byte(i)
	}
	algo, err := uapolicy.Symmetric(uri, nonce, nonce)
	require.NoError(t, err)

	c := &channelInstance{
		sc: &SecureChannel{cfg: &Config{
			SecurityMode:      ua.MessageSecurityModeSign,
			SecurityPolicyURI: uri,
		}},
		algo: algo,
	}

	m := &Message{
		MessageHeader: &MessageHeader{
			Header: &Header{
				MessageType: MessageTypeMessage,
				ChunkType:   ChunkTypeFinal,
			},
			SymmetricSecurityHeader: &SymmetricSecurityHeader{TokenID: 1},
			SequenceHeader: &SequenceHeader{
				SequenceNumber: 1,
				RequestID:      1,
			},
		},
	}

	// 12-byte message header + 4-byte symmetric security header (TokenID) + body.
	headerLength := 12 + m.SymmetricSecurityHeader.Len()
	plaintext := []byte{
		// Message Header: MSG / Final / MessageSize (patched by signAndEncrypt) / SecureChannelID
		0x4d, 0x53, 0x47, 0x46, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		// SymmetricSecurityHeader: TokenID
		0x01, 0x00, 0x00, 0x00,
		// Body
		0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
		0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10,
	}
	expectedBody := append([]byte(nil), plaintext[headerLength:]...)

	cipher, err := c.signAndEncrypt(m, append([]byte(nil), plaintext...))
	require.NoError(t, err, "error: message sign")

	// Sign mode appends a signature but does not encrypt, so the body on the wire
	// stays plaintext. Read the body between the header and the trailing signature.
	wireBody := cipher[headerLength : len(cipher)-algo.SignatureLength()]
	require.Equal(t, expectedBody, wireBody,
		"Sign mode must leave the symmetric body as plaintext, not ciphertext")

	chunk := new(MessageChunk)
	_, err = chunk.Decode(cipher)
	require.NoError(t, err, "error: message decode")

	plain, err := c.verifyAndDecrypt(chunk, cipher)
	require.NoError(t, err, "error: message verify")

	require.Equal(t, expectedBody, plain,
		"Sign mode roundtrip must return the original plaintext body")
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

const chunkedResponseTestTimeout = 2 * time.Second

func TestSendResponseWithContextWritesAllChunks(t *testing.T) {
	// Use a real TCP pair here so we exercise the exact chunk framing that a
	// peer would see on the wire, not just the internal message encoder.
	serverTCP, clientTCP := newTestTCPConnPair(t)
	defer serverTCP.Close()
	defer clientTCP.Close()

	uacpConn, err := uacp.NewConn(serverTCP, &uacp.Acknowledge{
		ReceiveBufSize: 64 * 1024,
		SendBufSize:    64 * 1024,
		MaxMessageSize: 2 * 1024 * 1024,
		MaxChunkCount:  512,
	})
	require.NoError(t, err)

	s := &SecureChannel{
		c: uacpConn,
		cfg: &Config{
			SecurityPolicyURI: ua.SecurityPolicyURINone,
			SecurityMode:      ua.MessageSecurityModeNone,
			RequestTimeout:    time.Second,
		},
		instances: make(map[uint32][]*channelInstance),
	}

	instance := newChannelInstance(s)
	instance.state = channelActive
	instance.secureChannelID = 1
	instance.securityTokenID = 1
	instance.maxBodySize = 256
	s.activeInstance = instance

	readDone := make(chan readChunksResult, 1)
	go func() {
		// Collect raw chunks until the final one arrives so the assertions below
		// can validate framing and sequence progression end-to-end.
		readDone <- readChunksUntilFinal(clientTCP)
	}()

	valueBytes := make([]byte, 4096)
	for i := range valueBytes {
		valueBytes[i] = byte(i)
	}

	v, err := ua.NewVariant(valueBytes)
	require.NoError(t, err)

	dv := &ua.DataValue{Value: v}
	dv.UpdateMask()

	resp := &ua.ReadResponse{
		ResponseHeader: &ua.ResponseHeader{
			Timestamp:          time.Now(),
			RequestHandle:      42,
			ServiceDiagnostics: &ua.DiagnosticInfo{},
			StringTable:        []string{},
			AdditionalHeader:   ua.NewExtensionObject(nil),
		},
		Results: []*ua.DataValue{dv},
	}

	require.NoError(t, s.SendResponseWithContext(context.Background(), 42, resp))

	select {
	case result := <-readDone:
		require.NoError(t, result.err)
		assertChunkedResponse(t, result.chunks)
	case <-time.After(chunkedResponseTestTimeout):
		t.Fatal("timed out waiting for response chunks")
	}
}

type readChunksResult struct {
	chunks [][]byte
	err    error
}

func readChunksUntilFinal(conn *net.TCPConn) readChunksResult {
	if err := conn.SetReadDeadline(time.Now().Add(chunkedResponseTestTimeout)); err != nil {
		return readChunksResult{err: err}
	}

	var chunks [][]byte
	for {
		// Read the fixed UACP/UASC header first so we know how large the current
		// chunk is before reading the remainder of its payload.
		hdr := make([]byte, 8)
		if _, err := io.ReadFull(conn, hdr); err != nil {
			return readChunksResult{err: err}
		}

		messageSize := binary.LittleEndian.Uint32(hdr[4:8])
		if messageSize < 12 {
			return readChunksResult{err: io.ErrUnexpectedEOF}
		}

		chunk := make([]byte, int(messageSize))
		copy(chunk, hdr)
		if _, err := io.ReadFull(conn, chunk[8:]); err != nil {
			return readChunksResult{err: err}
		}

		chunks = append(chunks, chunk)
		if chunk[3] == ChunkTypeFinal {
			return readChunksResult{chunks: chunks}
		}
	}
}

func assertChunkedResponse(t *testing.T, chunks [][]byte) {
	t.Helper()

	require.GreaterOrEqual(t, len(chunks), 2)

	var previousSequence uint32
	for i, chunk := range chunks {
		require.Equal(t, MessageTypeMessage, string(chunk[:3]))

		// Every chunk but the last should be marked intermediate so the test
		// proves the response was actually split, not just sent once.
		wantChunkType := byte(ChunkTypeIntermediate)
		if i == len(chunks)-1 {
			wantChunkType = ChunkTypeFinal
		}
		require.Equal(t, wantChunkType, chunk[3])

		messageSize := binary.LittleEndian.Uint32(chunk[4:8])
		require.Equal(t, len(chunk), int(messageSize))

		secureChannelID := binary.LittleEndian.Uint32(chunk[8:12])
		require.EqualValues(t, 1, secureChannelID)

		// Additional chunks must advance the sequence number; reusing the same
		// value would make the stream invalid for receivers.
		sequenceNumber := binary.LittleEndian.Uint32(chunk[16:20])
		if i > 0 {
			require.Greater(t, sequenceNumber, previousSequence)
		}
		previousSequence = sequenceNumber
	}
}

func newTestTCPConnPair(t *testing.T) (*net.TCPConn, *net.TCPConn) {
	t.Helper()

	// Keep the transport setup tiny and local so this regression test stays
	// close to the real wire behavior without dragging in a full listener stack.
	ln, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		if strings.Contains(err.Error(), "operation not permitted") {
			t.Skipf("tcp listen not permitted in this environment: %v", err)
		}
		t.Fatalf("ListenTCP() failed: %v", err)
	}
	defer ln.Close()

	accepted := make(chan *net.TCPConn, 1)
	acceptErr := make(chan error, 1)
	go func() {
		conn, err := ln.AcceptTCP()
		if err != nil {
			acceptErr <- err
			return
		}
		accepted <- conn
	}()

	client, err := net.DialTCP("tcp", nil, ln.Addr().(*net.TCPAddr))
	require.NoError(t, err)

	select {
	case server := <-accepted:
		return server, client
	case err := <-acceptErr:
		client.Close()
		t.Fatalf("AcceptTCP() failed: %v", err)
	case <-time.After(chunkedResponseTestTimeout):
		client.Close()
		t.Fatal("timed out waiting for accepted TCP connection")
	}

	return nil, nil
}
