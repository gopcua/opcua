// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package gopcua

import (
	"net"
)

// Dial creates a OPC UA Secure Channel connection.
func Dial(network string, laddr, raddr *net.TCPAddr) (*Conn, error) {
	var err error
	c := &Conn{}

	c.tcpConn, err = net.DialTCP("tcp", laddr, raddr)
	if err != nil {
		return nil, err
	}

	if err := Connect(c); err != nil {
		return nil, err
	}

	return c, nil
}

// Connect tries to establish OPC UA connection.
func Connect(c *Conn) error {
	return nil
}
