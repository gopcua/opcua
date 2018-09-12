// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uasc

import (
	"context"
	"net"
	"time"
)

// OpenSecureChannel acts like net.Dial for OPC UA Secure Conversation network.
//
// Currently security mode=None is only supported. If secMode is not set to
//
// The first param ctx is to be passed to monitorMessages(), which monitors and handles
// incoming messages automatically in another goroutine.
func OpenSecureChannel(ctx context.Context, transportConn net.Conn, secMode uint32, lifetime uint32, nonce []byte) (*SecureChannel, error) {
	return openSecureChannel(ctx, transportConn, secMode, lifetime, nonce, 5*time.Second, 3)
}

/* XXX - maybe useful for users to have them?
func OpenSecureChannelSecNone(ctx context.Context, transportConn net.Conn, lifetime uint32) (*SecureChannel, error) {
	return openSecureChannel(ctx, transportConn, services.SecModeNone, lifetime, nil, 5*time.Second, 3)
}

func OpenSecureChannelSecSign(ctx context.Context, transportConn net.Conn, lifetime uint32) (*SecureChannel, error) {
	return openSecureChannel(ctx, transportConn, services.SecModeSign, lifetime, nil, 5*time.Second, 3)
}

func OpenSecureChannelSecSignAndEncrypt(ctx context.Context, transportConn net.Conn, lifetime uint32, nonce []byte) (*SecureChannel, error) {
	return openSecureChannel(ctx, transportConn, services.SecModeSignAndEncrypt, lifetime, nonce, 5*time.Second, 3)
}
*/

func openSecureChannel(ctx context.Context, transportConn net.Conn, secMode, lifetime uint32, nonce []byte, interval time.Duration, maxRetry int) (*SecureChannel, error) {
	secChan := &SecureChannel{
		lowerConn: transportConn,
		state:     cliStateSecureChannelClosed,
		stateChan: make(chan secChanState),
		lenChan:   make(chan int),
		errChan:   make(chan error),
		rcvBuf:    make([]byte, 0xffff),
	}

	if err := secChan.OpenSecureChannelRequest(secMode, lifetime, nonce); err != nil {
		return nil, err
	}
	sent := 1

	go secChan.monitorMessages(ctx)
	for {
		if sent > maxRetry {
			return nil, ErrTimeout
		}

		select {
		case s := <-secChan.stateChan:
			switch s {
			case cliStateSecureChannelOpened:
				return secChan, nil
			default:
				continue
			}
		case err := <-secChan.errChan:
			return nil, err
		case <-time.After(interval):
			if err := secChan.OpenSecureChannelRequest(secMode, lifetime, nonce); err != nil {
				return nil, err
			}
			sent++
		}
	}
}

// CreateSession acts like net.Dial for OPC UA Secure Conversation network.
//
// The first param ctx is to be passed to monitorMessages(), which monitors and handles
// incoming messages automatically in another goroutine.
func CreateSession(ctx context.Context, transport net.Conn, appURI, prodURI string, appType uint32, server, endpoint string, timeout uint64, maxRespSize uint32, cert, nonce []byte, interval time.Duration, maxRetry int) (*Session, error) {
	session := &Session{
		lowerConn:   transport,
		state:       cliStateSessionClosed,
		stateChan:   make(chan sessionState),
		lenChan:     make(chan int),
		errChan:     make(chan error),
		rcvBuf:      make([]byte, 0xffff),
		isActivated: false,
	}

	if err := session.CreateSessionRequest(appURI, prodURI, appType, server, endpoint, timeout, maxRespSize, cert, nonce); err != nil {
		return nil, err
	}
	sent := 1

	go session.monitorMessages(ctx)
	for {
		if sent > maxRetry {
			return nil, ErrTimeout
		}

		select {
		case s := <-session.stateChan:
			switch s {
			case cliStateSessionCreated:
				return session, nil
			default:
				continue
			}
		case err := <-session.errChan:
			return nil, err
		case <-time.After(interval):
			if err := session.CreateSessionRequest(appURI, prodURI, appType, server, endpoint, timeout, maxRespSize, cert, nonce); err != nil {
				return nil, err
			}
			sent++
		}
	}
}
