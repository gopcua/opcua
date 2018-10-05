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
