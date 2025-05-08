// Copyright 2018-2020 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package server

import (
	"context"
	"crypto/rsa"
	"encoding/xml"
	"fmt"
	"log/slog"
	"net"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/gopcua/opcua/id"
	"github.com/gopcua/opcua/schema"
	"github.com/gopcua/opcua/ua"
	"github.com/gopcua/opcua/uacp"
	"github.com/gopcua/opcua/uapolicy"
)

//go:generate go run ../cmd/predefined-nodes/main.go

const defaultListenAddr = "opc.tcp://localhost:0"

// Server is a high-level OPC-UA Server
type Server struct {
	url string

	cfg *serverConfig

	mu         sync.Mutex
	status     *ua.ServerStatusDataType
	endpoints  []*ua.EndpointDescription
	namespaces []NameSpace

	l  *uacp.Listener
	cb *channelBroker
	sb *sessionBroker

	// nextSecureChannelID uint32

	// Service Handlers are methods called to respond to service requests from clients
	// All services should have a method here.
	handlers map[uint16]Handler

	SubscriptionService  *SubscriptionService
	MonitoredItemService *MonitoredItemService
}

type serverConfig struct {
	privateKey     *rsa.PrivateKey
	certificate    []byte
	applicationURI string

	endpoints []string

	applicationName  string
	manufacturerName string
	productName      string
	softwareVersion  string

	enabledSec  []security
	enabledAuth []authMode

	cap ServerCapabilities

	logger Logger
}

var capabilities = ServerCapabilities{
	OperationalLimits: OperationalLimits{
		MaxNodesPerRead: 32,
	},
}

type ServerCapabilities struct {
	OperationalLimits OperationalLimits
}

type OperationalLimits struct {
	MaxNodesPerRead uint32
}

type authMode struct {
	tokenType ua.UserTokenType
}

type security struct {
	secPolicy string
	secMode   ua.MessageSecurityMode
}

// New returns an initialized OPC-UA server.
// Call Start() afterwards to begin listening and serving connections
func New(opts ...Option) *Server {
	cfg := &serverConfig{
		cap:              capabilities,
		applicationName:  "GOPCUA",               // override with the ServerName option
		manufacturerName: "The gopcua Team",      // override with the ManufacturerName option
		productName:      "gopcua OPC/UA Server", // override with the ProductName option
		softwareVersion:  "0.0.0-dev",            // override with the SoftwareVersion option
		logger:           slog.Default(),
	}
	for _, opt := range opts {
		opt(cfg)
	}
	url := ""
	if len(cfg.endpoints) != 0 {
		url = cfg.endpoints[0]
	}
	if cfg.logger == nil {
		cfg.logger = &DiscardLogger{}
	}

	s := &Server{
		url:      url,
		cfg:      cfg,
		cb:       newChannelBroker(cfg.logger),
		sb:       newSessionBroker(cfg.logger),
		handlers: make(map[uint16]Handler),
		namespaces: []NameSpace{
			NewNameSpace("http://opcfoundation.org/UA/"), // ns:0
		},
		status: &ua.ServerStatusDataType{
			StartTime:   time.Now(),
			CurrentTime: time.Now(),
			State:       ua.ServerStateSuspended,
			BuildInfo: &ua.BuildInfo{
				ProductURI:       "https://github.com/gopcua/opcua",
				ManufacturerName: cfg.manufacturerName,
				ProductName:      cfg.productName,
				SoftwareVersion:  "0.0.0-dev",
				BuildNumber:      "",
				BuildDate:        time.Time{},
			},
			SecondsTillShutdown: 0,
			ShutdownReason:      &ua.LocalizedText{},
		},
	}

	// init server address space
	//for _, n := range PredefinedNodes() {
	//s.namespaces[0].AddNode(n)
	//}

	// this nodeset is pre-compiled into the binary and contains a known set of nodes
	// so it should *always* work ok.
	var nodes schema.UANodeSet
	xml.Unmarshal(schema.OpcUaNodeSet2, &nodes)

	n0, ok := s.namespaces[0].(*NodeNameSpace)
	n0.srv = s
	if !ok {
		// this should never happen because we just set namespace 0 to be a node namespace
		panic("Namespace 0 is not a node namespace!")
	}
	s.ImportNodeSet(&nodes)

	s.namespaces[0].AddNode(CurrentTimeNode())
	s.namespaces[0].AddNode(NamespacesNode(s))
	for _, n := range ServerStatusNodes(s, s.namespaces[0].Node(ua.NewNumericNodeID(0, id.Server))) {
		s.namespaces[0].AddNode(n)
	}
	for _, n := range ServerCapabilitiesNodes(s) {
		s.namespaces[0].AddNode(n)
	}

	return s
}

func (srv *Server) Session(hdr *ua.RequestHeader) *session {
	return srv.sb.Session(hdr.AuthenticationToken)
}

func (srv *Server) Namespace(id int) (NameSpace, error) {
	srv.mu.Lock()
	defer srv.mu.Unlock()
	if id < len(srv.namespaces) {
		return srv.namespaces[id], nil
	}
	return nil, fmt.Errorf("namespace %d not found", id)
}

func (srv *Server) Namespaces() []NameSpace {
	srv.mu.Lock()
	defer srv.mu.Unlock()
	return srv.namespaces
}

func (srv *Server) ChangeNotification(n *ua.NodeID) {
	srv.MonitoredItemService.ChangeNotification(n)
}

// for now, the address space of the server is split up into namespaces.
// this means that when we look up a node, we need to ask the specific namespace
// it belongs to for it instead of just a general lookup by ID
//
// the refRoot and refObjects flags can be used to automatically add a reference to the new Namespaces
// root or objects object respectively to the namespace 0
func (srv *Server) AddNamespace(ns NameSpace) int {
	srv.mu.Lock()
	defer srv.mu.Unlock()
	if idx := slices.Index(srv.namespaces, ns); idx >= 0 {
		return idx
	}
	ns.SetID(uint16(len(srv.namespaces)))
	srv.namespaces = append(srv.namespaces, ns)

	if ns.ID() == 0 {
		return 0

	}

	return len(srv.namespaces) - 1
}

func (srv *Server) Endpoints() []*ua.EndpointDescription {
	srv.mu.Lock()
	defer srv.mu.Unlock()
	return slices.Clone(srv.endpoints)
}

// Status returns the current server status.
func (srv *Server) Status() *ua.ServerStatusDataType {
	status := new(ua.ServerStatusDataType)
	srv.mu.Lock()
	*status = *srv.status
	srv.mu.Unlock()
	status.CurrentTime = time.Now()
	return status
}

// URLs returns opc endpoint that the server is listening on.
func (srv *Server) URLs() []string {
	return srv.cfg.endpoints
}

// Start initializes and starts a Server listening on addr
// If s was not initialized with NewServer(), addr defaults
// to localhost:0 to let the OS select a random port
func (srv *Server) Start(ctx context.Context) error {
	var err error

	if len(srv.cfg.endpoints) == 0 {
		return fmt.Errorf("cannot start server: no endpoints defined")
	}

	// Register all service handlers
	srv.initHandlers()

	if srv.url == "" {
		srv.url = defaultListenAddr
	}
	srv.l, err = uacp.Listen(ctx, srv.url, nil)
	if err != nil {
		return err
	}
	srv.cfg.logger.Info("Started listening on %v", srv.URLs())

	srv.initEndpoints()
	srv.setServerState(ua.ServerStateRunning)

	if srv.cb == nil {
		srv.cb = newChannelBroker(srv.cfg.logger)
	}

	go srv.acceptAndRegister(ctx, srv.l)
	go srv.monitorConnections(ctx)

	return nil
}

func (srv *Server) setServerState(state ua.ServerState) {
	srv.mu.Lock()
	srv.status.State = state
	srv.mu.Unlock()
}

// Close gracefully shuts the server down by closing all open connections,
// and stops listening on all endpoints
func (srv *Server) Close() error {
	srv.setServerState(ua.ServerStateShutdown)

	// Close the listener, preventing new sessions from starting
	if srv.l != nil {
		srv.l.Close()
	}

	// Shut down all secure channels and UACP connections
	return srv.cb.Close()
}

type temporary interface {
	Temporary() bool
}

func (srv *Server) acceptAndRegister(ctx context.Context, l *uacp.Listener) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			c, err := l.Accept(ctx)
			if err != nil {
				switch x := err.(type) {
				case *net.OpError:
					// socket closed. Cannot recover from this.
					srv.cfg.logger.Error("socket closed: %s", err)
					return
				case temporary:
					if x.Temporary() {
						continue
					}
				default:
					srv.cfg.logger.Error("error accepting connection: %s", err)
					continue
				}
			}

			go srv.cb.RegisterConn(ctx, c, srv.cfg.certificate, srv.cfg.privateKey)
			srv.cfg.logger.Info("registered connection: %s", c.RemoteAddr())
		}
	}
}

// monitorConnections reads messages off the secure channel connection and
// sends the message to the service handler
func (srv *Server) monitorConnections(ctx context.Context) {
	for ctx.Err() == nil {
		msg := srv.cb.ReadMessage(ctx)
		if msg == nil {
			continue // ctx is likely done, ctx.Err will be non-nil
		}
		if msg.Err != nil {
			srv.cfg.logger.Error("monitorConnections: Error received: %s\n", msg.Err)
			continue // todo(fs): close SC???
		}
		if resp := msg.Response(); resp != nil {
			srv.cfg.logger.Error("monitorConnections: Server received response %T ???", resp)
			continue // todo(fs): close SC???
		}
		srv.cfg.logger.Debug("monitorConnections: Received Message: %T", msg.Request())
		srv.cb.mu.RLock()
		sc, ok := srv.cb.s[msg.SecureChannelID]
		srv.cb.mu.RUnlock()
		if !ok {
			// if the secure channel ID is 0, this is probably a open secure channel request.
			srv.cfg.logger.Error("monitorConnections: Unknown SecureChannel: %d", msg.SecureChannelID)
			continue
		}

		// todo: should this be delegated to another goroutine in case handling this hangs?
		srv.handleService(ctx, sc, msg.RequestID, msg.Request())
	}
}

// initEndpoints builds the endpoint list from the server's configuration
func (srv *Server) initEndpoints() {
	var endpoints []*ua.EndpointDescription
	for _, sec := range srv.cfg.enabledSec {
		for _, url := range srv.cfg.endpoints {
			secLevel := uapolicy.SecurityLevel(sec.secPolicy, sec.secMode)

			ep := &ua.EndpointDescription{
				EndpointURL:   url, // todo: be able to listen on multiple adapters
				SecurityLevel: secLevel,
				Server: &ua.ApplicationDescription{
					ApplicationURI: srv.cfg.applicationURI,
					ProductURI:     "urn:github.com:gopcua:server",
					ApplicationName: &ua.LocalizedText{
						EncodingMask: ua.LocalizedTextText,
						Text:         srv.cfg.applicationName,
					},
					ApplicationType:     ua.ApplicationTypeServer,
					GatewayServerURI:    "",
					DiscoveryProfileURI: "",
					DiscoveryURLs:       srv.URLs(),
				},
				ServerCertificate:   srv.cfg.certificate,
				SecurityMode:        sec.secMode,
				SecurityPolicyURI:   sec.secPolicy,
				TransportProfileURI: "http://opcfoundation.org/UA-Profile/Transport/uatcp-uasc-uabinary",
			}

			for _, auth := range srv.cfg.enabledAuth {
				for _, authSec := range srv.cfg.enabledSec {
					if auth.tokenType == ua.UserTokenTypeAnonymous {
						authSec.secPolicy = "http://opcfoundation.org/UA/SecurityPolicy#None"
					}

					if auth.tokenType != ua.UserTokenTypeAnonymous && authSec.secPolicy == "http://opcfoundation.org/UA/SecurityPolicy#None" {
						continue
					}

					policyID := strings.ToLower(
						strings.TrimPrefix(auth.tokenType.String(), "UserTokenType") +
							"_" +
							strings.TrimPrefix(authSec.secPolicy, "http://opcfoundation.org/UA/SecurityPolicy#"),
					)

					var dup bool
					for _, uit := range ep.UserIdentityTokens {
						if uit.PolicyID == policyID {
							dup = true
							break
						}
					}

					if dup {
						continue
					}
					tok := &ua.UserTokenPolicy{
						PolicyID:          policyID,
						TokenType:         auth.tokenType,
						IssuedTokenType:   "",
						IssuerEndpointURL: "",
						SecurityPolicyURI: authSec.secPolicy,
					}

					ep.UserIdentityTokens = append(ep.UserIdentityTokens, tok)
				}
			}
			endpoints = append(endpoints, ep)
		}
	}

	srv.mu.Lock()
	srv.endpoints = endpoints
	srv.mu.Unlock()
}

func (srv *Server) Node(nid *ua.NodeID) *Node {
	ns := int(nid.Namespace())
	if ns < len(srv.namespaces) {
		return srv.namespaces[ns].Node(nid)
	}
	return nil
}
