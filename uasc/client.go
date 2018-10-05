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
	"github.com/wmnsk/gopcua/id"
	"github.com/wmnsk/gopcua/services"
)

// OpenSecureChannel acts like net.Dial for OPC UA Secure Conversation network.
//
// Currently security mode=None is only supported. If secMode is not set to
//
// The first param ctx is to be passed to monitorMessages(), which monitors and handles
// incoming messages automatically in another goroutine.
func OpenSecureChannel(ctx context.Context, transportConn net.Conn, cfg *Config) (*SecureChannel, error) {
	return openSecureChannel(ctx, transportConn, cfg, 5*time.Second, 3)
}

/* XXX - maybe useful for users to have them?
func OpenSecureChannelSecNone(ctx context.Context, transportConn net.Conn, lifetime uint32) (*SecureChannel, error) {
	return openSecureChannel(ctx, transportConn, services.SecModeNone, lifetime, nil, 5*time.Second, 3)
}

func OpenSecureChannelSecSign(ctx context.Context, transportConn net.Conn, lifetime uint32) (*SecureChannel, error) {
	return openSecureChannel(ctx, transportConn, services.SecModeSign, lifetime, nil, 5*time.Second, 3)
}

func OpenSecureChannelSecSignAndEncrypt(ctx context.Context, transportConn net.Conn) (*SecureChannel, error) {
	return openSecureChannel(ctx, transportConn, services.SecModeSignAndEncrypt, lifetime, nonce, 5*time.Second, 3)
}
*/

func openSecureChannel(ctx context.Context, transportConn net.Conn, cfg *Config, interval time.Duration, maxRetry int) (*SecureChannel, error) {
	secChan := &SecureChannel{
		mu:        new(sync.Mutex),
		lowerConn: transportConn,
		reqHeader: services.NewRequestHeader(
			datatypes.NewTwoByteNodeID(0), time.Time{}, 0, 0,
			0xffff, "", services.NewNullAdditionalHeader(), nil,
		),
		resHeader: services.NewResponseHeader(
			time.Time{}, 0, 0, services.NewNullDiagnosticInfo(),
			[]string{}, services.NewNullAdditionalHeader(), nil,
		),
		cfg:     cfg,
		state:   cliStateSecureChannelClosed,
		opened:  make(chan bool),
		lenChan: make(chan int),
		errChan: make(chan error),
		rcvBuf:  make([]byte, 0xffff),
	}

	if err := secChan.OpenSecureChannelRequest(); err != nil {
		return nil, err
	}
	sent := 1

	secChan.state = cliStateOpenSecureChannelSent
	go secChan.monitorMessages(ctx)
	for {
		if sent > maxRetry {
			return nil, ErrTimeout
		}

		select {
		case ok := <-secChan.opened:
			if ok {
				return secChan, nil
			}
		case err := <-secChan.errChan:
			return nil, err
		case <-time.After(interval):
			if err := secChan.OpenSecureChannelRequest(); err != nil {
				return nil, err
			}
			sent++
		}
	}
}

// NewSessionConfigClient creates a SessionConfig for client.
func NewSessionConfigClient(locales []string) *SessionConfig {
	return &SessionConfig{
		SessionTimeout: 0xffff,
		ClientDescription: services.NewApplicationDescription(
			"urn:gopcua:client", "urn:gopcua", "gopcua - OPC UA implementation in pure Golang",
			services.AppTypeClient, "", "", []string{""},
		),
		ClientSignature: services.NewSignatureData("", nil),
		ClientSoftwareCertificates: []*services.SignedSoftwareCertificate{
			services.NewSignedSoftwareCertificate(nil, nil),
		},
		LocaleIDs: locales,
		UserIdentityToken: datatypes.NewExtensionObject(
			datatypes.NewExpandedNodeID(
				false, false,
				datatypes.NewFourByteNodeID(0, id.AnonymousIdentityToken_Encoding_DefaultBinary),
				"", 0,
			),
			0x01, []byte("anonymous"),
		),
		UserTokenSignature: services.NewSignatureData("", nil),
	}
}

// CreateSession creates a session on top of SecureChannel.
func CreateSession(ctx context.Context, secChan *SecureChannel, cfg *SessionConfig, maxRetry int, interval time.Duration) (*Session, error) {
	session := &Session{
		mu:        new(sync.Mutex),
		secChan:   secChan,
		cfg:       cfg,
		state:     cliStateSessionClosed,
		created:   make(chan bool),
		activated: make(chan bool),
		lenChan:   make(chan int),
		errChan:   make(chan error),
		rcvBuf:    make([]byte, 0xffff),
	}

	if err := session.CreateSessionRequest(); err != nil {
		return nil, err
	}
	sent := 1

	session.state = cliStateCreateSessionSent
	go session.monitor(ctx)
	for {
		if sent > maxRetry {
			return nil, ErrTimeout
		}

		select {
		case ok := <-session.created:
			if ok {
				return session, nil
			}
		case err := <-session.errChan:
			return nil, err
		case <-time.After(interval):
			if err := session.CreateSessionRequest(); err != nil {
				return nil, err
			}
			sent++
		}
	}
}

// Activate activates the session.
func (s *Session) Activate() error {
	if err := s.ActivateSessionRequest(); err != nil {
		return err
	}
	sent := 0

	s.state = cliStateActivateSessionSent
	for {
		if sent > 3 {
			return ErrTimeout
		}

		select {
		case ok := <-s.activated:
			if ok {
				return nil
			}
		case err := <-s.errChan:
			return err
		case <-time.After(5 * time.Second):
			if err := s.ActivateSessionRequest(); err != nil {
				return err
			}
			sent++
		}
	}
}
