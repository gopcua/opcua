// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uasc

import (
	"context"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/wmnsk/gopcua/services"

	"github.com/wmnsk/gopcua/errors"
)

// Session is an implementation of the net.Conn interface for Session in OPC UA Secure Conversation.
//
// In UASC, there are two types of net.Conn: SecureChannel and Session. Each Conn is handled in different manner.
type Session struct {
	// Using net.Conn interface but basically Session should only be on the SecureChannel.
	lowerConn      net.Conn
	cfg            *Config
	lep, rep       string
	rcvBuf, sndBuf []byte
	state          sessionState
	stateChan      chan sessionState
	lenChan        chan int
	errChan        chan error
	appType        uint32
	isActivated    bool
	sessionID      uint32
}

// Read reads data from the connection.
// Read can be made to time out and return an Error with Timeout() == true
// after a fixed time limit; see SetDeadline and SetReadDeadline.
//
// If the data is one of Open/CloseSession, Create/Activate/CloseSession, it will be handled automatically.
func (s *Session) Read(b []byte) (n int, err error) {
	if !(s.state == cliStateSessionCreated || s.state == srvStateSessionCreated) {
		return 0, ErrSessionNotOpened
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

// Write writes data to the connection.
// Write can be made to time out and return an Error with Timeout() == true
// after a fixed time limit; see SetDeadline and SetWriteDeadline.
func (s *Session) Write(b []byte) (n int, err error) {
	if !(s.state == cliStateSessionCreated || s.state == srvStateSessionCreated) {
		return 0, ErrSessionNotOpened
	}
	select {
	case e := <-s.errChan:
		return 0, e
	default:
		return s.lowerConn.Write(b)
	}
}

// Activate activates a Session.
func (s *Session) Activate() error {
	if err := s.ActivateSessionRequest(); err != nil {
		return err
	}
	s.updateState(cliStateActivateSessionSent)

	for {
		select {
		case state := <-s.stateChan:
			switch state {
			case cliStateSessionActivated:
				return nil
			default:
				continue
			}
		case <-time.After(3 * time.Second):
			return errors.New("timed out while activating session")
		}
	}
}

// Close closes the connection.
// Any blocked Read or Write operations will be unblocked and return errors.
func (s *Session) Close() error {
	if err := s.CloseSessionRequest(false); err != nil {
		return err
	}
	var nextState sessionState
	switch s.appType {
	case services.AppTypeClient:
		nextState = cliStateCloseSessionSent
	case services.AppTypeServer:
		nextState = srvStateCloseSessionSent
	}
	s.updateState(nextState)

	for {
		select {
		case state := <-s.stateChan:
			switch state {
			case cliStateSessionClosed, srvStateSessionClosed:
				if err := s.lowerConn.Close(); err != nil {
					return err
				}
				close(s.errChan)
				close(s.lenChan)
				close(s.stateChan)
				return nil
			default:
				continue
			}
		case <-time.After(5 * time.Second):
			return errors.New("timed out while closing session")
		}
	}
}

// LocalAddr returns the local network address.
func (s *Session) LocalAddr() net.Addr {
	return s.lowerConn.LocalAddr()
}

// RemoteAddr returns the remote network address.
func (s *Session) RemoteAddr() net.Addr {
	return s.lowerConn.RemoteAddr()
}

// LocalEndpoint returns the local EndpointURL.
// This is expected to be called from server side of Session.
func (s *Session) LocalEndpoint() string {
	return s.lep
}

// RemoteEndpoint returns the remote EndpointURL.
// This is expected to be called from client side of Session.
func (s *Session) RemoteEndpoint() string {
	return s.rep
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
func (s *Session) SetDeadline(t time.Time) error {
	return s.lowerConn.SetDeadline(t)
}

// SetReadDeadline sets the deadline for future Read calls
// and any currently-blocked Read call.
// A zero value for t means Read will not time out.
func (s *Session) SetReadDeadline(t time.Time) error {
	return s.lowerConn.SetReadDeadline(t)
}

// SetWriteDeadline sets the deadline for future Write calls
// and any currently-blocked Write call.
// Even if write times out, it may return n > 0, indicating that
// some of the data was successfully written.
// A zero value for t means Write will not time out.
func (s *Session) SetWriteDeadline(t time.Time) error {
	return s.lowerConn.SetWriteDeadline(t)
}

type sessionState uint8

const (
	undefinedSession sessionState = iota
	secureChannelUnavailable
	cliStateSessionClosed
	cliStateCreateSessionSent
	cliStateSessionCreated
	cliStateActivateSessionSent
	cliStateSessionActivated
	cliStateCloseSessionSent
	srvStateSessionClosed
	srvStateSessionCreated
	srvStateSessionActivated
	srvStateCloseSessionSent
)

func (s *Session) updateState(c sessionState) {
	s.state = c
	s.stateChan <- s.state
}

func (s sessionState) String() string {
	switch s {
	case secureChannelUnavailable:
		return "secure channel connection unavailable"
	case cliStateSessionClosed:
		return "client session closed"
	case cliStateCreateSessionSent:
		return "client create session sent"
	case cliStateSessionCreated:
		return "client session created"
	case cliStateActivateSessionSent:
		return "client activate session sent"
	case cliStateSessionActivated:
		return "client session activated"
	case cliStateCloseSessionSent:
		return "client close session sent"
	case srvStateSessionClosed:
		return "server session closed"
	case srvStateSessionCreated:
		return "server session created"
	case srvStateSessionActivated:
		return "server session activated"
	case srvStateCloseSessionSent:
		return "server close session sent"
	default:
		return "unknown"
	}
}

// GetState returns the current sessionState of Session.
func (s *Session) GetState() string {
	if s == nil {
		return ""
	}
	return s.state.String()
}

func (s *Session) notifyLength(n int) {
	go func() {
		s.lenChan <- n
	}()
}

func (s *Session) monitorMessages(ctx context.Context) {
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
				if s.state == cliStateSessionCreated || s.state == srvStateSessionCreated {
					s.notifyLength(n)
				}
				continue
			}
			switch m := msg.Service.(type) {
			case *services.CreateSessionRequest:
				s.handleSrvCreateSessionRequest(m)
			case *services.CreateSessionResponse:
				s.handleSrvCreateSessionResponse(m)
			case *services.ActivateSessionRequest:
				s.handleSrvActivateSessionRequest(m)
			case *services.ActivateSessionResponse:
				s.handleSrvActivateSessionResponse(m)
			case *services.CloseSessionRequest:
				s.handleSrvCloseSessionRequest(m)
			case *services.CloseSessionResponse:
				s.handleSrvCloseSessionResponse(m)
			default:
				// pass to the user if type of msg is unknown.
				if s.state == cliStateSessionCreated || s.state == srvStateSessionCreated {
					s.notifyLength(n)
				}
			}
		}
	}
}

func (s *Session) handleSrvCreateSessionRequest(o *services.CreateSessionRequest) {
	switch s.state {
	// if state is closed, server accepts services.CreateSessionRequest.
	case srvStateSessionClosed:
		// proceed here.
	// client never accept services.CreateSessionRequest, just ignore Requests.
	case cliStateSessionClosed, cliStateCreateSessionSent, cliStateSessionCreated:
	// invalid sessionState. conn should be closed in error handler.
	default:
		s.errChan <- ErrInvalidState
	}
}

func (s *Session) handleSrvCreateSessionResponse(o *services.CreateSessionResponse) {
	switch s.state {
	// client accepts Acknowledge only after sending services.CreateSessionRequest.
	case cliStateCreateSessionSent:
	// if client conn is closed or established, just ignore Acknowledge.
	case cliStateSessionClosed, cliStateSessionCreated:
	// server never accept Acknowledge.
	case srvStateSessionClosed, srvStateSessionCreated:
	// invalid sessionState. conn should be closed in error handler.
	default:
	}
}

func (s *Session) handleSrvActivateSessionRequest(o *services.ActivateSessionRequest) {
	switch s.state {
	// if state is closed, server accepts services.ActivateSessionRequest.
	case srvStateSessionClosed:
		// proceed here
	// client never accept services.ActivateSessionRequest, just ignore Requests.
	case cliStateSessionClosed, cliStateCreateSessionSent, cliStateSessionCreated:
	// invalid sessionState. conn should be closed in error handler.
	default:
		s.errChan <- ErrInvalidState
	}
}

func (s *Session) handleSrvActivateSessionResponse(o *services.ActivateSessionResponse) {
	switch s.state {
	// client accepts Acknowledge only after sending services.ActivateSessionRequest.
	case cliStateCreateSessionSent:
	// if client conn is closed or established, just ignore Acknowledge.
	case cliStateSessionClosed, cliStateSessionCreated:
	// server never accept Acknowledge.
	case srvStateSessionClosed, srvStateSessionCreated:
	// invalid sessionState. conn should be closed in error handler.
	default:
	}
}

func (s *Session) handleSrvCloseSessionResponse(c *services.CloseSessionResponse) {
	switch s.state {
	// if client receives Error after sending services.CreateSessionRequest, notify error handler and switch sessionState to closed.
	case cliStateCreateSessionSent:
	// if client/server conn is established, just notify error to error handler.
	case cliStateSessionCreated, srvStateSessionCreated:
	// if client/server conn is closed, just ignore Error.
	case cliStateSessionClosed, srvStateSessionClosed:
	// invalid sessionState. conn should be closed in error handler.
	default:
	}
}

func (s *Session) handleSrvCloseSessionRequest(c *services.CloseSessionRequest) {
	switch s.state {
	// if client conn is closed, accept services.CloseSessionRequest.
	// XXX - not likely to hit this condition.
	case cliStateSessionClosed:
	// if client conn is opening/opened, client just ignore services.CloseSessionRequest.
	case cliStateCreateSessionSent, cliStateSessionCreated:
	// server never accept services.CloseSessionRequest.
	case srvStateSessionClosed, srvStateSessionCreated:
	// invalid sessionState. conn should be closed in error handler.
	default:
	}
}

// CreateSessionRequest sends CreateSessionRequest on top of UASC to Conn.
func (s *Session) CreateSessionRequest(appURI, prodURI string, appType uint32, server, endpoint string, timeout uint64, maxRespSize uint32, cert, nonce []byte) error {
	osc, err := New(
		services.NewCreateSessionRequest(
			time.Now(), appURI, prodURI, "gopcua",
			appType, server, s.rep, fmt.Sprintf("%v", time.Now().UnixNano()),
			nonce, cert, timeout, maxRespSize,
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

// CreateSessionResponse sends CreateSessionResponse on top of UASC to Conn.
func (s *Session) CreateSessionResponse(code uint32, token uint16, timeout uint64, cert, nonce []byte, alg string, sign []byte, maxRespSize uint32, endpoints ...*services.EndpointDescription) error {
	osc, err := New(
		services.NewCreateSessionResponse(
			time.Now(), code, services.NewNullDiagnosticInfo(), s.sessionID, token,
			timeout, nonce, cert, alg, sign, maxRespSize, endpoints...,
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

// CloseSessionRequest sends CloseSessionRequest on top of UASC to Conn.
func (s *Session) CloseSessionRequest(delete bool) error {
	csc, err := New(
		services.NewCloseSessionRequest(
			time.Now(), 0, 1, 0x03, 0xffff, "", delete,
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

// CloseSessionResponse sends CloseSessionResponse on top of UASC to Conn.
func (s *Session) CloseSessionResponse(code uint32) error {
	csc, err := New(
		services.NewCloseSessionResponse(
			time.Now(), 1, code, nil, []string{""},
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

// ActivateSessionRequest sends ActivateSessionRequest on top of UASC to Conn.
func (s *Session) ActivateSessionRequest() error {
	return nil
}

// ActivateSessionResponse sends ActivateSessionResponse on top of UASC to Conn.
func (s *Session) ActivateSessionResponse() error {
	return nil
}

// UASC Session-specific error definitions.
var (
	ErrInvalidSessionState  = errors.New("invalid sessionState")
	ErrSessionTimeout       = errors.New("timed out")
	ErrSessionReceivedError = errors.New("received Error message")
	ErrSessionNotOpened     = errors.New("connection not established")
)
