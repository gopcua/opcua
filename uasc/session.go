// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uasc

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/binary"
	"io"
	"net"
	"sync"
	"time"

	"github.com/wmnsk/gopcua/datatypes"
	"github.com/wmnsk/gopcua/services"
)

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

// todo(fs): this function should be removed since the caller should not read arbitrary bytes but full messages.
//
// Read reads data from the connection.
// Read can be made to time out and return an Error with Timeout() == true
// after a fixed time limit; see SetDeadline and SetReadDeadline.
//
// If the data is one of OpenSecureChannel or CloseSecureChannel, it will be handled automatically.
// func (s *Session) Read(b []byte) (n int, err error) {
// 	if !(s.state == cliStateSessionActivated || s.state == srvStateSessionActivated) {
// 		return 0, ErrSessionNotActivated
// 	}
// 	for {
// 		select {
// 		case n, ok := <-s.lenChan:
// 			if !ok {
// 				return 0, ErrSessionNotActivated
// 			}

// 			copy(b, s.rcvBuf[:n])
// 			return n, nil
// 			/*
// 				case time.After(s.readDeadline):
// 					return 0, ErrTimeout
// 			*/
// 		}
// 	}
// }

// ReadService reads the payload(=Service) from the connection.
// Which means the UASC Headers are omitted.
func (s *Session) ReadService() (services.Service, error) {
	if !(s.state == cliStateSessionActivated || s.state == srvStateSessionActivated) {
		return nil, ErrSessionNotActivated
	}
	return s.secChan.ReadService()
}

// todo(fs): this function should be removed since the caller should not write arbitrary bytes but full messages.
//
// Write writes data to the connection.
// Write can be made to time out and return an Error with Timeout() == true
// after a fixed time limit; see SetDeadline and SetWriteDeadline.
// func (s *Session) Write(b []byte) (n int, err error) {
// 	if s == nil || !(s.state == cliStateSessionActivated || s.state == srvStateSessionActivated) {
// 		return 0, ErrSessionNotActivated
// 	}
//
// 	return s.secChan.Write(b)
// }

// WriteService writes data to the connection.
// Unlike Write(), given b in WriteService() should only be serialized service.Service,
// while the UASC header is automatically set by the package.
// This enables writing arbitrary Service even if the service is not implemented in the package.
func (s *Session) WriteService(svc services.Service) error {
	if !(s.state == cliStateSessionActivated || s.state == srvStateSessionActivated) {
		return ErrSessionNotActivated
	}
	return s.secChan.WriteService(svc)
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
			if len(s.rcvBuf) < n {
				continue
			}

			msg := new(Message)
			_, err = msg.Decode(s.rcvBuf[:n])
			if err != nil {
				// pass to the user if msg is undecodable as UASC.
				go s.notifyLength(childCtx, n)
				continue
			}

			switch m := msg.Service.(type) {
			case *services.CreateSessionRequest:
				go s.handleCreateSessionRequest(m)
			case *services.CreateSessionResponse:
				go s.handleCreateSessionResponse(m)
			case *services.ActivateSessionRequest:
				go s.handleActivateSessionRequest(m)
			case *services.ActivateSessionResponse:
				go s.handleActivateSessionResponse(m)
			case *services.CloseSessionRequest:
				go s.handleCloseSessionRequest(m)
			case *services.CloseSessionResponse:
				go s.handleCloseSessionResponse(m)
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

		s.cfg.signatureToSend = services.NewSignatureDataFrom(cs.ClientCertificate, cs.ClientCertificate)
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
			/* XXX - should be handled properly when sign and encryption enabled.
			if err := validateSignature(cs.ServerSignature, s.cfg.mySignature); err != nil {
				s.errChan <- err
			}
			*/

			s.secChan.reqHeader.AuthenticationToken = cs.AuthenticationToken
			s.cfg.ServerEndpoints = cs.ServerEndpoints
			s.cfg.SessionTimeout = cs.RevisedSessionTimeout
			s.cfg.signatureToSend = services.NewSignatureDataFrom(cs.ServerCertificate, cs.ServerNonce)
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
		/* XXX - should be handled properly when sign and encryption enabled.
		if err := validateSignature(as.ClientSignature, s.cfg.mySignature); err != nil {
			if err := s.ActivateSessionResponse(status.BadUserSignatureInvalid); err != nil {
				s.errChan <- err
				return
			}
			s.errChan <- err
			return
		}
		*/

		s.cfg.LocaleIDs = append(s.cfg.LocaleIDs, as.LocaleIDs...)
		s.cfg.UserIdentityToken = as.UserIdentityToken.Value
		s.cfg.UserTokenSignature = as.UserTokenSignature

		if err := s.ActivateSessionResponse(0); err != nil {
			s.errChan <- err
			return
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
		for _, result := range as.Results {
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
	nonce := make([]byte, 32)
	if _, err := rand.Read(nonce); err != nil {
		return err
	}

	s.secChan.reqHeader.RequestHandle++
	s.secChan.reqHeader.Timestamp = time.Now()
	csr := services.NewCreateSessionRequest(
		s.secChan.reqHeader, s.cfg.ClientDescription, "", s.secChan.RemoteEndpoint(),
		"gopcua-"+time.Now().String(), nonce, s.secChan.cfg.Certificate, 0, 0,
	)

	if err := s.secChan.WriteService(csr); err != nil {
		s.secChan.reqHeader.RequestHandle--
		return err
	}

	// keep SignatureData to verify serverSignature in CreateSessionResponse.
	s.cfg.mySignature = services.NewSignatureDataFrom(s.secChan.cfg.Certificate, nonce)
	return nil
}

// CreateSessionResponse sends a CreateSessionResponse.
func (s *Session) CreateSessionResponse() error {
	sid := make([]byte, 4)
	if _, err := rand.Read(sid); err != nil {
		return err
	}
	sessID := binary.LittleEndian.Uint32(sid)
	nonce := make([]byte, 32)
	if _, err := rand.Read(nonce); err != nil {
		return err
	}

	s.secChan.resHeader.Timestamp = time.Now()
	csr := services.NewCreateSessionResponse(
		// XXX - Give AuthenticationToken as NodeID
		s.secChan.resHeader, datatypes.NewNumericNodeID(0, sessID), s.cfg.AuthenticationToken, 0xffff,
		nonce, s.secChan.cfg.Certificate, s.cfg.signatureToSend, 0xffff, s.cfg.ServerEndpoints...,
	)
	if err := s.secChan.WriteService(csr); err != nil {
		return err
	}

	// keep SignatureData to verify clientSignature in ActivateSessionRequest.
	s.cfg.mySignature = services.NewSignatureDataFrom(s.secChan.cfg.Certificate, nonce)
	return nil
}

// ActivateSessionRequest sends a ActivateSessionRequest.
func (s *Session) ActivateSessionRequest() error {
	s.secChan.reqHeader.RequestHandle++
	s.secChan.reqHeader.Timestamp = time.Now()
	asr := services.NewActivateSessionRequest(
		s.secChan.reqHeader, s.cfg.signatureToSend, s.cfg.LocaleIDs, s.cfg.UserIdentityToken,
		s.cfg.UserTokenSignature,
	)

	if err := s.secChan.WriteService(asr); err != nil {
		s.secChan.reqHeader.RequestHandle--
		return err
	}
	return nil
}

// ActivateSessionResponse sends a ActivateSessionResponse.
func (s *Session) ActivateSessionResponse(results ...uint32) error {
	nonce := make([]byte, 32)
	if _, err := rand.Read(nonce); err != nil {
		return err
	}

	s.secChan.resHeader.Timestamp = time.Now()
	asr := services.NewActivateSessionResponse(
		s.secChan.resHeader, nonce, results, []*datatypes.DiagnosticInfo{
			datatypes.NewNullDiagnosticInfo(),
		},
	)
	return s.secChan.WriteService(asr)
}

// CloseSessionRequest sends a CloseSessionRequest.
func (s *Session) CloseSessionRequest(delete bool) error {
	s.secChan.reqHeader.RequestHandle++
	s.secChan.reqHeader.Timestamp = time.Now()
	csr := services.NewCloseSessionRequest(
		s.secChan.reqHeader, delete,
	)

	if err := s.secChan.WriteService(csr); err != nil {
		s.secChan.reqHeader.RequestHandle--
		return err
	}
	return nil
}

// CloseSessionResponse sends a CloseSessionResponse.
func (s *Session) CloseSessionResponse() error {
	s.secChan.resHeader.Timestamp = time.Now()
	csr := services.NewCloseSessionResponse(s.secChan.resHeader)
	return s.secChan.WriteService(csr)
}

// ReadRequest sends a ReadRequest.
func (s *Session) ReadRequest(maxAge uint64, tsRet services.TimestampsToReturn, nodes ...*datatypes.ReadValueID) error {
	s.secChan.reqHeader.RequestHandle++
	s.secChan.reqHeader.Timestamp = time.Now()
	rdr := services.NewReadRequest(
		s.secChan.reqHeader, maxAge, tsRet, nodes...,
	)

	if err := s.secChan.WriteService(rdr); err != nil {
		s.secChan.reqHeader.RequestHandle--
		return err
	}
	return nil
}

// ReadResponse sends a ReadResponse.
func (s *Session) ReadResponse(results ...*datatypes.DataValue) error {
	s.secChan.resHeader.Timestamp = time.Now()
	rdr := services.NewReadResponse(
		s.secChan.resHeader, nil, results...,
	)
	return s.secChan.WriteService(rdr)
}

// WriteRequest sends a WriteRequest.
func (s *Session) WriteRequest(nodes ...*datatypes.WriteValue) error {
	s.secChan.reqHeader.RequestHandle++
	s.secChan.reqHeader.Timestamp = time.Now()
	wrr := services.NewWriteRequest(
		s.secChan.reqHeader, nodes...,
	)
	if err := s.secChan.WriteService(wrr); err != nil {
		s.secChan.reqHeader.RequestHandle--
		return err
	}
	return nil
}

// WriteResponse sends a WriteResponse.
func (s *Session) WriteResponse(results ...uint32) error {
	s.secChan.resHeader.Timestamp = time.Now()
	wrr := services.NewWriteResponse(
		s.secChan.resHeader, nil, results...,
	)
	return s.secChan.WriteService(wrr)
}

func validateSignature(received, expected *services.SignatureData) error {
	if received.Algorithm == "" || expected.Algorithm == "" {
		return nil
	}
	if received.Algorithm != expected.Algorithm {
		return ErrInvalidSignatureAlgorithm
	}
	if !bytes.Equal(received.Signature, expected.Signature) {
		return ErrInvalidSignatureData
	}
	return nil
}
