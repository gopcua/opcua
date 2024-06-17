// Copyright 2018-2020 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uasc

import (
	"encoding/binary"
	"math"
	"sync"
	"time"

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

	// this is the formula proposed by OPCUA - source node-opcua
	maxBodySize :=
		c.algo.PlaintextBlockSize()*
			((chunkSize-headerSize-symmetricAlgorithmHeader-c.algo.SignatureLength()-1)/c.algo.BlockSize()) -
			sequenceHeaderSize
	c.maxBodySize = uint32(maxBodySize - 12)

	// this is the formula proposed by ERN - source node-opcua
	// maxBlock := (chunkSize - headerSize) / c.algo.BlockSize()
	// c.maxBodySize = c.algo.PlaintextBlockSize()*maxBlock - sequenceHeaderSize - c.algo.SignatureLength() - 1
}

// signAndEncrypt encrypts the message bytes stored in b and returns the
// data signed and encrypted per the security policy information from the
// secure channel.
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
		paddingLength := plaintextBlockSize - ((len(b[headerLength:]) + c.algo.SignatureLength() + paddingBytes) % plaintextBlockSize)

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

func (c *channelInstance) verifyAndDecrypt(m *MessageChunk, r []byte) ([]byte, error) {
	if c.sc.cfg.SecurityMode == ua.MessageSecurityModeNone {
		return m.Data, nil
	}

	isAsymmetric := m.AsymmetricSecurityHeader != nil

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
