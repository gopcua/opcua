// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uacp

import (
	"net"

	"github.com/wmnsk/gopcua/errors"
	"github.com/wmnsk/gopcua/utils"
)

// Client is the configuration that OPC UA Connection Protocol client should have.
type Client struct {
	Endpoint          string
	ReceiveBufferSize uint32
	SendBufferSize    uint32
}

// NewClient creates a new Client with minimum mandatory parameters.
func NewClient(endpoint string, rcvBufSize uint32) *Client {
	return &Client{
		Endpoint:          endpoint,
		ReceiveBufferSize: rcvBufSize,
		SendBufferSize:    0xffff,
	}
}

// Dial acts like net.Dial for OPC UA Connection Protocol network.
//
// Currently the endpoint can only be specified in "opc.tcp://<addr[:port]>" format.
// If port is missing, ":4840" is automatically chosen.
// If laddr is nil, a local address is automatically chosen.
func (c *Client) Dial(laddr *net.TCPAddr) (*Conn, error) {
	network, raddr, err := utils.ResolveEndpoint(c.Endpoint)
	if err != nil {
		return nil, err
	}

	conn := &Conn{}
	conn.tcpConn, err = net.DialTCP(network, laddr, raddr)
	if err != nil {
		return nil, err
	}
	conn.rcvBuf = make([]byte, c.ReceiveBufferSize)

	if err := conn.Hello(c); err != nil {
		return nil, err
	}

	return conn, nil
}

// OpenConnection creates *uacp.Conn on top of the existing connection(net.Conn).
//
// Note: Currently conn only supports *net.TCPConn.
func (c *Client) OpenConnection(conn net.Conn) (*Conn, error) {
	switch cc := conn.(type) {
	case *net.TCPConn:
		uaConn := &Conn{
			tcpConn:  cc,
			endpoint: c.Endpoint,
			rcvBuf:   make([]byte, c.ReceiveBufferSize),
			sndBuf:   make([]byte, c.ReceiveBufferSize),
		}
		if err := uaConn.Hello(c); err != nil {
			return nil, err
		}
		return uaConn, nil
	default:
		return nil, errors.NewErrUnsupported(cc, "conn should be *net.TCPConn")
	}
}
