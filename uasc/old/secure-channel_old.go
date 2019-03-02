// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uasc

// // SecureChannel is an implementation of the net.Conn interface for Secure Channel in OPC UA Secure Conversation.
// //
// // In UASC, there are two types of net.Conn: SecureChannel and Session. Each Conn is handled in different manner.
// type SecureChannel struct {
// 	mu             *sync.Mutex
// 	lowerConn      net.Conn
// 	cfg            *Config
// 	reqHeader      *services.RequestHeader
// 	resHeader      *services.ResponseHeader
// 	rcvBuf, sndBuf []byte
// 	state          secChanState
// 	opened         chan bool
// 	lenChan        chan int
// 	errChan        chan error
// }

// // Read reads data from the connection.
// // Read can be made to time out and return an Error with Timeout() == true
// // after a fixed time limit; see SetDeadline and SetReadDeadline.
// //
// // If the data is one of OpenSecureChannel or CloseSecureChannel, it will be handled automatically.
// func (s *SecureChannel) Read(b []byte) (n int, err error) {
// 	n, err = s.read()
// 	if err != nil {
// 		return 0, err
// 	}
// 	copy(b, s.rcvBuf[:n])
// 	return n, nil
// }

// func (s *SecureChannel) ReadMessage() (*Message, error) {
// 	n, err := s.read()
// 	if err != nil {
// 		return nil, err
// 	}
// 	fmt.Println("SecureChannel.ReadMessage.Decode")
// 	m := new(Message)
// 	if _, err := m.Decode(s.rcvBuf[:n]); err != nil {
// 		return nil, err
// 	}
// 	return m, nil
// }

// // ReadService reads the payload(=Service) from the connection.
// // Which means the UASC Headers are omitted.
// func (s *SecureChannel) ReadService() (services.Service, error) {
// 	m, err := s.ReadMessage()
// 	if err != nil {
// 		return nil, err
// 	}
// 	log.Printf("uasc: recv %s%c", m.MessageType, m.ChunkType)
// 	return m.Service, nil
// }

// func (s *SecureChannel) read() (n int, err error) {
// 	if !(s.state == cliStateSecureChannelOpened || s.state == srvStateSecureChannelOpened) {
// 		return 0, ErrSecureChannelNotOpened
// 	}
// 	for {
// 		select {
// 		case n, ok := <-s.lenChan:
// 			if !ok {
// 				return 0, ErrSecureChannelNotOpened
// 			}
// 			return n, nil
// 		}
// 	}
// }

// // WriteService writes data to the connection.
// // Unlike Write(), given b in WriteService() should only be serialized service.Service,
// // while the UASC header is automatically set by the package.
// // This enables writing arbitrary Service even if the service is not implemented in the package.
// func (s *SecureChannel) WriteService(svc services.Service) error {
// 	if !(s.state == cliStateSecureChannelOpened || s.state == srvStateSecureChannelOpened) {
// 		return ErrSecureChannelNotOpened
// 	}
// 	s.cfg.SequenceNumber++
// 	return s.write(NewMessage(svc, s.cfg))
// }

// func (s *SecureChannel) write(msg *Message) error {
// 	b, err := ua.Encode(msg)
// 	if err != nil {
// 		return err
// 	}
// 	log.Printf("uasc: send %s%c %d bytes %x", msg.MessageType, msg.ChunkType, msg.MessageSize, b)
// 	if _, err := s.lowerConn.Write(b); err != nil {
// 		log.Printf("usac: send err:%s", err)
// 	}
// 	return err
// }

// // Close closes the connection.
// // Any blocked Read or Write operations will be unblocked and return errors.
// //
// // Before closing, client sends CloseSecureChannelRequest. Even if it fails, closing procedure does not stop.
// func (s *SecureChannel) Close() error {
// 	s.mu.Lock()
// 	defer s.mu.Unlock()

// 	err := s.CloseSecureChannelRequest()

// 	switch s.state {
// 	case cliStateCloseSecureChannelSent, cliStateOpenSecureChannelSent, cliStateSecureChannelOpened, cliStateSecureChannelClosed:
// 		s.state = cliStateSecureChannelClosed
// 	case srvStateCloseSecureChannelSent, srvStateSecureChannelOpened, srvStateSecureChannelClosed:
// 		s.state = srvStateSecureChannelClosed
// 	default:
// 		s.state = srvStateSecureChannelClosed
// 		return ErrInvalidState
// 	}

// 	s.close()
// 	return err
// }

// func (s *SecureChannel) close() {
// 	s.cfg = nil
// 	s.reqHeader = nil
// 	s.resHeader = nil
// 	s.rcvBuf = []byte{}
// 	s.sndBuf = []byte{}

// 	close(s.errChan)
// 	close(s.lenChan)
// 	close(s.opened)
// }

// // LocalAddr returns the local network address.
// func (s *SecureChannel) LocalAddr() net.Addr {
// 	return s.lowerConn.LocalAddr()
// }

// // RemoteAddr returns the remote network address.
// func (s *SecureChannel) RemoteAddr() net.Addr {
// 	return s.lowerConn.RemoteAddr()
// }

// // LocalEndpoint returns the local EndpointURL.
// //
// // This is expected to be called from server side of UACP Connection.
// // If transport connection is not *uacp.Conn, LocalEndpoint() returns "".
// func (s *SecureChannel) LocalEndpoint() string {
// 	conn, ok := s.lowerConn.(*uacp.Conn)
// 	if !ok {
// 		return ""
// 	}
// 	return conn.LocalEndpoint()
// }

// // RemoteEndpoint returns the remote EndpointURL.
// //
// // This is expected to be called from client side of SecureChannel.
// // If transport connection is not *uacp.Conn, RemoteEndpoint() returns "".
// func (s *SecureChannel) RemoteEndpoint() string {
// 	conn, ok := s.lowerConn.(*uacp.Conn)
// 	if !ok {
// 		return ""
// 	}
// 	return conn.RemoteEndpoint()
// }

// // SetDeadline sets the read and write deadlines associated
// // with the connection. It is equivalent to calling both
// // SetReadDeadline and SetWriteDeadline.
// //
// // A deadline is an absolute time after which I/O operations
// // fail with a timeout (see type Error) instead of
// // blocking. The deadline applies to all future and pending
// // I/O, not just the immediately following call to Read or
// // Write. After a deadline has been exceeded, the connection
// // can be refreshed by setting a deadline in the future.
// //
// // An idle timeout can be implemented by repeatedly extending
// // the deadline after successful Read or Write calls.
// //
// // A zero value for t means I/O operations will not time out.
// func (s *SecureChannel) SetDeadline(t time.Time) error {
// 	return s.lowerConn.SetDeadline(t)
// }

// // SetReadDeadline sets the deadline for future Read calls
// // and any currently-blocked Read call.
// // A zero value for t means Read will not time out.
// func (s *SecureChannel) SetReadDeadline(t time.Time) error {
// 	return s.lowerConn.SetReadDeadline(t)
// }

// // SetWriteDeadline sets the deadline for future Write calls
// // and any currently-blocked Write call.
// // Even if write times out, it may return n > 0, indicating that
// // some of the data was successfully written.
// // A zero value for t means Write will not time out.
// func (s *SecureChannel) SetWriteDeadline(t time.Time) error {
// 	return s.lowerConn.SetWriteDeadline(t)
// }

// func (s *SecureChannel) monitor(ctx context.Context) {
// 	childCtx, cancel := context.WithCancel(ctx)
// 	for {
// 		select {
// 		case <-ctx.Done():
// 			cancel()
// 			return
// 		default:
// 			n, err := s.lowerConn.Read(s.rcvBuf)
// 			if err != nil {
// 				if err == io.EOF {
// 					continue
// 				}
// 				cancel()
// 				return
// 			}
// 			if n == 0 {
// 				continue
// 			}
// 			if len(s.rcvBuf) < n {
// 				continue
// 			}
// 			log.Printf("uasc: recv %d bytes %x", n, s.rcvBuf[:n])

// 			msg := new(Message)
// 			_, err = msg.Decode(s.rcvBuf[:n])
// 			if err != nil {
// 				log.Printf("uasc: decode failed: %s", err)
// 				// pass to the user if msg is undecodable as UASC.
// 				go s.notifyLength(childCtx, n)
// 				continue
// 			}
// 			log.Printf("uasc: recv %s%c %x", msg.MessageType, msg.ChunkType, s.rcvBuf[:n])

// 			s.cfg.RequestID = msg.RequestID
// 			switch m := msg.Service.(type) {
// 			case *services.OpenSecureChannelRequest:
// 				go s.handleOpenSecureChannelRequest(m)
// 			case *services.OpenSecureChannelResponse:
// 				go s.handleOpenSecureChannelResponse(m)
// 			case *services.CloseSecureChannelRequest:
// 				go s.handleCloseSecureChannelRequest(m)
// 			case *services.CloseSecureChannelResponse:
// 				go s.handleCloseSecureChannelResponse(m)
// 			default:
// 				// pass to the user if type of msg is not
// 				// related to SecureChannel establishment.
// 				go s.notifyLength(childCtx, n)
// 			}
// 		}
// 	}
// }

// func (s *SecureChannel) notifyLength(ctx context.Context, n int) {
// 	select {
// 	case <-ctx.Done():
// 		return
// 	case s.lenChan <- n:
// 		return
// 	default:
// 		if !(s.state == cliStateSecureChannelOpened || s.state == srvStateSecureChannelOpened) {
// 			return
// 		}
// 	}
// }

// func (s *SecureChannel) handleOpenSecureChannelRequest(o *services.OpenSecureChannelRequest) {
// 	s.mu.Lock()
// 	defer s.mu.Unlock()

// 	switch s.state {
// 	// if state is closed, server accepts OpenSecureChannelRequest.
// 	case srvStateSecureChannelClosed:
// 		switch o.MessageSecurityMode {
// 		// accepts only if MessageSecurityMode is None.
// 		case services.SecModeNone:
// 			s.resHeader.RequestHandle = o.RequestHeader.RequestHandle
// 			if err := s.OpenSecureChannelResponse(0); err != nil {
// 				s.errChan <- err
// 			}
// 			s.state = srvStateSecureChannelOpened
// 			s.opened <- true
// 		// respond with BadSecurityModeRejected and notify server
// 		default:
// 			if err := s.OpenSecureChannelResponse(status.BadSecurityModeRejected); err != nil {
// 				s.errChan <- err
// 			}
// 			s.errChan <- ErrSecurityModeUnsupported
// 		}
// 	// if SecureChannel is already opened, respond with BadAlreadyExists.
// 	case srvStateSecureChannelOpened, srvStateCloseSecureChannelSent:
// 		if err := s.OpenSecureChannelResponse(status.BadAlreadyExists); err != nil {
// 			s.errChan <- err
// 		}
// 	// client never accept OpenSecureChannelRequest, just ignore it.
// 	case cliStateSecureChannelClosed, cliStateOpenSecureChannelSent, cliStateSecureChannelOpened, cliStateCloseSecureChannelSent:
// 	// invalid secChanState. conn should be closed in error handler.
// 	default:
// 		s.errChan <- ErrInvalidState
// 	}
// }

// func (s *SecureChannel) handleOpenSecureChannelResponse(o *services.OpenSecureChannelResponse) {
// 	s.mu.Lock()
// 	defer s.mu.Unlock()

// 	switch s.state {
// 	// client accepts OpenSecureChannelResponse only after sending OpenSecureChannelRequest.
// 	case cliStateOpenSecureChannelSent:
// 		switch o.ResponseHeader.ServiceResult {
// 		case 0: // Good
// 			s.cfg.SecureChannelID = o.SecurityToken.ChannelID
// 			s.cfg.SecurityTokenID = o.SecurityToken.TokenID
// 			s.state = cliStateSecureChannelOpened
// 			s.opened <- true
// 		case status.BadSecurityModeRejected:
// 			s.state = cliStateSecureChannelClosed
// 			s.errChan <- ErrRejected
// 		}
// 	// if client SecureChannel is closed or opened, just ignore OpenSecureChannelResponse.
// 	case cliStateSecureChannelClosed, cliStateSecureChannelOpened, cliStateCloseSecureChannelSent:
// 	// server never accept OpenSecureChannelResponse , just ignore it.
// 	case srvStateSecureChannelClosed, srvStateSecureChannelOpened, srvStateCloseSecureChannelSent:
// 	// invalid secChanState. SecureChannel should be closed in error handler.
// 	default:
// 		s.errChan <- ErrInvalidState
// 	}
// }

// func (s *SecureChannel) handleCloseSecureChannelRequest(c *services.CloseSecureChannelRequest) {
// 	s.mu.Lock()
// 	defer s.mu.Unlock()

// 	switch s.state {
// 	// if client SecureChannel is opened, accept CloseSecureChannelRequest.
// 	case cliStateSecureChannelOpened:
// 		s.reqHeader.RequestHandle = c.RequestHeader.RequestHandle
// 		if err := s.CloseSecureChannelResponse(0); err != nil {
// 			s.errChan <- err
// 		}
// 		s.state = cliStateCloseSecureChannelSent
// 	// if server SecureChannel is opened, accept CloseSecureChannelRequest.
// 	case srvStateSecureChannelOpened:
// 		s.reqHeader.RequestHandle = c.RequestHeader.RequestHandle
// 		if err := s.CloseSecureChannelResponse(0); err != nil {
// 			s.errChan <- err
// 		}
// 		s.state = srvStateCloseSecureChannelSent
// 	// if client/server SecureChannel is not opened, ignore CloseSecureChannelRequest.
// 	case cliStateSecureChannelClosed, cliStateOpenSecureChannelSent, cliStateCloseSecureChannelSent, srvStateSecureChannelClosed, srvStateCloseSecureChannelSent:
// 	// invalid secChanState. conn should be closed in error handler.
// 	default:
// 		s.errChan <- ErrInvalidState
// 	}
// }

// func (s *SecureChannel) handleCloseSecureChannelResponse(c *services.CloseSecureChannelResponse) {
// 	s.mu.Lock()
// 	defer s.mu.Unlock()

// 	switch s.state {
// 	// client accepts CloseSecureChannelResponse only after sending CloseSecureChannelResponse.
// 	case cliStateCloseSecureChannelSent:
// 		s.state = cliStateSecureChannelClosed
// 	// server accepts CloseSecureChannelResponse only after sending CloseSecureChannelResponse.
// 	case srvStateCloseSecureChannelSent:
// 		s.state = srvStateSecureChannelClosed
// 	// if client/server conn is opened, just ignore CloseSecureChannelResponse.
// 	case cliStateSecureChannelClosed, cliStateOpenSecureChannelSent, cliStateSecureChannelOpened, srvStateSecureChannelClosed, srvStateSecureChannelOpened:
// 	// invalid secChanState. conn should be closed in error handler.
// 	default:
// 		s.errChan <- ErrInvalidState
// 	}
// }

// // OpenSecureChannelRequest sends OpenSecureChannelRequest on top of UASC to SecureChannel.
// func (s *SecureChannel) OpenSecureChannelRequest() error {
// 	nonce := make([]byte, 32)
// 	if _, err := rand.Read(nonce); err != nil {
// 		return err
// 	}

// 	s.cfg.SequenceNumber++
// 	s.reqHeader.RequestHandle++
// 	s.reqHeader.Timestamp = time.Now()

// 	svc := services.NewOpenSecureChannelRequest(s.reqHeader, 0, services.ReqTypeIssue, s.cfg.SecurityMode, s.cfg.Lifetime, nonce)
// 	if err := s.write(NewMessage(svc, s.cfg)); err != nil {
// 		s.cfg.SequenceNumber--
// 		s.reqHeader.RequestHandle--
// 		return err
// 	}
// 	return nil
// }

// // OpenSecureChannelResponse sends OpenSecureChannelResponse on top of UASC to SecureChannel.
// func (s *SecureChannel) OpenSecureChannelResponse(code uint32) error {
// 	nonce := make([]byte, 32)
// 	if _, err := rand.Read(nonce); err != nil {
// 		return err
// 	}

// 	s.cfg.SequenceNumber++
// 	s.resHeader.ServiceResult = code
// 	s.resHeader.Timestamp = time.Now()

// 	svc := services.NewOpenSecureChannelResponse(
// 		s.resHeader, 0, services.NewChannelSecurityToken(
// 			s.cfg.SecureChannelID, s.cfg.SecurityTokenID, time.Now(), s.cfg.Lifetime,
// 		), nonce,
// 	)

// 	if err := s.write(NewMessage(svc, s.cfg)); err != nil {
// 		s.cfg.SequenceNumber--
// 		return err
// 	}
// 	return nil
// }

// // CloseSecureChannelRequest sends CloseSecureChannelRequest on top of UASC to SecureChannel.
// func (s *SecureChannel) CloseSecureChannelRequest() error {
// 	s.cfg.SequenceNumber++
// 	s.reqHeader.RequestHandle++
// 	s.reqHeader.Timestamp = time.Now()

// 	svc := services.NewCloseSecureChannelRequest(s.reqHeader, s.cfg.SecureChannelID)
// 	if err := s.write(NewMessage(svc, s.cfg)); err != nil {
// 		s.cfg.SequenceNumber--
// 		s.reqHeader.RequestHandle--
// 		return err
// 	}
// 	return nil
// }

// // CloseSecureChannelResponse sends CloseSecureChannelResponse on top of UASC to SecureChannel.
// func (s *SecureChannel) CloseSecureChannelResponse(code uint32) error {
// 	s.cfg.SequenceNumber++
// 	s.resHeader.ServiceResult = code
// 	s.resHeader.Timestamp = time.Now()

// 	svc := services.NewCloseSecureChannelResponse(s.resHeader)
// 	if err := s.write(NewMessage(svc, s.cfg)); err != nil {
// 		s.cfg.SequenceNumber--
// 		return err
// 	}
// 	return nil
// }

// // GetEndpointsRequest sends GetEndpointsRequest on top of UASC to SecureChannel.
// func (s *SecureChannel) GetEndpointsRequest(locales, uris []string) error {
// 	s.cfg.SequenceNumber++
// 	s.reqHeader.RequestHandle++
// 	s.reqHeader.Timestamp = time.Now()

// 	svc := services.NewGetEndpointsRequest(s.reqHeader, s.RemoteEndpoint(), locales, uris)
// 	if err := s.write(NewMessage(svc, s.cfg)); err != nil {
// 		s.cfg.SequenceNumber--
// 		s.reqHeader.RequestHandle--
// 		return err
// 	}
// 	return nil
// }

// // GetEndpointsResponse sends GetEndpointsResponse on top of UASC to SecureChannel.
// //
// // XXX - This is to be improved with some external configuration to describe endpoints infomation in the future release.
// func (s *SecureChannel) GetEndpointsResponse(code uint32, endpoints ...*services.EndpointDescription) error {
// 	s.cfg.SequenceNumber++
// 	s.resHeader.ServiceResult = code
// 	s.resHeader.Timestamp = time.Now()

// 	svc := services.NewGetEndpointsResponse(s.resHeader, endpoints...)
// 	if err := s.write(NewMessage(svc, s.cfg)); err != nil {
// 		s.cfg.SequenceNumber--
// 		return err
// 	}
// 	return nil
// }

// // FindServersRequest sends FindServersRequest on top of UASC to SecureChannel.
// func (s *SecureChannel) FindServersRequest(locales []string, servers ...string) error {
// 	s.cfg.SequenceNumber++
// 	s.reqHeader.RequestHandle++
// 	s.reqHeader.Timestamp = time.Now()

// 	svc := services.NewFindServersRequest(s.reqHeader, s.RemoteEndpoint(), locales, servers)
// 	if err := s.write(NewMessage(svc, s.cfg)); err != nil {
// 		s.cfg.SequenceNumber--
// 		s.reqHeader.RequestHandle--
// 		return err
// 	}
// 	return nil
// }

// // FindServersResponse sends FindServersResponse on top of UASC to SecureChannel.
// //
// // XXX - This is to be improved with some external configuration to describe application infomation in the future release.
// func (s *SecureChannel) FindServersResponse(code uint32, apps ...*services.ApplicationDescription) error {
// 	s.cfg.SequenceNumber++
// 	s.resHeader.ServiceResult = code
// 	s.resHeader.Timestamp = time.Now()

// 	svc := services.NewFindServersResponse(s.resHeader, apps...)
// 	if err := s.write(NewMessage(svc, s.cfg)); err != nil {
// 		s.cfg.SequenceNumber--
// 		return err
// 	}
// 	return nil
// }
