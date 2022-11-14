// Copyright 2018-2020 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package opcua

import (
	"context"
	"crypto/rand"
	"expvar"
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
	"github.com/gopcua/opcua/stats"
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
	defer c.CloseWithContext(ctx)
	res, err := c.GetEndpointsWithContext(ctx)
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
	atomicSechan atomic.Value // *uasc.SecureChannel
	sechanErr    chan error

	// atomicSession is the active atomicSession.
	atomicSession atomic.Value // *Session

	// subMux guards subs and pendingAcks.
	subMux sync.RWMutex

	// subs is the set of active subscriptions by id.
	subs map[uint32]*Subscription

	// pendingAcks contains the pending subscription acknowledgements
	// for all active subscriptions.
	pendingAcks []*ua.SubscriptionAcknowledgement

	// pausech pauses the subscription publish loop
	pausech chan struct{}

	// resumech resumes subscription publish loop
	resumech chan struct{}

	// mcancel stops subscription publish loop
	mcancel func()

	// timeout for sending PublishRequests
	atomicPublishTimeout atomic.Value // time.Duration

	// atomicState of the client
	atomicState atomic.Value // ConnState

	// list of cached atomicNamespaces on the server
	atomicNamespaces atomic.Value // []string

	// monitorOnce ensures only one connection monitor is running
	monitorOnce sync.Once

	// cfgerr contains an error that was captured in ApplyConfig.
	// Since the API does not allow to bubble the error up in NewClient
	// and we don't want to break existing code right away we carry the
	// error here and bubble it up during Dial and Connect.
	//
	// Note: Starting with v0.5 NewClient will return the error and this
	// variable needs to be removed.
	cfgerr error
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
//
// Note: Starting with v0.5 this function will will return an error.
func NewClient(endpoint string, opts ...Option) *Client {
	cfg := ApplyConfig(opts...)
	c := Client{
		endpointURL: endpoint,
		cfg:         cfg,
		sechanErr:   make(chan error, 1),
		subs:        make(map[uint32]*Subscription),
		pendingAcks: make([]*ua.SubscriptionAcknowledgement, 0),
		pausech:     make(chan struct{}, 2),
		resumech:    make(chan struct{}, 2),
		cfgerr:      cfg.Error(), // todo(fs): remove with v0.5.0 and return the error
	}
	c.pauseSubscriptions(context.Background())
	c.setPublishTimeout(uasc.MaxTimeout)
	c.setState(Closed)
	c.setSecureChannel(nil)
	c.setSession(nil)
	c.setNamespaces([]string{})
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
	// todo(fs): remove with v0.5.0
	if c.cfgerr != nil {
		return c.cfgerr
	}

	if c.SecureChannel() != nil {
		return errors.Errorf("already connected")
	}

	c.setState(Connecting)
	if err := c.Dial(ctx); err != nil {
		stats.RecordError(err)

		return err
	}

	s, err := c.CreateSessionWithContext(ctx, c.cfg.session)
	if err != nil {
		c.CloseWithContext(ctx)
		stats.RecordError(err)

		return err
	}

	if err := c.ActivateSessionWithContext(ctx, s); err != nil {
		c.CloseWithContext(ctx)
		stats.RecordError(err)

		return err
	}
	c.setState(Connected)

	mctx, mcancel := context.WithCancel(context.Background())
	c.mcancel = mcancel
	c.monitorOnce.Do(func() {
		go c.monitor(mctx)
		go c.monitorSubscriptions(mctx)
	})

	// todo(fs): we might need to guard this with an option in case of a broken
	// todo(fs): server. For the sake of simplicity we left the option out but
	// todo(fs): see the discussion in https://github.com/gopcua/opcua/pull/512
	// todo(fs): and you should find a commit that implements this option.
	if err := c.UpdateNamespacesWithContext(ctx); err != nil {
		c.CloseWithContext(ctx)
		stats.RecordError(err)

		return err
	}

	return nil
}

// monitor manages connection alteration
func (c *Client) monitor(ctx context.Context) {
	dlog := debug.NewPrefixLogger("client: monitor: ")

	dlog.Printf("start")
	defer dlog.Printf("done")

	defer c.mcancel()
	defer c.setState(Closed)

	action := none
	for {
		select {
		case <-ctx.Done():
			return

		case err, ok := <-c.sechanErr:
			stats.RecordError(err)

			// return if channel or connection is closed
			if !ok || err == io.EOF && c.State() == Closed {
				dlog.Print("closed")
				return
			}

			// tell the handler the connection is disconnected
			c.setState(Disconnected)
			dlog.Print("disconnected")

			if !c.cfg.sechan.AutoReconnect {
				// the connection is closed and should not be restored
				action = abortReconnect
				dlog.Print("auto-reconnect disabled")
				return
			}

			dlog.Print("auto-reconnecting")

			switch {
			case errors.Is(err, io.EOF):
				// the connection has been closed
				action = createSecureChannel

			case errors.Is(err, syscall.ECONNREFUSED):
				// the connection has been refused by the server
				action = abortReconnect

			case errors.Is(err, ua.StatusBadSecureChannelIDInvalid):
				// the secure channel has been rejected by the server
				action = createSecureChannel

			case errors.Is(err, ua.StatusBadSessionIDInvalid):
				// the session has been rejected by the server
				action = recreateSession

			case errors.Is(err, ua.StatusBadSubscriptionIDInvalid):
				// the subscription has been rejected by the server
				action = transferSubscriptions

			case errors.Is(err, ua.StatusBadCertificateInvalid):
				// todo(unknownet): recreate server certificate
				fallthrough

			default:
				// unknown error has occured
				action = createSecureChannel
			}

			c.setState(Disconnected)

			c.pauseSubscriptions(ctx)

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
						//
						// todo(fs): the two calls to Close() trigger a double-close on both the
						// todo(fs): secure channel and the UACP connection. I have guarded for this
						// todo(fs): with a sync.Once but that feels like a band-aid. We need to investigate
						// todo(fs): why we are trying to create a new secure channel when we shut the client
						// todo(fs): down.
						//
						// https://github.com/gopcua/opcua/pull/470
						c.conn.Close()
						if sc := c.SecureChannel(); sc != nil {
							sc.Close()
							c.setSecureChannel(nil)
						}

						c.setState(Reconnecting)

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

						s := c.Session()
						if s == nil {
							dlog.Printf("no session to restore")
							action = recreateSession
							continue
						}

						dlog.Printf("trying to restore session")
						if err := c.ActivateSessionWithContext(ctx, s); err != nil {
							dlog.Printf("restore session failed: %v", err)
							action = recreateSession
							continue
						}
						dlog.Printf("session restored")

						// todo(fs): see comment about guarding this with an option in Connect()
						dlog.Printf("trying to update namespaces")
						if err := c.UpdateNamespacesWithContext(ctx); err != nil {
							dlog.Printf("updating namespaces failed: %v", err)
							action = createSecureChannel
							continue
						}
						dlog.Printf("namespaces updated")

						action = restoreSubscriptions

					case recreateSession:
						dlog.Printf("action: recreateSession")

						// create a new session to replace the previous one

						dlog.Printf("trying to recreate session")
						s, err := c.CreateSessionWithContext(ctx, c.cfg.session)
						if err != nil {
							dlog.Printf("recreate session failed: %v", err)
							action = createSecureChannel
							continue
						}
						if err := c.ActivateSessionWithContext(ctx, s); err != nil {
							dlog.Printf("reactivate session failed: %v", err)
							action = createSecureChannel
							continue
						}
						dlog.Print("session recreated")

						// todo(fs): see comment about guarding this with an option in Connect()
						dlog.Printf("trying to update namespaces")
						if err := c.UpdateNamespacesWithContext(ctx); err != nil {
							dlog.Printf("updating namespaces failed: %v", err)
							action = createSecureChannel
							continue
						}
						dlog.Printf("namespaces updated")

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
						res, err := c.transferSubscriptions(ctx, subIDs)
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
							if err := c.republishSubscription(ctx, id, availableSeqs[id]); err != nil {
								dlog.Printf("republish of subscription %d failed", id)
								subsToRecreate = append(subsToRecreate, id)
							}
						}

						for _, id := range subsToRecreate {
							if err := c.recreateSubscription(ctx, id); err != nil {
								dlog.Printf("recreate subscripitions failed: %v", err)
								action = recreateSession
								continue
							}
						}

						c.setState(Connected)
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
			c.resumeSubscriptions(ctx)
			dlog.Printf("resumed subscriptions")
		}
	}
}

// Dial establishes a secure channel.
func (c *Client) Dial(ctx context.Context) error {
	// todo(fs): remove with v0.5.0
	if c.cfgerr != nil {
		return c.cfgerr
	}

	stats.Client().Add("Dial", 1)

	if c.SecureChannel() != nil {
		return errors.Errorf("secure channel already connected")
	}

	var err error
	var d = NewDialer(c.cfg)
	c.conn, err = d.Dial(ctx, c.endpointURL)
	if err != nil {
		return err
	}

	sc, err := uasc.NewSecureChannel(c.endpointURL, c.conn, c.cfg.sechan, c.sechanErr)
	if err != nil {
		c.conn.Close()
		return err
	}

	if err := sc.Open(ctx); err != nil {
		c.conn.Close()
		return err
	}
	c.setSecureChannel(sc)

	return nil
}

// Close closes the session and the secure channel.
//
// Note: Starting with v0.5 this method will require a context
// and the corresponding XXXWithContext(ctx) method will be removed.
func (c *Client) Close() error {
	return c.CloseWithContext(context.Background())
}

// Note: Starting with v0.5 this method is superseded by the non 'WithContext' method.
func (c *Client) CloseWithContext(ctx context.Context) error {
	stats.Client().Add("Close", 1)

	// try to close the session but ignore any error
	// so that we close the underlying channel and connection.
	c.CloseSessionWithContext(ctx)
	c.setState(Closed)

	if c.mcancel != nil {
		c.mcancel()
	}
	if c.SecureChannel() != nil {
		c.SecureChannel().Close()
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
	if c.conn != nil {
		c.conn.Close()
	}

	return nil
}

// State returns the current connection state.
func (c *Client) State() ConnState {
	return c.atomicState.Load().(ConnState)
}

func (c *Client) setState(s ConnState) {
	c.atomicState.Store(s)
	n := new(expvar.Int)
	n.Set(int64(s))
	stats.Client().Set("State", n)
}

// Namespaces returns the currently cached list of namespaces.
func (c *Client) Namespaces() []string {
	return c.atomicNamespaces.Load().([]string)
}

func (c *Client) setNamespaces(ns []string) {
	c.atomicNamespaces.Store(ns)
}

func (c *Client) publishTimeout() time.Duration {
	return c.atomicPublishTimeout.Load().(time.Duration)
}

func (c *Client) setPublishTimeout(d time.Duration) {
	c.atomicPublishTimeout.Store(d)
}

// SecureChannel returns the active secure channel.
func (c *Client) SecureChannel() *uasc.SecureChannel {
	return c.atomicSechan.Load().(*uasc.SecureChannel)
}

func (c *Client) setSecureChannel(sc *uasc.SecureChannel) {
	c.atomicSechan.Store(sc)
	stats.Client().Add("SecureChannel", 1)
}

// Session returns the active session.
func (c *Client) Session() *Session {
	return c.atomicSession.Load().(*Session)
}

func (c *Client) setSession(s *Session) {
	c.atomicSession.Store(s)
	stats.Client().Add("Session", 1)
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
//
// Note: Starting with v0.5 this method will require a context
// and the corresponding XXXWithContext(ctx) method will be removed.
func (c *Client) CreateSession(cfg *uasc.SessionConfig) (*Session, error) {
	return c.CreateSessionWithContext(context.Background(), cfg)
}

// Note: Starting with v0.5 this method is superseded by the non 'WithContext' method.
func (c *Client) CreateSessionWithContext(ctx context.Context, cfg *uasc.SessionConfig) (*Session, error) {
	if c.SecureChannel() == nil {
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
	// use c.SecureChannel().SendRequest() to enforce this.
	err := c.SecureChannel().SendRequestWithContext(ctx, req, nil, func(v interface{}) error {
		var res *ua.CreateSessionResponse
		if err := safeAssign(v, &res); err != nil {
			return err
		}

		err := c.SecureChannel().VerifySessionSignature(res.ServerCertificate, nonce, res.ServerSignature.Signature)
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
//
// Note: Starting with v0.5 this method will require a context
// and the corresponding XXXWithContext(ctx) method will be removed.
func (c *Client) ActivateSession(s *Session) error {
	return c.ActivateSessionWithContext(context.Background(), s)
}

// Note: Starting with v0.5 this method is superseded by the non 'WithContext' method.
func (c *Client) ActivateSessionWithContext(ctx context.Context, s *Session) error {
	if c.SecureChannel() == nil {
		return ua.StatusBadServerNotConnected
	}
	stats.Client().Add("ActivateSession", 1)
	sig, sigAlg, err := c.SecureChannel().NewSessionSignature(s.serverCertificate, s.serverNonce)
	if err != nil {
		log.Printf("error creating session signature: %s", err)
		return nil
	}

	switch tok := s.cfg.UserIdentityToken.(type) {
	case *ua.AnonymousIdentityToken:
		// nothing to do

	case *ua.UserNameIdentityToken:
		pass, passAlg, err := c.SecureChannel().EncryptUserPassword(s.cfg.AuthPolicyURI, s.cfg.AuthPassword, s.serverCertificate, s.serverNonce)
		if err != nil {
			log.Printf("error encrypting user password: %s", err)
			return err
		}
		tok.Password = pass
		tok.EncryptionAlgorithm = passAlg

	case *ua.X509IdentityToken:
		tokSig, tokSigAlg, err := c.SecureChannel().NewUserTokenSignature(s.cfg.AuthPolicyURI, s.serverCertificate, s.serverNonce)
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
	return c.SecureChannel().SendRequestWithContext(ctx, req, s.resp.AuthenticationToken, func(v interface{}) error {
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

		c.setSession(s)
		return nil
	})
}

// CloseSession closes the current session.
//
// See Part 4, 5.6.4
//
// Note: Starting with v0.5 this method will require a context
// and the corresponding XXXWithContext(ctx) method will be removed.
func (c *Client) CloseSession() error {
	return c.CloseSessionWithContext(context.Background())
}

// Note: Starting with v0.5 this method is superseded by the non 'WithContext' method.
func (c *Client) CloseSessionWithContext(ctx context.Context) error {
	stats.Client().Add("CloseSession", 1)
	if err := c.closeSession(ctx, c.Session()); err != nil {
		return err
	}
	c.setSession(nil)
	return nil
}

// closeSession closes the given session.
func (c *Client) closeSession(ctx context.Context, s *Session) error {
	if s == nil {
		return nil
	}
	req := &ua.CloseSessionRequest{DeleteSubscriptions: true}
	var res *ua.CloseSessionResponse
	return c.SendWithContext(ctx, req, func(v interface{}) error {
		return safeAssign(v, &res)
	})
}

// DetachSession removes the session from the client without closing it. The
// caller is responsible to close or re-activate the session. If the client
// does not have an active session the function returns no error.
//
// Note: Starting with v0.5 this method will require a context
// and the corresponding XXXWithContext(ctx) method will be removed.
func (c *Client) DetachSession() (*Session, error) {
	return c.DetachSessionWithContext(context.Background())
}

// Note: Starting with v0.5 this method is superseded by the non 'WithContext' method.
func (c *Client) DetachSessionWithContext(ctx context.Context) (*Session, error) {
	stats.Client().Add("DetachSession", 1)
	s := c.Session()
	c.setSession(nil)
	return s, nil
}

// Send sends the request via the secure channel and registers a handler for
// the response. If the client has an active session it injects the
// authentication token.
//
// Note: Starting with v0.5 this method will require a context
// and the corresponding XXXWithContext(ctx) method will be removed.
func (c *Client) Send(req ua.Request, h func(interface{}) error) error {
	return c.SendWithContext(context.Background(), req, h)
}

// Note: Starting with v0.5 this method is superseded by the non 'WithContext' method.
func (c *Client) SendWithContext(ctx context.Context, req ua.Request, h func(interface{}) error) error {
	stats.Client().Add("Send", 1)

	err := c.sendWithTimeout(ctx, req, c.cfg.sechan.RequestTimeout, h)
	stats.RecordError(err)

	return err
}

// sendWithTimeout sends the request via the secure channel with a custom timeout and registers a handler for
// the response. If the client has an active session it injects the
// authentication token.
func (c *Client) sendWithTimeout(ctx context.Context, req ua.Request, timeout time.Duration, h func(interface{}) error) error {
	if c.SecureChannel() == nil {
		return ua.StatusBadServerNotConnected
	}
	var authToken *ua.NodeID
	if s := c.Session(); s != nil {
		authToken = s.resp.AuthenticationToken
	}
	return c.SecureChannel().SendRequestWithTimeoutWithContext(ctx, req, authToken, timeout, h)
}

// Node returns a node object which accesses its attributes
// through this client connection.
func (c *Client) Node(id *ua.NodeID) *Node {
	return &Node{ID: id, c: c}
}

// GetEndpoints returns the list of available endpoints of the server.
//
// Note: Starting with v0.5 this method will require a context
// and the corresponding XXXWithContext(ctx) method will be removed.
func (c *Client) GetEndpoints() (*ua.GetEndpointsResponse, error) {
	return c.GetEndpointsWithContext(context.Background())
}

// Note: Starting with v0.5 this method is superseded by the non 'WithContext' method.
func (c *Client) GetEndpointsWithContext(ctx context.Context) (*ua.GetEndpointsResponse, error) {
	stats.Client().Add("GetEndpoints", 1)

	req := &ua.GetEndpointsRequest{
		EndpointURL: c.endpointURL,
	}
	var res *ua.GetEndpointsResponse
	err := c.SendWithContext(ctx, req, func(v interface{}) error {
		return safeAssign(v, &res)
	})
	return res, err
}

func cloneReadRequest(req *ua.ReadRequest) *ua.ReadRequest {
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
	return &ua.ReadRequest{
		MaxAge:             req.MaxAge,
		TimestampsToReturn: req.TimestampsToReturn,
		NodesToRead:        rvs,
	}
}

// Read executes a synchronous read request.
//
// By default, the function requests the value of the nodes
// in the default encoding of the server.
//
// Note: Starting with v0.5 this method will require a context
// and the corresponding XXXWithContext(ctx) method will be removed.
func (c *Client) Read(req *ua.ReadRequest) (*ua.ReadResponse, error) {
	return c.ReadWithContext(context.Background(), req)
}

// Note: Starting with v0.5 this method is superseded by the non 'WithContext' method.
func (c *Client) ReadWithContext(ctx context.Context, req *ua.ReadRequest) (*ua.ReadResponse, error) {
	stats.Client().Add("Read", 1)
	stats.Client().Add("NodesToRead", int64(len(req.NodesToRead)))

	// clone the request and the ReadValueIDs to set defaults without
	// manipulating them in-place.
	req = cloneReadRequest(req)

	var res *ua.ReadResponse
	err := c.SendWithContext(ctx, req, func(v interface{}) error {
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
//
// Note: Starting with v0.5 this method will require a context
// and the corresponding XXXWithContext(ctx) method will be removed.
func (c *Client) Write(req *ua.WriteRequest) (*ua.WriteResponse, error) {
	return c.WriteWithContext(context.Background(), req)
}

// Note: Starting with v0.5 this method is superseded by the non 'WithContext' method.
func (c *Client) WriteWithContext(ctx context.Context, req *ua.WriteRequest) (*ua.WriteResponse, error) {
	stats.Client().Add("Write", 1)
	stats.Client().Add("NodesToWrite", int64(len(req.NodesToWrite)))

	var res *ua.WriteResponse
	err := c.SendWithContext(ctx, req, func(v interface{}) error {
		return safeAssign(v, &res)
	})
	return res, err
}

func cloneBrowseRequest(req *ua.BrowseRequest) *ua.BrowseRequest {
	descs := make([]*ua.BrowseDescription, len(req.NodesToBrowse))
	for i, d := range req.NodesToBrowse {
		dc := &ua.BrowseDescription{}
		*dc = *d
		if dc.ReferenceTypeID == nil {
			dc.ReferenceTypeID = ua.NewNumericNodeID(0, id.References)
		}
		descs[i] = dc
	}
	reqc := &ua.BrowseRequest{
		View:                          req.View,
		RequestedMaxReferencesPerNode: req.RequestedMaxReferencesPerNode,
		NodesToBrowse:                 descs,
	}
	if reqc.View == nil {
		reqc.View = &ua.ViewDescription{}
	}
	if reqc.View.ViewID == nil {
		reqc.View.ViewID = ua.NewTwoByteNodeID(0)
	}
	return reqc
}

// Browse executes a synchronous browse request.
//
// Note: Starting with v0.5 this method will require a context
// and the corresponding XXXWithContext(ctx) method will be removed.
func (c *Client) Browse(req *ua.BrowseRequest) (*ua.BrowseResponse, error) {
	return c.BrowseWithContext(context.Background(), req)
}

// Note: Starting with v0.5 this method is superseded by the non 'WithContext' method.
func (c *Client) BrowseWithContext(ctx context.Context, req *ua.BrowseRequest) (*ua.BrowseResponse, error) {
	stats.Client().Add("Browse", 1)
	stats.Client().Add("NodesToBrowse", int64(len(req.NodesToBrowse)))

	// clone the request and the NodesToBrowse to set defaults without
	// manipulating them in-place.
	req = cloneBrowseRequest(req)

	var res *ua.BrowseResponse
	err := c.SendWithContext(ctx, req, func(v interface{}) error {
		return safeAssign(v, &res)
	})
	return res, err
}

// Call executes a synchronous call request for a single method.
//
// Note: Starting with v0.5 this method will require a context
// and the corresponding XXXWithContext(ctx) method will be removed.
func (c *Client) Call(req *ua.CallMethodRequest) (*ua.CallMethodResult, error) {
	return c.CallWithContext(context.Background(), req)
}

// Note: Starting with v0.5 this method is superseded by the non 'WithContext' method.
func (c *Client) CallWithContext(ctx context.Context, req *ua.CallMethodRequest) (*ua.CallMethodResult, error) {
	stats.Client().Add("Call", 1)

	creq := &ua.CallRequest{
		MethodsToCall: []*ua.CallMethodRequest{req},
	}
	var res *ua.CallResponse
	err := c.SendWithContext(ctx, creq, func(v interface{}) error {
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
//
// Note: Starting with v0.5 this method will require a context
// and the corresponding XXXWithContext(ctx) method will be removed.
func (c *Client) BrowseNext(req *ua.BrowseNextRequest) (*ua.BrowseNextResponse, error) {
	return c.BrowseNextWithContext(context.Background(), req)
}

// Note: Starting with v0.5 this method is superseded by the non 'WithContext' method.
func (c *Client) BrowseNextWithContext(ctx context.Context, req *ua.BrowseNextRequest) (*ua.BrowseNextResponse, error) {
	stats.Client().Add("BrowseNext", 1)

	var res *ua.BrowseNextResponse
	err := c.SendWithContext(ctx, req, func(v interface{}) error {
		return safeAssign(v, &res)
	})
	return res, err
}

// RegisterNodes registers node ids for more efficient reads.
//
// Part 4, Section 5.8.5
//
// Note: Starting with v0.5 this method will require a context
// and the corresponding XXXWithContext(ctx) method will be removed.
func (c *Client) RegisterNodes(req *ua.RegisterNodesRequest) (*ua.RegisterNodesResponse, error) {
	return c.RegisterNodesWithContext(context.Background(), req)
}

// Note: Starting with v0.5 this method is superseded by the non 'WithContext' method.
func (c *Client) RegisterNodesWithContext(ctx context.Context, req *ua.RegisterNodesRequest) (*ua.RegisterNodesResponse, error) {
	stats.Client().Add("RegisterNodes", 1)
	stats.Client().Add("NodesToRegister", int64(len(req.NodesToRegister)))

	var res *ua.RegisterNodesResponse
	err := c.SendWithContext(ctx, req, func(v interface{}) error {
		return safeAssign(v, &res)
	})
	return res, err
}

// UnregisterNodes unregisters node ids previously registered with RegisterNodes.
//
// Part 4, Section 5.8.6
//
// Note: Starting with v0.5 this method will require a context
// and the corresponding XXXWithContext(ctx) method will be removed.
func (c *Client) UnregisterNodes(req *ua.UnregisterNodesRequest) (*ua.UnregisterNodesResponse, error) {
	return c.UnregisterNodesWithContext(context.Background(), req)
}

// Note: Starting with v0.5 this method is superseded by the non 'WithContext' method.
func (c *Client) UnregisterNodesWithContext(ctx context.Context, req *ua.UnregisterNodesRequest) (*ua.UnregisterNodesResponse, error) {
	stats.Client().Add("UnregisterNodes", 1)
	stats.Client().Add("NodesToUnregister", int64(len(req.NodesToUnregister)))

	var res *ua.UnregisterNodesResponse
	err := c.SendWithContext(ctx, req, func(v interface{}) error {
		return safeAssign(v, &res)
	})
	return res, err
}

// Note: Starting with v0.5 this method will require a context
// and the corresponding XXXWithContext(ctx) method will be removed.
func (c *Client) HistoryReadRawModified(nodes []*ua.HistoryReadValueID, details *ua.ReadRawModifiedDetails) (*ua.HistoryReadResponse, error) {
	return c.HistoryReadRawModifiedWithContext(context.Background(), nodes, details)
}

// Note: Starting with v0.5 this method is superseded by the non 'WithContext' method.
func (c *Client) HistoryReadRawModifiedWithContext(ctx context.Context, nodes []*ua.HistoryReadValueID, details *ua.ReadRawModifiedDetails) (*ua.HistoryReadResponse, error) {
	stats.Client().Add("HistoryReadRawModified", 1)
	stats.Client().Add("HistoryReadValueID", int64(len(nodes)))

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
	err := c.SendWithContext(ctx, req, func(v interface{}) error {
		return safeAssign(v, &res)
	})
	return res, err
}

// NamespaceArray returns the list of namespaces registered on the server.
//
// Note: Starting with v0.5 this method will require a context
// and the corresponding XXXWithContext(ctx) method will be removed.
func (c *Client) NamespaceArray() ([]string, error) {
	return c.NamespaceArrayWithContext(context.Background())
}

// Note: Starting with v0.5 this method is superseded by the non 'WithContext' method.
func (c *Client) NamespaceArrayWithContext(ctx context.Context) ([]string, error) {
	stats.Client().Add("NamespaceArray", 1)
	node := c.Node(ua.NewNumericNodeID(0, id.Server_NamespaceArray))
	v, err := node.ValueWithContext(ctx)
	if err != nil {
		return nil, err
	}

	ns, ok := v.Value().([]string)
	if !ok {
		return nil, errors.Errorf("error fetching namespace array. id=%d, type=%T", v.Type(), v.Value())
	}
	return ns, nil
}

// FindNamespace returns the id of the namespace with the given name.
//
// Note: Starting with v0.5 this method will require a context
// and the corresponding XXXWithContext(ctx) method will be removed.
func (c *Client) FindNamespace(name string) (uint16, error) {
	return c.FindNamespaceWithContext(context.Background(), name)
}

// Note: Starting with v0.5 this method is superseded by the non 'WithContext' method.
func (c *Client) FindNamespaceWithContext(ctx context.Context, name string) (uint16, error) {
	stats.Client().Add("FindNamespace", 1)
	nsa, err := c.NamespaceArrayWithContext(ctx)
	if err != nil {
		return 0, err
	}
	for i, ns := range nsa {
		if ns == name {
			return uint16(i), nil
		}
	}
	return 0, errors.Errorf("namespace not found. name=%s", name)
}

// UpdateNamespaces updates the list of cached namespaces from the server.
//
// Note: Starting with v0.5 this method will require a context
// and the corresponding XXXWithContext(ctx) method will be removed.
func (c *Client) UpdateNamespaces() error {
	return c.UpdateNamespacesWithContext(context.Background())
}

// Note: Starting with v0.5 this method is superseded by the non 'WithContext' method.
func (c *Client) UpdateNamespacesWithContext(ctx context.Context) error {
	stats.Client().Add("UpdateNamespaces", 1)
	ns, err := c.NamespaceArrayWithContext(ctx)
	if err != nil {
		return err
	}
	c.setNamespaces(ns)
	return nil
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
