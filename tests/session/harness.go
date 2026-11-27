//go:build conformance

// Package session holds the gopcua session tests: one behaviour per file,
// with the clause it checks in the file's opening comment. This file holds
// the fixture they all run against; it is not a row.
//
// The fixture is the in-tree gopcua server on an ephemeral port, and a real
// gopcua client connected to it over the wire. The rows score the client's
// end of the session handshake: the UserIdentityToken it puts in its
// ActivateSession request after reading an endpoint's USERNAME
// UserTokenPolicies, and its refusal of a CreateSession response whose
// ServerSignature does not verify. The server lists those
// policies in the order the harness enables its security policies
// (server/server.go:400-435); a row that needs a list the server
// cannot build rewrites only the UserIdentityTokens of the real endpoint
// description.
//
// The in-tree ActivateSession handler ignores the UserIdentityToken, so the
// harness registers its own handler before Start, which RegisterHandler then
// keeps (server/service_handlers.go:96). It records the request and answers
// Good without looking up the session or verifying the client signature.
//
// Every helper here returns plain values. A row's test function turns a
// non-empty reason string into the failure or the decline, and registers the
// teardown, which is non-nil even when a stage failed.
package session

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"math/big"
	"net"
	"time"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/id"
	"github.com/gopcua/opcua/server"
	"github.com/gopcua/opcua/ua"
	"github.com/gopcua/opcua/uasc"
)

// connectBound bounds each stage of the exchange: a Connect, a
// GetEndpoints. A correct pair finishes in well under a second; the bound
// only lets a slow machine finish.
const connectBound = 10 * time.Second

// afterConnectBound is how long a row waits for the recorded ActivateSession
// request once Connect has returned. The handler records the request before
// it answers, so by then the request is already in the channel.
const afterConnectBound = time.Second

// serverNonceLength is the length of the ServerNonce the in-tree server's
// CreateSession sends (sessionNonceLength, server/session_service.go:18).
// The client appends that nonce to the password before encrypting it.
const serverNonceLength = 32

// rowUser and rowPass are the credentials every row's client activates
// with. They are literals so the byte-level row can look for rowPass in the
// decrypted password.
const (
	rowUser = "gopcua-conformance"
	rowPass = "conformance-password"
)

// rsaOAEPURI is the user-token encryption URI of Basic256Sha256, from
// Part 7's profile table as quoted at uapolicy/policyBasic256Sha256.go:38-42.
// It is a literal so the library does not grade itself.
const rsaOAEPURI = "http://www.w3.org/2001/04/xmlenc#rsa-oaep"

// sessionServer is the in-tree server with its ActivateSession handler
// replaced. key and cert are the server's own; a row whose replacement
// CreateSession handler must answer with the server's certificate reads
// cert. activate receives the first ActivateSession request; later ones
// are dropped.
type sessionServer struct {
	url  string
	key  *rsa.PrivateKey
	cert []byte

	activate chan *ua.ActivateSessionRequest
}

// startSessionServer starts the in-tree server on a loopback ephemeral
// port. It enables None, Basic128Rsa15 and Basic256Sha256, in that order, so
// every secured endpoint lists the legacy username_basic128rsa15 policy
// before its own. beforeStart, when given, runs after the ActivateSession
// handler is registered and before Start, so a row can register a handler
// of its own that initHandlers keeps (server/service_handlers.go:96); the
// ServerSignature row replaces CreateSession this way. It returns "" or
// the reason the fixture could not start.
func startSessionServer(beforeStart ...func(s *sessionServer, srv *server.Server)) (*sessionServer, func(), string) {
	port, why := freePort()
	if why != "" {
		return nil, func() {}, why
	}

	key, cert, why := mintCert("gopcua-conformance-session-server")
	if why != "" {
		return nil, func() {}, why
	}

	s := &sessionServer{
		url:      fmt.Sprintf("opc.tcp://127.0.0.1:%d", port),
		key:      key,
		cert:     cert,
		activate: make(chan *ua.ActivateSessionRequest, 1),
	}

	srv := server.New(
		server.Certificate(cert),
		server.PrivateKey(key),
		server.EnableSecurity("None", ua.MessageSecurityModeNone),
		server.EnableSecurity("Basic128Rsa15", ua.MessageSecurityModeSignAndEncrypt),
		server.EnableSecurity("Basic256Sha256", ua.MessageSecurityModeSignAndEncrypt),
		server.EnableAuthMode(ua.UserTokenTypeAnonymous),
		server.EnableAuthMode(ua.UserTokenTypeUserName),
		server.EndPoint("127.0.0.1", port),
	)

	// Registered before Start, so initHandlers keeps it.
	srv.RegisterHandler(id.ActivateSessionRequest_Encoding_DefaultBinary,
		s.recordActivateSession)
	for _, f := range beforeStart {
		f(s, srv)
	}

	if err := srv.Start(context.Background()); err != nil {
		return nil, func() {}, fmt.Sprintf("server start: %v", err)
	}
	return s, func() { _ = srv.Close() }, ""
}

// recordActivateSession records the request and answers it the way the
// in-tree handler answers a valid one: Good, with a fresh ServerNonce
// (server/session_service.go:114-128).
func (s *sessionServer) recordActivateSession(sc *uasc.SecureChannel, r ua.Request, reqID uint32) (ua.Response, error) {
	req, ok := r.(*ua.ActivateSessionRequest)
	if !ok {
		// Unreachable: the registry is keyed by the encoding id.
		return nil, ua.StatusBadRequestTypeInvalid
	}

	select {
	case s.activate <- req:
	default:
	}

	nonce := make([]byte, serverNonceLength)
	if _, err := rand.Read(nonce); err != nil {
		return nil, ua.StatusBadInternalError
	}

	return &ua.ActivateSessionResponse{
		ResponseHeader: &ua.ResponseHeader{
			Timestamp:          time.Now(),
			RequestHandle:      req.RequestHeader.RequestHandle,
			ServiceResult:      ua.StatusOK,
			ServiceDiagnostics: &ua.DiagnosticInfo{},
			StringTable:        []string{},
			AdditionalHeader:   ua.NewExtensionObject(nil),
		},
		ServerNonce: nonce,
	}, nil
}

// awaitActivateSession waits up to within for the recorded ActivateSession
// request. It returns "" or the reason none arrived; the row declines on
// that, since it cannot tell which side failed.
func (s *sessionServer) awaitActivateSession(within time.Duration) (*ua.ActivateSessionRequest, string) {
	select {
	case req := <-s.activate:
		return req, ""
	case <-time.After(within):
		return nil, fmt.Sprintf("no ActivateSession request arrived within %v", within)
	}
}

// endpoint fetches the server's endpoints with GetEndpoints and returns the
// one with the given security policy and mode. It returns "" or the reason it
// failed. fromGopcua is false only when connectBound ran out, which a loaded
// machine can cause; otherwise both halves are gopcua.
func (s *sessionServer) endpoint(policy string, mode ua.MessageSecurityMode) (ep *ua.EndpointDescription, why string, fromGopcua bool) {
	ctx, cancel := context.WithTimeout(context.Background(), connectBound)
	defer cancel()
	eps, err := opcua.GetEndpoints(ctx, s.url)
	if err != nil {
		return nil, fmt.Sprintf("GetEndpoints from %s: %v", s.url, err), ctx.Err() == nil
	}
	for _, ep := range eps {
		if ep.SecurityPolicyURI == policy && ep.SecurityMode == mode {
			return ep, "", false
		}
	}
	return nil, fmt.Sprintf("no endpoint of %s has security policy %q and mode %s",
		s.url, policy, mode), true
}

// withUsernameTokens returns a shallow copy of ep with its
// UserIdentityTokens replaced by tokens.
func withUsernameTokens(ep *ua.EndpointDescription, tokens ...*ua.UserTokenPolicy) *ua.EndpointDescription {
	clone := *ep
	clone.UserIdentityTokens = tokens
	return &clone
}

// usernameToken returns the request's UserNameIdentityToken, or the reason
// there is none. unscoreable reports that the token could not be read at all;
// otherwise the token decoded as another type, which is a plain failure.
func usernameToken(req *ua.ActivateSessionRequest) (tok *ua.UserNameIdentityToken, why string, unscoreable bool) {
	if req.UserIdentityToken == nil {
		return nil, "the ActivateSession request carries no UserIdentityToken at all", true
	}
	tok, ok := req.UserIdentityToken.Value.(*ua.UserNameIdentityToken)
	if ok {
		return tok, "", false
	}
	if req.UserIdentityToken.Value == nil {
		return nil, "the server's decoder left the UserIdentityToken untyped", true
	}
	return nil, fmt.Sprintf("the ActivateSession request carried a %T, "+
		"not the UserNameIdentityToken the client was configured to send",
		req.UserIdentityToken.Value), false
}

// launchClient builds a client with SecurityFromEndpoint(ep) and
// AuthUsername and runs its Connect in the background, under a context
// bounded by connectBound. waitConnect blocks until Connect returns and
// reports false only if Connect is still running connectBound after the
// call; a row calls it before tearing the server down. why is "" or the
// reason the client could not be built, and fromGopcua reports whether
// opcua.NewClient, not the fixture, failed.
func launchClient(s *sessionServer, ep *ua.EndpointDescription, user, pass string) (waitConnect func() bool, why string, fromGopcua bool) {
	key, cert, why := mintCert("gopcua-conformance-session-client")
	if why != "" {
		return func() bool { return true }, why, false
	}

	client, err := opcua.NewClient(s.url,
		opcua.PrivateKey(key),
		opcua.Certificate(cert),
		opcua.SecurityFromEndpoint(ep, ua.UserTokenTypeUserName),
		opcua.AuthUsername(user, pass),
		opcua.AutoReconnect(false),
	)
	if err != nil {
		return func() bool { return true }, fmt.Sprintf("build the client: %v", err), true
	}

	ctx, cancel := context.WithTimeout(context.Background(), connectBound)
	done := make(chan struct{})
	go func() {
		_ = client.Connect(ctx)
		_ = client.Close(context.Background())
		cancel()
		close(done)
	}()
	return func() bool {
		select {
		case <-done:
			return true
		case <-time.After(connectBound):
			return false
		}
	}, "", false
}

// mintCert mints a self-signed 2048-bit RSA key and certificate in memory.
// 2048 bits is the only size both Basic128Rsa15 (at most,
// uapolicy/policyBasic128Rsa15.go:73-74) and Basic256Sha256 (at least,
// uapolicy/policyBasic256Sha256.go:86) accept. It returns "" or the reason
// minting failed.
func mintCert(commonName string) (key *rsa.PrivateKey, cert []byte, why string) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, fmt.Sprintf("generate an RSA key: %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: commonName},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage: x509.KeyUsageDigitalSignature |
			x509.KeyUsageKeyEncipherment | x509.KeyUsageDataEncipherment,
	}
	cert, err = x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		return nil, nil, fmt.Sprintf("create the certificate: %v", err)
	}
	return key, cert, ""
}

// freePort returns an unused loopback TCP port, or the reason there is none.
// It duplicates tests/server's freePort; moving both into tests/internal is
// left to a change that may edit tests/server.
func freePort() (int, string) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, fmt.Sprintf("reserve port: %v", err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	if err := l.Close(); err != nil {
		return 0, fmt.Sprintf("release port: %v", err)
	}
	return port, ""
}
