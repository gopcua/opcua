// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package opcua

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"log"
	"reflect"
	"sort"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/gopcua/opcua/debug"
	"github.com/gopcua/opcua/errors"
	"github.com/gopcua/opcua/id"
	"github.com/gopcua/opcua/ua"
	"github.com/gopcua/opcua/uacp"
	"github.com/gopcua/opcua/uasc"
)

// GetEndpoints returns the available endpoint descriptions for the server.
func GetEndpoints(endpoint string) ([]*ua.EndpointDescription, error) {
	c := NewClient(endpoint)
	if err := c.Dial(context.Background()); err != nil {
		return nil, err
	}
	defer c.Close()
	res, err := c.GetEndpoints()
	if err != nil {
		return nil, err
	}
	return res.Endpoints, nil
}

// SelectEndpoint returns the endpoint with the highest security level which matches
// security policy and security mode. policy and mode can be omitted so that
// only one of them has to match.
// todo(fs): should this function return an error?
func SelectEndpoint(endpoints []*ua.EndpointDescription, policy string, mode ua.MessageSecurityMode) *ua.EndpointDescription {
	if len(endpoints) == 0 {
		return nil
	}

	sort.Sort(bySecurityLevel(endpoints))
	policy = ua.FormatSecurityPolicyURI(policy)

	// don't care -> return highest security level
	if policy == "" && mode == ua.MessageSecurityModeInvalid {
		return endpoints[0]
	}

	for _, p := range endpoints {
		// match only security mode
		if policy == "" && p.SecurityMode == mode {
			return p
		}

		// match only security policy
		if p.SecurityPolicyURI == policy && mode == ua.MessageSecurityModeInvalid {
			return p
		}

		// match both
		if p.SecurityPolicyURI == policy && p.SecurityMode == mode {
			return p
		}
	}
	return nil
}

type bySecurityLevel []*ua.EndpointDescription

func (a bySecurityLevel) Len() int           { return len(a) }
func (a bySecurityLevel) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a bySecurityLevel) Less(i, j int) bool { return a[i].SecurityLevel < a[j].SecurityLevel }

type ClientNotification uint

const (
	ClientNotificationStartReconnection ClientNotification = iota
	ClientNotificationReconnection
	ClientNotificationReconnectionCancel
	ClientNotificationReconnectionAttemptHasFailed
	ClientNotificationAfterReconnection
	ClientNotificationConnectionReestablished
	ClientNotificationConnectionLost
)

// Client is a high-level client for an OPC/UA server.
// It establishes a secure channel and a session.
type Client struct {
	// endpointURL is the endpoint URL the client connects to.
	endpointURL string

	// cfg is the configuration for the secure channel.
	cfg *uasc.Config

	// sessionCfg is the configuration for the session.
	sessionCfg *uasc.SessionConfig

	// sechan is the open secure channel.
	sechan *uasc.SecureChannel

	// session is the active session.
	session atomic.Value // *Session

	sessions map[*Session]struct{}

	// cancelMonitor cancels the monitorChannel goroutine
	cancelMonitor       context.CancelFunc
	cancelSecureChannel context.CancelFunc

	// once initializes session
	once sync.Once

	notifs map[ClientNotification]chan interface{}
}

// NewClient creates a new Client.
//
// When no options are provided the new client is created from
// DefaultClientConfig() and DefaultSessionConfig(). If no authentication method
// is configured, a UserIdentityToken for anonymous authentication will be set.
// See #Client.CreateSession for details.
//
// To modify configuration you can provide any number of Options as opts. See
// #Option for details.
//
// https://godoc.org/github.com/gopcua/opcua#Option
func NewClient(endpoint string, opts ...Option) *Client {
	cfg, sessionCfg := ApplyConfig(opts...)

	return &Client{
		endpointURL: endpoint,
		cfg:         cfg,
		sessionCfg:  sessionCfg,
		sessions:    make(map[*Session]struct{}),
		notifs:      make(map[ClientNotification]chan interface{}),
	}
}

// Connect establishes a secure channel and creates a new session.
func (c *Client) Connect(ctx context.Context) (err error) {
	if ctx == nil {
		ctx = context.Background()
	}

	if c.sechan != nil {
		return errors.Errorf("already connected")
	}
	if err := c.Dial(ctx); err != nil {
		return err
	}
	s, err := c.CreateSession(c.sessionCfg)
	if err != nil {
		_ = c.Close()
		return err
	}

	if err := c.ActivateSession(s); err != nil {
		_ = c.Close()
		return err
	}

	ctx, s.cancelKeepAlive = context.WithCancel(ctx)
	go s.keepAliveManager.Run(ctx)

	return nil
}

// Dial establishes a secure channel.
func (c *Client) Dial(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}

	c.once.Do(func() { c.session.Store((*Session)(nil)) })
	if c.sechan != nil {
		return errors.Errorf("secure channel already connected")
	}
	conn, err := uacp.Dial(ctx, c.endpointURL)
	if err != nil {
		return err
	}
	sechan, err := uasc.NewSecureChannel(c.endpointURL, conn, c.cfg)
	if err != nil {
		_ = conn.Close()
		return err
	}
	c.sechan = sechan

	ctx, c.cancelMonitor = context.WithCancel(ctx)
	go c.monitorChannel(ctx)
	// Create secure channel from monitor context
	ctx, c.cancelSecureChannel = context.WithCancel(ctx)
	go c.sechan.Run(ctx)

	if err := sechan.Open(); err != nil {
		c.cancelMonitor()
		c.cancelSecureChannel()

		_ = conn.Close()
		c.sechan = nil
		return err
	}

	return nil
}

func (c *Client) monitorChannel(ctx context.Context) {
	for {
		notif := c.sechan.Notif
		select {
		case <-ctx.Done():
			return
		case <-notif(uasc.SecureChannelNotificationClose):
			if c.cfg.AutoReconnect {

				c.Session().keepAliveManager.Suspend()
				c.notify(ClientNotificationConnectionLost, struct{}{})

				if err := c.recreateSecureChannel(ctx); err != nil {
					debug.Printf("opcua: recreate secure channel has failed")
					continue
				}
				c.notify(ClientNotificationConnectionReestablished, struct{}{})
				if err := c.notifyConnectionReestablished(); err != nil {
					debug.Printf("opcua: reestablishing connection has failed")
					if err := c.Close(); err != nil {
						// callback err ??
					}
					continue
				}
				c.Session().keepAliveManager.Resume()
			} else {
				c.destroySecureChannel()
			}
			// todo: add event handler
			// case <-notif(uasc.SecureChannelNotificationLifetime75):
			// case <-notif(uasc.SecureChannelNotificationSecurityTokenRenewed):
			// case <-notif(uasc.SecureChannelNotificationReceiveResponse):
			// case <-notif(uasc.SecureChannelNotificationError):
		}
	}
}

// Close closes the session and the secure channel.
func (c *Client) Close() error {
	// try to close the session but ignore any error
	// so that we close the underlying channel and connection.
	_ = c.CloseSession()
	if c.cancelMonitor != nil {
		c.cancelMonitor()
	}
	if c.cancelSecureChannel != nil {
		c.cancelSecureChannel()
	}

	return c.sechan.Close()
}

// Session returns the active session.
func (c *Client) Session() *Session {
	return c.session.Load().(*Session)
}

// CreateSession creates a new session which is not yet activated and not
// associated with the client. Call ActivateSession to both activate and
// associate the session with the client.
//
// If no UserIdentityToken is given explicitly before calling CreateSesion,
// it automatically sets anonymous identity token with the same PolicyID
// that the server sent in Create Session Response. The default PolicyID
// "Anonymous" wii be set if it's missing in response.
//
// See Part 4, 5.6.2
func (c *Client) CreateSession(cfg *uasc.SessionConfig) (*Session, error) {
	if c.sechan == nil {
		return nil, errors.Errorf("secure channel not connected")
	}

	nonce := make([]byte, 32)
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	req := &ua.CreateSessionRequest{
		ClientDescription:       cfg.ClientDescription,
		EndpointURL:             c.endpointURL,
		SessionName:             fmt.Sprintf("gopcua-%d", time.Now().UnixNano()),
		ClientNonce:             nonce,
		ClientCertificate:       c.cfg.Certificate,
		RequestedSessionTimeout: float64(cfg.SessionTimeout / time.Millisecond),
	}

	var s *Session
	// for the CreateSessionRequest the authToken is always nil.
	// use c.sechan.Send() to enforce this.
	err := c.sechan.SendRequest(req, nil, func(v interface{}) error {
		var res *ua.CreateSessionResponse
		if err := safeAssign(v, &res); err != nil {
			return err
		}

		err := c.sechan.VerifySessionSignature(res.ServerCertificate, nonce, res.ServerSignature.Signature)
		if err != nil {
			log.Printf("error verifying session signature: %s", err)
			return nil
		}

		// Ensure we have a valid identity token that the server will accept before trying to activate a session
		if c.sessionCfg.UserIdentityToken == nil {
			opt := AuthAnonymous()
			opt(c.cfg, c.sessionCfg)

			p := anonymousPolicyID(res.ServerEndpoints)
			opt = AuthPolicyID(p)
			opt(c.cfg, c.sessionCfg)
		}

		s = &Session{
			c:                 c,
			cfg:               cfg,
			resp:              res,
			serverNonce:       res.ServerNonce,
			serverCertificate: res.ServerCertificate,
			keepAliveManager:  c.createKeepAliveManager(),
		}

		s.publishEngine = newPublishEngine(s)

		c.sessions[s] = struct{}{}

		return nil
	})
	return s, err
}

func (c *Client) sessionIsClosed() bool {
	if c.Session() != nil {
		return false
	}
	return true
}

func (c *Client) createKeepAliveManager() *KeepAliveManager {
	return NewKeepAliveManager(
		c.sessionCfg.SessionTimeout,
		func(state *KeepAliveState) error {

			// Add test if reconnecting then callback
			res, err := c.Read(
				&ua.ReadRequest{
					MaxAge: 2000,
					NodesToRead: []*ua.ReadValueID{
						&ua.ReadValueID{
							NodeID:      ua.NewNumericNodeID(0, 2259),
							AttributeID: ua.AttributeIDValue,
						},
					},
					TimestampsToReturn: ua.TimestampsToReturnBoth,
				},
			)
			if err != nil {
				return err
			}
			if len(res.Results) != 1 {
				return errors.Errorf("Unconsistant result")
			}

			dataValue := res.Results[0]
			if dataValue.Status == ua.StatusOK {
				switch value := dataValue.Value.Value().(type) {
				case int32:
					newState := ua.ServerState(value)
					lastKnownState, ok := state.LastKnownState().(ua.ServerState)
					if ok {
						if newState != lastKnownState {
							// Warning
						}
					}
					state.SetLastKnownState(newState)
				}
			} else {
				return dataValue.Status
			}
			return nil
		},
	)
}

const defaultAnonymousPolicyID = "Anonymous"

func anonymousPolicyID(endpoints []*ua.EndpointDescription) string {
	for _, e := range endpoints {
		if e.SecurityMode != ua.MessageSecurityModeNone || e.SecurityPolicyURI != ua.SecurityPolicyURINone {
			continue
		}

		for _, t := range e.UserIdentityTokens {
			if t.TokenType == ua.UserTokenTypeAnonymous {
				return t.PolicyID
			}
		}
	}

	return defaultAnonymousPolicyID
}

// ActivateSession activates the session and associates it with the client. If
// the client already has a session it will be closed. To retain the current
// session call DetachSession.
//
// See Part 4, 5.6.3
func (c *Client) ActivateSession(s *Session) error {
	sig, sigAlg, err := c.sechan.NewSessionSignature(s.serverCertificate, s.serverNonce)
	if err != nil {
		log.Printf("error creating session signature: %s", err)
		return nil
	}

	switch tok := s.cfg.UserIdentityToken.(type) {
	case *ua.AnonymousIdentityToken:
		// nothing to do

	case *ua.UserNameIdentityToken:
		pass, passAlg, err := c.sechan.EncryptUserPassword(s.cfg.AuthPolicyURI, s.cfg.AuthPassword, s.serverCertificate, s.serverNonce)
		if err != nil {
			log.Printf("error encrypting user password: %s", err)
			return err
		}
		tok.Password = pass
		tok.EncryptionAlgorithm = passAlg

	case *ua.X509IdentityToken:
		tokSig, tokSigAlg, err := c.sechan.NewUserTokenSignature(s.cfg.AuthPolicyURI, s.serverCertificate, s.serverNonce)
		if err != nil {
			log.Printf("error creating session signature: %s", err)
			return err
		}
		s.cfg.UserTokenSignature = &ua.SignatureData{
			Algorithm: tokSigAlg,
			Signature: tokSig,
		}

	case *ua.IssuedIdentityToken:
		tok.EncryptionAlgorithm = ""
	}

	req := &ua.ActivateSessionRequest{
		ClientSignature: &ua.SignatureData{
			Algorithm: sigAlg,
			Signature: sig,
		},
		ClientSoftwareCertificates: nil,
		LocaleIDs:                  s.cfg.LocaleIDs,
		UserIdentityToken:          ua.NewExtensionObject(s.cfg.UserIdentityToken),
		UserTokenSignature:         s.cfg.UserTokenSignature,
	}
	return c.sechan.SendRequest(req, s.resp.AuthenticationToken, func(v interface{}) error {
		var res *ua.ActivateSessionResponse
		if err := safeAssign(v, &res); err != nil {
			return err
		}

		// save the nonce for the next request
		s.serverNonce = res.ServerNonce

		if err := c.CloseSession(); err != nil {
			// try to close the newly created session but report
			// only the initial error.
			_ = c.closeSession(s)
			return err
		}
		c.session.Store(s)
		return nil
	})
}

// CloseSession closes the current session.
//
// See Part 4, 5.6.4
func (c *Client) CloseSession() error {
	if err := c.closeSession(c.Session()); err != nil {
		return err
	}
	c.session.Store((*Session)(nil))
	return nil
}

// closeSession closes the given session.
func (c *Client) closeSession(s *Session) error {
	if s == nil {
		return nil
	}
	delete(c.sessions, s)

	if s.cancelKeepAlive != nil {
		s.cancelKeepAlive()
	}

	if s.publishEngine != nil {
		s.publishEngine.Terminate()
		s.publishEngine = nil
	}

	req := &ua.CloseSessionRequest{DeleteSubscriptions: true}
	var res *ua.CloseSessionResponse
	return c.Send(req, func(v interface{}) error {
		return safeAssign(v, &res)
	})
}

// DetachSession removes the session from the client without closing it. The
// caller is responsible to close or re-activate the session. If the client
// does not have an active session the function returns no error.
func (c *Client) DetachSession() (*Session, error) {
	s := c.Session()
	c.session.Store((*Session)(nil))
	return s, nil
}

func (c *Client) repairSessions() error {
	for s := range c.sessions {
		if err := s.repairSession(); err != nil {
			return err
		}
	}
	return nil
}

// Send sends the request via the secure channel and registers a handler for
// the response. If the client has an active session it injects the
// authentication token.
func (c *Client) Send(req ua.Request, h func(interface{}) error) error {
	return c.sendWithTimeout(req, c.cfg.RequestTimeout, h)
}

// sendWithTimeout sends the request via the secure channel with a custom timeout and registers a handler for
// the response. If the client has an active session it injects the
// authentication token.
func (c *Client) sendWithTimeout(req ua.Request, timeout time.Duration, h func(interface{}) error) error {
	var authToken *ua.NodeID
	if s := c.Session(); s != nil {
		authToken = s.resp.AuthenticationToken
	}
	return c.sechan.SendRequestWithTimeout(req, authToken, timeout, h)
}

// Node returns a node object which accesses its attributes
// through this client connection.
func (c *Client) Node(id *ua.NodeID) *Node {
	return &Node{ID: id, c: c}
}

func (c *Client) GetEndpoints() (*ua.GetEndpointsResponse, error) {
	req := &ua.GetEndpointsRequest{
		EndpointURL: c.endpointURL,
	}
	var res *ua.GetEndpointsResponse
	err := c.Send(req, func(v interface{}) error {
		return safeAssign(v, &res)
	})
	return res, err
}

// Read executes a synchronous read request.
//
// By default, the function requests the value of the nodes
// in the default encoding of the server.
func (c *Client) Read(req *ua.ReadRequest) (*ua.ReadResponse, error) {
	// clone the request and the ReadValueIDs to set defaults without
	// manipulating them in-place.
	rvs := make([]*ua.ReadValueID, len(req.NodesToRead))
	for i, rv := range req.NodesToRead {
		rc := &ua.ReadValueID{}
		*rc = *rv
		if rc.AttributeID == 0 {
			rc.AttributeID = ua.AttributeIDValue
		}
		if rc.DataEncoding == nil {
			rc.DataEncoding = &ua.QualifiedName{}
		}
		rvs[i] = rc
	}
	req = &ua.ReadRequest{
		MaxAge:             req.MaxAge,
		TimestampsToReturn: req.TimestampsToReturn,
		NodesToRead:        rvs,
	}

	var res *ua.ReadResponse
	err := c.Send(req, func(v interface{}) error {
		return safeAssign(v, &res)
	})
	return res, err
}

// Write executes a synchronous write request.
func (c *Client) Write(req *ua.WriteRequest) (*ua.WriteResponse, error) {
	var res *ua.WriteResponse
	err := c.Send(req, func(v interface{}) error {
		return safeAssign(v, &res)
	})
	return res, err
}

// Browse executes a synchronous browse request.
func (c *Client) Browse(req *ua.BrowseRequest) (*ua.BrowseResponse, error) {
	var res *ua.BrowseResponse
	err := c.Send(req, func(v interface{}) error {
		return safeAssign(v, &res)
	})
	return res, err
}

// Call executes a synchronous call request for a single method.
func (c *Client) Call(req *ua.CallMethodRequest) (*ua.CallMethodResult, error) {
	creq := &ua.CallRequest{
		MethodsToCall: []*ua.CallMethodRequest{req},
	}
	var res *ua.CallResponse
	err := c.Send(creq, func(v interface{}) error {
		return safeAssign(v, &res)
	})
	if err != nil {
		return nil, err
	}
	if len(res.Results) != 1 {
		return nil, ua.StatusBadUnknownResponse
	}
	return res.Results[0], nil
}

// BrowseNext executes a synchronous browse request.
func (c *Client) BrowseNext(req *ua.BrowseNextRequest) (*ua.BrowseNextResponse, error) {
	var res *ua.BrowseNextResponse
	err := c.Send(req, func(v interface{}) error {
		r, ok := v.(*ua.BrowseNextResponse)
		if !ok {
			return errors.Errorf("invalid response: %T", v)
		}
		res = r
		return nil
	})
	return res, err
}

// Subscribe creates a Subscription with given parameters. Parameters that have not been set
// (have zero values) are overwritten with default values.
// See opcua.DefaultSubscription* constants
func (c *Client) Subscribe(params *SubscriptionParameters) (*Subscription, error) {
	if params == nil {
		params = &SubscriptionParameters{}
	}
	params.setDefaults()
	req := &ua.CreateSubscriptionRequest{
		RequestedPublishingInterval: float64(params.Interval / time.Millisecond),
		RequestedLifetimeCount:      params.LifetimeCount,
		RequestedMaxKeepAliveCount:  params.MaxKeepAliveCount,
		PublishingEnabled:           true,
		MaxNotificationsPerPublish:  params.MaxNotificationsPerPublish,
		Priority:                    params.Priority,
	}

	var res *ua.CreateSubscriptionResponse
	err := c.Send(req, func(v interface{}) error {
		return safeAssign(v, &res)
	})
	if err != nil {
		return nil, err
	}
	if res.ResponseHeader.ServiceResult != ua.StatusOK {
		return nil, res.ResponseHeader.ServiceResult
	}

	sub := &Subscription{
		SubscriptionID:            res.SubscriptionID,
		params:                    params,
		publishEngine:             c.Session().publishEngine,
		RevisedPublishingInterval: time.Duration(res.RevisedPublishingInterval) * time.Millisecond,
		RevisedLifetimeCount:      res.RevisedLifetimeCount,
		RevisedMaxKeepAliveCount:  res.RevisedMaxKeepAliveCount,
		monitoredItems:            []*MonitoredItem{},
		lastSequenceNumber:        0,
		Notifs:                    params.Notifs,
		c:                         c,
	}

	if err := c.Session().publishEngine.RegisterSubscription(sub); err != nil {
		return nil, err
	}

	return sub, nil
}

func (c *Client) StartPublishEngine(ctx context.Context) {
	session, ok := c.session.Load().(*Session)
	if ok && session != nil {
		session.publishEngine.Run(ctx)
	}
}

func (c *Client) SuspendPublishEngine() {
	session, ok := c.session.Load().(*Session)
	if ok && session != nil {
		session.publishEngine.Suspend()
	}
}

func (c *Client) ResumePublishEngine() {
	session, ok := c.session.Load().(*Session)
	if ok && session != nil {
		session.publishEngine.Resume()
	}
}

func (c *Client) Notif(key ClientNotification) chan interface{} {
	notif, ok := c.notifs[key]
	if !ok {
		notif = make(chan interface{}, 1)
		c.notifs[key] = notif
	}
	return notif
}

func (c *Client) notify(key ClientNotification, val interface{}) {
	notif, ok := c.notifs[key]
	if ok {
		notif <- val
	}
}

func (c *Client) recreateSecureChannel(ctx context.Context) error {
	debug.Printf("opcua: recreate secure channel...")

	// todo: add test, endpoint server is known ?
	// Test if is already reconnecting

	c.notify(ClientNotificationStartReconnection, struct{}{})

	if err := c.destroySecureChannel(); err != nil {
		return err
	}
	resultNotif := make(chan error, 1)

	failAndRetry := func(err error, retryNotif chan struct{}) {
		debug.Printf("opcua: %s", err.Error())

		if false /*todo: c.reconnectionIsCanceled*/ {
			c.notify(ClientNotificationReconnectionCancel, struct{}{})
			resultNotif <- errors.Errorf("Secure channel reconnection has been canceled")
			return
		}
		c.notify(ClientNotificationReconnectionAttemptHasFailed, err)

		timer := time.NewTimer(100 * time.Millisecond)

		select {
		case <-ctx.Done():
		case <-timer.C:
			retryNotif <- struct{}{}
		}
	}

	attemptToRecreateSecureChannel := func(errNotif chan error) {

		if c.cancelSecureChannel != nil {
			c.cancelSecureChannel()
			c.cancelSecureChannel = nil
		}

		conn, err := uacp.Dial(ctx, c.endpointURL)
		if err != nil {
			errNotif <- err
			return
		}
		sechan, err := uasc.NewSecureChannel(c.endpointURL, conn, c.cfg)
		if err != nil {
			_ = conn.Close()
			errNotif <- err
			return
		}
		c.sechan = sechan

		ctx, c.cancelSecureChannel = context.WithCancel(ctx)
		go c.sechan.Run(ctx)

		if err := sechan.Open(); err != nil {
			c.cancelMonitor()

			_ = conn.Close()
			c.sechan = nil
			errNotif <- err
			return
		}
		errNotif <- nil
	}

	go func() {
		errNotif := make(chan error, 1)
		retryNotif := make(chan struct{}, 1)
		go attemptToRecreateSecureChannel(errNotif)
		for {
			select {
			case <-ctx.Done():
				return
			case <-retryNotif:
				go attemptToRecreateSecureChannel(errNotif)
			case err := <-errNotif:
				if err != nil {
					if err == syscall.ECONNREFUSED {
						resultNotif <- err
						return
					}
					if false /*Backoff aborted*/ {
						failAndRetry(err, retryNotif)
						continue
					}
					if err == ua.StatusBadCertificateInvalid {
						// todo: recreate server certificate
						if err := c.sechan.Open(); err != nil {
							failAndRetry(err, retryNotif)
							continue
						}
						resultNotif <- nil
						return
					}
					failAndRetry(err, retryNotif)
					continue
				}
				debug.Printf("opcua: secure channel reconnected")
				resultNotif <- nil
				return
			}
		}
	}()
	return <-resultNotif
}

func (c *Client) notifyConnectionReestablished() error {
	return c.repairSessions()
}

func (c *Client) destroySecureChannel() error {
	if c.sechan != nil {
		debug.Printf("opcua: destroying secure channel")

		if err := c.sechan.Close(); err != nil && err != io.EOF {
			return err
		}
		c.sechan = nil
	}
	return nil
}

func (c *Client) HistoryReadRawModified(nodes []*ua.HistoryReadValueID, details *ua.ReadRawModifiedDetails) (*ua.HistoryReadResponse, error) {
	// Part 4, 5.10.3 HistoryRead
	req := &ua.HistoryReadRequest{
		TimestampsToReturn: ua.TimestampsToReturnBoth,
		NodesToRead:        nodes,
		// Part 11, 6.4 HistoryReadDetails parameters
		HistoryReadDetails: &ua.ExtensionObject{
			TypeID:       ua.NewFourByteExpandedNodeID(0, id.ReadRawModifiedDetails_Encoding_DefaultBinary),
			EncodingMask: ua.ExtensionObjectBinary,
			Value:        details,
		},
	}

	var res *ua.HistoryReadResponse
	err := c.Send(req, func(v interface{}) error {
		return safeAssign(v, &res)
	})
	return res, err
}

type KeepAliveManager struct {
	PingTimeout              time.Duration
	checkInterval            time.Duration
	lastResponseReceivedTime time.Time
	KeepAliveState
	sendPing    func(state *KeepAliveState) error
	chanSuspend chan struct{}
	chanResume  chan struct{}
	checkTimer  *time.Timer
}

type KeepAliveState struct {
	lastKnownState interface{}
}

func (k *KeepAliveState) LastKnownState() interface{} {
	return k.lastKnownState
}

func (k *KeepAliveState) SetLastKnownState(state interface{}) {
	k.lastKnownState = state
}

func NewKeepAliveManager(srvTimeout time.Duration, sendPing func(state *KeepAliveState) error) *KeepAliveManager {
	pingTimeout := srvTimeout * 2 / 3
	return &KeepAliveManager{
		PingTimeout:   pingTimeout,
		checkInterval: pingTimeout / 3,
		sendPing:      sendPing,
		chanSuspend:   make(chan struct{}, 1),
		chanResume:    make(chan struct{}, 1),
	}
}

func (k *KeepAliveManager) Run(ctx context.Context) {
	afterCheck := make(chan error, 1)

	k.checkTimer = time.NewTimer(k.checkInterval)

	checkKeepAlive := func() {
		now := time.Now()

		timeSinceLastServerContact := now.Sub(k.lastResponseReceivedTime)
		if timeSinceLastServerContact < k.PingTimeout {
			afterCheck <- nil
			return
		}
		if err := k.sendPing(&k.KeepAliveState); err != nil {
			afterCheck <- err
			return
		}
		afterCheck <- nil
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-k.checkTimer.C:
			go checkKeepAlive()
		case err := <-afterCheck:
			k.checkTimer.Reset(k.checkInterval)
			if err != nil {
				k.Suspend()
				k.checkTimer.Stop()
			}

		case <-k.chanSuspend:
			select {
			case <-ctx.Done():
				return
			case <-k.chanResume:
				continue
			}
		}
	}
}

func (k *KeepAliveManager) Suspend() {
	if k.checkTimer != nil {
		k.checkTimer.Stop()
	}
	k.chanSuspend <- struct{}{}
}

func (k *KeepAliveManager) Resume() {
	k.chanResume <- struct{}{}
}

func (k *KeepAliveManager) Notify() {
	k.checkTimer.Reset(k.checkInterval)
	k.lastResponseReceivedTime = time.Now()
}

// safeAssign implements a type-safe assign from T to *T.
func safeAssign(t, ptrT interface{}) error {
	if reflect.TypeOf(t) != reflect.TypeOf(ptrT).Elem() {
		return InvalidResponseTypeError{t, ptrT}
	}

	// this is *ptrT = t
	reflect.ValueOf(ptrT).Elem().Set(reflect.ValueOf(t))
	return nil
}

type InvalidResponseTypeError struct {
	got, want interface{}
}

func (e InvalidResponseTypeError) Error() string {
	return fmt.Sprintf("invalid response: got %T want %T", e.got, e.want)
}
