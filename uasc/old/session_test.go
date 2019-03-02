// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uasc

// func setUpSession(ctx context.Context) (*Session, *Session, error) {
// 	cliChan, srvChan, err := setUpSecureChannel(ctx)
// 	if err != nil {
// 		return nil, nil, err
// 	}

// 	srvSessionChan := make(chan *Session)
// 	errChan := make(chan error)
// 	go func() {
// 		srvCfg := NewServerSessionConfig(srvChan)
// 		srvSession, err := ListenAndAcceptSession(ctx, srvChan, srvCfg)
// 		if err != nil {
// 			errChan <- err
// 		}
// 		srvSessionChan <- srvSession
// 	}()

// 	cliCfg := NewClientSessionConfig([]string{}, datatypes.NewAnonymousIdentityToken("anonymous"))
// 	cliSession, err := CreateSession(ctx, cliChan, cliCfg, 3, 5*time.Second)
// 	if err != nil {
// 		return nil, nil, err
// 	}

// 	if err := cliSession.Activate(); err != nil {
// 		return nil, nil, err
// 	}

// 	select {
// 	case srvSession := <-srvSessionChan:
// 		return cliSession, srvSession, nil
// 	case err := <-errChan:
// 		return nil, nil, err
// 	case <-time.After(10 * time.Second):
// 		return nil, nil, errors.New("timed out")
// 	}
// }

// func TestClientSessionWrite(t *testing.T) {
// 	ctx := context.Background()
// 	ctx, cancel := context.WithCancel(ctx)
// 	defer cancel()

// 	cliSession, srvSession, err := setUpSession(ctx)
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	if err := cliSession.WriteService(msg); err != nil {
// 		t.Fatal(err)
// 	}

// 	got, err := srvSession.ReadService()
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	if diff := cmp.Diff(got, msg); diff != "" {
// 		t.Error(diff)
// 	}
// }

// func TestServerSessionWrite(t *testing.T) {
// 	ctx := context.Background()
// 	ctx, cancel := context.WithCancel(ctx)
// 	defer cancel()

// 	cliSession, srvSession, err := setUpSession(ctx)
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	if err := srvSession.WriteService(msg); err != nil {
// 		t.Fatal(err)
// 	}

// 	got, err := cliSession.ReadService()
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	if diff := cmp.Diff(got, msg); diff != "" {
// 		t.Error(diff)
// 	}
// }

// func TestClientSessionClose(t *testing.T) {
// 	ctx := context.Background()
// 	ctx, cancel := context.WithCancel(ctx)
// 	defer cancel()

// 	cliSession, _, err := setUpSession(ctx)
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	if err := cliSession.Close(); err != nil {
// 		t.Fatal(err)
// 	}
// }

// func TestServerSessionClose(t *testing.T) {
// 	ctx := context.Background()
// 	ctx, cancel := context.WithCancel(ctx)
// 	defer cancel()

// 	_, srvSession, err := setUpSession(ctx)
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	if err := srvSession.Close(); err != nil {
// 		t.Fatal(err)
// 	}
// }

// func TestSessionLocalEndpoint(t *testing.T) {
// 	ctx := context.Background()
// 	ctx, cancel := context.WithCancel(ctx)
// 	defer cancel()

// 	_, srvSession, err := setUpSession(ctx)
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	if diff := cmp.Diff(srvSession.LocalEndpoint(), endpoint); diff != "" {
// 		t.Error(diff)
// 	}
// }

// func TestSessionRemoteEndpoint(t *testing.T) {
// 	ctx := context.Background()
// 	ctx, cancel := context.WithCancel(ctx)
// 	defer cancel()

// 	cliSession, _, err := setUpSession(ctx)
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	if diff := cmp.Diff(cliSession.RemoteEndpoint(), endpoint); diff != "" {
// 		t.Error(diff)
// 	}
// }
