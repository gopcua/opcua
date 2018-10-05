// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uacp

import (
	"context"
	"net"
	"sync"
	"time"

	"github.com/wmnsk/gopcua/utils"
)

// Dial acts like net.Dial for OPC UA Connection Protocol network.
//
// Currently the endpoint can only be specified in "opc.tcp://<addr[:port]>/path" format.
//
// The first param ctx is to be passed to monitor(), which monitors and handles
// incoming messages automatically in another goroutine.
//
// If port is missing, ":4840" is automatically chosen.
// If laddr is nil, a local address is automatically chosen.
func Dial(ctx context.Context, endpoint string) (*Conn, error) {
	return dial(ctx, endpoint, 5*time.Second, 3)
}

// DialTimeout is Dial with retransmission interval and max retransmission count.
func DialTimeout(ctx context.Context, endpoint string, interval time.Duration, maxRetry int) (*Conn, error) {
	return dial(ctx, endpoint, interval, maxRetry)
}

func dial(ctx context.Context, endpoint string, interval time.Duration, maxRetry int) (*Conn, error) {
	network, raddr, err := utils.ResolveEndpoint(endpoint)
	if err != nil {
		return nil, err
	}

	conn := &Conn{
		mu:          new(sync.Mutex),
		state:       cliStateClosed,
		established: make(chan bool),
		lenChan:     make(chan int),
		errChan:     make(chan error),
		rcvBuf:      make([]byte, 0xffff),
		rep:         endpoint,
	}
	conn.lowerConn, err = net.Dial(network, raddr.String())
	if err != nil {
		return nil, err
	}

	if err := conn.Hello(); err != nil {
		return nil, err
	}
	sent := 1

	go conn.monitor(ctx)
	for {
		if sent > maxRetry {
			return nil, ErrTimeout
		}

		select {
		case s := <-conn.established:
			if s {
				return conn, nil
			}
		case err := <-conn.errChan:
			return nil, err
		case <-time.After(interval):
			if err := conn.Hello(); err != nil {
				return nil, err
			}
			sent++
		}
	}
}
