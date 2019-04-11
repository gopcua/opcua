// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uasc

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"fmt"
	"io"
	"log"
	"reflect"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gopcua/opcua/debug"
	"github.com/gopcua/opcua/securitypolicy"
	"github.com/gopcua/opcua/ua"
	"github.com/gopcua/opcua/uacp"
)

const (
	secureChannelCreated int32 = iota
	secureChannelOpen
	secureChannelClosed
)

type Response struct {
	V   interface{}
	Err error
}

type SecureChannel struct {
	EndpointURL string

	// c is the uacp connection.
	c *uacp.Conn

	// cfg is the configuration for the secure channel.
	cfg *Config

	// reqhdr is the header for the next request.
	reqhdr *ua.RequestHeader

	// quit signals the termination of the recv loop.
	quit chan struct{}

	// state is the state of the secure channel.
	// Must be accessed with atomic.LoadInt32/StoreInt32
	state int32

	// mu guards handler which contains the response channels
	// for the outstanding requests. The key is the request
	// handle which is part of the Request and Response headers.
	mu      sync.Mutex
	handler map[uint32]chan Response

	enc *securitypolicy.EncryptionAlgorithm
}

func NewSecureChannel(endpoint string, c *uacp.Conn, cfg *Config) (*SecureChannel, error) {
	if c == nil {
		return nil, fmt.Errorf("no connection")
	}
	if cfg == nil {
		return nil, fmt.Errorf("no secure channel config")
	}

	// always reset the secure channel id
	cfg.SecureChannelID = 0

	return &SecureChannel{
		EndpointURL: endpoint,
		c:           c,
		cfg:         cfg,
		reqhdr: &ua.RequestHeader{
			AuthenticationToken: ua.NewTwoByteNodeID(0),
			Timestamp:           time.Now(),
			TimeoutHint:         0xffff,
			AdditionalHeader:    ua.NewExtensionObject(nil),
		},
		state:   secureChannelCreated,
		quit:    make(chan struct{}),
		handler: make(map[uint32]chan Response),
	}, nil
}

func (s *SecureChannel) Open() error {
	go s.recv()
	return s.openSecureChannel()
}

func (s *SecureChannel) Close() error {
	if err := s.closeSecureChannel(); err != nil {
		log.Print("failed to send close secure channel request")
	}
	close(s.quit)
	return s.c.Close()
}

func (s *SecureChannel) LocalEndpoint() string {
	return s.EndpointURL
}

func (s *SecureChannel) setState(n int32) {
	atomic.StoreInt32(&s.state, n)
}

func (s *SecureChannel) hasState(n int32) bool {
	return atomic.LoadInt32(&s.state) == n
}

func (s *SecureChannel) openSecureChannel() error {
	var err error
	var localKey *rsa.PrivateKey
	var remoteKey *rsa.PublicKey

	// Set the encryption methods to Asymmetric with the appropriate
	// public keys.  OpenSecureChannel is always encrypted with the
	// asymmetric algorithms.
	// The default value of the encryption algorithm method is the
	// SecurityModeNone so no additional work is required for that case
	if s.cfg.SecurityMode != ua.MessageSecurityModeNone {
		localKey = s.cfg.LocalKey
		// todo(dh): move this into the securitypolicy package proper or
		// adjust the Asymmetric method to receive a certificate instead
		remoteCert, err := x509.ParseCertificate(s.cfg.RemoteCertificate)
		if err != nil {
			return err
		}
		var ok bool
		remoteKey, ok = remoteCert.PublicKey.(*rsa.PublicKey)
		if !ok {
			return ua.StatusBadCertificateInvalid
		}
	}

	s.enc, err = securitypolicy.Asymmetric(s.cfg.SecurityPolicyURI, localKey, remoteKey)
	if err != nil {
		return err
	}

	nonce := make([]byte, s.enc.NonceLength())
	if _, err := rand.Read(nonce); err != nil {
		return err
	}

	req := &ua.OpenSecureChannelRequest{
		ClientProtocolVersion: 0,
		RequestType:           ua.SecurityTokenRequestTypeIssue,
		SecurityMode:          s.cfg.SecurityMode,
		ClientNonce:           nonce,
		RequestedLifetime:     s.cfg.Lifetime,
	}

	return s.Send(req, nil, func(v interface{}) error {
		resp, ok := v.(*ua.OpenSecureChannelResponse)
		if !ok {
			return fmt.Errorf("got %T, want OpenSecureChannelResponse", req)
		}
		s.cfg.SecurityTokenID = resp.SecurityToken.TokenID

		s.enc, err = securitypolicy.Symmetric(s.cfg.SecurityPolicyURI, nonce, resp.ServerNonce)
		if err != nil {
			return err
		}

		s.setState(secureChannelOpen)
		return nil
	})
}

// closeSecureChannel sends CloseSecureChannelRequest on top of UASC to SecureChannel.
func (s *SecureChannel) closeSecureChannel() error {
	req := &ua.CloseSecureChannelRequest{}

	defer s.setState(secureChannelClosed)
	return s.Send(req, nil, nil)
}

// Send sends the service request and calls h with the response.
func (s *SecureChannel) Send(svc interface{}, authToken *ua.NodeID, h func(interface{}) error) error {
	ch, err := s.SendAsync(svc, authToken)
	if err != nil {
		return err
	}

	if h == nil {
		return nil
	}

	// todo(fs): handle timeout
	resp := <-ch
	if resp.Err != nil {
		return resp.Err
	}
	return h(resp.V)
}

// SendAsync sends the service request and returns a channel which will receive the
// response when it arrives.
func (s *SecureChannel) SendAsync(svc interface{}, authToken *ua.NodeID) (resp chan Response, err error) {
	typeID := ua.ServiceTypeID(svc)
	if typeID == 0 {
		return nil, fmt.Errorf("unknown service %T. Did you call register?", svc)
	}
	if authToken == nil {
		authToken = ua.NewTwoByteNodeID(0)
	}
	// the request header is always the first field
	val := reflect.ValueOf(svc)
	val.Elem().Field(0).Set(reflect.ValueOf(s.reqhdr))
	// update counters and reset them on error
	s.cfg.SequenceNumber++
	s.reqhdr.AuthenticationToken = authToken
	s.reqhdr.RequestHandle++
	s.reqhdr.Timestamp = time.Now()
	defer func() {
		if err != nil {
			s.cfg.SequenceNumber--
			s.reqhdr.RequestHandle--
		}
	}()

	// encode the message
	m := NewMessage(svc, typeID, s.cfg)
	b, err := m.Encode()
	if err != nil {
		return nil, err
	}
	reqid := m.SequenceHeader.RequestID

	// encrypt the message prior to sending it
	// if SecurityMode == None, this returns the byte stream untouched
	b, err = s.signAndEncrypt(m, b)

	// send the message
	if _, err := s.c.Write(b); err != nil {
		return nil, err
	}
	debug.Printf("conn %d/%d: send %T with %d bytes", s.c.ID(), reqid, svc, len(b))

	// register the handler
	resp = make(chan Response)
	s.mu.Lock()
	s.handler[reqid] = resp
	s.mu.Unlock()
	return resp, nil
}

func (s *SecureChannel) readchunk() (*MessageChunk, error) {
	// read a full message from the underlying conn.
	b, err := s.c.Receive()
	if err == io.EOF || s.hasState(secureChannelClosed) {
		return nil, io.EOF
	}
	if errf, ok := err.(*uacp.Error); ok {
		return nil, errf
	}
	if err != nil {
		return nil, fmt.Errorf("sechan: read header failed: %s %#v", err, err)
	}

	const hdrlen = 12
	h := new(Header)
	if _, err := h.Decode(b[:hdrlen]); err != nil {
		return nil, fmt.Errorf("sechan: decode header failed: %s", err)
	}

	// drop if the channel id does not match
	if s.cfg.SecureChannelID > 0 && s.cfg.SecureChannelID != h.SecureChannelID {
		return nil, fmt.Errorf("sechan: secure channel id mismatch: got 0x%04x, want 0x%04x", h.SecureChannelID, s.cfg.SecureChannelID)
	}

	// todo(fs): check for ERRF here

	// decode the other headers
	m := new(MessageChunk)
	if _, err := m.Decode(b); err != nil {
		return nil, fmt.Errorf("sechan: decode chunk failed: %s", err)
	}

	// todo(fs): check for MSGA here

	// decrypt the block
	m.Data, err = s.verifyAndDecrypt(m, b)
	if err != nil {
		return nil, err
	}

	n, err := m.SequenceHeader.Decode(m.Data)
	if err != nil {
		return nil, fmt.Errorf("sechan: decode sequence header failed: %s", err)
	}
	m.Data = m.Data[n:]

	if s.cfg.SecureChannelID == 0 {
		s.cfg.SecureChannelID = h.SecureChannelID
		debug.Printf("conn %d/%d: set secure channel id to %d", s.c.ID(), m.SequenceHeader.RequestID, s.cfg.SecureChannelID)
	}

	return m, nil
}

// recv receives message chunks from the secure channel, decodes and forwards
// them to the registered callback channel, if there is one. Otherwise,
// the message is dropped.
func (s *SecureChannel) recv() {
	// chunks maps request id to message chunks
	chunks := map[uint32][]*MessageChunk{}

	for {
		select {
		case <-s.quit:
			return

		default:
			chunk, err := s.readchunk()
			if err == io.EOF {
				return
			}
			if errf, ok := err.(*uacp.Error); ok {
				s.notifyCallers(errf)
				s.Close()
				return
			}
			if err != nil {
				debug.Printf("error received while receiving chunk: %s", err)
				continue
			}

			hdr := chunk.Header
			reqid := chunk.SequenceHeader.RequestID
			debug.Printf("conn %d/%d: recv %s%c with %d bytes", s.c.ID(), reqid, hdr.MessageType, hdr.ChunkType, hdr.MessageSize)

			if hdr.ChunkType != 'F' {
				chunks[reqid] = append(chunks[reqid], chunk)
				if n := len(chunks[reqid]); uint32(n) > s.c.MaxChunkCount() {
					// todo(fs): send error
					delete(chunks, reqid)
					s.notifyCaller(reqid, nil, fmt.Errorf("too many chunks: %d > %d", n, s.c.MaxChunkCount()))
				}
				continue
			}

			// merge chunks
			all := append(chunks[reqid], chunk)
			delete(chunks, reqid)
			b, err := mergeChunks(all)
			if err != nil {
				// todo(fs): send error
				s.notifyCaller(reqid, nil, fmt.Errorf("chunk merge error: %v", err))
				continue
			}

			if uint32(len(b)) > s.c.MaxMessageSize() {
				// todo(fs): send error
				s.notifyCaller(reqid, nil, fmt.Errorf("message too large: %d > %d", uint32(len(b)), s.c.MaxMessageSize()))
				continue
			}

			// since we are not decoding the ResponseHeader separately
			// we need to drop every message that has an error since we
			// cannot get to the RequestHandle in the ResponseHeader.
			// To fix this we must a) decode the ResponseHeader separately
			// and subsequently remove it and the TypeID from all service
			// structs and tests. We also need to add a deadline to all
			// handlers and check them periodically to time them out.
			_, svc, err := ua.DecodeService(b)
			if err != nil {
				s.notifyCaller(reqid, nil, err)
				continue
			}

			// extract the ServiceStatus field from the
			// ResponseHeader which is always the first
			// field in the struct.
			//
			// If the service status is not OK then bubble
			// that error up to the caller.
			val := reflect.ValueOf(svc)
			field0 := val.Elem().Field(0).Interface()
			if hdr, ok := field0.(*ua.ResponseHeader); ok {
				debug.Printf("conn %d/%d: res:%v", s.c.ID(), reqid, hdr.ServiceResult)
				if hdr.ServiceResult != ua.StatusOK {
					s.notifyCaller(reqid, svc, hdr.ServiceResult)
					return
				}
			}
			s.notifyCaller(reqid, svc, err)
		}
	}
}

func (s *SecureChannel) notifyCallers(err error) {
	s.mu.Lock()
	var reqids []uint32
	for id := range s.handler {
		reqids = append(reqids, id)
	}
	for _, id := range reqids {
		s.notifyCallerLock(id, nil, err)
	}
	s.mu.Unlock()
}

func (s *SecureChannel) notifyCaller(reqid uint32, svc interface{}, err error) {
	s.mu.Lock()
	s.notifyCallerLock(reqid, svc, err)
	s.mu.Unlock()
}

func (s *SecureChannel) notifyCallerLock(reqid uint32, svc interface{}, err error) {
	if err != nil {
		debug.Printf("conn %d/%d: %v", s.c.ID(), reqid, err)
	} else {
		debug.Printf("conn %d/%d: recv %T", s.c.ID(), reqid, svc)
	}

	// check if we have a pending request handler for this response.
	ch := s.handler[reqid]
	delete(s.handler, reqid)

	// no handler -> next response
	if ch == nil {
		debug.Printf("conn %d/%d: no handler for %T", s.c.ID(), reqid, svc)
		return
	}

	// send response to caller
	go func() {
		ch <- Response{svc, err}
		close(ch)
	}()
}

func mergeChunks(chunks []*MessageChunk) ([]byte, error) {
	if len(chunks) == 0 {
		return nil, nil
	}
	if len(chunks) == 1 {
		return chunks[0].Data, nil
	}

	// todo(fs): check if this is correct and necessary
	// sort.Sort(bySequence(chunks))

	var b []byte
	var seqnr uint32
	for _, c := range chunks {
		if c.SequenceHeader.SequenceNumber == seqnr {
			continue // duplicate chunk
		}
		seqnr = c.SequenceHeader.SequenceNumber
		b = append(b, c.Data...)
	}
	return b, nil
}

// todo(fs): we only need this if we need to sort chunks. Need to check the spec
// type bySequence []*MessageChunk

// func (a bySequence) Len() int      { return len(a) }
// func (a bySequence) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
// func (a bySequence) Less(i, j int) bool {
// 	return a[i].SequenceHeader.SequenceNumber < a[j].SequenceHeader.SequenceNumber
// }
