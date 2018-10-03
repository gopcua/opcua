// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uasc

import (
	"context"
	"io"
	"net"
	"time"

	"github.com/wmnsk/gopcua/errors"
	"github.com/wmnsk/gopcua/services"
	"github.com/wmnsk/gopcua/status"
	"github.com/wmnsk/gopcua/uacp"
)

// Config represents a configuration which UASC client/server has in common.
type Config struct {
	SecureChannelID   uint32
	SecurityPolicyURI string
	Certificate       []byte
	Thumbprint        []byte
	RequestID         uint32
	SecurityTokenID   uint32
	SequenceNumber    uint32
}

// NewConfig creates a new Config.
func NewConfig(chanID uint32, policyURI string, cert, thumbprint []byte, reqID, tokenID uint32) *Config {
	return &Config{
		SecureChannelID:   chanID,
		SecurityPolicyURI: policyURI,
		Certificate:       cert,
		Thumbprint:        thumbprint,
		RequestID:         reqID,
		SecurityTokenID:   tokenID,
		SequenceNumber:    0,
	}
}

// SecureChannel is an implementation of the net.Conn interface for Secure Channel in OPC UA Secure Conversation.
//
// In UASC, there are two types of net.Conn: SecureChannel and Session. Each Conn is handled in different manner.
type SecureChannel struct {
	lowerConn      net.Conn
	cfg            *Config
	reqHeader      *services.RequestHeader
	resHeader      *services.ResponseHeader
	rcvBuf, sndBuf []byte
	state          secChanState
	stateChan      chan secChanState
	lenChan        chan int
	errChan        chan error
}

// Read reads data from the connection.
// Read can be made to time out and return an Error with Timeout() == true
// after a fixed time limit; see SetDeadline and SetReadDeadline.
//
// If the data is one of OpenSecureChannel or CloseSecureChannel, it will be handled automatically.
func (s *SecureChannel) Read(b []byte) (n int, err error) {
	if !(s.state == cliStateSecureChannelOpened || s.state == srvStateSecureChannelOpened) {
		return 0, ErrSecureChannelNotOpened
	}
	for {
		select {
		case n := <-s.lenChan:
			copy(b, s.rcvBuf[:n])
			return n, nil
		case e := <-s.errChan:
			return 0, e
		default:
			continue
		}
	}
}

// ReadService reads the payload(=Service) from the connection.
// Which means the UASC Headers are omitted.
func (s *SecureChannel) ReadService(b []byte) (n int, err error) {
	if !(s.state == cliStateSecureChannelOpened || s.state == srvStateSecureChannelOpened) {
		return 0, ErrSecureChannelNotOpened
	}
	for {
		select {
		case n := <-s.lenChan:
			sc, err := Decode(s.rcvBuf[:n])
			if err != nil {
				return 0, err
			}
			copy(b, sc.SequenceHeader.Payload)
			return int(sc.MessageSize), nil
		case e := <-s.errChan:
			return 0, e
		default:
			continue
		}
	}
}

// Write writes data to the connection.
// Write can be made to time out and return an Error with Timeout() == true
// after a fixed time limit; see SetDeadline and SetWriteDeadline.
func (s *SecureChannel) Write(b []byte) (n int, err error) {
	if s == nil || !(s.state == cliStateSecureChannelOpened || s.state == srvStateSecureChannelOpened) {
		return 0, ErrSecureChannelNotOpened
	}
	select {
	case e := <-s.errChan:
		return 0, e
	default:
		return s.lowerConn.Write(b)
	}
}

// WriteService writes data to the connection.
// Unlike Write(), given b in WriteService() should only be serialized service.Service,
// while the UASC header is automatically set by the package.
// This enables writing arbitrary Service even if the service is not implemented in the package.
func (s *SecureChannel) WriteService(b []byte) (n int, err error) {
	if !(s.state == cliStateSecureChannelOpened || s.state == srvStateSecureChannelOpened) {
		return 0, ErrSecureChannelNotOpened
	}
	select {
	case e := <-s.errChan:
		return 0, e
	default:
		s.cfg.SequenceNumber++

		msg := New(nil, s.cfg)
		msg.MessageSize += uint32(len(b))
		serialized, err := msg.Serialize()
		if err != nil {
			return 0, err
		}
		serialized = append(serialized, b...)

		if _, err := s.lowerConn.Write(serialized); err != nil {
			return 0, err
		}

		return int(msg.MessageSize), nil
	}
}

// Close closes the connection.
// Any blocked Read or Write operations will be unblocked and return errors.
//
// Before closing, client sends CloseSecureChannelRequest, but it does not care the result of that request.
func (s *SecureChannel) Close() error {
	switch s.state {
	case cliStateSecureChannelClosed, cliStateOpenSecureChannelSent, cliStateSecureChannelOpened:
		s.CloseSecureChannelRequest()
		s.updateState(cliStateSecureChannelClosed)
	case srvStateCloseSecureChannelSent, srvStateSecureChannelOpened:
		s.updateState(srvStateSecureChannelClosed)
	}

	s.cfg.SequenceNumber = 0
	s.reqHeader.RequestHandle = 0
	s.resHeader.RequestHandle = 0
	close(s.errChan)
	close(s.lenChan)
	close(s.stateChan)

	return s.lowerConn.Close()
}

// LocalAddr returns the local network address.
func (s *SecureChannel) LocalAddr() net.Addr {
	return s.lowerConn.LocalAddr()
}

// RemoteAddr returns the remote network address.
func (s *SecureChannel) RemoteAddr() net.Addr {
	return s.lowerConn.RemoteAddr()
}

// LocalEndpoint returns the local EndpointURL.
//
// This is expected to be called from server side of UACP Connection.
// If transport connection is not *uacp.Conn, LocalEndpoint() returns "".
func (s *SecureChannel) LocalEndpoint() string {
	conn, ok := s.lowerConn.(*uacp.Conn)
	if !ok {
		return ""
	}
	return conn.LocalEndpoint()
}

// RemoteEndpoint returns the remote EndpointURL.
//
// This is expected to be called from client side of SecureChannel.
// If transport connection is not *uacp.Conn, RemoteEndpoint() returns "".
func (s *SecureChannel) RemoteEndpoint() string {
	conn, ok := s.lowerConn.(*uacp.Conn)
	if !ok {
		return ""
	}
	return conn.RemoteEndpoint()
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
func (s *SecureChannel) SetDeadline(t time.Time) error {
	return s.lowerConn.SetDeadline(t)
}

// SetReadDeadline sets the deadline for future Read calls
// and any currently-blocked Read call.
// A zero value for t means Read will not time out.
func (s *SecureChannel) SetReadDeadline(t time.Time) error {
	return s.lowerConn.SetReadDeadline(t)
}

// SetWriteDeadline sets the deadline for future Write calls
// and any currently-blocked Write call.
// Even if write times out, it may return n > 0, indicating that
// some of the data was successfully written.
// A zero value for t means Write will not time out.
func (s *SecureChannel) SetWriteDeadline(t time.Time) error {
	return s.lowerConn.SetWriteDeadline(t)
}

type secChanState uint8

const (
	undefined secChanState = iota
	transportUnavailable
	cliStateSecureChannelClosed
	cliStateOpenSecureChannelSent
	cliStateSecureChannelOpened
	cliStateCloseSecureChannelSent
	srvStateSecureChannelClosed
	srvStateSecureChannelOpened
	srvStateCloseSecureChannelSent
)

func (s *SecureChannel) updateState(c secChanState) {
	s.state = c
	s.stateChan <- s.state
}

func (s secChanState) String() string {
	switch s {
	case transportUnavailable:
		return "transport connection unavailable"
	case cliStateSecureChannelClosed:
		return "client secure channel closed"
	case cliStateOpenSecureChannelSent:
		return "client open secure channel sent"
	case cliStateSecureChannelOpened:
		return "client secure channel opened"
	case cliStateCloseSecureChannelSent:
		return "client close secure channel sent"
	case srvStateSecureChannelClosed:
		return "server secure channel closed"
	case srvStateSecureChannelOpened:
		return "server secure channel opened"
	case srvStateCloseSecureChannelSent:
		return "server close secure channel sent"
	default:
		return "unknown"
	}
}

// GetState returns the current secChanState of SecureChannel.
func (s *SecureChannel) GetState() string {
	if s == nil {
		return ""
	}
	return s.state.String()
}

func (s *SecureChannel) notifyLength(n int) {
	go func() {
		s.lenChan <- n
	}()
}

func (s *SecureChannel) monitorMessages(ctx context.Context) {
	defer s.Close()
	s.updateState(s.state)

	for {
		select {
		case <-ctx.Done():
			return
		default:
			n, err := s.lowerConn.Read(s.rcvBuf)
			if err != nil {
				if err == io.EOF {
					continue
				}
				s.Close()
			}
			if n == 0 {
				continue
			}

			msg, err := Decode(s.rcvBuf[:n])
			if err != nil {
				// pass to the user if msg is undecodable as UASC.
				if s.state == cliStateSecureChannelOpened || s.state == srvStateSecureChannelOpened {
					s.notifyLength(n)
				}
				continue
			}
			switch m := msg.Service.(type) {
			case *services.OpenSecureChannelRequest:
				s.handleOpenSecureChannelRequest(m)
			case *services.OpenSecureChannelResponse:
				s.handleOpenSecureChannelResponse(m)
			case *services.CloseSecureChannelRequest:
				s.handleCloseSecureChannelRequest(m)
			case *services.CloseSecureChannelResponse:
				s.handleCloseSecureChannelResponse(m)
			default:
				// pass to the user if type of msg is unknown.
				if s.state == cliStateSecureChannelOpened || s.state == srvStateSecureChannelOpened {
					s.notifyLength(n)
				}
			}
		}
	}
}

func (s *SecureChannel) handleOpenSecureChannelRequest(o *services.OpenSecureChannelRequest) {
	switch s.state {
	// if state is closed, server accepts OpenSecureChannelRequest.
	case srvStateSecureChannelClosed:
		switch o.MessageSecurityMode {
		// accepts only if MessageSecurityMode is None.
		case services.SecModeNone:
			s.resHeader.RequestHandle = o.RequestHandle
			if err := s.OpenSecureChannelResponse(0, 0, 0xffff, nil); err != nil {
				s.errChan <- err
			}
			s.updateState(srvStateSecureChannelOpened)
		// respond with BadSecurityModeRejected and notify server
		default:
			if err := s.OpenSecureChannelResponse(status.BadSecurityModeRejected, 0, 0xffff, nil); err != nil {
				s.errChan <- err
			}
			s.errChan <- ErrSecurityModeUnsupported
		}
	// if SecureChannel is already opened, respond with BadAlreadyExists.
	case srvStateSecureChannelOpened, srvStateCloseSecureChannelSent:
		if err := s.OpenSecureChannelResponse(status.BadAlreadyExists, 0, 0xffff, nil); err != nil {
			s.errChan <- err
		}
	// client never accept OpenSecureChannelRequest, just ignore it.
	case cliStateSecureChannelClosed, cliStateOpenSecureChannelSent, cliStateSecureChannelOpened, cliStateCloseSecureChannelSent:
	// invalid secChanState. conn should be closed in error handler.
	default:
		s.errChan <- ErrInvalidState
	}
}

func (s *SecureChannel) handleOpenSecureChannelResponse(o *services.OpenSecureChannelResponse) {
	switch s.state {
	// client accepts OpenSecureChannelResponse only after sending OpenSecureChannelRequest.
	case cliStateOpenSecureChannelSent:
		switch o.ServiceResult {
		case 0: // Good
			s.cfg.SecureChannelID = o.SecurityToken.ChannelID
			s.cfg.SecurityTokenID = o.SecurityToken.TokenID
			s.updateState(cliStateSecureChannelOpened)
		case status.BadSecurityModeRejected:
			s.errChan <- ErrRejected
			s.updateState(cliStateSecureChannelClosed)
		}
	// if client SecureChannel is closed or opened, just ignore OpenSecureChannelResponse.
	case cliStateSecureChannelClosed, cliStateSecureChannelOpened, cliStateCloseSecureChannelSent:
	// server never accept OpenSecureChannelResponse , just ignore it.
	case srvStateSecureChannelClosed, srvStateSecureChannelOpened, srvStateCloseSecureChannelSent:
	// invalid secChanState. SecureChannel should be closed in error handler.
	default:
		s.errChan <- ErrInvalidState
	}
}

func (s *SecureChannel) handleCloseSecureChannelRequest(c *services.CloseSecureChannelRequest) {
	switch s.state {
	// if client SecureChannel is opened, accept CloseSecureChannelRequest.
	case cliStateSecureChannelOpened:
		s.reqHeader.RequestHandle = c.RequestHandle
		if err := s.CloseSecureChannelResponse(0); err != nil {
			s.errChan <- err
		}
		s.updateState(cliStateCloseSecureChannelSent)
	// if server SecureChannel is opened, accept CloseSecureChannelRequest.
	case srvStateSecureChannelOpened:
		s.reqHeader.RequestHandle = c.RequestHandle
		if err := s.CloseSecureChannelResponse(0); err != nil {
			s.errChan <- err
		}
		s.updateState(srvStateCloseSecureChannelSent)
	// if client/server SecureChannel is not opened, ignore CloseSecureChannelRequest.
	case cliStateSecureChannelClosed, cliStateOpenSecureChannelSent, cliStateCloseSecureChannelSent, srvStateSecureChannelClosed, srvStateCloseSecureChannelSent:
	// invalid secChanState. conn should be closed in error handler.
	default:
		s.errChan <- ErrInvalidState
	}
}

func (s *SecureChannel) handleCloseSecureChannelResponse(c *services.CloseSecureChannelResponse) {
	switch s.state {
	// client accepts CloseSecureChannelResponse only after sending CloseSecureChannelResponse.
	case cliStateCloseSecureChannelSent:
		s.updateState(cliStateSecureChannelClosed)
	// server accepts CloseSecureChannelResponse only after sending CloseSecureChannelResponse.
	case srvStateCloseSecureChannelSent:
		s.updateState(srvStateSecureChannelClosed)
	// if client/server conn is opened, just ignore CloseSecureChannelResponse.
	case cliStateSecureChannelClosed, cliStateOpenSecureChannelSent, cliStateSecureChannelOpened, srvStateSecureChannelClosed, srvStateSecureChannelOpened:
	// invalid secChanState. conn should be closed in error handler.
	default:
		s.errChan <- ErrInvalidState
	}
}

// OpenSecureChannelRequest sends OpenSecureChannelRequest on top of UASC to SecureChannel.
func (s *SecureChannel) OpenSecureChannelRequest(secMode, lifetime uint32, nonce []byte) error {
	s.cfg.SequenceNumber++
	s.reqHeader.RequestHandle++
	s.reqHeader.Timestamp = time.Now()
	osc, err := New(
		services.NewOpenSecureChannelRequest(
			s.reqHeader, 0, services.ReqTypeIssue, secMode, lifetime, nonce,
		), s.cfg,
	).Serialize()
	if err != nil {
		return err
	}

	if _, err := s.lowerConn.Write(osc); err != nil {
		return err
	}
	return nil
}

// OpenSecureChannelResponse sends OpenSecureChannelResponse on top of UASC to SecureChannel.
func (s *SecureChannel) OpenSecureChannelResponse(code, token, lifetime uint32, nonce []byte) error {
	s.cfg.SequenceNumber++
	s.resHeader.ServiceResult = code
	s.resHeader.Timestamp = time.Now()
	osc, err := New(
		services.NewOpenSecureChannelResponse(
			s.resHeader, 0, services.NewChannelSecurityToken(
				s.cfg.SecureChannelID, token, time.Now(), lifetime,
			), nonce,
		), s.cfg,
	).Serialize()
	if err != nil {
		return err
	}

	if _, err := s.lowerConn.Write(osc); err != nil {
		return err
	}
	return nil
}

// CloseSecureChannelRequest sends CloseSecureChannelRequest on top of UASC to SecureChannel.
func (s *SecureChannel) CloseSecureChannelRequest() error {
	s.cfg.SequenceNumber++
	s.reqHeader.RequestHandle++
	s.reqHeader.Timestamp = time.Now()
	csc, err := New(
		services.NewCloseSecureChannelRequest(
			s.reqHeader, s.cfg.SecureChannelID,
		), s.cfg,
	).Serialize()
	if err != nil {
		return err
	}

	if _, err := s.lowerConn.Write(csc); err != nil {
		return err
	}
	return nil
}

// CloseSecureChannelResponse sends CloseSecureChannelResponse on top of UASC to SecureChannel.
func (s *SecureChannel) CloseSecureChannelResponse(code uint32) error {
	s.cfg.SequenceNumber++
	s.resHeader.ServiceResult = code
	s.resHeader.Timestamp = time.Now()
	csc, err := New(
		services.NewCloseSecureChannelResponse(s.resHeader), s.cfg,
	).Serialize()
	if err != nil {
		return err
	}

	if _, err := s.lowerConn.Write(csc); err != nil {
		return err
	}
	return nil
}

// GetEndpointsRequest sends GetEndpointsRequest on top of UASC to SecureChannel.
func (s *SecureChannel) GetEndpointsRequest(locales, uris []string) error {
	s.reqHeader.RequestHandle++
	s.reqHeader.Timestamp = time.Now()
	gep, err := services.NewGetEndpointsRequest(
		s.reqHeader, s.RemoteEndpoint(), locales, uris,
	).Serialize()
	if err != nil {
		return err
	}

	if _, err := s.WriteService(gep); err != nil {
		return err
	}
	return nil
}

// GetEndpointsResponse sends GetEndpointsResponse on top of UASC to SecureChannel.
//
// XXX - This is to be improved with some external configuration to describe endpoints infomation in the future release.
func (s *SecureChannel) GetEndpointsResponse(code uint32, endpoints ...*services.EndpointDescription) error {
	s.resHeader.ServiceResult = code
	s.resHeader.Timestamp = time.Now()
	gep, err := services.NewGetEndpointsResponse(
		s.resHeader, endpoints...,
	).Serialize()
	if err != nil {
		return err
	}

	if _, err := s.WriteService(gep); err != nil {
		return err
	}
	return nil
}

// FindServersRequest sends FindServersRequest on top of UASC to SecureChannel.
func (s *SecureChannel) FindServersRequest(locales, servers []string) error {
	s.reqHeader.RequestHandle++
	s.reqHeader.Timestamp = time.Now()
	fsr, err := services.NewFindServersRequest(
		s.reqHeader, s.RemoteEndpoint(), locales, servers,
	).Serialize()
	if err != nil {
		return err
	}

	if _, err := s.WriteService(fsr); err != nil {
		return err
	}
	return nil
}

// FindServersResponse sends FindServersResponse on top of UASC to SecureChannel.
//
// XXX - This is to be improved with some external configuration to describe application infomation in the future release.
func (s *SecureChannel) FindServersResponse(code uint32, apps ...*services.ApplicationDescription) error {
	s.resHeader.ServiceResult = code
	s.resHeader.Timestamp = time.Now()
	fsr, err := services.NewFindServersResponse(
		s.resHeader, apps...,
	).Serialize()
	if err != nil {
		return err
	}

	if _, err := s.WriteService(fsr); err != nil {
		return err
	}
	return nil
}

// UASC-specific error definitions.
// XXX - to be integrated in errors package.
var (
	ErrInvalidState            = errors.New("invalid state")
	ErrUnexpectedMessage       = errors.New("got unexpected message")
	ErrTimeout                 = errors.New("timed out")
	ErrSecureChannelNotOpened  = errors.New("connection not established")
	ErrSecurityModeUnsupported = errors.New("got request with unsupported SecurityMode")
	ErrRejected                = errors.New("rejected by server")
)
