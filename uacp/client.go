// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uacp

import (
	"net"
	"time"

	"github.com/wmnsk/gopcua/errors"
	"github.com/wmnsk/gopcua/utils"
)

// Client is the configuration that OPC UA Connection Protocol client should have.
type Client struct {
	Endpoint           string
	ReceiveBufferSize  uint32
	SendBufferSize     uint32
	RetransmitInterval time.Duration
	MaxRetransmit      int
}

// NewClient creates a new Client with minimum mandatory parameters.
func NewClient(endpoint string, rcvBufSize uint32, interval time.Duration, maxRetry int) *Client {
	return &Client{
		Endpoint:           endpoint,
		ReceiveBufferSize:  rcvBufSize,
		SendBufferSize:     0xffff,
		RetransmitInterval: interval,
		MaxRetransmit:      maxRetry,
	}
}

// Dial acts like net.Dial for OPC UA Connection Protocol network.
//
// Currently the endpoint can only be specified in "opc.tcp://<addr[:port]>/path" format.
//
// If port is missing, ":4840" is automatically chosen.
//
// If laddr is nil, a local address is automatically chosen.
func (c *Client) Dial(laddr *net.TCPAddr) (*Conn, error) {
	network, raddr, err := utils.ResolveEndpoint(c.Endpoint)
	if err != nil {
		return nil, err
	}

	conn := &Conn{
		cliCfg:    c,
		state:     cliStateClosed,
		stateChan: make(chan state),
		lenChan:   make(chan int),
		errChan:   make(chan error),
		rcvBuf:    make([]byte, c.ReceiveBufferSize),
		rep:       c.Endpoint,
	}
	conn.tcpConn, err = net.DialTCP(network, laddr, raddr)
	if err != nil {
		return nil, err
	}

	if err := conn.Hello(c); err != nil {
		return nil, err
	}
	sent := 1

	go conn.startFSM()
	for {
		if sent > c.MaxRetransmit {
			return nil, errors.New("timed out")
		}

		select {
		case s := <-conn.stateChan:
			switch s {
			case cliStateEstablished:
				return conn, nil
			default:
				continue
			}
		case err := <-conn.errChan:
			return nil, err
		case <-time.After(c.RetransmitInterval):
			if err := conn.Hello(c); err != nil {
				return nil, err
			}
			sent++
		}
	}
}

// OpenConnection creates *uacp.Conn on top of the existing connection(net.Conn).
//
// Note: Currently conn only supports *net.TCPConn.
func (c *Client) OpenConnection(conn net.Conn) (*Conn, error) {
	switch cc := conn.(type) {
	case *net.TCPConn:
		uaConn := &Conn{
			tcpConn: cc,
			rep:     c.Endpoint,
			rcvBuf:  make([]byte, c.ReceiveBufferSize),
			sndBuf:  make([]byte, c.ReceiveBufferSize),
		}
		if err := uaConn.Hello(c); err != nil {
			return nil, err
		}
		return uaConn, nil
	default:
		return nil, errors.NewErrUnsupported(cc, "conn should be *net.TCPConn")
	}
}
