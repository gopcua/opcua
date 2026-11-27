//go:build conformance

// A CreateSession response whose ServerSignature does not verify fails the
// Connect: the client returns an error naming the session signature check
// and never sends an ActivateSession request.
//
// The clause: OPC 10000-4 §6.1.8: "The ClientSignature and UserTokenSignature
// shall not be computed until the ServerSignature is validated." A client
// that computed them anyway would offer the server a ClientSignature for a
// session whose ServerSignature it never accepted.
//
// Neighbouring rows: empty_token_policy_means_the_channels_policy_test.go
// and legacy_first_username_policies_use_the_endpoints_own_test.go score the
// UserIdentityToken of the ActivateSession request this row shows is never
// sent when the ServerSignature does not verify.
//
// The row runs against the in-tree server's Basic256Sha256 SignAndEncrypt
// endpoint, with the server's CreateSession handler replaced by one the row
// owns, which answers with the signature the secure channel would sign, one
// byte flipped, so the response is wrong in its signature and nowhere
// else. See harness.go.
//
// The row asserts that the tampered response was served, that Connect
// returns an error naming the session signature check, and that no
// ActivateSession request arrives within afterConnectBound of Connect
// returning. It asserts nothing about the error's wording beyond the words
// "session signature", nor about the channel after the refusal.
//
// gopcua passes this row: CreateSession returns the verification error and
// Connect hands it back (client.go:860-866). Before that, the error was
// only logged and CreateSession returned no session, which left Connect to
// read one that did not exist. No reference stack was read for this row;
// the tampered response is the row's own, not a stack's output.
package session

import (
	"context"
	"crypto/rand"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/id"
	"github.com/gopcua/opcua/server"
	"github.com/gopcua/opcua/ua"
	"github.com/gopcua/opcua/uasc"
)

func TestUnverifiedServerSignatureFailsConnect(t *testing.T) {
	created := make(chan *ua.CreateSessionRequest, 1)
	s, teardown, why := startSessionServer(func(s *sessionServer, srv *server.Server) {
		srv.RegisterHandler(id.CreateSessionRequest_Encoding_DefaultBinary,
			tamperedCreateSession(s.cert, created))
	})
	t.Cleanup(teardown)
	if why != "" {
		t.Fatalf("could not set the test up: %s", why)
	}

	ep, why, fromGopcua := s.endpoint(ua.SecurityPolicyURIBasic256Sha256, ua.MessageSecurityModeSignAndEncrypt)
	if why != "" {
		if fromGopcua {
			t.Errorf("gopcua's own behaviour made this run unscoreable: %s", why)
		} else {
			t.Fatalf("could not set the test up: %s", why)
		}
		return
	}

	waitConnect, why, fromGopcua := launchAnonymousClient(s, ep)
	if why != "" {
		if fromGopcua {
			t.Errorf("gopcua's own behaviour made this run unscoreable: %s", why)
		} else {
			t.Fatalf("could not set the test up: %s", why)
		}
		return
	}

	connectErr, returned := waitConnect()
	if !returned {
		t.Fatalf("could not set the test up: Connect did not return within %v", connectBound)
	}

	// The refusal can only be scored if the tampered response was served;
	// otherwise Connect failed before the signature was ever checked.
	select {
	case <-created:
	default:
		t.Fatalf("could not set the test up: the server never served the tampered CreateSession response")
	}

	if connectErr == nil {
		t.Errorf("Connect succeeded against a server whose ServerSignature does not verify")
	} else if !strings.Contains(connectErr.Error(), "session signature") {
		t.Errorf("Connect failed with %q, which does not name the session signature check", connectErr)
	}

	// §6.1.8: no ClientSignature may be computed until the ServerSignature
	// is validated, so no ActivateSession request may exist once Connect
	// has returned. The bound only covers a request already in flight.
	select {
	case req := <-s.activate:
		t.Errorf("an ActivateSession request was sent despite the unverified ServerSignature: %v", req)
	case <-time.After(afterConnectBound):
	}

	if t.Failed() {
		return
	}
	t.Logf("Connect refused the tampered ServerSignature with %q", connectErr)
}

// launchAnonymousClient builds a client with SecurityFromEndpoint(ep) for
// the Anonymous token and runs its Connect in the background, under a
// context bounded by connectBound. waitConnect blocks until Connect returns
// and reports its error, or reports that Connect is still running
// connectBound after the call; a row calls it before tearing the server
// down. why is "" or the reason the client could not be built, and
// fromGopcua reports whether opcua.NewClient, not the fixture, failed.
func launchAnonymousClient(s *sessionServer, ep *ua.EndpointDescription) (waitConnect func() (error, bool), why string, fromGopcua bool) {
	key, cert, why := mintCert("gopcua-conformance-session-client")
	if why != "" {
		return func() (error, bool) { return nil, true }, why, false
	}

	client, err := opcua.NewClient(s.url,
		opcua.PrivateKey(key),
		opcua.Certificate(cert),
		opcua.SecurityFromEndpoint(ep, ua.UserTokenTypeAnonymous),
		opcua.AutoReconnect(false),
	)
	if err != nil {
		return func() (error, bool) { return nil, true }, fmt.Sprintf("build the client: %v", err), true
	}

	ctx, cancel := context.WithTimeout(context.Background(), connectBound)
	connectErr := make(chan error, 1)
	go func() {
		connectErr <- client.Connect(ctx)
		_ = client.Close(context.Background())
		cancel()
	}()
	return func() (error, bool) {
		select {
		case err := <-connectErr:
			return err, true
		case <-time.After(connectBound):
			return nil, false
		}
	}, "", false
}

// tamperedCreateSession answers a CreateSession request with the signature
// the secure channel would sign, with one byte flipped, so the response is
// wrong in its signature and nowhere else. It records the request it
// answered.
func tamperedCreateSession(serverCertificate []byte, recorded chan<- *ua.CreateSessionRequest) server.Handler {
	return func(sc *uasc.SecureChannel, r ua.Request, reqID uint32) (ua.Response, error) {
		req, ok := r.(*ua.CreateSessionRequest)
		if !ok {
			return nil, ua.StatusBadRequestTypeInvalid
		}
		select {
		case recorded <- req:
		default:
		}

		sig, alg, err := sc.NewSessionSignature(req.ClientCertificate, req.ClientNonce)
		if err != nil {
			return nil, ua.StatusBadInternalError
		}
		if len(sig) == 0 {
			return nil, ua.StatusBadInternalError
		}
		sig[0] ^= 0xff

		nonce := make([]byte, serverNonceLength)
		if _, err := rand.Read(nonce); err != nil {
			return nil, ua.StatusBadInternalError
		}
		return &ua.CreateSessionResponse{
			ResponseHeader: &ua.ResponseHeader{
				Timestamp:     time.Now(),
				RequestHandle: req.RequestHeader.RequestHandle,
				ServiceResult: ua.StatusOK,
			},
			SessionID:             ua.NewNumericNodeID(1, 1),
			AuthenticationToken:   ua.NewNumericNodeID(1, 2),
			RevisedSessionTimeout: 60000,
			ServerNonce:           nonce,
			ServerCertificate:     serverCertificate,
			ServerSignature: &ua.SignatureData{
				Algorithm: alg,
				Signature: sig,
			},
		}, nil
	}
}
