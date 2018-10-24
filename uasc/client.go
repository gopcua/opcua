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

// OpenSecureChannel acts like net.Dial for OPC UA Secure Conversation network.
//
// Currently security mode=None is only supported. If secMode is not set to
//
// The first param ctx is to be passed to monitor(), which monitors and handles
// incoming messages automatically in another goroutine.
func OpenSecureChannel(ctx context.Context, transportConn net.Conn, cfg *Config, interval time.Duration, maxRetry int) (*SecureChannel, error) {
	if err := cfg.validate("client"); err != nil {
		return nil, err
	}

	secChan := &SecureChannel{
		mu:        new(sync.Mutex),
		lowerConn: transportConn,
		reqHeader: services.NewRequestHeader(
			datatypes.NewTwoByteNodeID(0), time.Time{}, 0, 0,
			0xffff, "", services.NewNullAdditionalHeader(),
		),
		resHeader: services.NewResponseHeader(
			time.Time{}, 0, 0, datatypes.NewNullDiagnosticInfo(),
			[]string{}, services.NewNullAdditionalHeader(),
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
	go secChan.monitor(ctx)
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
