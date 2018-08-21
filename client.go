// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package gopcua

import (
	"net"
	"time"

	"github.com/wmnsk/gopcua/errors"
	"github.com/wmnsk/gopcua/services"
	"github.com/wmnsk/gopcua/uasc"
)

type Client struct {
	*Conn
	Config *uasc.Config
}

// Dial creates a OPC UA Secure Channel connection.
func Dial(network string, laddr, raddr *net.TCPAddr, cfg *uasc.Config) (*Conn, error) {
	var err error
	c := &Conn{}

	c.tcpConn, err = net.DialTCP("tcp", laddr, raddr)
	if err != nil {
		return nil, err
	}

	if err := Connect(c, cfg); err != nil {
		return nil, err
	}

	return c, nil
}

// Connect tries to establish OPC UA connection.
func Connect(c *Conn, cfg *uasc.Config) error {
	cli := &Client{
		Conn:   c,
		Config: cfg,
	}

	if err := cli.OpenSecureChannel(); err != nil {
		return err
	}

	endpoints, err := cli.GetEndpoints()
	if err != nil {
		return err
	}

	for _, endpoint := range endpoints {
		if endpoint.MessageSecurityMode == cli.Config.SecurityMode {
			cli.Config.ServerCertificate = endpoint.ServerCertificate.Get()
			cli.Config.ServerURI = endpoint.Server.ApplicationURI.Get()

			return nil
		}

		return &errors.ErrUnsupported{endpoint, "no matching security mode found."}
	}

	return nil
}

// OpenSecureChannel sends OpenSecureChannelRequest and retrieve values from OpenSecureChannelResonse.
//
// If the response is not OpenSecureChannel, it returns error.
func (c *Client) OpenSecureChannel() error {
	opn, err := uasc.New(
		services.NewOpenSecureChannelRequest(
			time.Now(),
			0,
			1,
			0,
			6000000,
			"",
			0,
			0,
			services.SecModeNone,
			6000000,
			nil,
		),
		&uasc.Config{
			SecureChannelID: 1,
		},
	).Serialize()

	if _, err := c.Conn.Write(opn); err != nil {
		return err
	}

	buf := make([]byte, 10000)
	n, err := c.Conn.Read(buf)
	if err != nil {
		return err
	}

	res, err := uasc.Decode(buf[:n])
	if err != nil {
		return err
	}

	switch srv := res.Service.(type) {
	case *services.OpenSecureChannelResponse:
		c.Config.SecureChannelID = srv.SecurityToken.ChannelID
		c.Config.SecurityTokenID = srv.SecurityToken.TokenID
	default:
		return &errors.ErrInvalidType{
			Type:   srv,
			Action: "open secure channel",
			Msg:    "should be *services.OpenSecureChannelResponse",
		}
	}

	return nil
}

// CloseSecureChannel sends CloseSecureChannelRequest and retrieve values from CloseSecureChannelResonse.
//
// If the response is not CloseSecureChannel, it returns error and closes the underlying TCP Connection.
// XXX - not implemented yet.
func (c *Client) CloseSecureChannel() error {
	return nil
}

// GetEndpoints sends GetEndpointsRequest and retrieve values from GetEndpointsResonse.
//
// If the response is not GetEndpoints, it returns error without any other actions.
// XXX - not implemented yet.
func (c *Client) GetEndpoints() ([]*services.EndpointDescription, error) {
	return nil, nil
}
