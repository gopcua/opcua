// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uacp

import (
	"fmt"
	"net"
	"time"

	"github.com/wmnsk/gopcua/errors"
)

// Conn is an implementation of the net.Conn interface for OPC UA Connection Protocol.
type Conn struct {
	tcpConn  *net.TCPConn
	endpoint string
	rcvBuf   []byte
	sndBuf   []byte
}

// Read reads data from the connection.
// Read can be made to time out and return an Error with Timeout() == true
// after a fixed time limit; see SetDeadline and SetReadDeadline.
func (c *Conn) Read(b []byte) (n int, err error) {
	return c.tcpConn.Read(b)
}

// Write writes data to the connection.
// Write can be made to time out and return an Error with Timeout() == true
// after a fixed time limit; see SetDeadline and SetWriteDeadline.
func (c *Conn) Write(b []byte) (n int, err error) {
	return c.tcpConn.Write(b)
}

// Close closes the connection.
// Any blocked Read or Write operations will be unblocked and return errors.
func (c *Conn) Close() error {
	if err := c.tcpConn.Close(); err != nil {
		return err
	}

	c = nil
	return nil
}

// LocalAddr returns the local network address.
func (c *Conn) LocalAddr() net.Addr {
	return c.tcpConn.LocalAddr()
}

// RemoteAddr returns the remote network address.
func (c *Conn) RemoteAddr() net.Addr {
	return c.tcpConn.RemoteAddr()
}

// SetDeadline sets the read and write deadlines associated
// with the connection. It is equivalent to calling both
// SetReadDeadline and SetWriteDeadline.
//
// A deadline is an absolute time after which I/O operations
// fail with a timeout (see type Error) instead of
// blocking. The deadline applies to all future and pending
// I/O, not just the immediately following call to Read or
// Write. After a deadline has been exceeded, the connection
// can be refreshed by setting a deadline in the future.
//
// An idle timeout can be implemented by repeatedly extending
// the deadline after successful Read or Write calls.
//
// A zero value for t means I/O operations will not time out.
func (c *Conn) SetDeadline(t time.Time) error {
	return c.tcpConn.SetDeadline(t)
}

// SetReadDeadline sets the deadline for future Read calls
// and any currently-blocked Read call.
// A zero value for t means Read will not time out.
func (c *Conn) SetReadDeadline(t time.Time) error {
	return c.tcpConn.SetReadDeadline(t)
}

// SetWriteDeadline sets the deadline for future Write calls
// and any currently-blocked Write call.
// Even if write times out, it may return n > 0, indicating that
// some of the data was successfully written.
// A zero value for t means Write will not time out.
func (c *Conn) SetWriteDeadline(t time.Time) error {
	return c.tcpConn.SetWriteDeadline(t)
}

// Hello sends UACP Hello message and checks the reponse.
//
// Note: This is exported for those who want to debug, but might be made private in the future.
func (c *Conn) Hello(cli *Client) error {
	hel, err := NewHello(0, cli.ReceiveBufferSize, cli.SendBufferSize, 0, cli.Endpoint).Serialize()
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
		cli.SendBufferSize = msg.ReceiveBufSize
		return nil
	case *Error:
		return fmt.Errorf("received Error. Reason: %s", msg.Reason.Get())
	default:
		return errors.NewErrInvalidType(msg, "initiating UACP", ".")
	}
}
