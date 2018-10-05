// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uasc

import (
	"context"
	"net"
	"sync"
	"time"

	"github.com/wmnsk/gopcua/datatypes"
	"github.com/wmnsk/gopcua/services"
)

// ListenAndAcceptSecureChannel starts UASC server on top of established transport connection.
func ListenAndAcceptSecureChannel(ctx context.Context, transport net.Conn, cfg *Config) (*SecureChannel, error) {
	secChan := &SecureChannel{
		mu:        new(sync.Mutex),
		lowerConn: transport,
		reqHeader: services.NewRequestHeader(
			datatypes.NewTwoByteNodeID(0), time.Now(), 0, 0,
			0xffff, "", services.NewNullAdditionalHeader(), nil,
		),
		resHeader: services.NewResponseHeader(
			time.Now(), 0, 0, services.NewNullDiagnosticInfo(),
			[]string{}, services.NewNullAdditionalHeader(), nil,
		),
		cfg:     cfg,
		state:   srvStateSecureChannelClosed,
		opened:  make(chan bool),
		lenChan: make(chan int),
		errChan: make(chan error),
		rcvBuf:  make([]byte, 0xffff),
	}

	go secChan.monitorMessages(ctx)
	for {
		select {
		case ok := <-secChan.opened:
			if ok {
				return secChan, nil
			}
		case err := <-secChan.errChan:
			return nil, err
		}
	}
}

// NewSessionConfigServer creates a new SessionConfigServer for server.
func NewSessionConfigServer(secChan *SecureChannel) *SessionConfig {
	return &SessionConfig{
		AuthenticationToken: datatypes.NewFourByteNodeID(0, uint16(time.Now().UnixNano())),
		SessionTimeout:      0xffff,
		ServerEndpoints: []*services.EndpointDescription{
			services.NewEndpointDescription(
				secChan.LocalEndpoint(), services.NewApplicationDescription(
					"urn:gopcua:client", "urn:gopcua", "gopcua - OPC UA implementation in pure Golang",
					services.AppTypeServer, "", "", []string{""},
				),
				secChan.cfg.Certificate, secChan.cfg.SecurityMode, secChan.cfg.SecurityPolicyURI,
				services.NewUserTokenPolicyArray(
					[]*services.UserTokenPolicy{
						services.NewUserTokenPolicy("", 0, "", "", ""),
					},
				), "", 0,
			),
		},
		ServerSignature: services.NewSignatureData("", nil),
	}
}

// ListenAndAcceptSession starts UASC server on top of established transport connection.
func ListenAndAcceptSession(ctx context.Context, secChan *SecureChannel, cfg *SessionConfig) (*Session, error) {
	session := &Session{
		mu:        new(sync.Mutex),
		secChan:   secChan,
		cfg:       cfg,
		state:     srvStateSessionClosed,
		created:   make(chan bool),
		activated: make(chan bool),
		lenChan:   make(chan int),
		errChan:   make(chan error),
		rcvBuf:    make([]byte, 0xffff),
	}

	go session.monitor(ctx)
	for {
		select {
		case ok := <-session.activated:
			if ok {
				return session, nil
			}
		case err := <-session.errChan:
			return nil, err
		}
	}
}
