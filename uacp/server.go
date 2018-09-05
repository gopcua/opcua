// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uacp

import (
	"fmt"
	"net"

	"github.com/wmnsk/gopcua/utils"
)

// Server is the configuration that OPC UA Connection Protocol server should have.
type Server struct {
	Endpoint          string
	ReceiveBufferSize uint32
	SendBufferSize    uint32
}

// NewServer creates a new Server with minimum mandatory parameters.
func NewServer(endpoint string, rcvBufSize uint32) *Server {
	return &Server{
		Endpoint:          endpoint,
		ReceiveBufferSize: rcvBufSize,
		SendBufferSize:    0xffff,
	}
}

// Listener is a OPC UA Connection Protocol network listener.
type Listener struct {
	tcpListener *net.TCPListener
	srv         *Server
}

// Listen acts like net.Listen for OPC UA Connection Protocol networks.
//
// Currently the endpoint can only be specified in "opc.tcp://<addr[:port]>/path" format.
//
// If the IP field of laddr is nil or an unspecified IP address, ListenTCP listens on all available unicast and anycast IP addresses of the local system.
// If the Port field of laddr is 0, a port number is automatically chosen.
func (s *Server) Listen() (*Listener, error) {
	network, laddr, err := utils.ResolveEndpoint(s.Endpoint)
	if err != nil {
		return nil, err
	}

	lis := &Listener{srv: s}
	lis.tcpListener, err = net.ListenTCP(network, laddr)
	if err != nil {
		return nil, err
	}

	return lis, nil
}

// Accept accepts the next incoming call and returns the new connection.
func (l *Listener) Accept() (*Conn, error) {
	var err error

	conn := &Conn{}
	conn.tcpConn, err = l.tcpListener.AcceptTCP()
	if err != nil {
		return nil, err
	}
	conn.rcvBuf = make([]byte, l.srv.ReceiveBufferSize)

	n, err := conn.tcpConn.Read(conn.rcvBuf)
	if err != nil {
		return nil, err
	}

	message, err := Decode(conn.rcvBuf[:n])
	if err != nil {
		return nil, err
	}

	switch msg := message.(type) {
	case *Hello:
		spath, _ := utils.GetPath(l.srv.Endpoint)
		cpath, err := utils.GetPath(msg.EndPointURL.Get())
		if err != nil || cpath != spath {
			if err := conn.Error(BadTCPEndpointURLInvalid, fmt.Sprintf("Endpoint: %s does not exist", msg.EndPointURL.Get())); err != nil {
				return nil, err
			}
			return nil, fmt.Errorf("cannot accept due to invalid EndpointURL: %s", msg.EndPointURL.Get())
		}
		l.srv.SendBufferSize = msg.ReceiveBufSize
		if err := conn.Acknowledge(l.srv); err != nil {
			return nil, err
		}
	default:
		if err := conn.Error(BadTCPMessageTypeInvalid, "Expected Hello"); err != nil {
			return nil, err
		}
	}

	return conn, nil
}
