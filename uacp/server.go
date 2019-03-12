// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uacp

import (
	"context"
	"fmt"
)

type Listener struct{}

func (a *Listener) Close() error {
	return nil
}

func (l *Listener) Accept(ctx context.Context) (*Conn, error) {
	return nil, nil
}

func Listen(endpoint string, rcvBufSize uint32) (*Listener, error) {
	return nil, fmt.Errorf("not implemented")
}

// // Listener is a OPC UA Connection Protocol network listener.
// type Listener struct {
// 	lowerListener          net.Listener
// 	endpoint               string
// 	rcvBufSize, sndBufSize uint32
// }

// // Listen acts like net.Listen for OPC UA Connection Protocol networks.
// //
// // Currently the endpoint can only be specified in "opc.tcp://<addr[:port]>/path" format.
// //
// // If the IP field of laddr is nil or an unspecified IP address, Listen listens on all available unicast and anycast IP addresses of the local system.
// // If the Port field of laddr is 0, a port number is automatically chosen.
// func Listen(endpoint string, rcvBufSize uint32) (*Listener, error) {
// 	network, laddr, err := utils.ResolveEndpoint(endpoint)
// 	if err != nil {
// 		return nil, err
// 	}

// 	lis := &Listener{
// 		endpoint:   endpoint,
// 		rcvBufSize: rcvBufSize,
// 		sndBufSize: 0xffff,
// 	}
// 	lis.lowerListener, err = net.Listen(network, laddr.String())
// 	if err != nil {
// 		return nil, err
// 	}

// 	return lis, nil
// }

// // Accept accepts the next incoming call and returns the new connection.
// //
// // The first param ctx is to be passed to monitor(), which monitors and handles
// // incoming messages automatically in another goroutine.
// func (l *Listener) Accept(ctx context.Context) (*Conn, error) {
// 	var err error

// 	conn := &Conn{
// 		mu:          new(sync.Mutex),
// 		state:       srvStateClosed,
// 		established: make(chan bool),
// 		lenChan:     make(chan int),
// 		errChan:     make(chan error),
// 		rcvBuf:      make([]byte, l.rcvBufSize),
// 		lep:         l.endpoint,
// 	}
// 	conn.lowerConn, err = l.lowerListener.Accept()
// 	if err != nil {
// 		return nil, err
// 	}

// 	go conn.monitor(ctx)
// 	select {
// 	case ok := <-conn.established:
// 		if ok {
// 			return conn, nil
// 		}
// 	case err := <-conn.errChan:
// 		return nil, err
// 	}

// 	return nil, nil
// }

// // Close closes the Listener.
// func (l *Listener) Close() error {
// 	if err := l.lowerListener.Close(); err != nil {
// 		return err
// 	}

// 	l.endpoint = ""
// 	l.rcvBufSize = 0
// 	l.sndBufSize = 0
// 	return nil
// }

// // Addr returns the listener's network address.
// func (l *Listener) Addr() net.Addr {
// 	return l.lowerListener.Addr()
// }

// // Endpoint returns the listener's EndpointURL.
// func (l *Listener) Endpoint() string {
// 	return l.endpoint
// }
