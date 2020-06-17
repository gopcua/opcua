// Copyright 2018-2020 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uasc

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"io"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gopcua/opcua/debug"
	"github.com/gopcua/opcua/errors"
	"github.com/gopcua/opcua/ua"
	"github.com/gopcua/opcua/uacp"
	"github.com/gopcua/opcua/uapolicy"
)

const (
	timeoutLeniency = 250 * time.Millisecond
	MaxTimeout      = math.MaxUint32 * time.Millisecond
)

type response struct {
	ReqID uint32
	SCID  uint32
	V     interface{}
	Err   error
}

type SecureChannel struct {
	endpointURL string

	// c is the uacp connection
	c *uacp.Conn

	// cfg is the configuration for the secure channel.
	cfg *Config

	// time returns the current time. When not set it defaults to time.Now().
	time func() time.Time

	// closing is channel used to indicate to go routines that the secure channel is closing
	closing chan struct{}

	// closingMu is used to protect the _changing_ of the mutex
	// i.e. when we _read_ from the closing chan we acquire a read lock, and when in `reset`, we acquire a write lock
	closingMu sync.RWMutex

	// startDispatcher ensures only one dispatcher is running
	startDispatcher sync.Once

	// requestID is a "global" counter shared between multiple channels and tokens
	requestID   uint32
	requestIDMu sync.Mutex

	// instances maps secure channel IDs to a list to channel states
	instances      map[uint32][]*channelInstance
	activeInstance *channelInstance
	instancesMu    sync.Mutex

	// handles maps request IDs to response channels
	handlers   map[uint32]chan *response
	handlersMu sync.Mutex

	// chunks maintains a temporary list of chunks for a given request ID
	chunks   map[uint32][]*MessageChunk
	chunksMu sync.Mutex

	// openingInstance is a temporary var that allows the dispatcher know how to handle a open channel request
	// note: we only allow a single "open" request in flight at any point in time. The mutex is held for the entire
	// duration of the "open" request.
	openingInstance *channelInstance
	openingMu       sync.Mutex
}

func NewSecureChannel(endpoint string, c *uacp.Conn, cfg *Config) (*SecureChannel, error) {
	if c == nil {
		return nil, errors.Errorf("no connection")
	}

	if cfg == nil {
		return nil, errors.Errorf("no secure channel config")
	}

	if cfg.SecurityPolicyURI != ua.SecurityPolicyURINone {
		if cfg.SecurityMode == ua.MessageSecurityModeNone {
			return nil, errors.Errorf("invalid channel config: Security policy '%s' cannot be used with '%s'", cfg.SecurityPolicyURI, cfg.SecurityMode)
		}
		if cfg.LocalKey == nil {
			return nil, errors.Errorf("invalid channel config: Security policy '%s' requires a private key", cfg.SecurityPolicyURI)
		}
	}

	// Force the security mode to None if the policy is also None
	// TODO: I don't like that a SecureChannel changes the incoming config
	if cfg.SecurityPolicyURI == ua.SecurityPolicyURINone {
		cfg.SecurityMode = ua.MessageSecurityModeNone
	}

	s := &SecureChannel{
		endpointURL: endpoint,
		c:           c,
		cfg:         cfg,
		requestID:   cfg.RequestIDSeed,
	}

	s.reset()

	return s, nil
}

func (s *SecureChannel) reset() {
	s.closingMu.Lock()
	defer s.closingMu.Unlock()

	// note: we _don't_ reset s.requestID
	s.closing = make(chan struct{})
	s.startDispatcher = sync.Once{}
	s.instances = make(map[uint32][]*channelInstance)
	s.chunks = make(map[uint32][]*MessageChunk)
	s.handlers = make(map[uint32]chan *response)
	s.activeInstance = nil
	s.openingInstance = nil
}

func (s *SecureChannel) getActiveChannelInstance() (*channelInstance, error) {
	s.instancesMu.Lock()
	defer s.instancesMu.Unlock()
	if s.activeInstance == nil {
		return nil, errors.Errorf("sechan: secure channel not open.")
	}
	return s.activeInstance, nil
}

func (s *SecureChannel) dispatcher() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s.closingMu.RLock()
	defer s.closingMu.RUnlock()

	for {
		select {
		case <-s.closing:
			return
		default:
			resp := s.receive(ctx)
			if resp.Err == io.EOF {
				return
			}

			if resp.Err != nil {
				debug.Printf("uasc %d/%d: err: %v", s.c.ID(), resp.ReqID, resp.Err)
			} else {
				debug.Printf("uasc %d/%d: recv %T", s.c.ID(), resp.ReqID, resp.V)
			}

			ch, ok := s.popHandler(resp.ReqID)

			if !ok {
				debug.Printf("uasc %d/%d: no handler for %T", s.c.ID(), resp.ReqID, resp.V)
				continue
			}

			debug.Printf("sending %T to handler\n", resp.V)
			select {
			case ch <- resp:
			default:
				// this should never happen since the chan is of size one
				debug.Printf("unexpected state. channel write should always succeed.")
			}
		}
	}
}

// receive receives message chunks from the secure channel, decodes and forwards
// them to the registered callback channel, if there is one. Otherwise,
// the message is dropped.
func (s *SecureChannel) receive(ctx context.Context) *response {
	for {
		select {
		case <-ctx.Done():
			return &response{Err: ctx.Err()}

		default:
			chunk, err := s.readChunk()
			if err == io.EOF {
				debug.Printf("uasc readChunk EOF")
				return &response{Err: err}
			}

			if err != nil {
				return &response{Err: err}
			}

			hdr := chunk.Header
			reqID := chunk.SequenceHeader.RequestID

			resp := &response{
				ReqID: reqID,
				SCID:  chunk.MessageHeader.Header.SecureChannelID,
			}

			debug.Printf("uasc %d/%d: recv %s%c with %d bytes", s.c.ID(), reqID, hdr.MessageType, hdr.ChunkType, hdr.MessageSize)

			s.chunksMu.Lock()

			switch hdr.ChunkType {
			case 'A':
				delete(s.chunks, reqID)
				s.chunksMu.Unlock()

				msga := new(MessageAbort)
				if _, err := msga.Decode(chunk.Data); err != nil {
					debug.Printf("conn %d/%d: invalid MSGA chunk. %s", s.c.ID(), reqID, err)
					resp.Err = ua.StatusBadDecodingError
					return resp
				}

				return &response{ReqID: reqID, Err: ua.StatusCode(msga.ErrorCode)}

			case 'C':
				s.chunks[reqID] = append(s.chunks[reqID], chunk)
				if n := len(s.chunks[reqID]); uint32(n) > s.c.MaxChunkCount() {
					delete(s.chunks, reqID)
					s.chunksMu.Unlock()
					resp.Err = errors.Errorf("too many chunks: %d > %d", n, s.c.MaxChunkCount())
					return resp
				}
				s.chunksMu.Unlock()
				continue
			}

			// merge chunks
			all := append(s.chunks[reqID], chunk)
			delete(s.chunks, reqID)

			s.chunksMu.Unlock()

			b, err := mergeChunks(all)
			if err != nil {
				resp.Err = err
				return resp
			}

			if uint32(len(b)) > s.c.MaxMessageSize() {
				resp.Err = errors.Errorf("message too large: %d > %d", uint32(len(b)), s.c.MaxMessageSize())
				return resp
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
				resp.Err = err
				return resp
			}

			resp.V = svc

			// If the service status is not OK then bubble
			// that error up to the caller.
			if r, ok := svc.(ua.Response); ok {
				if status := r.Header().ServiceResult; status != ua.StatusOK {
					resp.Err = status
					return resp
				}
			}

			return resp
		}
	}
}

func (s *SecureChannel) readChunk() (*MessageChunk, error) {
	// read a full message from the underlying conn.
	b, err := s.c.Receive()
	if err == io.EOF || len(b) == 0 {
		return nil, io.EOF
	}

	if err != nil {
		return nil, errors.Errorf("sechan: read header failed: %s %#v", err, err)
	}

	const hdrlen = 12 // TODO: move to pkg level const
	h := new(Header)
	if _, err := h.Decode(b[:hdrlen]); err != nil {
		return nil, errors.Errorf("sechan: decode header failed: %s", err)
	}

	// decode the other headers
	m := new(MessageChunk)
	if _, err := m.Decode(b); err != nil {
		return nil, errors.Errorf("sechan: decode chunk failed: %s", err)
	}

	var decryptWith *channelInstance

	switch m.MessageType {
	case "OPN":
		debug.Printf("uasc OPN Request")

		// Make sure we have a valid security header
		if m.AsymmetricSecurityHeader == nil {
			return nil, ua.StatusBadDecodingError // todo(dh): check if this is the correct error
		}

		if s.openingInstance == nil {
			return nil, errors.Errorf("sechan: invalid state. openingInstance is nil.")
		}

		if m.SecurityPolicyURI != ua.SecurityPolicyURINone {
			s.cfg.RemoteCertificate = m.AsymmetricSecurityHeader.SenderCertificate
			debug.Printf("Setting securityPolicy to %s", m.SecurityPolicyURI)
		}

		s.cfg.SecurityPolicyURI = m.SecurityPolicyURI

		decryptWith = s.openingInstance
	case "CLO":
		return nil, io.EOF
	case "MSG":
		// nop
	default:
		return nil, errors.Errorf("sechan: unknown message type: %s", m.MessageType)
	}

	// Decrypt the block and put data back into m.Data
	m.Data, err = s.verifyAndDecrypt(m, b, decryptWith)
	if err != nil {
		return nil, err
	}

	n, err := m.SequenceHeader.Decode(m.Data)
	if err != nil {
		return nil, errors.Errorf("sechan: decode sequence header failed: %s", err)
	}
	m.Data = m.Data[n:]

	return m, nil
}

// verifyAndDecrypt verifies and optionally decrypts a message. if `instance` is given, then it will only use that
// state. Otherwise it will look up states by channel ID and try each.
func (s *SecureChannel) verifyAndDecrypt(m *MessageChunk, b []byte, instance *channelInstance) ([]byte, error) {
	if instance != nil {
		return instance.verifyAndDecrypt(m, b)
	}

	instances := s.getInstancesBySecureChannelID(m.MessageHeader.SecureChannelID)
	if len(instances) == 0 {
		return nil, errors.Errorf("sechan: unable to find instance for SecureChannelID=%d", m.MessageHeader.SecureChannelID)
	}

	var (
		err      error
		verified []byte
	)

	for i := len(instances) - 1; i >= 0; i-- {
		// instances[i].Lock()
		if verified, err = instances[i].verifyAndDecrypt(m, b); err == nil {
			// instances[i].Unlock()
			return verified, nil
		}
		// instances[i].Unlock()
		debug.Printf("attempting an older channel state...")
	}

	return nil, err
}

func (s *SecureChannel) getInstancesBySecureChannelID(id uint32) []*channelInstance {
	s.instancesMu.Lock()
	defer s.instancesMu.Unlock()

	instances := s.instances[id]
	if instances == nil {
		return nil
	}

	// return a copy of the slice in case a renewal is triggered
	cpy := make([]*channelInstance, len(instances))
	copy(cpy, instances)

	return instances
}

func (s *SecureChannel) LocalEndpoint() string {
	return s.endpointURL
}

func (s *SecureChannel) Open(ctx context.Context) error {
	return s.open(ctx, nil, ua.SecurityTokenRequestTypeIssue)
}

func (s *SecureChannel) open(ctx context.Context, instance *channelInstance, requestType ua.SecurityTokenRequestType) error {
	// TODO: do something with the context

	s.openingMu.Lock()
	defer s.openingMu.Unlock()

	if s.openingInstance != nil {
		return errors.Errorf("sechan: invalid state. openingInstance must be nil when opening a new secure channel.")
	}

	var (
		err       error
		localKey  *rsa.PrivateKey
		remoteKey *rsa.PublicKey
	)

	s.startDispatcher.Do(func() {
		go s.dispatcher()
	})

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
		if remoteKey, ok = remoteCert.PublicKey.(*rsa.PublicKey); !ok {
			return ua.StatusBadCertificateInvalid
		}
	}

	algo, err := uapolicy.Asymmetric(s.cfg.SecurityPolicyURI, localKey, remoteKey)
	if err != nil {
		return err
	}

	s.openingInstance = newChannelInstance(s)

	if requestType == ua.SecurityTokenRequestTypeRenew {
		// TODO: lock? sequenceNumber++?
		// this seems racy. if another request goes out while the other open request is in flight then won't an error
		// be raised on the server? can the sequenceNumber be as "global" as the request ID?
		s.openingInstance.sequenceNumber = instance.sequenceNumber
		s.openingInstance.secureChannelID = instance.secureChannelID
	}

	// trigger cleanup after we are all done
	defer func() {
		if s.openingInstance == nil || s.openingInstance.state != channelActive {
			debug.Printf("failed to open a new secure channel")
		}
		s.openingInstance = nil
	}()

	reqID := s.nextRequestID()

	s.openingInstance.algo = algo

	localNonce, err := algo.MakeNonce()
	if err != nil {
		return err
	}

	req := &ua.OpenSecureChannelRequest{
		ClientProtocolVersion: 0,
		RequestType:           requestType,
		SecurityMode:          s.cfg.SecurityMode,
		ClientNonce:           localNonce,
		RequestedLifetime:     s.cfg.Lifetime,
	}

	return s.sendRequestWithTimeout(req, reqID, s.openingInstance, nil, s.cfg.RequestTimeout, func(v interface{}) error {
		resp, ok := v.(*ua.OpenSecureChannelResponse)
		if !ok {
			return errors.Errorf("got %T, want OpenSecureChannelResponse", v)
		}
		return s.handleOpenSecureChannelResponse(resp, localNonce, s.openingInstance)
	})
}

func (s *SecureChannel) handleOpenSecureChannelResponse(resp *ua.OpenSecureChannelResponse, localNonce []byte, instance *channelInstance) (err error) {
	instance.state = channelActive
	instance.secureChannelID = resp.SecurityToken.ChannelID
	instance.securityTokenID = resp.SecurityToken.TokenID
	instance.createdAt = resp.SecurityToken.CreatedAt
	instance.revisedLifetime = time.Millisecond * time.Duration(resp.SecurityToken.RevisedLifetime)

	// allow the client to specify a lifetime that is smaller
	if int64(s.cfg.Lifetime) < int64(instance.revisedLifetime/time.Millisecond) {
		instance.revisedLifetime = time.Millisecond * time.Duration(s.cfg.Lifetime)
	}

	if instance.algo, err = uapolicy.Symmetric(s.cfg.SecurityPolicyURI, localNonce, resp.ServerNonce); err != nil {
		return err
	}

	s.instancesMu.Lock()
	defer s.instancesMu.Unlock()

	if _, ok := s.instances[resp.SecurityToken.ChannelID]; ok {
		// since there's already an existing entry for this SecureChannelID it means we are in a renewal
		s.instances[resp.SecurityToken.ChannelID] = append(
			s.instances[resp.SecurityToken.ChannelID],
			s.openingInstance,
		)
	} else {
		s.instances[resp.SecurityToken.ChannelID] = []*channelInstance{s.openingInstance}
	}

	s.activeInstance = instance

	debug.Printf("received security token: channelID=%d tokenID=%d createdAt=%s lifetime=%s", instance.secureChannelID, instance.securityTokenID, instance.createdAt.Format(time.RFC3339), instance.revisedLifetime)

	if s.cfg.SecurityMode != ua.MessageSecurityModeNone {
		go s.scheduleRenewal(instance)
		go s.scheduleExpiration(instance)
	}

	return
}

func (s *SecureChannel) scheduleRenewal(instance *channelInstance) {
	// https://reference.opcfoundation.org/v104/Core/docs/Part4/5.5.2/#5.5.2.1
	// Clients should request a new SecurityToken after 75 % of its lifetime has elapsed. This should ensure that
	// clients will receive the new SecurityToken before the old one actually expire
	const renewAfter = 0.75
	when := time.Second * time.Duration(instance.revisedLifetime.Seconds()*renewAfter)

	debug.Printf("channelID %d will be refreshed in %s (%s)", instance.secureChannelID, when, time.Now().UTC().Add(when).Format(time.RFC3339))

	t := time.NewTimer(when)
	defer t.Stop()

	s.closingMu.RLock()
	defer s.closingMu.RUnlock()

	select {
	case <-s.closing:
		return
	case <-t.C:
	}

	// TODO: where should this error go?
	_ = s.renew(instance)
}

func (s *SecureChannel) renew(instance *channelInstance) error {
	// lock ensure no one else renews this at the same time
	instance.Lock()
	defer instance.Unlock()

	return s.open(context.Background(), instance, ua.SecurityTokenRequestTypeRenew)
}

func (s *SecureChannel) scheduleExpiration(instance *channelInstance) {
	// https://reference.opcfoundation.org/v104/Core/docs/Part4/5.5.2/#5.5.2.1
	// Clients should accept Messages secured by an expired SecurityToken for up to 25 % of the token lifetime.
	const expireAfter = 1.25
	when := instance.createdAt.Add(time.Second * time.Duration(instance.revisedLifetime.Seconds()*expireAfter))

	debug.Printf("channelID %d/%d will expire at %s", instance.secureChannelID, instance.securityTokenID, when.UTC().Format(time.RFC3339))

	t := time.NewTimer(time.Until(when))

	s.closingMu.RLock()
	defer s.closingMu.RUnlock()

	select {
	case <-s.closing:
		return
	case <-t.C:
	}

	s.instancesMu.Lock()
	defer s.instancesMu.Unlock()

	oldInstances := s.instances[instance.securityTokenID]

	s.instances[instance.securityTokenID] = []*channelInstance{}

	for _, oldInstance := range oldInstances {
		if oldInstance.secureChannelID != instance.secureChannelID {
			// something has gone horribly wrong!
			debug.Printf("secureChannelID mismatch during scheduleExpiration!")
		}
		if oldInstance.securityTokenID == instance.securityTokenID {
			continue
		}
		s.instances[instance.securityTokenID] = append(
			s.instances[instance.securityTokenID],
			oldInstance,
		)
	}
}

func (s *SecureChannel) sendRequestWithTimeout(
	req ua.Request,
	reqID uint32,
	instance *channelInstance,
	authToken *ua.NodeID,
	timeout time.Duration,
	h func(interface{}) error) error {

	respRequired := h != nil

	ch, err := s.sendAsyncWithTimeout(req, reqID, instance, authToken, respRequired, timeout)
	if err != nil {
		return err
	}

	if !respRequired {
		return nil
	}

	// `+ timeoutLeniency` to give the server a chance to respond to TimeoutHint
	timer := time.NewTimer(timeout + timeoutLeniency)
	defer timer.Stop()

	select {
	case resp := <-ch:
		if resp.Err != nil {
			if resp.V != nil {
				_ = h(resp.V) // ignore result because resp.Err takes precedence
			}
			return resp.Err
		}
		return h(resp.V)
	case <-timer.C:
		s.popHandler(reqID)
		return ua.StatusBadTimeout
	}
}

func (s *SecureChannel) popHandler(reqID uint32) (chan *response, bool) {
	s.handlersMu.Lock()
	defer s.handlersMu.Unlock()

	ch, ok := s.handlers[reqID]
	if ok {
		delete(s.handlers, reqID)
	}
	return ch, ok
}

func (s *SecureChannel) Renew(ctx context.Context) error {
	instance, err := s.getActiveChannelInstance()
	if err != nil {
		return err
	}

	return s.renew(instance)
}

// SendRequest sends the service request and calls h with the response.
func (s *SecureChannel) SendRequest(req ua.Request, authToken *ua.NodeID, h func(interface{}) error) error {
	return s.SendRequestWithTimeout(req, authToken, s.cfg.RequestTimeout, h)
}

func (s *SecureChannel) SendRequestWithTimeout(req ua.Request, authToken *ua.NodeID, timeout time.Duration, h func(interface{}) error) error {
	active, err := s.getActiveChannelInstance()
	if err != nil {
		return err
	}

	return s.sendRequestWithTimeout(req, s.nextRequestID(), active, authToken, timeout, h)
}

func (s *SecureChannel) sendAsyncWithTimeout(
	req ua.Request,
	reqID uint32,
	instance *channelInstance,
	authToken *ua.NodeID,
	respRequired bool,
	timeout time.Duration,
) (<-chan *response, error) {

	instance.Lock()

	m, err := instance.newRequestMessage(req, reqID, authToken, timeout)
	if err != nil {
		instance.Unlock()
		return nil, err
	}

	b, err := m.Encode()
	if err != nil {
		instance.Unlock()
		return nil, err
	}

	b, err = instance.signAndEncrypt(m, b)
	if err != nil {
		instance.Unlock()
		return nil, err
	}

	instance.Unlock()

	var resp chan *response

	if respRequired {
		// register the handler if a callback was passed
		resp = make(chan *response, 1)

		s.handlersMu.Lock()

		if s.handlers[reqID] != nil {
			s.handlersMu.Unlock()
			return nil, errors.Errorf("error: duplicate handler registration for request id %d", reqID)
		}

		s.handlers[reqID] = resp
		s.handlersMu.Unlock()
	}

	// send the message
	var n int
	if n, err = s.c.Write(b); err != nil {
		return nil, err
	}

	atomic.AddUint64(&instance.bytesSent, uint64(n))
	atomic.AddUint32(&instance.messagesSent, 1)

	debug.Printf("uasc %d/%d: send %T with %d bytes", s.c.ID(), reqID, req, len(b))

	return resp, nil
}

func (s *SecureChannel) nextRequestID() uint32 {
	s.requestIDMu.Lock()
	defer s.requestIDMu.Unlock()

	s.requestID++
	if s.requestID == 0 {
		s.requestID = 1
	}

	return s.requestID
}

// Close closes an existing secure channel
func (s *SecureChannel) Close() error {
	debug.Printf("uasc Close()")

	defer func() {
		close(s.closing)
		s.reset()
	}()

	err := s.SendRequest(&ua.CloseSecureChannelRequest{}, nil, nil)

	if err != nil {
		return err
	}

	return io.EOF
}

func (s *SecureChannel) timeNow() time.Time {
	if s.time != nil {
		return s.time()
	}
	return time.Now()
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

	var (
		b     []byte
		seqnr uint32
	)

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
