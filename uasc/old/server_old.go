// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uasc

// // ListenAndAcceptSecureChannel starts UASC server on top of established transport connection.
// func ListenAndAcceptSecureChannel(ctx context.Context, transport net.Conn, cfg *Config) (*SecureChannel, error) {
// 	if err := cfg.validate("server"); err != nil {
// 		return nil, err
// 	}

// 	secChan := &SecureChannel{
// 		mu:        new(sync.Mutex),
// 		lowerConn: transport,
// 		reqHeader: services.NewRequestHeader(
// 			datatypes.NewTwoByteNodeID(0), time.Now(), 0, 0,
// 			0xffff, "", services.NewNullAdditionalHeader(),
// 		),
// 		resHeader: services.NewResponseHeader(
// 			time.Now(), 0, 0, &datatypes.DiagnosticInfo{},
// 			[]string{}, services.NewNullAdditionalHeader(),
// 		),
// 		cfg:     cfg,
// 		state:   srvStateSecureChannelClosed,
// 		opened:  make(chan bool),
// 		lenChan: make(chan int),
// 		errChan: make(chan error),
// 		rcvBuf:  make([]byte, 0xffff),
// 	}

// 	go secChan.monitor(ctx)
// 	for {
// 		select {
// 		case ok := <-secChan.opened:
// 			if ok {
// 				return secChan, nil
// 			}
// 		case err := <-secChan.errChan:
// 			return nil, err
// 		}
// 	}
// }

// // ListenAndAcceptSession starts UASC server on top of established transport connection.
// func ListenAndAcceptSession(ctx context.Context, secChan *SecureChannel, cfg *SessionConfig) (*Session, error) {
// 	session := &Session{
// 		mu:        new(sync.Mutex),
// 		secChan:   secChan,
// 		cfg:       cfg,
// 		state:     srvStateSessionClosed,
// 		created:   make(chan bool),
// 		activated: make(chan bool),
// 		lenChan:   make(chan int),
// 		errChan:   make(chan error),
// 		rcvBuf:    make([]byte, 0xffff),
// 	}

// 	go session.monitor(ctx)
// 	for {
// 		select {
// 		case ok := <-session.activated:
// 			if ok {
// 				return session, nil
// 			}
// 		case err := <-session.errChan:
// 			return nil, err
// 		}
// 	}
// }
