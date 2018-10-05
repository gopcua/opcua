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
// The first param ctx is to be passed to monitorMessages(), which monitors and handles
// incoming messages automatically in another goroutine.
func OpenSecureChannel(ctx context.Context, transportConn net.Conn, cfg *Config, secMode uint32, lifetime uint32, nonce []byte) (*SecureChannel, error) {
	return openSecureChannel(ctx, transportConn, cfg, secMode, lifetime, nonce, 5*time.Second, 3)
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

func openSecureChannel(ctx context.Context, transportConn net.Conn, cfg *Config, secMode, lifetime uint32, nonce []byte, interval time.Duration, maxRetry int) (*SecureChannel, error) {
	secChan := &SecureChannel{
		mu:        new(sync.Mutex),
		lowerConn: transportConn,
		reqHeader: services.NewRequestHeader(
			datatypes.NewTwoByteNodeID(0), time.Now(), 0, 0,
			0xffff, "", services.NewNullAdditionalHeader(), nil,
		),
		resHeader: services.NewResponseHeader(
			time.Now(), 0, 0, services.NewNullDiagnosticInfo(),
			[]string{}, services.NewNullAdditionalHeader(), nil,
		),
		cfg:     cfg,
		state:   cliStateSecureChannelClosed,
		opened:  make(chan bool),
		lenChan: make(chan int),
		errChan: make(chan error),
		rcvBuf:  make([]byte, 0xffff),
	}

	if err := secChan.OpenSecureChannelRequest(secMode, lifetime, nonce); err != nil {
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
			if err := secChan.OpenSecureChannelRequest(secMode, lifetime, nonce); err != nil {
				return nil, err
			}
			sent++
		}
	}
}
