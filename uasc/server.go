// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uasc

import (
	"context"
	"net"

	"github.com/wmnsk/gopcua/services"
)

// ListenAndAcceptSecureChannel starts UASC server on top of established transport connection.
func ListenAndAcceptSecureChannel(ctx context.Context, transport net.Conn, cfg *Config) (*SecureChannel, error) {
	s := &SecureChannel{
		lowerConn: transport,
		cfg:       cfg,
		state:     srvStateSecureChannelClosed,
		stateChan: make(chan secChanState),
		lenChan:   make(chan int),
		errChan:   make(chan error),
		rcvBuf:    make([]byte, 0xffff),
	}

	var message *Message
	n, err := s.lowerConn.Read(s.rcvBuf)
	if err != nil {
		return nil, err
	}

	message, err = Decode(s.rcvBuf[:n])
	if err != nil {
		return nil, err
	}

	switch msg := message.Service.(type) {
	case *services.OpenSecureChannelRequest:
		go s.handleOpenSecureChannelRequest(msg)
	default:
		return nil, ErrUnexpectedMessage
	}

	go s.monitorMessages(ctx)
	for {
		select {
		case state := <-s.stateChan:
			switch state {
			case srvStateSecureChannelOpened:
				return s, nil
			}
		case err := <-s.errChan:
			return nil, err
		}
	}
}
