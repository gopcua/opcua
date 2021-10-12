// Copyright 2018-2020 opcua authors. All rights reserved.
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
func GetEndpoints(ctx context.Context, endpoint string, opts ...Option) ([]*ua.EndpointDescription, error) {
	opts = append(opts, AutoReconnect(false))
	c := NewClient(endpoint, opts...)
	if err := c.Dial(ctx); err != nil {
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

	sort.Sort(sort.Reverse(bySecurityLevel(endpoints)))
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

// Client is a high-level client for an OPC/UA server.
// It establishes a secure channel and a session.
type Client struct {
	// endpointURL is the endpoint URL the client connects to.
	endpointURL string

	// cfg is the configuration for the client.
	cfg *Config

	// conn is the open connection
	conn *uacp.Conn

	// sechan is the open secure channel.
	sechan    *uasc.SecureChannel
	sechanErr chan error

	// session is the active session.
	session atomic.Value // *Session

	// subs is the set of active subscriptions by id.
	subs   map[uint32]*Subscription
	subMux sync.RWMutex

	pendingAcks []*ua.SubscriptionAcknowledgement

	pausech  chan struct{} // pauses subscription publish loop
	resumech chan struct{} // resumes subscription publish loop
	mcancel  func()        // stops subscription publish loop

	// timeout for sending PublishRequests
	publishTimeout atomic.Value

	// state of the client
	state atomic.Value // ConnState

	// monitorOnce ensures only one connection monitor is running
	monitorOnce sync.Once

	// sessionOnce initializes the session
	sessionOnce sync.Once
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
	cfg := ApplyConfig(opts...)
	c := Client{
		endpointURL: endpoint,
		cfg:         cfg,
		sechanErr:   make(chan error, 1),
		subs:        make(map[uint32]*Subscription),
		pausech:     make(chan struct{}, 2),
		resumech:    make(chan struct{}, 2),
		pendingAcks: []*ua.SubscriptionAcknowledgement{},
	}
	c.publishTimeout.Store(uasc.MaxTimeout)
	c.pauseSubscriptions()
	c.state.Store(Closed)
	return &c
}

// reconnectAction is a list of actions for the client reconnection logic.
type reconnectAction uint8

const (
	none reconnectAction = iota // no reconnection action

	createSecureChannel   // recreate secure channel action
	restoreSession        // ask the server to repair session
	recreateSession       // ask the client to repair session
	restoreSubscriptions  // republish or recreate subscriptions
	transferSubscriptions // move subscriptions from one session to another
	abortReconnect        // the reconnecting is not possible
)

// Connect establishes a secure channel and creates a new session.
func (c *Client) Connect(ctx context.Context) (err error) {
	if c.sechan != nil {
		return errors.Errorf("already connected")
	}

	c.state.Store(Connecting)
	if err := c.Dial(ctx); err != nil {
		return err
	}
	s, err := c.CreateSession(c.cfg.session)
	if err != nil {
		_ = c.Close()
		return err
	}
	if err := c.ActivateSession(s); err != nil {
		_ = c.Close()
		return err
	}
	c.state.Store(Connected)

	mctx, mcancel := context.WithCancel(context.Background())
	c.mcancel = mcancel
	c.monitorOnce.Do(func() {
		go c.monitor(mctx)
		go c.monitorSubscriptions(mctx)
	})

	return nil
}

// monitor manages connection alteration
func (c *Client) monitor(ctx context.Context) {
	dlog := debug.NewPrefixLogger("client: monitor: ")

	dlog.Printf("start")
	defer dlog.Printf("done")

	defer c.mcancel()
	defer c.state.Store(Closed)

	action := none
	for {
		select {
		case <-ctx.Done():
			return

		case err, ok := <-c.sechanErr:
			// return if channel or connection is closed
			if !ok || err == io.EOF && c.State() == Closed {
				dlog.Print("closed")
				return
			}

			// tell the handler the connection is disconnected
			c.state.Store(Disconnected)
			dlog.Print("disconnected")

			if !c.cfg.sechan.AutoReconnect {
				// the connection is closed and should not be restored
				action = abortReconnect
				dlog.Print("auto-reconnect disabled")
				return
			}

			dlog.Print("auto-reconnecting")

			switch err {
			case io.EOF:
				// the connection has been closed
				action = createSecureChannel

			case syscall.ECONNREFUSED:
				// the connection has been refused by the server
				action = abortReconnect

			default:
				switch x := err.(type) {
				case *uacp.Error:
					switch ua.StatusCode(x.ErrorCode) {
					case ua.StatusBadSecureChannelIDInvalid:
						// the secure channel has been rejected by the server
						action = createSecureChannel

					case ua.StatusBadSessionIDInvalid:
						// the session has been rejected by the server
						action = recreateSession

					case ua.StatusBadSubscriptionIDInvalid:
						// the subscription has been rejected by the server
						action = transferSubscriptions

					case ua.StatusBadCertificateInvalid:
						// todo(unknownet): recreate server certificate
						fallthrough

					default:
						// unknown error has occured
						action = createSecureChannel
					}

				default:
					// unknown error has occured
					action = createSecureChannel
				}
			}

			c.state.Store(Disconnected)

			c.pauseSubscriptions()

			var (
				subsToRepublish []uint32 // subscription ids for which to send republish requests
				subsToRecreate  []uint32 // subscription ids which need to be recreated as new subscriptions
				availableSeqs   map[uint32][]uint32
			)

			for action != none {

				select {
				case <-ctx.Done():
					return

				default:
					switch action {

					case createSecureChannel:
						dlog.Printf("action: createSecureChannel")

						// recreate a secure channel by brute forcing
						// a reconnection to the server

						// close previous secure channel
						_ = c.conn.Close()
						c.sechan.Close()
						c.sechan = nil

						c.state.Store(Reconnecting)

						dlog.Printf("trying to recreate secure channel")
						for {
							if err := c.Dial(ctx); err != nil {
								select {
								case <-ctx.Done():
									return
								case <-time.After(c.cfg.sechan.ReconnectInterval):
									dlog.Printf("trying to recreate secure channel")
									continue
								}
							}
							break
						}
						dlog.Printf("secure channel recreated")
						action = restoreSession

					case restoreSession:
						dlog.Printf("action: restoreSession")

						// try to reactivate the session,
						// This only works if the session is still open on the server
						// otherwise recreate it

						dlog.Printf("trying to restore session")
						s, err := c.DetachSession()
						if err != nil {
							action = createSecureChannel
							continue
						}
						if err := c.ActivateSession(s); err != nil {
							dlog.Printf("restore session failed")
							action = recreateSession
							continue
						}
						dlog.Printf("session restored")
						action = restoreSubscriptions

					case recreateSession:
						dlog.Printf("action: recreateSession")

						// create a new session to replace the previous one

						dlog.Printf("trying to recreate session")
						s, err := c.CreateSession(c.cfg.session)
						if err != nil {
							dlog.Printf("recreate session failed: %v", err)
							action = createSecureChannel
							continue
						}
						if err := c.ActivateSession(s); err != nil {
							dlog.Printf("reactivate session failed: %v", err)
							action = createSecureChannel
							continue
						}
						action = transferSubscriptions

					case transferSubscriptions:
						dlog.Printf("action: transferSubscriptions")

						// transfer subscriptions from the old to the new session
						// and try to republish the subscriptions.
						// Restore the subscriptions where republishing fails.

						subIDs := c.SubscriptionIDs()

						availableSeqs = map[uint32][]uint32{}
						subsToRecreate = nil
						subsToRepublish = nil

						// try to transfer all subscriptions to the new session and
						// recreate them all if that fails.
						res, err := c.transferSubscriptions(subIDs)
						switch {
						case err != nil:
							dlog.Printf("transfer subscriptions failed. Recreating all subscriptions: %v", err)
							subsToRepublish = nil
							subsToRecreate = subIDs

						default:
							// otherwise, try a republish for the subscriptions that were transferred
							// and recreate the rest.
							for i := range res.Results {
								transferResult := res.Results[i]
								switch transferResult.StatusCode {
								case ua.StatusBadSubscriptionIDInvalid:
									dlog.Printf("sub %d: transfer subscription failed", subIDs[i])
									subsToRecreate = append(subsToRecreate, subIDs[i])

								default:
									subsToRepublish = append(subsToRepublish, subIDs[i])
									availableSeqs[subIDs[i]] = transferResult.AvailableSequenceNumbers
								}
							}
						}

						action = restoreSubscriptions

					case restoreSubscriptions:
						dlog.Printf("action: restoreSubscriptions")

						// try to republish the previous subscriptions from the server
						// otherwise restore them.
						// Assume that subsToRecreate and subsToRepublish have been
						// populated in the previous step.

						for _, id := range subsToRepublish {
							if err := c.republishSubscription(id, availableSeqs[id]); err != nil {
								dlog.Printf("republish of subscription %d failed", id)
								subsToRecreate = append(subsToRecreate, id)
							}
						}

						for _, id := range subsToRecreate {
							if err := c.recreateSubscription(id); err != nil {
								dlog.Printf("recreate subscripitions failed: %v", err)
								action = recreateSession
								continue
							}
						}

						c.state.Store(Connected)
						action = none

					case abortReconnect:
						dlog.Printf("action: abortReconnect")

						// non recoverable disconnection
						// stop the client

						// todo(unknownet): should we store the error?
						dlog.Printf("reconnection not recoverable")
						return
					}
				}
			}

			// clear sechan errors from reconnection
			for len(c.sechanErr) > 0 {
				<-c.sechanErr
			}

			dlog.Printf("resuming subscriptions")
			c.resumeSubscriptions()
			dlog.Printf("resumed subscriptions")
		}
	}
}

// Dial establishes a secure channel.
func (c *Client) Dial(ctx context.Context) error {
	c.sessionOnce.Do(func() {
		c.session.Store((*Session)(nil))
	})

	if c.sechan != nil {
		return errors.Errorf("secure channel already connected")
	}

	var err error
	var d = NewDialer(c.cfg)
	c.conn, err = d.Dial(ctx, c.endpointURL)
	if err != nil {
		return err
	}

	c.sechan, err = uasc.NewSecureChannel(c.endpointURL, c.conn, c.cfg.sechan, c.sechanErr)
	if err != nil {
		_ = c.conn.Close()
		return err
	}

	return c.sechan.Open(ctx)
}

// Close closes the session and the secure channel.
func (c *Client) Close() error {
	// try to close the session but ignore any error
	// so that we close the underlying channel and connection.
	c.CloseSession()
	c.state.Store(Closed)

	if c.mcancel != nil {
		c.mcancel()
	}
	if c.sechan != nil {
		c.sechan.Close()
	}

	// https://github.com/gopcua/opcua/pull/462
	//
	// do not close the c.sechanErr channel since it leads to
	// race conditions and it gets garbage collected anyway.
	// There is nothing we can do with this error while
	// shutting down the client so I think it is safe to ignore
	// them.

	// close the connection but ignore the error since there isn't
	// anything we can do about it anyway
	c.conn.Close()

	return nil
}

func (c *Client) State() ConnState {
	return c.state.Load().(ConnState)
}

// Session returns the active session.
func (c *Client) Session() *Session {
	return c.session.Load().(*Session)
}

// sessionClosed returns true when there is no session.
func (c *Client) sessionClosed() bool {
	return c.Session() == nil
}

// Session is a OPC/UA session as described in Part 4, 5.6.
type Session struct {
	cfg *uasc.SessionConfig

	// resp is the response to the CreateSession request which contains all
	// necessary parameters to activate the session.
	resp *ua.CreateSessionResponse

	// serverCertificate is the certificate used to generate the signatures for
	// the ActivateSessionRequest methods
	serverCertificate []byte

	// serverNonce is the secret nonce received from the server during Create and Activate
	// Session response. Used to generate the signatures for the ActivateSessionRequest
	// and User Authorization
	serverNonce []byte
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
		return nil, ua.StatusBadServerNotConnected
	}

	nonce := make([]byte, 32)
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	name := cfg.SessionName
	if name == "" {
		name = fmt.Sprintf("gopcua-%d", time.Now().UnixNano())
	}

	req := &ua.CreateSessionRequest{
		ClientDescription:       cfg.ClientDescription,
		EndpointURL:             c.endpointURL,
		SessionName:             name,
		ClientNonce:             nonce,
		ClientCertificate:       c.cfg.sechan.Certificate,
		RequestedSessionTimeout: float64(cfg.SessionTimeout / time.Millisecond),
	}

	var s *Session
	// for the CreateSessionRequest the authToken is always nil.
	// use c.sechan.SendRequest() to enforce this.
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
		if c.cfg.session.UserIdentityToken == nil {
			opt := AuthAnonymous()
			opt(c.cfg)

			p := anonymousPolicyID(res.ServerEndpoints)
			opt = AuthPolicyID(p)
			opt(c.cfg)
		}

		s = &Session{
			cfg:               cfg,
			resp:              res,
			serverNonce:       res.ServerNonce,
			serverCertificate: res.ServerCertificate,
		}

		return nil
	})
	return s, err
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
	if c.sechan == nil {
		return ua.StatusBadServerNotConnected
	}
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

		// close the previous session
		//
		// https://github.com/gopcua/opcua/issues/474
		//
		// We decided not to check the error of CloseSession() since we
		// can't do much about it anyway and it creates a race in the
		// re-connection logic.
		c.CloseSession()

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

// Send sends the request via the secure channel and registers a handler for
// the response. If the client has an active session it injects the
// authentication token.
func (c *Client) Send(req ua.Request, h func(interface{}) error) error {
	return c.sendWithTimeout(req, c.cfg.sechan.RequestTimeout, h)
}

// sendWithTimeout sends the request via the secure channel with a custom timeout and registers a handler for
// the response. If the client has an active session it injects the
// authentication token.
func (c *Client) sendWithTimeout(req ua.Request, timeout time.Duration, h func(interface{}) error) error {
	if c.sechan == nil {
		return ua.StatusBadServerNotConnected
	}
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
		err := safeAssign(v, &res)

		// If the client cannot decode an extension object then its
		// value will be nil. However, since the EO was known to the
		// server the StatusCode for that data value will be OK. We
		// therefore check for extension objects with nil values and set
		// the status code to StatusBadDataTypeIDUnknown.
		if err == nil {
			for _, dv := range res.Results {
				if dv.Value == nil {
					continue
				}
				val := dv.Value.Value()
				if eo, ok := val.(*ua.ExtensionObject); ok && eo.Value == nil {
					dv.Status = ua.StatusBadDataTypeIDUnknown
				}
			}
		}

		return err
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
		return safeAssign(v, &res)
	})
	return res, err
}

// RegisterNodes registers node ids for more efficient reads.
// Part 4, Section 5.8.5
func (c *Client) RegisterNodes(req *ua.RegisterNodesRequest) (*ua.RegisterNodesResponse, error) {
	var res *ua.RegisterNodesResponse
	err := c.Send(req, func(v interface{}) error {
		return safeAssign(v, &res)
	})
	return res, err
}

// UnregisterNodes unregisters node ids previously registered with RegisterNodes.
// Part 4, Section 5.8.6
func (c *Client) UnregisterNodes(req *ua.UnregisterNodesRequest) (*ua.UnregisterNodesResponse, error) {
	var res *ua.UnregisterNodesResponse
	err := c.Send(req, func(v interface{}) error {
		return safeAssign(v, &res)
	})
	return res, err
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

// safeAssign implements a type-safe assign from T to *T.
func safeAssign(t, ptrT interface{}) error {
	if reflect.TypeOf(t) != reflect.TypeOf(ptrT).Elem() {
		return InvalidResponseTypeError{t, ptrT}
	}

	// this is *ptrT = t
	reflect.ValueOf(ptrT).Elem().Set(reflect.ValueOf(t))
	return nil
}

func uint32SliceContains(n uint32, a []uint32) bool {
	for _, v := range a {
		if n == v {
			return true
		}
	}
	return false
}

type InvalidResponseTypeError struct {
	got, want interface{}
}

func (e InvalidResponseTypeError) Error() string {
	return fmt.Sprintf("invalid response: got %T want %T", e.got, e.want)
}
