// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uasc

import (
	"context"
	"crypto/rand"
	"io"
	"net"
	"sync"
	"time"

	"github.com/wmnsk/gopcua/datatypes"
	"github.com/wmnsk/gopcua/errors"
	"github.com/wmnsk/gopcua/services"
)

// SessionConfig is a set of common configurations used in Session.
type SessionConfig struct {
	SessionTimeout                   uint64
	AuthenticationToken              datatypes.NodeID
	ClientDescription                *services.ApplicationDescription
	ServerEndpoints                  []*services.EndpointDescription
	ClientSignature, ServerSignature *services.SignatureData
	ClientSoftwareCertificates       []*services.SignedSoftwareCertificate
	LocaleIDs                        []string
	UserIdentityToken                *datatypes.ExtensionObject
	UserTokenSignature               *services.SignatureData
}

// Session is an implementation of the net.Conn interface for Session in OPC UA Secure Conversation.
//
// In UASC, there are two types of net.Conn: SecureChannel and Session. Each Conn is handled in different manner.
type Session struct {
	mu             *sync.Mutex
	secChan        *SecureChannel
	cfg            *SessionConfig
	state          sessionState
	created        chan bool
	activated      chan bool
	lenChan        chan int
	errChan        chan error
	sndBuf, rcvBuf []byte
}

// Read reads data from the connection.
// Read can be made to time out and return an Error with Timeout() == true
// after a fixed time limit; see SetDeadline and SetReadDeadline.
//
// If the data is one of OpenSecureChannel or CloseSecureChannel, it will be handled automatically.
func (s *Session) Read(b []byte) (n int, err error) {
	if !(s.state == cliStateSessionActivated || s.state == srvStateSessionActivated) {
		return 0, ErrSessionNotActivated
	}
	for {
		select {
		case n := <-s.lenChan:
			copy(b, s.rcvBuf[:n])
			return n, nil
			/*
				case time.After(s.readDeadline):
					return 0, ErrTimeout
			*/
		}
	}
}

// ReadService reads the payload(=Service) from the connection.
// Which means the UASC Headers are omitted.
func (s *Session) ReadService(b []byte) (n int, err error) {
	if !(s.state == cliStateSessionActivated || s.state == srvStateSessionActivated) {
		return 0, ErrSessionNotActivated
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
			/*
				case time.After(s.readDeadline):
					return 0, ErrTimeout
			*/
		}
	}
}

// Write writes data to the connection.
// Write can be made to time out and return an Error with Timeout() == true
// after a fixed time limit; see SetDeadline and SetWriteDeadline.
func (s *Session) Write(b []byte) (n int, err error) {
	if s == nil || !(s.state == cliStateSessionActivated || s.state == srvStateSessionActivated) {
		return 0, ErrSessionNotActivated
	}

	return s.secChan.Write(b)
}

// WriteService writes data to the connection.
// Unlike Write(), given b in WriteService() should only be serialized service.Service,
// while the UASC header is automatically set by the package.
// This enables writing arbitrary Service even if the service is not implemented in the package.
func (s *Session) WriteService(b []byte) (n int, err error) {
	if !(s.state == cliStateSessionActivated || s.state == srvStateSessionActivated) {
		return 0, ErrSessionNotActivated
	}
	return s.secChan.WriteService(b)
}

// Close closes the connection.
// Any blocked Read or Write operations will be unblocked and return errors.
//
// Before closing, client sends CloseSessionRequest. Even if it fails, closing procedure does not stop.
func (s *Session) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	err := s.CloseSessionRequest(true)

	switch s.state {
	case cliStateCreateSessionSent, cliStateActivateSessionSent, cliStateCloseSessionSent, cliStateSessionCreated, cliStateSessionActivated:
		s.state = cliStateSessionCreated
	case srvStateSessionCreated, srvStateSessionActivated, srvStateSessionClosed:
		s.state = srvStateSessionClosed
	default:
		s.state = srvStateSessionClosed
		return ErrInvalidState
	}

	s.close()
	return err
}

func (s *Session) close() {
	s.cfg = nil
	s.rcvBuf = []byte{}
	s.sndBuf = []byte{}
	s.secChan = nil

	close(s.errChan)
	close(s.lenChan)
	close(s.created)
	close(s.activated)
}

// LocalAddr returns the local network address.
func (s *Session) LocalAddr() net.Addr {
	return s.secChan.LocalAddr()
}

// RemoteAddr returns the remote network address.
func (s *Session) RemoteAddr() net.Addr {
	return s.secChan.RemoteAddr()
}

// LocalEndpoint returns the local EndpointURL.
//
// This is expected to be called from server side of UACP Connection.
// If transport connection is not *uacp.Conn, LocalEndpoint() returns "".
func (s *Session) LocalEndpoint() string {
	return s.secChan.LocalEndpoint()
}

// RemoteEndpoint returns the remote EndpointURL.
//
// This is expected to be called from client side of SecureChannel.
// If transport connection is not *uacp.Conn, RemoteEndpoint() returns "".
func (s *Session) RemoteEndpoint() string {
	return s.secChan.RemoteEndpoint()
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
	return s.secChan.SetDeadline(t)
}

// SetReadDeadline sets the deadline for future Read calls
// and any currently-blocked Read call.
// A zero value for t means Read will not time out.
func (s *Session) SetReadDeadline(t time.Time) error {
	return s.secChan.SetReadDeadline(t)
}

// SetWriteDeadline sets the deadline for future Write calls
// and any currently-blocked Write call.
// Even if write times out, it may return n > 0, indicating that
// some of the data was successfully written.
// A zero value for t means Write will not time out.
func (s *Session) SetWriteDeadline(t time.Time) error {
	return s.secChan.SetWriteDeadline(t)
}

func (s *Session) monitor(ctx context.Context) {
	childCtx, cancel := context.WithCancel(ctx)

	for {
		select {
		case <-ctx.Done():
			cancel()
			return
		default:
			n, err := s.secChan.Read(s.rcvBuf)
			if err != nil {
				if err == io.EOF {
					continue
				}
				cancel()
				return
			}
			if n == 0 {
				continue
			}

			msg, err := Decode(s.rcvBuf[:n])
			if err != nil {
				// pass to the user if msg is undecodable as UASC.
				go s.notifyLength(childCtx, n)
				continue
			}

			switch m := msg.Service.(type) {
			case *services.CreateSessionRequest:
				s.handleCreateSessionRequest(m)
			case *services.CreateSessionResponse:
				s.handleCreateSessionResponse(m)
			case *services.ActivateSessionRequest:
				s.handleActivateSessionRequest(m)
			case *services.ActivateSessionResponse:
				s.handleActivateSessionResponse(m)
			case *services.CloseSessionRequest:
				s.handleCloseSessionRequest(m)
			case *services.CloseSessionResponse:
				s.handleCloseSessionResponse(m)
			default:
				// pass to the user if type of msg is unknown.
				go s.notifyLength(childCtx, n)
			}
		}
	}
}

func (s *Session) notifyLength(ctx context.Context, n int) {
	select {
	case <-ctx.Done():
		return
	case s.lenChan <- n:
		return
	}
}

func (s *Session) handleCreateSessionRequest(cs *services.CreateSessionRequest) {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch s.state {
	case srvStateSessionClosed, srvStateSessionCreated, srvStateSessionActivated:
		s.cfg.ClientDescription = cs.ClientDescription
		s.sndBuf = make([]byte, cs.MaxResponseMessageSize)

		if err := s.CreateSessionResponse(); err != nil {
			s.errChan <- err
		}

		s.state = srvStateSessionCreated
		// s.created <- true
		return
	default:
		s.errChan <- ErrInvalidState
	}
}

func (s *Session) handleCreateSessionResponse(cs *services.CreateSessionResponse) {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch s.state {
	case cliStateCreateSessionSent:
		if cs.ResponseHeader.ServiceResult == 0 {
			s.secChan.reqHeader.AuthenticationToken = cs.AuthenticationToken
			s.cfg.ServerEndpoints = cs.ServerEndpoints.EndpointDescriptions
			s.cfg.SessionTimeout = cs.RevisedSessionTimeout
			s.cfg.ServerSignature = cs.ServerSignature
			s.sndBuf = make([]byte, cs.MaxRequestMessageSize)

			s.state = cliStateSessionCreated
			s.created <- true
			return
		}
		s.errChan <- ErrUnexpectedMessage
	default:
		s.errChan <- ErrInvalidState
	}
}

func (s *Session) handleActivateSessionRequest(as *services.ActivateSessionRequest) {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch s.state {
	case srvStateSessionCreated, srvStateSessionActivated:
		if string(as.AuthenticationToken.GetIdentifier()) != string(s.cfg.AuthenticationToken.GetIdentifier()) {
			s.errChan <- ErrInvalidAuthenticationToken
		}

		s.cfg.ClientSignature = as.ClientSignature
		s.cfg.ClientSoftwareCertificates = as.ClientSoftwareCertificates.Certificates
		for _, str := range as.LocaleIDs.Strings {
			s.cfg.LocaleIDs = append(s.cfg.LocaleIDs, str.Get())
		}
		s.cfg.UserIdentityToken = as.UserIdentityToken
		s.cfg.UserTokenSignature = as.UserTokenSignature

		if err := s.ActivateSessionResponse(0); err != nil {
			s.errChan <- err
		}
		s.state = srvStateSessionActivated
		s.activated <- true
		return
	default:
		s.errChan <- ErrInvalidState
	}
}

func (s *Session) handleActivateSessionResponse(as *services.ActivateSessionResponse) {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch s.state {
	case cliStateActivateSessionSent:
		for _, result := range as.Results.Values {
			if result != 0 {
				s.errChan <- ErrRejected
			}
		}
		s.state = cliStateSessionActivated
		s.activated <- true
		return
	default:
		s.errChan <- ErrInvalidState
	}
}

func (s *Session) handleCloseSessionRequest(cs *services.CloseSessionRequest) {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch s.state {
	case srvStateSessionCreated, srvStateSessionActivated:
		/* XXX - not implemented yet.
		if cs.DeleteSubscriptions.Value == 0 {

		}
		*/
		if err := s.CloseSessionResponse(); err != nil {
			s.errChan <- err
		}
		s.state = srvStateSessionClosed
		return
	default:
		s.errChan <- ErrInvalidState
	}
}

func (s *Session) handleCloseSessionResponse(cs *services.CloseSessionResponse) {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch s.state {
	case cliStateCloseSessionSent:
		s.state = cliStateSessionClosed
		return
	default:
		s.errChan <- ErrInvalidState
	}
}

// CreateSessionRequest sends a CreateSessionRequest.
func (s *Session) CreateSessionRequest() error {
	s.secChan.reqHeader.RequestHandle++
	s.secChan.reqHeader.Timestamp = time.Now()

	nonce := make([]byte, 32)
	if _, err := rand.Read(nonce); err != nil {
		return err
	}
	csr, err := services.NewCreateSessionRequest(
		s.secChan.reqHeader, s.cfg.ClientDescription, "", s.secChan.RemoteEndpoint(),
		"gopcua-"+time.Now().String(), nonce, s.secChan.cfg.Certificate, 0, 0,
	).Serialize()
	if err != nil {
		return err
	}

	if _, err := s.secChan.WriteService(csr); err != nil {
		return err
	}
	return nil
}

// CreateSessionResponse sends a CreateSessionResponse.
func (s *Session) CreateSessionResponse() error {
	s.secChan.resHeader.Timestamp = time.Now()

	nonce := make([]byte, 32)
	if _, err := rand.Read(nonce); err != nil {
		return err
	}
	csr, err := services.NewCreateSessionResponse(
		// XXX - Give AuthenticationToken as NodeID
		s.secChan.resHeader, uint32(time.Now().UnixNano()), s.cfg.AuthenticationToken.GetIdentifier(), 0xffff,
		nonce, s.secChan.cfg.Certificate, s.cfg.ServerSignature.Algorithm.Get(),
		s.cfg.ServerSignature.Signature.Get(), 0xffff, s.cfg.ServerEndpoints...,
	).Serialize()
	if err != nil {
		return err
	}

	if _, err := s.secChan.WriteService(csr); err != nil {
		return err
	}
	return nil
}

// ActivateSessionRequest sends a ActivateSessionRequest.
func (s *Session) ActivateSessionRequest() error {
	s.secChan.reqHeader.RequestHandle++
	s.secChan.reqHeader.Timestamp = time.Now()

	asr, err := services.NewActivateSessionRequest(
		s.secChan.reqHeader, s.cfg.ClientSignature,
		s.cfg.ClientSoftwareCertificates, s.cfg.LocaleIDs,
		s.cfg.UserIdentityToken, s.cfg.UserTokenSignature,
	).Serialize()
	if err != nil {
		return err
	}

	if _, err := s.secChan.WriteService(asr); err != nil {
		return err
	}
	return nil
}

// ActivateSessionResponse sends a ActivateSessionResponse.
func (s *Session) ActivateSessionResponse(results ...uint32) error {
	s.secChan.resHeader.Timestamp = time.Now()

	nonce := make([]byte, 32)
	if _, err := rand.Read(nonce); err != nil {
		return err
	}
	asr, err := services.NewActivateSessionResponse(
		s.secChan.resHeader, nonce, results, []*services.DiagnosticInfo{
			services.NewNullDiagnosticInfo(),
		},
	).Serialize()
	if err != nil {
		return err
	}

	if _, err := s.secChan.WriteService(asr); err != nil {
		return err
	}
	return nil
}

// CloseSessionRequest sends a CloseSessionRequest.
func (s *Session) CloseSessionRequest(delete bool) error {
	s.secChan.reqHeader.RequestHandle++
	s.secChan.reqHeader.Timestamp = time.Now()

	csr, err := services.NewCloseSessionRequest(
		s.secChan.reqHeader, delete,
	).Serialize()
	if err != nil {
		return err
	}

	if _, err := s.secChan.WriteService(csr); err != nil {
		return err
	}
	return nil
}

// CloseSessionResponse sends a CloseSessionResponse.
func (s *Session) CloseSessionResponse() error {
	s.secChan.resHeader.Timestamp = time.Now()

	asr, err := services.NewCloseSessionResponse(s.secChan.resHeader).Serialize()
	if err != nil {
		return err
	}

	if _, err := s.secChan.WriteService(asr); err != nil {
		return err
	}
	return nil
}

type sessionState uint8

const (
	cliStateSessionClosed sessionState = iota
	cliStateCreateSessionSent
	cliStateSessionCreated
	cliStateActivateSessionSent
	cliStateSessionActivated
	cliStateCloseSessionSent
	srvStateSessionClosed
	srvStateSessionCreated
	srvStateActivateSessionSent
	srvStateSessionActivated
	srvStateCloseSessionSent
)

func (s sessionState) String() string {
	switch s {
	case cliStateSessionClosed:
		return "cliStateSessionClosed"
	case cliStateCreateSessionSent:
		return "cliStateCreateSessionSent"
	case cliStateSessionCreated:
		return "cliStateSessionCreated"
	case cliStateActivateSessionSent:
		return "cliStateActivateSessionSent"
	case cliStateSessionActivated:
		return "cliStateSessionActivated"
	case cliStateCloseSessionSent:
		return "cliStateCloseSessionSent"
	case srvStateSessionClosed:
		return "srvStateSessionClosed"
	case srvStateSessionCreated:
		return "srvStateSessionCreated"
	case srvStateActivateSessionSent:
		return "srvStateActivateSessionSent"
	case srvStateSessionActivated:
		return "srvStateSessionActivated"
	case srvStateCloseSessionSent:
		return "srvStateCloseSessionSent"
	default:
		return ""
	}
}

// Errors
var (
	ErrInvalidAuthenticationToken = errors.New("invalid AuthenticationToken")
	ErrSessionNotActivated        = errors.New("session is not activated")
)
