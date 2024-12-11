// Copyright 2018-2020 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uasc

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/binary"
	"io"
	"log"
	"math"
	"net"
	"strings"
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

type channelKind int

const (
	client channelKind = iota
	server
)

// ResponseHandler handles the response of a service request and
// is used by the client.
type ResponseHandler func(ua.Response) error

// MessageBody is the content of a secure channel message sent
// between a client and a server and represents a service
// request or response.
type MessageBody struct {
	RequestID       uint32
	SecureChannelID uint32
	Err             error

	body any
}

func (b MessageBody) Request() ua.Request {
	if x, ok := b.body.(ua.Request); ok {
		return x
	}
	return nil
}

func (b MessageBody) Response() ua.Response {
	if x, ok := b.body.(ua.Response); ok {
		return x
	}
	return nil
}

type conditionLocker struct {
	bLock   bool
	lockMu  sync.Mutex
	lockCnd *sync.Cond
}

func newConditionLocker() *conditionLocker {
	c := &conditionLocker{}
	c.lockCnd = sync.NewCond(&c.lockMu)
	return c
}

func (c *conditionLocker) lock() {
	c.lockMu.Lock()
	c.bLock = true
	c.lockMu.Unlock()
}

func (c *conditionLocker) unlock() {
	c.lockMu.Lock()
	c.bLock = false
	c.lockMu.Unlock()
	c.lockCnd.Broadcast()
}

func (c *conditionLocker) waitIfLock() {
	c.lockMu.Lock()
	for c.bLock {
		c.lockCnd.Wait()
	}
	c.lockMu.Unlock()
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
	closing      chan struct{}
	disconnected chan struct{}

	// startDispatcher ensures only one dispatcher is running
	startDispatcher sync.Once

	// requestID is a "global" counter shared between multiple channels and tokens
	requestID   uint32
	requestIDMu sync.Mutex

	// instances maps secure channel IDs to a list to channel states
	instances      map[uint32][]*channelInstance
	activeInstance *channelInstance
	instancesMu    sync.Mutex

	// prevent sending msg when secure channel renewal occurs
	reqLocker  *conditionLocker
	rcvLocker  *conditionLocker
	pendingReq sync.WaitGroup

	// handles maps requestIDs to response channels
	handlers   map[uint32]chan *MessageBody
	handlersMu sync.Mutex

	// chunks maintains a temporary list of chunks for a given request ID
	chunks   map[uint32][]*MessageChunk
	chunksMu sync.Mutex

	// openingInstance is a temporary var that allows the dispatcher know how to handle a open channel request
	// note: we only allow a single "open" request in flight at any point in time. The mutex is held for the entire
	// duration of the "open" request.
	openingInstance *channelInstance
	openingMu       sync.Mutex

	// errorCh receive dispatcher errors
	errch chan<- error

	// required for the server channel

	// // secureChannelID is a unique identifier for the SecureChannel assigned by the Server.
	// // If a Server receives a SecureChannelId which it does not recognize it shall return an
	// // appropriate transport layer error.
	// //
	// // When a Server starts the first SecureChannelId used should be a value that is likely to
	// // be unique after each restart. This ensures that a Server restart does not cause
	// // previously connected Clients to accidentally ‘reuse’ SecureChannels that did not belong
	// // to them.
	// secureChannelID uint32

	// // sequenceNumber is a monotonically increasing sequence number assigned by the sender to each
	// // MessageChunk sent over the SecureChannel.
	// sequenceNumber uint32

	// // securityTokenID is a unique identifier for the SecureChannel SecurityToken used to secure the Message.
	// // This identifier is returned by the Server in an OpenSecureChannel response Message.
	// // If a Server receives a TokenId which it does not recognize it shall return an appropriate
	// // transport layer error.
	// securityTokenID uint32

	// kind indicates whether this is a server or a client channel.
	kind channelKind

	closeOnce sync.Once
}

func NewSecureChannel(endpoint string, c *uacp.Conn, cfg *Config, errCh chan<- error) (*SecureChannel, error) {
	return newSecureChannel(endpoint, c, cfg, client, errCh, 0, 0, 0)
}

func NewServerSecureChannel(endpoint string, c *uacp.Conn, cfg *Config, errCh chan<- error, secureChannelID, sequenceNumber, securityTokenID uint32) (*SecureChannel, error) {
	s, err := newSecureChannel(endpoint, c, cfg, server, errCh, secureChannelID, sequenceNumber, securityTokenID)
	if err != nil {
		return nil, err
	}

	s.openingInstance = newChannelInstance(s)
	s.openingInstance.secureChannelID = secureChannelID
	s.openingInstance.sequenceNumber = sequenceNumber
	s.openingInstance.securityTokenID = securityTokenID

	return s, nil
}

func newSecureChannel(endpoint string, c *uacp.Conn, cfg *Config, kind channelKind, errCh chan<- error, secureChannelID, sequenceNumber, securityTokenID uint32) (*SecureChannel, error) {
	if c == nil {
		return nil, errors.Errorf("no connection")
	}

	if cfg == nil {
		return nil, errors.Errorf("no secure channel config")
	}

	if errCh == nil {
		return nil, errors.Errorf("no error channel")
	}

	switch {
	case cfg.SecurityPolicyURI == ua.SecurityPolicyURINone && cfg.SecurityMode != ua.MessageSecurityModeNone:
		return nil, errors.Errorf("invalid channel config: Security policy '%s' cannot be used with '%s'", cfg.SecurityPolicyURI, cfg.SecurityMode)
	case cfg.SecurityPolicyURI != ua.SecurityPolicyURINone && cfg.SecurityMode == ua.MessageSecurityModeNone:
		return nil, errors.Errorf("invalid channel config: Security policy '%s' cannot be used with '%s'", cfg.SecurityPolicyURI, cfg.SecurityMode)
	case cfg.SecurityPolicyURI != ua.SecurityPolicyURINone && cfg.LocalKey == nil:
		return nil, errors.Errorf("invalid channel config: Security policy '%s' requires a private key", cfg.SecurityPolicyURI)
	}

	s := &SecureChannel{
		endpointURL: endpoint,
		c:           c,
		cfg:         cfg,
		requestID:   cfg.RequestIDSeed,
		kind:        kind,
		// secureChannelID: secureChannelID,
		// sequenceNumber:  sequenceNumber,
		// securityTokenID: securityTokenID,
		reqLocker:    newConditionLocker(),
		rcvLocker:    newConditionLocker(),
		errch:        errCh,
		closing:      make(chan struct{}),
		disconnected: make(chan struct{}),
		instances:    make(map[uint32][]*channelInstance),
		chunks:       make(map[uint32][]*MessageChunk),
		handlers:     make(map[uint32]chan *MessageBody),
	}

	return s, nil
}

func (s *SecureChannel) RemoteAddr() net.Addr {
	return s.c.TCPConn.RemoteAddr()
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

	defer func() {
		close(s.disconnected)
	}()

	for {
		select {
		case <-s.closing:
			return
		default:
			msg := s.Receive(ctx)
			if msg.Err != nil {
				select {
				case <-s.closing:
					return
				default:
					select {
					case s.errch <- msg.Err:
					default:
					}
				}
			}

			if msg.Err == io.EOF {
				return
			}

			if msg.Err != nil {
				debug.Printf("uasc %d/%d: err: %v", s.c.ID(), msg.RequestID, msg.Err)
			} else {
				debug.Printf("uasc %d/%d: recv %T", s.c.ID(), msg.RequestID, msg.body)
			}

			ch, ok := s.popHandler(msg.RequestID)

			if !ok {
				debug.Printf("uasc %d/%d: no handler for %T", s.c.ID(), msg.RequestID, msg.body)
				continue
			}

			// HACK
			if _, ok := msg.Response().(*ua.OpenSecureChannelResponse); ok {
				s.rcvLocker.lock()
			}

			debug.Printf("uasc %d/%d: sending %T to handler", s.c.ID(), msg.RequestID, msg.body)
			select {
			case ch <- msg:
			default:
				// this should never happen since the chan is of size one
				debug.Printf("uasc %d/%d: unexpected state. channel write should always succeed.", s.c.ID(), msg.RequestID)
			}

			s.rcvLocker.waitIfLock()
		}
	}
}

// Receive receives message chunks from the secure channel, decodes and forwards
// them to the registered callback channel, if there is one. Otherwise,
// the message is dropped.
func (s *SecureChannel) Receive(ctx context.Context) *MessageBody {
	for {
		select {
		case <-ctx.Done():
			return &MessageBody{Err: ctx.Err()}

		default:
			chunk, err := s.readChunk()
			if err == io.EOF {
				debug.Printf("uasc %d: readChunk EOF", s.c.ID())
				return &MessageBody{Err: err}
			}

			if err != nil {
				return &MessageBody{Err: err}
			}

			hdr := chunk.Header
			reqID := chunk.SequenceHeader.RequestID

			strdat := string(chunk.Data)
			if strings.Contains(strdat, "CurrentTime") {
				debug.Printf("Requested CurrentTime.")
			}

			msg := &MessageBody{
				RequestID:       reqID,
				SecureChannelID: chunk.MessageHeader.Header.SecureChannelID,
			}

			debug.Printf("uasc %d/%d: recv %s%c with %d bytes", s.c.ID(), reqID, hdr.MessageType, hdr.ChunkType, hdr.MessageSize)

			s.chunksMu.Lock()

			switch hdr.ChunkType {
			case 'A':
				delete(s.chunks, reqID)
				s.chunksMu.Unlock()

				msga := new(MessageAbort)
				if _, err := msga.Decode(chunk.Data); err != nil {
					debug.Printf("uasc %d/%d: invalid MSGA chunk. %s", s.c.ID(), reqID, err)
					msg.Err = ua.StatusBadDecodingError
					return msg
				}

				return &MessageBody{RequestID: reqID, Err: ua.StatusCode(msga.ErrorCode)}

			case 'C':
				s.chunks[reqID] = append(s.chunks[reqID], chunk)
				if n := len(s.chunks[reqID]); uint32(n) > s.c.MaxChunkCount() {
					delete(s.chunks, reqID)
					s.chunksMu.Unlock()
					msg.Err = errors.Errorf("too many chunks: %d > %d", n, s.c.MaxChunkCount())
					return msg
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
				msg.Err = err
				return msg
			}

			if uint32(len(b)) > s.c.MaxMessageSize() {
				msg.Err = errors.Errorf("message too large: %d > %d", uint32(len(b)), s.c.MaxMessageSize())
				return msg
			}

			// todo(fs): Why are we talking about ResponseHeaders here?
			// todo(fs): this should apply to both requests and responses

			// since we are not decoding the ResponseHeader separately
			// we need to drop every message that has an error since we
			// cannot get to the RequestHandle in the ResponseHeader.
			// To fix this we must a) decode the ResponseHeader separately
			// and subsequently remove it and the TypeID from all service
			// structs and tests. We also need to add a deadline to all
			// handlers and check them periodically to time them out.
			_, body, err := ua.DecodeService(b)
			if err != nil {
				msg.Err = err
				return msg
			}

			msg.body = body

			// todo(fs): not sure this is correct
			if req, ok := msg.Request().(*ua.OpenSecureChannelRequest); ok {
				err := s.handleOpenSecureChannelRequest(reqID, req)
				if err != nil {
					debug.Printf("uasc %d/%d: handling %T failed: %v", s.c.ID(), reqID, req, err)
					return &MessageBody{Err: err}
				}
				return &MessageBody{}
			}

			// If the service status is not OK then bubble
			// that error up to the caller.
			if resp := msg.Response(); resp != nil {
				if status := resp.Header().ServiceResult; status != ua.StatusOK {
					msg.Err = status
					return msg
				}
			}

			return msg
		}
	}
}

func (s *SecureChannel) readChunk() (*MessageChunk, error) {
	// read a full message from the underlying conn.
	buf := uacp.AllocBuffer(int(s.c.ReceiveBufSize()))
	defer uacp.FreeBuffer(buf)
	b, err := s.c.Receive(buf.Bytes())
	if err == io.EOF || len(b) == 0 {
		return nil, io.EOF
	}
	// do not wrap this error since it hides conn error
	var uacperr *uacp.Error
	if errors.As(err, &uacperr) {
		return nil, err
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
		// Make sure we have a valid security header
		if m.AsymmetricSecurityHeader == nil {
			return nil, ua.StatusBadDecodingError // todo(dh): check if this is the correct error
		}

		if s.openingInstance == nil {
			return nil, errors.Errorf("sechan: invalid state. openingInstance is nil.")
		}

		s.cfg.SecurityPolicyURI = m.SecurityPolicyURI
		if m.SecurityPolicyURI != ua.SecurityPolicyURINone {
			s.cfg.RemoteCertificate = m.AsymmetricSecurityHeader.SenderCertificate
			debug.Printf("uasc %d: setting securityPolicy to %s", s.c.ID(), m.SecurityPolicyURI)

			// TODO: where does this need to actually be set?  It should be independent of the securitypolicy
			// but here I'm just assuming it is sign and encrypt when we have anything other than None
			// The problem is we need to know whether the message has to be decrypted before we can look at the
			// security mode field in the open request to see if it is encrypted!
			s.cfg.SecurityMode = ua.MessageSecurityModeSignAndEncrypt
			remoteCert, err := x509.ParseCertificate(s.cfg.RemoteCertificate)
			if err != nil {
				return nil, err
			}
			remoteKey, ok := remoteCert.PublicKey.(*rsa.PublicKey)
			if !ok {
				return nil, ua.StatusBadCertificateInvalid
			}
			algo, err := uapolicy.Asymmetric(s.cfg.SecurityPolicyURI, s.openingInstance.sc.cfg.LocalKey, remoteKey)
			if err != nil {
				return nil, err
			}

			s.openingInstance.algo = algo
		}

		decryptWith = s.openingInstance
	case "CLO":
		return nil, io.EOF
	case "MSG":
		// noop
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
		if verified, err = instances[i].verifyAndDecrypt(m, b); err == nil {
			return verified, nil
		}
		debug.Printf("uasc %d: attempting an older channel state...", s.c.ID())
	}

	return nil, err
}

func (s *SecureChannel) getInstancesBySecureChannelID(id uint32) []*channelInstance {
	s.instancesMu.Lock()
	defer s.instancesMu.Unlock()

	instances := s.instances[id]
	if len(instances) == 0 {
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
	debug.Printf("sc.open")
	defer s.rcvLocker.unlock()

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
	// s.openingInstance.secureChannelID = s.secureChannelID
	// s.openingInstance.sequenceNumber = s.sequenceNumber
	// s.openingInstance.securityTokenID = s.securityTokenID

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
			debug.Printf("uasc %d: failed to open a new secure channel", s.c.ID())
		}
		s.openingInstance = nil
	}()

	reqID := s.nextRequestID()

	s.openingInstance.algo = algo
	s.openingInstance.SetMaximumBodySize(int(s.c.SendBufSize()))

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

	return s.sendRequestWithTimeout(ctx, req, reqID, s.openingInstance, nil, s.cfg.RequestTimeout, func(v ua.Response) error {
		debug.Printf("OpenSecureChannelResponse handler")
		resp, ok := v.(*ua.OpenSecureChannelResponse)
		if !ok {
			return errors.Errorf("got %T, want OpenSecureChannelResponse", v)
		}
		return s.handleOpenSecureChannelResponse(resp, localNonce, s.openingInstance)
	})
}

func (s *SecureChannel) handleOpenSecureChannelResponse(resp *ua.OpenSecureChannelResponse, localNonce []byte, instance *channelInstance) (err error) {
	debug.Printf("sc.handleOpenSecureChannelResponse")
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

	instance.SetMaximumBodySize(int(s.c.SendBufSize()))

	s.instancesMu.Lock()
	defer s.instancesMu.Unlock()

	s.instances[resp.SecurityToken.ChannelID] = append(
		s.instances[resp.SecurityToken.ChannelID],
		s.openingInstance,
	)

	s.activeInstance = instance

	debug.Printf("uasc %d: received security token. channelID=%d tokenID=%d createdAt=%s lifetime=%s", s.c.ID(), instance.secureChannelID, instance.securityTokenID, instance.createdAt.Format(time.RFC3339), instance.revisedLifetime)

	// depending on whether the channel is used in a client
	// or a server we need to trigger different behavior.
	// client channels trigger token renewals and need to cleanup old
	// channel crypto configs. server channels only need to do the
	// channel cleanup.
	switch s.kind {
	case client:
		go s.scheduleRenewal(instance)
		go s.scheduleExpiration(instance)

	case server:
		go s.scheduleExpiration(instance)
	}

	return
}

func (s *SecureChannel) handleOpenSecureChannelRequest(reqID uint32, svc ua.Request) error {
	debug.Printf("handleOpenSecureChannelRequest: Got OPN Request\n")

	var err error

	req, ok := svc.(*ua.OpenSecureChannelRequest)
	if !ok {
		debug.Printf("Expected OpenSecureChannel Request, got %T\n", svc)
	}

	// Part 6.7.4: https://reference.opcfoundation.org/Core/Part6/v105/docs/6.7.4
	// todo(fs): check that ClientProtocolVersion matches HELLO.Version
	// todo(fs): respond with Bad_ProtocolVersionUnsupported if they don't match
	// todo(fs): also make sure that ServerProtocolVersion in the response is the
	// todo(fs): the same as ACK.Version. Return the error if we don't support
	// todo(fs): the client protocol version.
	// todo(fs): how do we get the HELLO.Version here?
	// if hello.Version != req.ClientProtocolVersion || req.ClientProtocolVersion != 0 {
	if req.ClientProtocolVersion != 0 {
		return ua.StatusBadProtocolVersionUnsupported
	}

	// Part 6.7.4: The AuthenticationToken should be nil. ???
	if req.RequestHeader.AuthenticationToken.IntID() != 0 {
		return ua.StatusBadSecureChannelTokenUnknown
	}

	s.cfg.Lifetime = req.RequestedLifetime
	s.cfg.SecurityMode = req.SecurityMode

	// I had to do the encryption setup in the chunk decoding logic because you have to
	// decrypt the thing before you even know you have an open message.
	// so this is redundant.
	var (
		localKey  *rsa.PrivateKey
		remoteKey *rsa.PublicKey
	)

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

	instance := s.openingInstance
	instance.algo = algo
	instance.sc.requestID = req.RequestHeader.RequestHandle // todo(fs): is this correct?

	nonce := make([]byte, instance.algo.NonceLength())
	if _, err := rand.Read(nonce); err != nil {
		return err
	}
	resp := &ua.OpenSecureChannelResponse{
		ResponseHeader: &ua.ResponseHeader{
			Timestamp:          s.timeNow(),
			RequestHandle:      req.RequestHeader.RequestHandle,
			ServiceDiagnostics: &ua.DiagnosticInfo{},
			StringTable:        []string{},
			AdditionalHeader:   ua.NewExtensionObject(nil),
		},
		ServerProtocolVersion: 0,
		SecurityToken: &ua.ChannelSecurityToken{
			ChannelID:       instance.secureChannelID,
			TokenID:         instance.securityTokenID,
			CreatedAt:       s.timeNow(),
			RevisedLifetime: req.RequestedLifetime,
		},
		ServerNonce: nonce,
	}

	ctx := context.Background() // todo(fs): fixme
	if err := s.sendResponseWithContext(ctx, instance, reqID, resp); err != nil {
		return err
	}

	instance.algo, err = uapolicy.Symmetric(s.cfg.SecurityPolicyURI, nonce, req.ClientNonce)
	if err != nil {
		return err
	}

	instance.state = channelActive // todo(fs): is this correct?
	// s.setState(secureChannelOpen)

	s.instancesMu.Lock()
	s.instances[instance.secureChannelID] = append(
		s.instances[instance.secureChannelID],
		instance,
	)
	s.activeInstance = instance
	s.instancesMu.Unlock()

	return nil
}

func (s *SecureChannel) scheduleRenewal(instance *channelInstance) {
	// https://reference.opcfoundation.org/v104/Core/docs/Part4/5.5.2/#5.5.2.1
	// Clients should request a new SecurityToken after 75 % of its lifetime has elapsed. This should ensure that
	// clients will receive the new SecurityToken before the old one actually expire
	const renewAfter = 0.75
	when := time.Second * time.Duration(instance.revisedLifetime.Seconds()*renewAfter)

	debug.Printf("uasc %d: security token is refreshed at %s (%s). channelID=%d tokenID=%d", s.c.ID(), time.Now().UTC().Add(when).Format(time.RFC3339), when, instance.secureChannelID, instance.securityTokenID)

	t := time.NewTimer(when)
	defer t.Stop()

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
	s.reqLocker.lock()
	defer s.reqLocker.unlock()
	s.pendingReq.Wait()
	instance.Lock()
	defer instance.Unlock()

	return s.open(context.Background(), instance, ua.SecurityTokenRequestTypeRenew)
}

func (s *SecureChannel) scheduleExpiration(instance *channelInstance) {
	// https://reference.opcfoundation.org/v104/Core/docs/Part4/5.5.2/#5.5.2.1
	// Clients should accept Messages secured by an expired SecurityToken for up to 25 % of the token lifetime.
	const expireAfter = 1.25
	when := instance.createdAt.Add(time.Second * time.Duration(instance.revisedLifetime.Seconds()*expireAfter))

	debug.Printf("uasc %d: security token expires at %s. channelID=%d tokenID=%d", s.c.ID(), when.UTC().Format(time.RFC3339), instance.secureChannelID, instance.securityTokenID)

	t := time.NewTimer(time.Until(when))
	defer t.Stop()

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
			debug.Printf("uasc %d: secureChannelID mismatch during scheduleExpiration!", s.c.ID())
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
	ctx context.Context,
	req ua.Request,
	reqID uint32,
	instance *channelInstance,
	authToken *ua.NodeID,
	timeout time.Duration,
	h ResponseHandler) error {

	s.pendingReq.Add(1)
	respRequired := h != nil

	ch, err := s.sendAsyncWithTimeout(ctx, req, reqID, instance, authToken, respRequired, timeout)
	s.pendingReq.Done()
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
	case <-ctx.Done():
		s.popHandler(reqID)
		return ctx.Err()
	case <-s.disconnected:
		s.popHandler(reqID)
		return io.EOF
	case msg := <-ch:
		if msg.Err != nil {
			if msg.Response() != nil {
				_ = h(msg.Response()) // ignore result because msg.Err takes precedence
			}
			return msg.Err
		}
		return h(msg.Response())
	case <-timer.C:
		s.popHandler(reqID)
		return ua.StatusBadTimeout
	}
}

func (s *SecureChannel) popHandler(reqID uint32) (chan *MessageBody, bool) {
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

func (s *SecureChannel) SendRequest(ctx context.Context, req ua.Request, authToken *ua.NodeID, h ResponseHandler) error {
	// SendRequest sends the service request and calls h with the response.
	return s.SendRequestWithTimeout(ctx, req, authToken, s.cfg.RequestTimeout, h)
}

func (s *SecureChannel) SendRequestWithTimeout(ctx context.Context, req ua.Request, authToken *ua.NodeID, timeout time.Duration, h ResponseHandler) error {
	s.reqLocker.waitIfLock()
	active, err := s.getActiveChannelInstance()
	if err != nil {
		return err
	}

	return s.sendRequestWithTimeout(ctx, req, s.nextRequestID(), active, authToken, timeout, h)
}

func (s *SecureChannel) sendAsyncWithTimeout(
	ctx context.Context,
	req ua.Request,
	reqID uint32,
	instance *channelInstance,
	authToken *ua.NodeID,
	respRequired bool,
	timeout time.Duration,
) (<-chan *MessageBody, error) {

	instance.Lock()
	defer instance.Unlock()

	m, err := instance.newRequestMessage(req, reqID, authToken, timeout)
	if err != nil {
		return nil, err
	}

	var resp chan *MessageBody

	if respRequired {
		// register the handler if a callback was passed
		resp = make(chan *MessageBody, 1)

		s.handlersMu.Lock()

		if s.handlers[reqID] != nil {
			s.handlersMu.Unlock()
			return nil, errors.Errorf("error: duplicate handler registration for request id %d", reqID)
		}

		s.handlers[reqID] = resp
		s.handlersMu.Unlock()
	}

	chunks, err := m.EncodeChunks(instance.maxBodySize)
	if err != nil {
		return nil, err
	}

	for i, chunk := range chunks {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		if i > 0 { // fix sequence number on subsequent chunks
			number := instance.nextSequenceNumber()
			log.Printf("burning sequence number %d", number)
			binary.LittleEndian.PutUint32(chunk[16:], uint32(number))
		}

		chunk, err = instance.signAndEncrypt(m, chunk)
		if err != nil {
			return nil, err
		}

		// send the message
		var n int
		if n, err = s.c.Write(chunk); err != nil {
			return nil, err
		}

		atomic.AddUint64(&instance.bytesSent, uint64(n))
		atomic.AddUint32(&instance.messagesSent, 1)

		debug.Printf("uasc %d/%d: send %T with %d bytes", s.c.ID(), reqID, req, len(chunk))
	}

	return resp, nil
}

func (s *SecureChannel) SendResponseWithContext(ctx context.Context, reqID uint32, resp ua.Response) error {
	return s.sendResponseWithContext(ctx, nil, reqID, resp)
}

func (s *SecureChannel) SendMsgWithContext(ctx context.Context, instance *channelInstance, reqID uint32, resp any) error {
	typeID := ua.ServiceTypeID(resp)
	if typeID == 0 {
		return errors.Errorf("uasc: unknown service %T. Did you call register?", resp)
	}

	var err error
	if instance == nil {
		instance, err = s.getActiveChannelInstance()
		if err != nil {
			return err
		}
	}

	// we need to get a lock on the sequence number so we are sure to send them in the correct order.
	// encode the message
	m := instance.newMessage(resp, typeID, reqID)
	b, err := m.Encode()
	if err != nil {
		return err
	}

	// encrypt the message prior to sending it
	// if SecurityMode == None, this returns the byte stream untouched
	b, err = instance.signAndEncrypt(m, b)
	if err != nil {
		return err
	}

	// send the message
	n, err := s.c.Write(b)
	if err != nil {
		return err
	}

	// todo(fs): what if len(b) != n? Can this happen?
	if len(b) != n {
		return errors.Errorf("uasc: incomplete message %T sent len=%d sent=%d", resp, len(b), n)
	}

	atomic.AddUint64(&instance.bytesSent, uint64(n))
	atomic.AddUint32(&instance.messagesSent, 1)

	debug.Printf("uasc %d/%d: send %T with %d bytes", s.c.ID(), reqID, resp, len(b))

	return nil
}

func (s *SecureChannel) sendResponseWithContext(ctx context.Context, instance *channelInstance, reqID uint32, resp ua.Response) error {
	typeID := ua.ServiceTypeID(resp)
	if typeID == 0 {
		return errors.Errorf("uasc: unknown service %T. Did you call register?", resp)
	}

	var err error
	if instance == nil {
		instance, err = s.getActiveChannelInstance()
		if err != nil {
			return err
		}
	}
	instance.Lock()
	defer instance.Unlock()

	// encode the message
	m := instance.newMessage(resp, typeID, reqID)
	b, err := m.Encode()
	if err != nil {
		log.Printf("Error encoding msg: %v", err)
		return err
	}

	// encrypt the message prior to sending it
	// if SecurityMode == None, this returns the byte stream untouched
	b, err = instance.signAndEncrypt(m, b)
	if err != nil {
		return err
	}

	// send the message
	n, err := s.c.Write(b)
	if err != nil {
		return err
	}

	// todo(fs): what if len(b) != n? Can this happen?
	if len(b) != n {
		return errors.Errorf("uasc: incomplete message %T sent len=%d sent=%d", resp, len(b), n)
	}

	atomic.AddUint64(&instance.bytesSent, uint64(n))
	atomic.AddUint32(&instance.messagesSent, 1)

	debug.Printf("uasc %d/%d: send %T with %d bytes", s.c.ID(), reqID, resp, len(b))

	return nil
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
func (s *SecureChannel) Close() (err error) {
	// https://github.com/gopcua/opcua/pull/470
	// guard against double close until we found the root cause
	err = io.EOF
	s.closeOnce.Do(func() { err = s.close() })
	return
}

func (s *SecureChannel) close() error {
	debug.Printf("uasc %d: Close()", s.c.ID())

	defer func() {
		close(s.closing)
	}()

	s.reqLocker.unlock()
	s.rcvLocker.unlock()

	select {
	case <-s.disconnected:
		return io.EOF
	default:
	}

	err := s.SendRequest(context.Background(), &ua.CloseSecureChannelRequest{}, nil, nil)
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
