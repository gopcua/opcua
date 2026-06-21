// Copyright 2018-2020 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uasc

import (
	"encoding/binary"
	"math"
	"sync"
	"time"

	"github.com/gopcua/opcua/debug"
	"github.com/gopcua/opcua/errors"
	"github.com/gopcua/opcua/id"
	"github.com/gopcua/opcua/ua"
	"github.com/gopcua/opcua/uapolicy"
)

type instanceState int

const (
	channelOpening instanceState = iota
	channelActive
)

type channelInstance struct {
	sync.Mutex
	sc              *SecureChannel
	state           instanceState
	createdAt       time.Time
	revisedLifetime time.Duration
	secureChannelID uint32
	securityTokenID uint32
	sequenceNumber  uint32
	algo            *uapolicy.EncryptionAlgorithm
	maxBodySize     uint32

	bytesSent uint64 // atomic.Load/Store - needs to be aligned for 32bit systems
	// bytesReceived    uint64
	messagesSent uint32 // atomic.Load/Store
	// messagesReceived uint32
}

func newChannelInstance(sc *SecureChannel) *channelInstance {
	return &channelInstance{
		sc:    sc,
		state: channelOpening,
	}
}

func (c *channelInstance) nextSequenceNumber() uint32 {
	// lock must be held
	c.sequenceNumber++
	if c.sequenceNumber > math.MaxUint32-1023 {
		c.sequenceNumber = 1
	}

	return c.sequenceNumber
}

func (c *channelInstance) newRequestMessage(req ua.Request, reqID uint32, authToken *ua.NodeID, timeout time.Duration) (*Message, error) {
	typeID := ua.ServiceTypeID(req)
	if typeID == 0 {
		return nil, errors.Errorf("unknown service %T. Did you call register?", req)
	}
	if authToken == nil {
		authToken = ua.NewTwoByteNodeID(0)
	}

	reqHdr := &ua.RequestHeader{
		AuthenticationToken: authToken,
		Timestamp:           c.sc.timeNow(),
		RequestHandle:       reqID, // TODO: can I cheat like this?
	}

	if timeout > 0 && timeout < c.sc.cfg.RequestTimeout {
		timeout = c.sc.cfg.RequestTimeout
	}
	reqHdr.TimeoutHint = uint32(timeout / time.Millisecond)
	req.SetHeader(reqHdr)

	// encode the message
	return c.newMessage(req, typeID, reqID), nil
}

func (c *channelInstance) newMessage(srv interface{}, typeID uint16, requestID uint32) *Message {
	sequenceNumber := c.nextSequenceNumber()
	debug.Printf("got sequence number %d", sequenceNumber)

	switch typeID {
	case id.OpenSecureChannelRequest_Encoding_DefaultBinary, id.OpenSecureChannelResponse_Encoding_DefaultBinary:
		// Do not send the thumbprint for security mode None
		// even if we have a certificate.
		//
		// See https://github.com/gopcua/opcua/issues/259
		thumbprint := c.sc.cfg.Thumbprint
		if c.sc.cfg.SecurityMode == ua.MessageSecurityModeNone {
			thumbprint = nil
		}

		return &Message{
			MessageHeader: &MessageHeader{
				Header:                   NewHeader(MessageTypeOpenSecureChannel, ChunkTypeFinal, c.secureChannelID),
				AsymmetricSecurityHeader: NewAsymmetricSecurityHeader(c.sc.cfg.SecurityPolicyURI, c.sc.cfg.Certificate, thumbprint),
				SequenceHeader:           NewSequenceHeader(sequenceNumber, requestID),
			},
			TypeID:  ua.NewFourByteExpandedNodeID(0, typeID),
			Service: srv,
		}

	case id.CloseSecureChannelRequest_Encoding_DefaultBinary, id.CloseSecureChannelResponse_Encoding_DefaultBinary:
		return &Message{
			MessageHeader: &MessageHeader{
				Header:                  NewHeader(MessageTypeCloseSecureChannel, ChunkTypeFinal, c.secureChannelID),
				SymmetricSecurityHeader: NewSymmetricSecurityHeader(c.securityTokenID),
				SequenceHeader:          NewSequenceHeader(sequenceNumber, requestID),
			},
			TypeID:  ua.NewFourByteExpandedNodeID(0, typeID),
			Service: srv,
		}

	default:
		return &Message{
			MessageHeader: &MessageHeader{
				Header:                  NewHeader(MessageTypeMessage, ChunkTypeFinal, c.secureChannelID),
				SymmetricSecurityHeader: NewSymmetricSecurityHeader(c.securityTokenID),
				SequenceHeader:          NewSequenceHeader(sequenceNumber, requestID),
			},
			TypeID:  ua.NewFourByteExpandedNodeID(0, typeID),
			Service: srv,
		}
	}
}

func (c *channelInstance) SetMaximumBodySize(chunkSize int) {
	sequenceHeaderSize := 8
	headerSize := 12
	symmetricAlgorithmHeader := 4

	// signAndEncrypt appends one PaddingSize byte (two if ExtraPaddingSize is
	// present, i.e. for signatures longer than 256 bytes) before padding the
	// plaintext to a block boundary.
	paddingSizeBytes := 1
	if c.algo.RemoteSignatureLength() > 256 {
		paddingSizeBytes = 2
	}

	// OPC UA Part 6, 6.7.2.5 defines
	//
	//   MaxBodySize = PlainTextBlockSize *
	//       Floor((MessageChunkSize - HeaderSize - 1) / CipherTextBlockSize) -
	//       SequenceHeaderSize - SignatureSize
	//
	// where the -1 reserves room for the PaddingSize byte. Subtracting it
	// inside the Floor only takes effect when chunkSize-headerSize is an exact
	// multiple of the cipher block size. For any other chunk size (e.g. the
	// default 65535) a maximum-size body pads out to one cipher block more
	// than fits in the chunk. Reserving the PaddingSize byte(s) outside the
	// Floor instead is exact: a body of maxBodySize fills the chunk to the
	// last whole cipher block and one more byte no longer fits, for every
	// chunk size (see TestMaxBodySizeFitsChunk).
	maxBodySize :=
		c.algo.PlaintextBlockSize()*
			((chunkSize-headerSize-symmetricAlgorithmHeader)/c.algo.BlockSize()) -
			sequenceHeaderSize - c.algo.SignatureLength() - paddingSizeBytes
	c.maxBodySize = uint32(maxBodySize)
}

// signAndEncrypt encrypts the message bytes stored in b and returns the
// data signed and encrypted per the security policy information from the
// secure channel.
//
// Unlike verifyAndDecrypt, this has no asymmetric carve-out for SecurityMode
// None. That is correct only because a sender carries its real (non-None)
// SecurityMode when emitting an OPN; add an asymmetric guard here if the crypto
// is ever restructured. This function and verifyAndDecrypt stay in lockstep on
// the isAsymmetric clause: changing one without the other re-breaks symmetric
// Sign traffic.
//
// TODO: split the merged asymmetric/symmetric crypto into separate functions,
// as open62541 (signAndEncryptAsym/Sym) and the OPC Foundation .NET reference
// (WriteAsymmetric/SymmetricMessage) do. Any such split must preserve
// verifyAndDecrypt's asymmetric carve-out for the SecurityMode==None window, or
// symmetric Sign traffic re-breaks (#815/#817).
func (c *channelInstance) signAndEncrypt(m *Message, b []byte) ([]byte, error) {
	// Nothing to do
	if c.sc.cfg.SecurityMode == ua.MessageSecurityModeNone {
		return b, nil
	}

	isAsymmetric := m.MessageHeader.AsymmetricSecurityHeader != nil

	var headerLength int

	if isAsymmetric {
		headerLength = 12 + m.AsymmetricSecurityHeader.Len()
	} else {
		headerLength = 12 + m.SymmetricSecurityHeader.Len()
	}

	var encryptedLength int
	if c.sc.cfg.SecurityMode == ua.MessageSecurityModeSignAndEncrypt || isAsymmetric {
		plaintextBlockSize := c.algo.PlaintextBlockSize()
		extraPadding := c.algo.RemoteSignatureLength() > 256
		paddingBytes := 1
		if extraPadding {
			paddingBytes = 2
		}
		// The PaddingSize byte(s) and Padding must fill the plaintext up to a
		// block boundary. If it is already block-aligned no padding is needed
		// beyond the PaddingSize byte(s) themselves (cf. open62541
		// ua_securechannel_crypto.c, padding calculation).
		remainder := (len(b[headerLength:]) + c.algo.SignatureLength() + paddingBytes) % plaintextBlockSize
		paddingLength := 0
		if remainder != 0 {
			paddingLength = plaintextBlockSize - remainder
		}

		// appends paddingLength Padding bytes plus one PaddingSize byte,
		// each holding the paddingLength value (OPC UA Part 6, 6.7.2.5)
		for i := 0; i <= paddingLength; i++ {
			b = append(b, byte(paddingLength))
		}
		if extraPadding {
			b = append(b, byte(paddingLength>>8))
		}
		encryptedLength = ((len(b[headerLength:]) + c.algo.SignatureLength()) / plaintextBlockSize) * c.algo.BlockSize()
	} else { // MessageSecurityModeSign
		encryptedLength = len(b[headerLength:]) + c.algo.SignatureLength()
	}

	// Fix header size to account for signing / encryption
	binary.LittleEndian.PutUint32(b[4:], uint32(headerLength+encryptedLength))
	m.Header.MessageSize = uint32(headerLength + encryptedLength)

	signature, err := c.algo.Signature(b)
	if err != nil {
		return nil, ua.StatusBadSecurityChecksFailed
	}

	b = append(b, signature...)
	p := b[headerLength:]
	if c.sc.cfg.SecurityMode == ua.MessageSecurityModeSignAndEncrypt || isAsymmetric {
		p, err = c.algo.Encrypt(p)
		if err != nil {
			return nil, ua.StatusBadSecurityChecksFailed
		}
	}
	return append(b[:headerLength], p...), nil
}

// verifyAndDecrypt verifies and optionally decrypts an incoming chunk.
//
// An asymmetric OpenSecureChannel under a real security policy is always signed
// and encrypted, even while the channel's SecurityMode still reads None during
// the handshake (OPC UA Part 6 v1.05 §6.7.4). The SecurityMode==None case below
// therefore returns raw only for a genuinely unsecured chunk: the explicit #None
// policy, or any symmetric message regardless of policy. This carve-out is the
// replacement for the readChunk SecurityMode override (added in #817); that
// override is no longer present and this guard takes its place.
func (c *channelInstance) verifyAndDecrypt(m *MessageChunk, r []byte) ([]byte, error) {
	isAsymmetric := m.AsymmetricSecurityHeader != nil

	// Return raw only for a genuinely unsecured chunk; an asymmetric OPN under a
	// real policy must still be decrypted even while SecurityMode reads None.
	if c.sc.cfg.SecurityMode == ua.MessageSecurityModeNone &&
		(c.sc.cfg.SecurityPolicyURI == ua.SecurityPolicyURINone || !isAsymmetric) {
		return m.Data, nil
	}

	headerLength := 12

	if isAsymmetric {
		headerLength += m.AsymmetricSecurityHeader.Len()
	} else {
		headerLength += m.SymmetricSecurityHeader.Len()
	}

	b := make([]byte, len(r))
	copy(b, r)

	if c.sc.cfg.SecurityMode == ua.MessageSecurityModeSignAndEncrypt || isAsymmetric {
		p, err := c.algo.Decrypt(b[headerLength:])
		if err != nil {
			return nil, ua.StatusBadSecurityChecksFailed
		}
		b = append(b[:headerLength], p...)
	}

	signature := b[len(b)-c.algo.RemoteSignatureLength():]
	messageToVerify := b[:len(b)-c.algo.RemoteSignatureLength()]

	if err := c.algo.VerifySignature(messageToVerify, signature); err != nil {
		return nil, ua.StatusBadSecurityChecksFailed
	}

	var paddingLength int
	if c.sc.cfg.SecurityMode == ua.MessageSecurityModeSignAndEncrypt || isAsymmetric {
		paddingLength = int(messageToVerify[len(messageToVerify)-1])
		if c.algo.SignatureLength() > 256 {
			paddingLength <<= 8
			paddingLength += int(messageToVerify[len(messageToVerify)-2])
			paddingLength += 1
		}
		paddingLength += 1
	}

	b = messageToVerify[headerLength : len(messageToVerify)-paddingLength]

	return b, nil
}
