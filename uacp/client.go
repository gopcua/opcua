// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uacp

import (
	"fmt"
	"net"
	"strings"

	"github.com/wmnsk/gopcua/errors"
)

// ClientConfig is the configuration that OPC UA Connection Protocol client should have.
type ClientConfig struct {
	Endpoint          string
	SendBufferSize    uint32
	ReceiveBufferSize uint32
}

// NewClientConfig creates a new ClientConfig with minimum mandatory parameters.
func NewClientConfig(endpoint string, rcvBufSize uint32) *ClientConfig {
	return &ClientConfig{
		Endpoint:          endpoint,
		ReceiveBufferSize: rcvBufSize,
		SendBufferSize:    0,
	}
}

// Dial acts like Dial for OPC UA network.
//
// Currently the endpoint can only be specified in "opc.tcp://<addr[:port]>" format.
// If port is missing, ":4840" is automatically chosen.
// If laddr is nil, a local address is automatically chosen.
func Dial(cfg *ClientConfig, laddr *net.TCPAddr) (*Conn, error) {
	network, raddr, err := resolveEndpoint(cfg.Endpoint)
	if err != nil {
		return nil, err
	}

	conn := &Conn{}
	conn.tcpConn, err = net.DialTCP(network, laddr, raddr)
	if err != nil {
		return nil, err
	}
	conn.rcvBuf = make([]byte, int(cfg.ReceiveBufferSize))
	if err := conn.Hello(cfg); err != nil {
		return nil, err
	}

	return conn, nil
}

// Hello sends UACP Hello message and checks the reponse.
//
// Note: This is exported for those who wants to debug, but might be made private in the future.
func (c *Conn) Hello(cfg *ClientConfig) error {
	hel, err := NewHello(0, cfg.SendBufferSize, cfg.ReceiveBufferSize, 0, cfg.Endpoint).Serialize()
	if err != nil {
		return err
	}

	if _, err := c.tcpConn.Write(hel); err != nil {
		return err
	}

	n, err := c.tcpConn.Read(c.rcvBuf)
	if err != nil {
		return err
	}

	message, err := Decode(c.rcvBuf[:n])
	if err != nil {
		return err
	}

	switch msg := message.(type) {
	case *Acknowledge:
		cfg.SendBufferSize = msg.ReceiveBufSize
		return nil
	case *Error:
		return fmt.Errorf("received Error. Reason: %s", msg.Reason.Get())
	default:
		return errors.NewErrInvalidType(msg, "initiating UACP", ".")
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
