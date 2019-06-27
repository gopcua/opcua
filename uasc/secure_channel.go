// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uasc

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"fmt"
	"io"
	"math"
	"reflect"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gopcua/opcua/debug"
	"github.com/gopcua/opcua/ua"
	"github.com/gopcua/opcua/uacp"
	"github.com/gopcua/opcua/uapolicy"
)

const (
	secureChannelCreated int32 = iota
	secureChannelOpen
	secureChannelClosed
	timeoutLeniency = 250 * time.Millisecond
	MaxTimeout      = math.MaxUint32 * time.Millisecond
)

type Response struct {
	ReqID uint32
	SCID  uint32
	V     interface{}
	Err   error
}

type SecureChannel struct {
	EndpointURL string

	// c is the uacp connection.
	c *uacp.Conn

	// cfg is the configuration for the secure channel.
	cfg *Config

	// reqhdrproto is the prototype for the header for the next request.
	reqhdrproto *ua.RequestHeader

	// state is the state of the secure channel.
	// Must be accessed with atomic.LoadInt32/StoreInt32
	state int32

	// mu guards handler which contains the response channels
	// for the outstanding requests. The key is the request
	// handle which is part of the Request and Response headers.
	mu      sync.Mutex
	handler map[uint32]chan Response

	chunks map[uint32][]*MessageChunk

	enc *uapolicy.EncryptionAlgorithm
}

func NewSecureChannel(endpoint string, c *uacp.Conn, cfg *Config) (*SecureChannel, error) {
	if c == nil {
		return nil, fmt.Errorf("no connection")
	}
	if cfg == nil {
		return nil, fmt.Errorf("no secure channel config")
	}

	if cfg.SecurityPolicyURI != ua.SecurityPolicyURINone {
		if cfg.SecurityMode == ua.MessageSecurityModeNone {
			return nil, fmt.Errorf("invalid channel config: Security policy '%s' cannot be used with '%s'", cfg.SecurityPolicyURI, cfg.SecurityMode)
		}
		if cfg.LocalKey == nil {
			return nil, fmt.Errorf("invalid channel config: Security policy '%s' requires a private key", cfg.SecurityPolicyURI)
		}
	}

	// Force the security mode to None if the policy is also None
	if cfg.SecurityPolicyURI == ua.SecurityPolicyURINone {
		cfg.SecurityMode = ua.MessageSecurityModeNone
	}

	return &SecureChannel{
		EndpointURL: endpoint,
		c:           c,
		cfg:         cfg,
		reqhdrproto: &ua.RequestHeader{
			TimeoutHint:      uint32(cfg.RequestTimeout / time.Millisecond),
			AdditionalHeader: ua.NewExtensionObject(nil),
		},
		state:   secureChannelCreated,
		handler: make(map[uint32]chan Response),
		chunks:  make(map[uint32][]*MessageChunk),
	}, nil
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

// Send sends the service request and calls h with the response.
func (s *SecureChannel) Send(svc interface{}, authToken *ua.NodeID, h func(interface{}) error) error {
	return s.SendWithTimeout(svc, authToken, s.cfg.RequestTimeout, h)
}

// SendWithTimeout sends the service request and calls h with the response with a specific timeout.
func (s *SecureChannel) SendWithTimeout(svc interface{}, authToken *ua.NodeID, timeout time.Duration, h func(interface{}) error) error {
	respRequired := h != nil

	ch, reqid, err := s.SendAsync(svc, authToken, respRequired)
	if err != nil {
		return err
	}

	if !respRequired {
		return nil
	}

	select {
	case resp := <-ch:
		if resp.Err != nil {
			if resp.V != nil {
				_ = h(resp.V) // ignore result because resp.Err takes precedence
			}
			return resp.Err
		}
		return h(resp.V)
	case <-time.After(timeout + timeoutLeniency): // `+ timeoutLeniency` to give the server a chance to respond to TimeoutHint
		s.mu.Lock()
		s.popHandlerLock(reqid)
		s.mu.Unlock()
		return ua.StatusBadTimeout
	}
}

// SendAsync sends the service request and returns a channel which will receive the
// response when it arrives.
func (s *SecureChannel) SendAsync(svc interface{}, authToken *ua.NodeID, respReq bool) (resp chan Response, reqID uint32, err error) {
	return s.sendAsyncWithTimeout(svc, authToken, respReq, s.cfg.RequestTimeout)
}

// sendAsyncWithTimeout sends the service request with a specific timeout and returns a channel which will receive the
// response when it arrives.
func (s *SecureChannel) sendAsyncWithTimeout(svc interface{}, authToken *ua.NodeID, respReq bool, timeout time.Duration) (resp chan Response, reqID uint32, err error) {
	typeID := ua.ServiceTypeID(svc)
	if typeID == 0 {
		return nil, 0, fmt.Errorf("unknown service %T. Did you call register?", svc)
	}

	s.mu.Lock()
	// prepare the header value for the next request
	reqhdr := s.newRequestHeaderLock(authToken, timeout)
	// the request header is always the first field
	val := reflect.ValueOf(svc)
	rHdr := val.Elem().Field(0)
	if _, ok := rHdr.Interface().(*ua.RequestHeader); ok {
		rHdr.Set(reflect.ValueOf(reqhdr))
	}

	s.cfg.SequenceNumber++
	s.cfg.RequestID++
	// encode the message
	m := NewMessage(svc, typeID, s.cfg)
	reqid := m.SequenceHeader.RequestID
	s.mu.Unlock()
	b, err := m.Encode()
	if err != nil {
		return nil, reqid, err
	}

	// encrypt the message prior to sending it
	// if SecurityMode == None, this returns the byte stream untouched
	b, err = s.signAndEncrypt(m, b)
	if err != nil {
		return nil, reqid, err
	}

	// send the message
	if _, err := s.c.Write(b); err != nil {
		return nil, reqid, err
	}
	debug.Printf("uasc %d/%d: send %T with %d bytes", s.c.ID(), reqid, svc, len(b))

	// register the handler if a callback was passed
	if !respReq {
		return nil, 0, nil
	}
	resp = make(chan Response)
	s.mu.Lock()
	if s.handler[reqid] != nil {
		s.mu.Unlock()
		return nil, reqid, fmt.Errorf("error: duplicate handler registration for request id %d", reqid)
	}
	s.handler[reqid] = resp
	s.mu.Unlock()
	return resp, reqid, nil
}

func (s *SecureChannel) newRequestHeaderLock(authToken *ua.NodeID, timeout time.Duration) *ua.RequestHeader {
	hdr := &ua.RequestHeader{}
	s.reqhdrproto.RequestHandle++
	*hdr = *s.reqhdrproto

	if authToken == nil {
		authToken = ua.NewTwoByteNodeID(0)
	}
	hdr.AuthenticationToken = authToken
	if hdr.AdditionalHeader == nil {
		hdr.AdditionalHeader = ua.NewExtensionObject(nil)
	}
	hdr.Timestamp = time.Now()
	if timeout > 0 && timeout < s.cfg.RequestTimeout {
		timeout = s.cfg.RequestTimeout
	}
	hdr.TimeoutHint = uint32(timeout / time.Millisecond)
	return hdr
}

func (s *SecureChannel) readChunk() (*MessageChunk, error) {
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

	// decode the other headers
	m := new(MessageChunk)
	if _, err := m.Decode(b); err != nil {
		return nil, fmt.Errorf("sechan: decode chunk failed: %s", err)
	}

	// OPN Request, initialize encryption
	// todo(dh): How to account for renew requests?
	switch m.MessageType {
	case "OPN":
		debug.Printf("uasc: OPN Request")
		// Make sure we have a valid security header
		if m.AsymmetricSecurityHeader == nil {
			return nil, ua.StatusBadDecodingError // todo(dh): check if this is the correct error
		}

		// Load the remote certificates from the security header, if present
		var remoteKey *rsa.PublicKey
		if m.SecurityPolicyURI != ua.SecurityPolicyURINone {
			remoteKey, err = uapolicy.PublicKey(m.AsymmetricSecurityHeader.SenderCertificate)
			if err != nil {
				return nil, err
			}

			s.cfg.RemoteCertificate = m.AsymmetricSecurityHeader.SenderCertificate
			debug.Printf("Setting securityPolicy to %s", m.SecurityPolicyURI)
		}

		s.cfg.SecurityPolicyURI = m.SecurityPolicyURI
		s.cfg.RequestID = m.RequestID

		s.enc, err = uapolicy.Asymmetric(m.SecurityPolicyURI, s.cfg.LocalKey, remoteKey)
		if err != nil {
			return nil, err
		}

	case "CLO":
		if !s.hasState(secureChannelOpen) {
			return nil, ua.StatusBadSecureChannelIDInvalid
		}

		// We received the close request so no response is necessary.
		// Returning io.EOF signals to the calling methods that the channel is to be shut down
		s.setState(secureChannelClosed)

		return nil, io.EOF

	case "MSG":
	}

	// Decrypts the block and returns data back into m.Data
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
		debug.Printf("uasc %d/%d: set secure channel id to %d", s.c.ID(), m.SequenceHeader.RequestID, s.cfg.SecureChannelID)
	}

	return m, nil
}

// Receive waits for a complete message to be read from the channel and
// sends it back to the caller.  If the caller was initiated from a
// Send(), the message is directed to the registered callback function
// and Receive() does not return. Otherwise, if no handler is detected,
// the Receive returns with the message as a return value.
// This behaviour means that anticipated results are automatically directed back to
// their callers but unsolicited messages are sent to the caller of
// Receive() to handle.
func (s *SecureChannel) Receive(ctx context.Context) Response {
	for {
		select {
		case <-ctx.Done():
			return Response{Err: io.EOF}
		default:
			reqid, svc, err := s.receive(ctx)
			if _, ok := err.(*uacp.Error); ok || err == io.EOF {
				s.notifyCallers(ctx, err)
				s.Close()
				return Response{
					ReqID: reqid,
					SCID:  s.cfg.SecureChannelID,
					V:     svc,
					Err:   err,
				}
			}
			if err != nil {
				debug.Printf("uasc %d/%d: err: %v", s.c.ID(), reqid, err)
			} else {
				debug.Printf("uasc %d/%d: recv %T", s.c.ID(), reqid, svc)
			}

			// todo: validate request ID / check that it is increasing correctly
			s.cfg.RequestID = reqid

			switch svc.(type) {
			case *ua.OpenSecureChannelRequest:
				err := s.handleOpenSecureChannelRequest(svc)
				if err != nil {
					return Response{
						Err: err,
					}
				}
				continue
			}

			// check if we have a pending request handler for this response.
			s.mu.Lock()
			ch, ok := s.handler[reqid]
			delete(s.handler, reqid)
			s.mu.Unlock()
			if !ok {
				debug.Printf("uasc %d/%d: no handler for %T, returning result to caller", s.c.ID(), reqid, svc)
				return Response{
					ReqID: reqid,
					SCID:  s.cfg.SecureChannelID,
					V:     svc,
					Err:   err,
				}
			}

			// send response to caller
			go func() {
				debug.Printf("sending %T to handler\n", svc)
				r := Response{
					ReqID: reqid,
					SCID:  s.cfg.SecureChannelID,
					V:     svc,
					Err:   err,
				}
				select {
				case <-ctx.Done():
				case ch <- r:
				}
			}()
		}
	}
}

// receive receives message chunks from the secure channel, decodes and forwards
// them to the registered callback channel, if there is one. Otherwise,
// the message is dropped.
func (s *SecureChannel) receive(ctx context.Context) (uint32, interface{}, error) {

	for {
		select {
		case <-ctx.Done():
			return 0, nil, nil

		default:
			chunk, err := s.readChunk()
			if err == io.EOF {
				return 0, nil, err
			}
			if errf, ok := err.(*uacp.Error); ok {
				s.notifyCallers(ctx, errf)
				return 0, nil, errf
			}
			if err != nil {
				debug.Printf("error received while receiving chunk: %s", err)
				continue
			}

			hdr := chunk.Header
			reqid := chunk.SequenceHeader.RequestID
			debug.Printf("uasc %d/%d: recv %s%c with %d bytes", s.c.ID(), reqid, hdr.MessageType, hdr.ChunkType, hdr.MessageSize)

			switch hdr.ChunkType {
			case 'A':
				delete(s.chunks, reqid)

				msga := new(MessageAbort)
				if _, err := msga.Decode(chunk.Data); err != nil {
					debug.Printf("conn %d/%d: invalid MSGA chunk. %s", s.c.ID(), reqid, err)
					return reqid, nil, ua.StatusBadDecodingError
				}

				return reqid, nil, ua.StatusCode(msga.ErrorCode)

			case 'C':
				s.chunks[reqid] = append(s.chunks[reqid], chunk)
				if n := len(s.chunks[reqid]); uint32(n) > s.c.MaxChunkCount() {
					delete(s.chunks, reqid)
					return reqid, nil, fmt.Errorf("too many chunks: %d > %d", n, s.c.MaxChunkCount())
				}
				continue
			}

			// merge chunks
			all := append(s.chunks[reqid], chunk)
			delete(s.chunks, reqid)
			b, err := mergeChunks(all)
			if err != nil {
				return reqid, nil, fmt.Errorf("chunk merge error: %v", err)
			}

			if uint32(len(b)) > s.c.MaxMessageSize() {
				return reqid, nil, fmt.Errorf("message too large: %d > %d", uint32(len(b)), s.c.MaxMessageSize())
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
				return reqid, nil, err
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
				debug.Printf("uasc %d/%d: res:%v", s.c.ID(), reqid, hdr.ServiceResult)
				if hdr.ServiceResult != ua.StatusOK {
					return reqid, svc, hdr.ServiceResult
				}
			}
			return reqid, svc, err
		}
	}
}

func (s *SecureChannel) notifyCallers(ctx context.Context, err error) {
	s.mu.Lock()
	var reqids []uint32
	for id := range s.handler {
		reqids = append(reqids, id)
	}
	for _, id := range reqids {
		s.notifyCallerLock(ctx, id, nil, err)
	}
	s.mu.Unlock()
}

func (s *SecureChannel) notifyCallerLock(ctx context.Context, reqid uint32, svc interface{}, err error) {
	if err != nil {
		debug.Printf("uasc %d/%d: %v", s.c.ID(), reqid, err)
	} else {
		debug.Printf("uasc %d/%d: recv %T", s.c.ID(), reqid, svc)
	}

	// check if we have a pending request handler for this response.
	ch := s.popHandlerLock(reqid)

	// no handler -> next response
	if ch == nil {
		debug.Printf("uasc %d/%d: no handler for %T", s.c.ID(), reqid, svc)
		return
	}

	// send response to caller
	go func() {
		r := Response{
			ReqID: reqid,
			SCID:  s.cfg.SecureChannelID,
			V:     svc,
			Err:   err,
		}
		select {
		case <-ctx.Done():
		case ch <- r:
		}
		close(ch)
	}()
}

// Open opens a new secure channel with a server
func (s *SecureChannel) Open() error {
	return s.openSecureChannel()
}

// Close closes an existing secure channel
func (s *SecureChannel) Close() error {
	if err := s.closeSecureChannel(); err != nil && err != io.EOF {
		debug.Printf("failed to send close secure channel request: %s", err)
	}

	if err := s.c.Close(); err != nil && err != io.EOF {
		debug.Printf("failed to close transport connection: %s", err)
	}

	return io.EOF
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
		// todo(dh): move this into the uapolicy package proper or
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

	s.enc, err = uapolicy.Asymmetric(s.cfg.SecurityPolicyURI, localKey, remoteKey)
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

		s.enc, err = uapolicy.Symmetric(s.cfg.SecurityPolicyURI, nonce, resp.ServerNonce)
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
	// Don't send the CloseSecureChannel message if it was never fully opened (due to ERR, etc)
	if !s.hasState(secureChannelOpen) {
		return io.EOF
	}

	err := s.Send(req, nil, nil)
	if err != nil {
		return err
	}

	return io.EOF
}

func (s *SecureChannel) handleOpenSecureChannelRequest(svc interface{}) error {
	debug.Printf("handleOpenSecureChannelRequest: Got OPN Request\n")

	var err error

	req, ok := svc.(*ua.OpenSecureChannelRequest)
	if !ok {
		debug.Printf("Expected OpenSecureChannel Request, got %T\n", svc)
	}

	s.cfg.Lifetime = req.RequestedLifetime
	s.cfg.SecurityMode = req.SecurityMode

	nonce := make([]byte, s.enc.NonceLength())
	if _, err := rand.Read(nonce); err != nil {
		return err
	}
	resp := &ua.OpenSecureChannelResponse{
		ResponseHeader: &ua.ResponseHeader{
			Timestamp:          time.Now(),
			RequestHandle:      req.RequestHeader.RequestHandle,
			ServiceDiagnostics: &ua.DiagnosticInfo{},
			StringTable:        []string{},
			AdditionalHeader:   ua.NewExtensionObject(nil),
		},
		ServerProtocolVersion: 0,
		SecurityToken: &ua.ChannelSecurityToken{
			ChannelID:       s.cfg.SecureChannelID,
			TokenID:         s.cfg.SecurityTokenID,
			CreatedAt:       time.Now(),
			RevisedLifetime: req.RequestedLifetime,
		},
		ServerNonce: nonce,
	}

	err = s.Send(resp, nil, nil)
	if err != nil {
		return err
	}

	s.enc, err = uapolicy.Symmetric(s.cfg.SecurityPolicyURI, nonce, req.ClientNonce)
	if err != nil {
		return err
	}
	s.setState(secureChannelOpen)

	return nil
}

func (s *SecureChannel) popHandlerLock(reqid uint32) chan Response {
	ch := s.handler[reqid]
	delete(s.handler, reqid)
	return ch
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
