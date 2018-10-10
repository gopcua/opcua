// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uasc

/*
func TestSession(t *testing.T) {
	ep := "opc.tcp://127.0.0.1:4840/foo/bar"
	policyURI := "http://opcfoundation.org/UA/SecurityPolicy#None"
	ln, err := uacp.Listen(ep, 0xffff)
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	cfg := NewConfig(
		1, services.SecModeNone, policyURI, nil, nil, 1000, 0, 1,
	)

	secChanOK := make(chan bool)
	sessionOK := make(chan bool)
	go func() {
		defer ln.Close()
		srvConn, err := ln.Accept(ctx)
		if err != nil {
			t.Fatal(err)
		}

		secChan, err := ListenAndAcceptSecureChannel(ctx, srvConn, cfg)
		if err != nil {
			t.Fatal(err)
		}
		secChanOK <- true

		svrCfg := NewSessionConfigServer(
			secChan,
			services.NewSignatureData("", nil),
			[]*services.SignedSoftwareCertificate{
				services.NewSignedSoftwareCertificate(nil, nil),
			},
		)
		if _, err := ListenAndAcceptSession(ctx, secChan, svrCfg); err != nil {
			t.Error(err)
		}
		sessionOK <- true
	}()

	cliConn, err := uacp.Dial(ctx, ep)
	if err != nil {
		t.Fatal(err)
	}

	secChan, err := OpenSecureChannel(ctx, cliConn, cfg)
	if err != nil {
		t.Fatal(err)
	}

	select {
	case _, ok := <-secChanOK:
		if !ok {
			t.Fatalf("timed out")
		}
	case <-time.After(10 * time.Second):
		t.Fatalf("timed out")
	}

	cliCfg := NewSessionConfigClient(
		[]string{"ja-JP"},
		datatypes.NewAnonymousIdentityToken("anonymous"),
	)
	session, err := CreateSession(ctx, secChan, cliCfg, 3, 5*time.Second)
	if err != nil {
		t.Error(err)
	}

	if err := session.Activate(); err != nil {
		t.Error(err)
	}

	select {
	case _, ok := <-sessionOK:
		if !ok {
			t.Fatalf("timed out")
		}
	case <-time.After(10 * time.Second):
		t.Fatalf("timed out")
	}
}
*/
