// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package server

import (
	"context"
	"crypto/rsa"
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/gopcua/opcua/debug"
	"github.com/gopcua/opcua/ua"
	"github.com/gopcua/opcua/uacp"
	"github.com/gopcua/opcua/uapolicy"
)

const defaultListenAddr = "opc.tcp://localhost:0"

// Server is a high-level OPC-UA Server
type Server struct {
	url string

	cfg *serverConfig

	mu        sync.Mutex
	Endpoints []*ua.EndpointDescription

	l  *uacp.Listener
	cb *channelBroker

	nextSecureChannelID uint32

	sb *sessionBroker

	// Service Handlers are methods called to respond to service requests from clients
	// All services should have a method here.
	serviceHandlers map[uint16]handlerFunc
}

type serverConfig struct {
	privateKey     *rsa.PrivateKey
	certificate    []byte
	applicationURI string

	enabledSec  []security
	enabledAuth []authMode
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
func New(url string, opts ...Option) *Server {
	s := &Server{
		url: url,
		cfg: &serverConfig{},
		cb:  newChannelBroker(),
		sb:  newSessionBroker(),
	}
	for _, opt := range opts {
		opt(s.cfg)
	}
	return s
}

// URL returns opc endpoint that the server is listening on.
func (s *Server) URL() string {
	if s.l != nil {
		return fmt.Sprintf("opc.tcp://%s", s.l.Addr())
	}

	return ""
}

// Start initializes and starts a Server listening on addr
// If s was not initialized with NewServer(), addr defaults
// to localhost:0 to let the OS select a random port
func (s *Server) Start(ctx context.Context) error {
	var err error

	// Register all service handlers
	s.initHandlers()

	if s.url == "" {
		s.url = defaultListenAddr
	}
	s.l, err = uacp.Listen(s.url, nil)
	if err != nil {
		return err
	}
	log.Printf("Started listening on %s", s.URL())

	s.generateEndpoints()

	if s.cb == nil {
		s.cb = newChannelBroker()
	}

	go s.acceptAndRegister(ctx, s.l)
	go s.monitorConnections(ctx)

	return nil
}

// Shutdown gracefully shuts the server down by closing all open connections,
// and stops listening on all endpoints
func (s *Server) Shutdown() error {
	// Close the listener, preventing new sessions from starting
	s.l.Close()

	// Shut down all secure channels and UACP connections
	s.cb.CloseAll()

	return nil
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
				if nErr, ok := err.(temporary); ok && nErr.Temporary() {
					continue
				}
				debug.Printf("error accepting connection: %s\n", err)
				break
			}

			go s.cb.RegisterConn(ctx, c, s.cfg.certificate, s.cfg.privateKey)
			debug.Printf("registered connection: %s\n", c.LocalAddr())
		}
	}
}

// monitorConnections reads messages off the secure channel connection and
// sends the message to the service handler
func (s *Server) monitorConnections(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			msg := s.cb.ReadMessage(ctx)
			if msg.Err != nil {
				debug.Printf("monitorConnections: Error received: %s\n", msg.Err)
			}
			debug.Printf("monitorConnections: Received Message: %T\n", msg.V)
			s.cb.mu.RLock()
			sc, ok := s.cb.s[msg.SCID]
			s.cb.mu.RUnlock()
			if !ok {
				debug.Printf("monitorConnections: Unknown SecureChannel: %d", msg.SCID)
				continue
			}

			// todo: should this be delegated to another goroutine in case handling this hangs?
			s.handleService(sc, msg.V.(ua.Request))
		}
	}
}

// generateEndpoints builds the endpoint list from the server's configuration
func (s *Server) generateEndpoints() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.Endpoints == nil {
		s.Endpoints = make([]*ua.EndpointDescription, 0)
	}

	for _, sec := range s.cfg.enabledSec {
		secLevel := uapolicy.SecurityLevel(sec.secPolicy, sec.secMode)

		ep := &ua.EndpointDescription{
			EndpointURL:   s.URL(), // todo: be able to listen on multiple adapters
			SecurityLevel: secLevel,
			Server: &ua.ApplicationDescription{
				ApplicationURI: s.cfg.applicationURI,
				ProductURI:     "urn:github.com:gopcua:server",
				ApplicationName: &ua.LocalizedText{
					EncodingMask: ua.LocalizedTextText,
					Text:         "GOPCUA",
				},
				ApplicationType:     ua.ApplicationTypeServer,
				GatewayServerURI:    "",
				DiscoveryProfileURI: "",
				DiscoveryURLs:       []string{s.l.Addr().String()},
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
		s.Endpoints = append(s.Endpoints, ep)
	}

	return nil
}
