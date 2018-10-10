// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uasc

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/wmnsk/gopcua/errors"
	"github.com/wmnsk/gopcua/services"
	"github.com/wmnsk/gopcua/uacp"
)

var (
	endpoint  = "opc.tcp://127.0.0.1:4840/foo/bar"
	policyURI = "http://opcfoundation.org/UA/SecurityPolicy#None"
	cliCfg    = NewClientConfig(policyURI, nil, nil, 3333, services.SecModeNone, 3600000)
	srvCfg    = NewServerConfig(policyURI, nil, nil, 1111, services.SecModeNone, 2222, 3600000)
	msg       = []byte{0xde, 0xad, 0xbe, 0xef}
)

func setUpSecureChannel(ctx context.Context) (*SecureChannel, *SecureChannel, error) {

	ln, err := uacp.Listen(endpoint, 0xffff)
	if err != nil {
		return nil, nil, err
	}
	defer ln.Close()

	srvChanChan := make(chan *SecureChannel)
	errChan := make(chan error)
	go func() {
		defer ln.Close()
		srvConn, err := ln.Accept(ctx)
		if err != nil {
			errChan <- err
		}

		srvChan, err := ListenAndAcceptSecureChannel(ctx, srvConn, srvCfg)
		if err != nil {
			errChan <- err
		}
		srvChanChan <- srvChan
	}()

	cliConn, err := uacp.Dial(ctx, endpoint)
	if err != nil {
		return nil, nil, err
	}

	cliChan, err := OpenSecureChannel(ctx, cliConn, cliCfg, 5*time.Second, 3)
	if err != nil {
		return nil, nil, err
	}

	select {
	case srvChan := <-srvChanChan:
		return cliChan, srvChan, nil
	case err := <-errChan:
		return nil, nil, err
	case <-time.After(10 * time.Second):
		return nil, nil, errors.New("timed out")
	}
}

func TestClientWrite(t *testing.T) {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	cliChan, srvChan, err := setUpSecureChannel(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := cliChan.Write(msg); err != nil {
		t.Fatal(err)
	}

	buf := make([]byte, 1024)
	n, err := srvChan.Read(buf)
	if err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(buf[:n], msg); diff != "" {
		t.Error(diff)
	}
}

func TestServerWrite(t *testing.T) {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	cliChan, srvChan, err := setUpSecureChannel(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := srvChan.Write(msg); err != nil {
		t.Fatal(err)
	}

	buf := make([]byte, 1024)
	n, err := cliChan.Read(buf)
	if err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(buf[:n], msg); diff != "" {
		t.Error(diff)
	}
}

func TestGetEndpointRequest(t *testing.T) {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	cliChan, srvChan, err := setUpSecureChannel(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if err := cliChan.GetEndpointsRequest([]string{"ja-JP"}, []string{"uri"}); err != nil {
		t.Fatal(err)
	}

	buf := make([]byte, 1024)
	n, err := srvChan.ReadService(buf)
	if err != nil {
		t.Fatal(err)
	}
	msg, err := services.Decode(buf[:n])
	if err != nil {
		t.Fatal(err)
	}

	if _, ok := msg.(*services.GetEndpointsRequest); !ok {
		t.Error("failed to assert type")
	}
}

func TestGetEndpointResponse(t *testing.T) {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	cliChan, srvChan, err := setUpSecureChannel(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if err := srvChan.GetEndpointsResponse(0, services.NewEndpointDescription(
		srvChan.LocalEndpoint(), services.NewApplicationDescription(
			"", "", "", services.AppTypeServer, "", "", []string{""},
		), nil, services.SecModeNone, policyURI, services.NewUserTokenPolicyArray(
			[]*services.UserTokenPolicy{
				services.NewUserTokenPolicy("id", 0, "", "", ""),
			},
		), "", 0,
	)); err != nil {
		t.Fatal(err)
	}

	buf := make([]byte, 1024)
	n, err := cliChan.ReadService(buf)
	if err != nil {
		t.Fatal(err)
	}
	msg, err := services.Decode(buf[:n])
	if err != nil {
		t.Fatal(err)
	}

	if _, ok := msg.(*services.GetEndpointsResponse); !ok {
		t.Error("failed to assert type")
	}
}

func TestFindServerRequest(t *testing.T) {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	cliChan, srvChan, err := setUpSecureChannel(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if err := cliChan.FindServersRequest([]string{"ja-JP"}, "server"); err != nil {
		t.Fatal(err)
	}

	buf := make([]byte, 1024)
	n, err := srvChan.ReadService(buf)
	if err != nil {
		t.Fatal(err)
	}
	msg, err := services.Decode(buf[:n])
	if err != nil {
		t.Fatal(err)
	}

	if _, ok := msg.(*services.FindServersRequest); !ok {
		t.Error("failed to assert type")
	}
}

func TestFindServerResponse(t *testing.T) {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	cliChan, srvChan, err := setUpSecureChannel(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if err := srvChan.FindServersResponse(
		0, services.NewApplicationDescription(
			"", "", "", services.AppTypeServer, "", "", []string{""},
		),
	); err != nil {
		t.Fatal(err)
	}

	buf := make([]byte, 1024)
	n, err := cliChan.ReadService(buf)
	if err != nil {
		t.Fatal(err)
	}
	msg, err := services.Decode(buf[:n])
	if err != nil {
		t.Fatal(err)
	}

	if _, ok := msg.(*services.FindServersResponse); !ok {
		t.Error("failed to assert type")
	}
}

func TestLocalEndpoint(t *testing.T) {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	_, srvChan, err := setUpSecureChannel(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(srvChan.LocalEndpoint(), endpoint); diff != "" {
		t.Error(diff)
	}
}

func TestRemoteEndpoint(t *testing.T) {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	cliChan, _, err := setUpSecureChannel(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(cliChan.RemoteEndpoint(), endpoint); diff != "" {
		t.Error(diff)
	}
}
