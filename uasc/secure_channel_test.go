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
