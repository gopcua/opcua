// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uacp

import (
	"net"
	"strings"

	"github.com/wmnsk/gopcua/errors"
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

// Dial acts like net.Dial for OPC UA network.
//
// Currently the endpoint can only be specified in "opc.tcp://<addr[:port]>" format.
// If port is missing, ":4840" is automatically chosen.
// If laddr is nil, a local address is automatically chosen.
func (c *Client) Dial(laddr *net.TCPAddr) (*Conn, error) {
	network, raddr, err := resolveEndpoint(c.Endpoint)
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

func resolveEndpoint(ep string) (network string, raddr *net.TCPAddr, err error) {
	elems := strings.Split(ep, "/")
	if elems[0] != "opc.tcp:" {
		return "", nil, errors.NewErrUnsupported(elems[0], "should be in \"opc.tcp://<addr:port>\" format.")
	}

	addr := elems[2]
	if !strings.Contains(addr, ":") {
		addr += ":4840"
	}

	network = "tcp"
	raddr, err = net.ResolveTCPAddr("tcp", addr)
	return
}
