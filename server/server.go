// Copyright 2018-2020 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package server

import (
	"context"
	"crypto/rsa"
	"encoding/xml"
	"fmt"
	"log"
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
	}
	for _, opt := range opts {
		opt(cfg)
	}
	url := ""
	if len(cfg.endpoints) != 0 {
		url = cfg.endpoints[0]
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
		log.Panic("Namespace 0 is not a node namespace!")
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

func (s *Server) Session(hdr *ua.RequestHeader) *session {
	return s.sb.Session(hdr.AuthenticationToken)
}

func (s *Server) Namespace(id int) (NameSpace, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if id < len(s.namespaces) {
		return s.namespaces[id], nil
	}
	return nil, fmt.Errorf("namespace %d not found", id)
}

func (s *Server) Namespaces() []NameSpace {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.namespaces
}

func (s *Server) ChangeNotification(n *ua.NodeID) {
	s.MonitoredItemService.ChangeNotification(n)
}

// for now, the address space of the server is split up into namespaces.
// this means that when we look up a node, we need to ask the specific namespace
// it belongs to for it instead of just a general lookup by ID
//
// the refRoot and refObjects flags can be used to automatically add a reference to the new Namespaces
// root or objects object respectively to the namespace 0
func (s *Server) AddNamespace(ns NameSpace) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	if idx := slices.Index(s.namespaces, ns); idx >= 0 {
		return idx
	}
	ns.SetID(uint16(len(s.namespaces)))
	s.namespaces = append(s.namespaces, ns)

	if ns.ID() == 0 {
		return 0

	}

	return len(s.namespaces) - 1
}

func (s *Server) Endpoints() []*ua.EndpointDescription {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.endpoints)
}

// Status returns the current server status.
func (s *Server) Status() *ua.ServerStatusDataType {
	status := new(ua.ServerStatusDataType)
	s.mu.Lock()
	*status = *s.status
	s.mu.Unlock()
	status.CurrentTime = time.Now()
	return status
}

// URLs returns opc endpoint that the server is listening on.
func (s *Server) URLs() []string {
	return s.cfg.endpoints
}

// Start initializes and starts a Server listening on addr
// If s was not initialized with NewServer(), addr defaults
// to localhost:0 to let the OS select a random port
func (s *Server) Start(ctx context.Context) error {
	var err error

	if len(s.cfg.endpoints) == 0 {
		return fmt.Errorf("cannot start server: no endpoints defined")
	}

	// Register all service handlers
	s.initHandlers()

	if s.url == "" {
		s.url = defaultListenAddr
	}
	s.l, err = uacp.Listen(s.url, nil)
	if err != nil {
		return err
	}
	log.Printf("Started listening on %v", s.URLs())

	s.initEndpoints()
	s.setServerState(ua.ServerStateRunning)

	if s.cb == nil {
		s.cb = newChannelBroker(s.cfg.logger)
	}

	go s.acceptAndRegister(ctx, s.l)
	go s.monitorConnections(ctx)

	return nil
}

func (s *Server) setServerState(state ua.ServerState) {
	s.mu.Lock()
	s.status.State = state
	s.mu.Unlock()
}

// Close gracefully shuts the server down by closing all open connections,
// and stops listening on all endpoints
func (s *Server) Close() error {
	s.setServerState(ua.ServerStateShutdown)

	// Close the listener, preventing new sessions from starting
	if s.l != nil {
		s.l.Close()
	}

	// Shut down all secure channels and UACP connections
	return s.cb.Close()
}

type temporary interface {
	Temporary() bool
}

func (s *Server) acceptAndRegister(ctx context.Context, l *uacp.Listener) {
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
					if s.cfg.logger != nil {
						s.cfg.logger.Error("socket closed: %s", err)
					}
					return
				case temporary:
					if x.Temporary() {
						continue
					}
				default:
					if s.cfg.logger != nil {
						s.cfg.logger.Error("error accepting connection: %s", err)
					}
					continue
				}
			}

			go s.cb.RegisterConn(ctx, c, s.cfg.certificate, s.cfg.privateKey)
			if s.cfg.logger != nil {
				s.cfg.logger.Info("registered connection: %s", c.RemoteAddr())
			}
		}
	}
}

// monitorConnections reads messages off the secure channel connection and
// sends the message to the service handler
func (s *Server) monitorConnections(ctx context.Context) {
	for ctx.Err() == nil {
		msg := s.cb.ReadMessage(ctx)
		if msg == nil {
			continue // ctx is likely done, ctx.Err will be non-nil
		}
		if msg.Err != nil {
			if s.cfg.logger != nil {
				s.cfg.logger.Error("monitorConnections: Error received: %s\n", msg.Err)
			}
			continue // todo(fs): close SC???
		}
		if resp := msg.Response(); resp != nil {
			if s.cfg.logger != nil {
				s.cfg.logger.Error("monitorConnections: Server received response %T ???", resp)
			}
			continue // todo(fs): close SC???
		}
		if s.cfg.logger != nil {
			s.cfg.logger.Debug("monitorConnections: Received Message: %T", msg.Request())
		}
		s.cb.mu.RLock()
		sc, ok := s.cb.s[msg.SecureChannelID]
		s.cb.mu.RUnlock()
		if !ok {
			// if the secure channel ID is 0, this is probably a open secure channel request.
			if s.cfg.logger != nil && msg.SecureChannelID != 0 {
				s.cfg.logger.Error("monitorConnections: Unknown SecureChannel: %d", msg.SecureChannelID)
			}
			continue
		}

		// todo: should this be delegated to another goroutine in case handling this hangs?
		s.handleService(ctx, sc, msg.RequestID, msg.Request())
	}
}

// initEndpoints builds the endpoint list from the server's configuration
func (s *Server) initEndpoints() {
	var endpoints []*ua.EndpointDescription
	for _, sec := range s.cfg.enabledSec {
		for _, url := range s.cfg.endpoints {
			secLevel := uapolicy.SecurityLevel(sec.secPolicy, sec.secMode)

			ep := &ua.EndpointDescription{
				EndpointURL:   url, // todo: be able to listen on multiple adapters
				SecurityLevel: secLevel,
				Server: &ua.ApplicationDescription{
					ApplicationURI: s.cfg.applicationURI,
					ProductURI:     "urn:github.com:gopcua:server",
					ApplicationName: &ua.LocalizedText{
						EncodingMask: ua.LocalizedTextText,
						Text:         s.cfg.applicationName,
					},
					ApplicationType:     ua.ApplicationTypeServer,
					GatewayServerURI:    "",
					DiscoveryProfileURI: "",
					DiscoveryURLs:       s.URLs(),
				},
				ServerCertificate:   s.cfg.certificate,
				SecurityMode:        sec.secMode,
				SecurityPolicyURI:   sec.secPolicy,
				TransportProfileURI: "http://opcfoundation.org/UA-Profile/Transport/uatcp-uasc-uabinary",
			}

			for _, auth := range s.cfg.enabledAuth {
				for _, authSec := range s.cfg.enabledSec {
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

	s.mu.Lock()
	s.endpoints = endpoints
	s.mu.Unlock()
}

func (s *Server) Node(nid *ua.NodeID) *Node {
	ns := int(nid.Namespace())
	if ns < len(s.namespaces) {
		return s.namespaces[ns].Node(nid)
	}
	return nil
}
