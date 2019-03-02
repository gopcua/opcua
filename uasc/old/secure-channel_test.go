// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uasc

// var (
// 	endpoint  = "opc.tcp://127.0.0.1:4840/foo/bar"
// 	policyURI = "http://opcfoundation.org/UA/SecurityPolicy#None"
// 	cliCfg    = NewClientConfig(policyURI, nil, nil, 3333, services.SecModeNone, 3600000)
// 	srvCfg    = NewServerConfig(policyURI, nil, nil, 1111, services.SecModeNone, 2222, 3600000)
// 	msg       = services.NewReadRequest(
// 		services.NewRequestHeader(
// 			datatypes.NewTwoByteNodeID(0),
// 			time.Date(2018, time.August, 10, 23, 0, 0, 0, time.UTC),
// 			1, 0, 0, "", services.NewNullAdditionalHeader(),
// 		),
// 		0, services.TimestampsToReturnBoth,
// 		datatypes.NewReadValueID(
// 			datatypes.NewFourByteNodeID(0, 2256),
// 			datatypes.IntegerIDValue,
// 			"", 0, "",
// 		),
// 	)
// 	// msg       = []byte{0xde, 0xad, 0xbe, 0xef}
// )

// func setUpSecureChannel(ctx context.Context) (*SecureChannel, *SecureChannel, error) {
// 	ln, err := uacp.Listen(endpoint, 0xffff)
// 	if err != nil {
// 		return nil, nil, err
// 	}
// 	defer ln.Close()

// 	srvChanChan := make(chan *SecureChannel)
// 	errChan := make(chan error)
// 	go func() {
// 		srvConn, err := ln.Accept(ctx)
// 		if err != nil {
// 			errChan <- err
// 		}

// 		srvChan, err := ListenAndAcceptSecureChannel(ctx, srvConn, srvCfg)
// 		if err != nil {
// 			errChan <- err
// 		}
// 		srvChanChan <- srvChan
// 	}()

// 	cliConn, err := uacp.Dial(ctx, endpoint)
// 	if err != nil {
// 		return nil, nil, err
// 	}

// 	cliChan, err := OpenSecureChannel(ctx, cliConn, cliCfg, 5*time.Second, 3)
// 	if err != nil {
// 		return nil, nil, err
// 	}

// 	select {
// 	case srvChan := <-srvChanChan:
// 		return cliChan, srvChan, nil
// 	case err := <-errChan:
// 		return nil, nil, err
// 	case <-time.After(10 * time.Second):
// 		return nil, nil, errors.New("timed out")
// 	}
// }

// func TestClientWrite(t *testing.T) {
// 	ctx := context.Background()
// 	ctx, cancel := context.WithCancel(ctx)
// 	defer cancel()

// 	cliChan, srvChan, err := setUpSecureChannel(ctx)
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	if err := cliChan.WriteService(msg); err != nil {
// 		t.Fatal(err)
// 	}

// 	got, err := srvChan.ReadService()
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	verify.Values(t, "", got, msg)
// }

// func TestServerWrite(t *testing.T) {
// 	ctx := context.Background()
// 	ctx, cancel := context.WithCancel(ctx)
// 	defer cancel()

// 	cliChan, srvChan, err := setUpSecureChannel(ctx)
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	if err := srvChan.WriteService(msg); err != nil {
// 		t.Fatal(err)
// 	}

// 	got, err := cliChan.ReadService()
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	verify.Values(t, "", got, msg)
// }

// func TestClientClose(t *testing.T) {
// 	ctx := context.Background()
// 	ctx, cancel := context.WithCancel(ctx)
// 	defer cancel()

// 	cliChan, _, err := setUpSecureChannel(ctx)
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	if err := cliChan.Close(); err != nil {
// 		t.Fatal(err)
// 	}
// }

// func TestServerClose(t *testing.T) {
// 	ctx := context.Background()
// 	ctx, cancel := context.WithCancel(ctx)
// 	defer cancel()

// 	_, srvChan, err := setUpSecureChannel(ctx)
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	if err := srvChan.Close(); err != nil {
// 		t.Fatal(err)
// 	}
// }

// func TestGetEndpointRequest(t *testing.T) {
// 	ctx := context.Background()
// 	ctx, cancel := context.WithCancel(ctx)
// 	defer cancel()

// 	cliChan, srvChan, err := setUpSecureChannel(ctx)
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	if err := cliChan.GetEndpointsRequest([]string{"ja-JP"}, []string{"uri"}); err != nil {
// 		t.Fatal(err)
// 	}

// 	msg, err := srvChan.ReadService()
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	if _, ok := msg.(*services.GetEndpointsRequest); !ok {
// 		t.Error("failed to assert type")
// 	}
// }

// func TestGetEndpointResponse(t *testing.T) {
// 	ctx := context.Background()
// 	ctx, cancel := context.WithCancel(ctx)
// 	defer cancel()

// 	cliChan, srvChan, err := setUpSecureChannel(ctx)
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	if err := srvChan.GetEndpointsResponse(0, services.NewEndpointDescription(
// 		srvChan.LocalEndpoint(), services.NewApplicationDescription(
// 			"", "", "", services.AppTypeServer, "", "", []string{""},
// 		), nil, services.SecModeNone, policyURI,
// 		[]*services.UserTokenPolicy{
// 			services.NewUserTokenPolicy("id", 0, "", "", ""),
// 		}, "", 0,
// 	)); err != nil {
// 		t.Fatal(err)
// 	}

// 	msg, err := cliChan.ReadService()
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	if _, ok := msg.(*services.GetEndpointsResponse); !ok {
// 		t.Error("failed to assert type")
// 	}
// }

// func TestFindServerRequest(t *testing.T) {
// 	ctx := context.Background()
// 	ctx, cancel := context.WithCancel(ctx)
// 	defer cancel()

// 	cliChan, srvChan, err := setUpSecureChannel(ctx)
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	if err := cliChan.FindServersRequest([]string{"ja-JP"}, "server"); err != nil {
// 		t.Fatal(err)
// 	}

// 	msg, err := srvChan.ReadService()
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	if _, ok := msg.(*services.FindServersRequest); !ok {
// 		t.Error("failed to assert type")
// 	}
// }

// func TestFindServerResponse(t *testing.T) {
// 	ctx := context.Background()
// 	ctx, cancel := context.WithCancel(ctx)
// 	defer cancel()

// 	cliChan, srvChan, err := setUpSecureChannel(ctx)
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	if err := srvChan.FindServersResponse(
// 		0, services.NewApplicationDescription(
// 			"", "", "", services.AppTypeServer, "", "", []string{""},
// 		),
// 	); err != nil {
// 		t.Fatal(err)
// 	}

// 	msg, err := cliChan.ReadService()
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	if _, ok := msg.(*services.FindServersResponse); !ok {
// 		t.Error("failed to assert type")
// 	}
// }

// func TestLocalEndpoint(t *testing.T) {
// 	ctx := context.Background()
// 	ctx, cancel := context.WithCancel(ctx)
// 	defer cancel()

// 	_, srvChan, err := setUpSecureChannel(ctx)
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	verify.Values(t, "", srvChan.LocalEndpoint(), endpoint)
// }

// func TestRemoteEndpoint(t *testing.T) {
// 	ctx := context.Background()
// 	ctx, cancel := context.WithCancel(ctx)
// 	defer cancel()

// 	cliChan, _, err := setUpSecureChannel(ctx)
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	verify.Values(t, "", cliChan.RemoteEndpoint(), endpoint)
// }
