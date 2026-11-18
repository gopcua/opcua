//go:build conformance

// Package session holds the gopcua session tests: one behaviour per file,
// with the clause it checks in the file's opening comment. This file holds
// the fixture they all run against; it is not a row.
//
// The fixture is the in-tree gopcua server - production code, not a
// scripted one - brought up on an ephemeral port, and a real gopcua client
// connected to it over the wire. The conformance unit behind the rows,
// OPC 10000-7's "Security User Name Password Client" (User Token - User
// Name Password Client Facet), binds the client half: the rows score what
// the client puts in the UserIdentityToken of its ActivateSession request
// after reading the USERNAME UserTokenPolicies an EndpointDescription
// lists. The in-tree server is what presents those descriptions:
// initEndpoints (server/server.go:373) builds each endpoint's
// UserIdentityTokens from the enabled auth modes crossed with the enabled
// security policies, one USERNAME policy per policy, in the order the
// policies were enabled (server/server.go:400-435), so the order this
// harness enables them in is the order the client meets them in. Where a
// row needs an endpoint list that order cannot produce, it rewrites the
// UserIdentityTokens of the endpoint description it hands the client -
// the narrowest surgery that presents the list, because everything else
// about the description (its URL, its certificate, its policy and mode)
// is the real server's own - and its header says so.
//
// The server does not check user credentials at all: the in-tree
// ActivateSession handler reads the UserIdentityToken and does nothing
// with it (server/session_service.go), so the token's PolicyID and
// EncryptionAlgorithm show in one place only - the request itself - and
// the harness wraps the handler to record it.
// (*server.Server).RegisterHandler (server/service_handlers.go:96) only
// writes a handler that is not yet registered, so a handler registered
// before Start is the one that stays: initHandlers' own RegisterHandler
// for ActivateSession runs at Start (server/server.go:250), finds ours
// already there, and leaves it. The wrapper records the request and then
// answers it with the in-tree handler's own answer to a request whose
// session it finds and whose client signature verifies - a Good
// ServiceResult and a fresh 32-byte ServerNonce, the answer the handler
// builds and returns (server/session_service.go:114-128) - and it
// reproduces nothing else of that handler. The in-tree one also
// resolves the request's AuthenticationToken to a session, refusing
// with BadSessionIDInvalid when there is none, and runs
// sc.VerifySessionSignature on the client's signature, refusing with
// BadSecurityChecksFailed when that fails
// (server/session_service.go:101-112); the wrapper looks up nothing and
// verifies nothing, so a client completes its Connect here whatever
// token it sent. Whether Connect succeeded is a symptom the rows do not
// score.
//
// Every helper here returns plain values. A helper does not call
// t.Errorf or t.Fatalf; a row's test function turns a non-empty reason
// string into the failure or the decline, and it, not the fixture,
// registers the teardown with t.Cleanup. A stage that fails leaves the
// harness partly up: teardown is non-nil for whatever came up, and the
// row still owns calling it.
//
// The package name is session, as the directory is; the production
// package github.com/gopcua/opcua/server is a different import path.
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

// connectBound is how long a stage of the exchange gets: a Connect, a
// GetEndpoints. The bound is not a timing assumption: a correct client
// and the loopback server under it finish each stage in well under a
// second, so the bound exists to let a slow machine finish, not to wait
// out a broken one; the teardown releases a pair that never does. The
// wait for the recorded ActivateSession has its own shorter bound,
// afterConnectBound, because it runs only once Connect has returned.
const connectBound = 10 * time.Second

// afterConnectBound is how long a row waits for the recorded
// ActivateSession request once the client's Connect has returned. The
// wrapper records a request before it answers it, and the client's
// Connect does not return before the answer arrives, so a Connect that
// has returned has already put the request in the channel; the grace
// covers the handoff alone and is not a second connectBound's wait for a
// request that can no longer arrive.
const afterConnectBound = time.Second

// serverNonceLength is how many bytes the in-tree server's CreateSession
// puts in the ServerNonce it hands the client (sessionNonceLength,
// server/session_service.go:18), and therefore how many nonce bytes the
// plaintext of a password the client encrypted carries after the password
// itself. The row that reads that plaintext names where the length comes
// from; the number is a literal here and never read from the server
// package, whose constant is unexported.
const serverNonceLength = 32

// rowUser and rowPass are the credentials every row's client activates
// with. The password is the plaintext the byte-level row later looks for
// inside the decrypted Password, so it is a literal here and never chosen
// by the library.
const rowUser = "gopcua-conformance"

// rowPass is the password every row's client activates with; a literal for
// the reason rowUser gives.
const rowPass = "conformance-password"

// rsaOAEPURI is the user-token encryption URI of every RSA-OAEP-SHA1
// policy - Basic256Sha256's above all, the policy the rows' channel runs
// - from Part 7's profile table for Basic256Sha256, which the tree quotes
// at uapolicy/policyBasic256Sha256.go:38-42: "The URI is
// http://www.w3.org/2001/04/xmlenc#rsa-oaep." A literal, never read from
// uapolicy, whose EncryptionURI() would let the library grade itself.
const rsaOAEPURI = "http://www.w3.org/2001/04/xmlenc#rsa-oaep"

// sessionServer is the in-tree server with its ActivateSession handler
// wrapped: the endpoint it listens on, the minted certificate and key it
// signs and decrypts with, and the channel the first ActivateSession
// request arrives on. A request the channel already holds is not
// displaced: the wrapper records the first and drops the rest, because a
// client that sends one Connect sends one activation, and a row that
// needed the second would say so in its header.
type sessionServer struct {
	url  string
	cert []byte // the server's minted DER certificate
	key  *rsa.PrivateKey

	activate chan *ua.ActivateSessionRequest
}

// startSessionServer brings the in-tree server up on the loopback
// interface, on an ephemeral port, with its ActivateSession handler
// wrapped to record the request, and returns it, a teardown for
// everything that came up, and "" - or, when a stage failed, the reason
// the run cannot be scored. Every stage here is the fixture's own half -
// reserving the port, minting the key pair, bringing the server up - so
// every one of its failures is a decline, the same event the server
// package declines for its own harness (tests/server/harness.go returns
// "server start: %v" and its rows print "could not set the test up: %s"
// for it); the caller states that side, and a flag returned here could
// only ever have said it one way.
//
// The security policies are enabled in this order - None, then
// Basic128Rsa15 in mode SignAndEncrypt, then Basic256Sha256 in mode
// SignAndEncrypt - because initEndpoints walks enabledSec twice, once per
// endpoint and once per endpoint's UserIdentityTokens, in that order
// (server/server.go:375 and 401), so the USERNAME policies of every
// secured endpoint list the legacy Basic128Rsa15 first and the endpoint's
// own Basic256Sha256 after it. That is the list the server's own construction
// produces, and it is the list of the interop row the package exists to
// hold; the harness does not reorder anything to get it.
//
// Both auth modes are enabled - Anonymous, then UserName - so the
// endpoint lists carry both kinds of token policy, and the client has a
// USERNAME policy to meet in every endpoint the rows connect to.
func startSessionServer() (*sessionServer, func(), string) {
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
		cert:     cert,
		key:      key,
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

	// Registered before Start, per the package comment: initHandlers'
	// own registration at Start finds this one and leaves it.
	srv.RegisterHandler(id.ActivateSessionRequest_Encoding_DefaultBinary,
		s.recordActivateSession)

	if err := srv.Start(context.Background()); err != nil {
		return nil, func() {}, fmt.Sprintf("server start: %v", err)
	}
	return s, func() { _ = srv.Close() }, ""
}

// recordActivateSession is the wrapped ActivateSession handler: it
// delivers the request to the row that is waiting for it - the first
// one; a later activation finds the channel full and is dropped - and
// answers with the in-tree handler's own answer to a request whose
// session it finds and whose client signature verifies - a Good
// ServiceResult and a fresh nonce (server/session_service.go:114-128).
// It reproduces nothing else of that handler: the in-tree one also
// resolves the request's AuthenticationToken to a session, refusing
// with BadSessionIDInvalid when there is none, and runs
// sc.VerifySessionSignature on the client's signature, refusing with
// BadSecurityChecksFailed when that fails
// (server/session_service.go:101-112); this wrapper does neither. It
// checks nothing about the request's UserIdentityToken, the way the
// in-tree handler does not: the request is the evidence, and the rows
// hold the verdict.
func (s *sessionServer) recordActivateSession(sc *uasc.SecureChannel, r ua.Request, reqID uint32) (ua.Response, error) {
	req, ok := r.(*ua.ActivateSessionRequest)
	if !ok {
		// The handler registry is keyed by the request's encoding id, so
		// only an ActivateSessionRequest can reach here; the answer is
		// the in-tree handler's own to a request it cannot cast
		// (server/session_service.go:96-99).
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

// awaitActivateSession waits up to `within` for the client's ActivateSession
// request to arrive and returns it with "". Otherwise it returns nil and
// the reason the run cannot be scored: a client that never sent one and a
// server that never answered its half land in the same reason, and the
// row declines on it, because a red the row cannot pin on the library is
// not a score.
func (s *sessionServer) awaitActivateSession(within time.Duration) (*ua.ActivateSessionRequest, string) {
	select {
	case req := <-s.activate:
		return req, ""
	case <-time.After(within):
		return nil, fmt.Sprintf("no ActivateSession request arrived within %v", within)
	}
}

// endpoint fetches the server's endpoint descriptions over the wire - a
// real GetEndpoints from a real client, the same discovery an application
// drives before it connects - and returns the one carrying the security
// policy and mode it is asked for, with "" - or, when the exchange failed
// or no endpoint matches, the reason the run cannot be scored, because
// both halves of that exchange are the library under test: the in-tree
// server's list is gopcua's own behaviour, and so is a GetEndpoints that
// would not complete. Every failure here is gopcua's, so the caller
// scores it as gopcua's own behaviour without asking this helper for the
// side, which could only ever have said one thing.
func (s *sessionServer) endpoint(policy string, mode ua.MessageSecurityMode) (*ua.EndpointDescription, string) {
	ctx, cancel := context.WithTimeout(context.Background(), connectBound)
	defer cancel()
	eps, err := opcua.GetEndpoints(ctx, s.url)
	if err != nil {
		return nil, fmt.Sprintf("GetEndpoints from %s: %v", s.url, err)
	}
	for _, ep := range eps {
		if ep.SecurityPolicyURI == policy && ep.SecurityMode == mode {
			return ep, ""
		}
	}
	return nil, fmt.Sprintf("no endpoint of %s carries security policy %q and mode %s; "+
		"the server's own list is what a row's stimulus comes from, and it does not hold "+
		"the one this row needs", s.url, policy, mode)
}

// usernamePolicies is the USERNAME UserTokenPolicies of the endpoint it is
// given, in the order the endpoint lists them. SecurityFromEndpoint walks
// that order (config.go:429), so it is the order the row scores.
func usernamePolicies(ep *ua.EndpointDescription) []*ua.UserTokenPolicy {
	var out []*ua.UserTokenPolicy
	for _, t := range ep.UserIdentityTokens {
		if t.TokenType == ua.UserTokenTypeUserName {
			out = append(out, t)
		}
	}
	return out
}

// withUsernameTokens returns a copy of the endpoint description with its
// UserIdentityTokens replaced by the ones it is given: the narrowest
// rewrite that presents an endpoint list the in-tree server's own
// construction cannot produce, because every other field of the
// description - its URL, its certificate, its security policy and mode -
// stays the real server's own. The copy is shallow and the rows never
// mutate what they are handed, so the caller's endpoint is not touched.
func withUsernameTokens(ep *ua.EndpointDescription, tokens ...*ua.UserTokenPolicy) *ua.EndpointDescription {
	clone := *ep
	clone.UserIdentityTokens = tokens
	return &clone
}

// usernameToken returns the UserNameIdentityToken the recorded
// ActivateSession request carried in its UserIdentityToken, with "" - or,
// when the request carries no such token, the reason and unscoreable,
// which says which kind of not-scoring it is: an ExtensionObject the
// server's decoder left untyped is a request the row cannot read, so the
// run cannot be scored; a token of another type decoded fine and the row
// scores it as the failure it is, because the client was configured to
// send a UserNameIdentityToken and did not. Either way the bytes are the
// client's own; the split only says whether the row can read them.
func usernameToken(req *ua.ActivateSessionRequest) (tok *ua.UserNameIdentityToken, why string, unscoreable bool) {
	if req.UserIdentityToken == nil {
		return nil, "the ActivateSession request carries no UserIdentityToken at all", true
	}
	tok, ok := req.UserIdentityToken.Value.(*ua.UserNameIdentityToken)
	if ok {
		return tok, "", false
	}
	if req.UserIdentityToken.Value == nil {
		return nil, "the UserIdentityToken is an ExtensionObject the server's decoder " +
			"left untyped, so the row cannot read the token the client named", true
	}
	return nil, fmt.Sprintf("the ActivateSession request carried a %T instead of "+
		"the UserNameIdentityToken the client was configured to send, so the "+
		"endpoint's USERNAME policy was never used at all",
		req.UserIdentityToken.Value), false
}

// launchClient builds the client under test and runs its Connect on its
// own goroutine, under a context bounded by connectBound. The row passes
// the endpoint description whose USERNAME policies the client is to read -
// the real one the server returned, or the row's rewrite of it - and the
// username and password it is to activate with; the harness mints the
// client's own certificate pair, adds SecurityFromEndpoint and
// AuthUsername and AutoReconnect(false), so the behaviour the rows score -
// which policy the client picks and how it encrypts the password - is the
// library's own, and a failed connect is a returned error and not a retry
// loop. The goroutine closes the client after Connect returns, so no
// client outlives the row; Connect's own error is a symptom and no row
// asserts it.
//
// It returns waitConnect, which blocks until that goroutine is done -
// bounded by connectBound, so a Connect that ignores its own context
// cannot hold the teardown past the bound - and "" - or, when the client
// could not be built, the reason the run cannot be scored, a wait that
// returns at once, and which side caused it: the certificate this
// function mints is the fixture's own half, so a minting failure comes
// from the environment's random source and is a decline, while
// opcua.NewClient is the library under test and a client it would not
// build is gopcua's own behaviour, which the row scores as unscoreable
// and not a decline. A row waits for the client to finish before it
// tears the server down: the recorded request arrives mid-Connect, and a
// teardown that closes the server underneath a Connect still in flight
// fails the client's own reads - the pair wedge the teardown then waits
// out - for the fixture's timing and not for anything the row scores.
func launchClient(s *sessionServer, ep *ua.EndpointDescription, user, pass string) (waitConnect func(), why string, fromGopcua bool) {
	key, cert, why := mintCert("gopcua-conformance-session-client")
	if why != "" {
		return func() {}, why, false
	}

	client, err := opcua.NewClient(s.url,
		opcua.PrivateKey(key),
		opcua.Certificate(cert),
		opcua.SecurityFromEndpoint(ep, ua.UserTokenTypeUserName),
		opcua.AuthUsername(user, pass),
		opcua.AutoReconnect(false),
	)
	if err != nil {
		return func() {}, fmt.Sprintf("build the client: %v", err), true
	}

	ctx, cancel := context.WithTimeout(context.Background(), connectBound)
	done := make(chan struct{})
	go func() {
		_ = client.Connect(ctx)
		_ = client.Close(context.Background())
		cancel()
		close(done)
	}()
	return func() {
		select {
		case <-done:
		case <-time.After(connectBound):
		}
	}, "", false
}

// mintCert mints one self-signed RSA key and X.509 certificate in memory,
// for the common name it is given: the server's own pair, and each
// client's. Nothing is written to disk and no key material is committed:
// every run mints its own. The key is 2048 bits, and the size is not a
// free choice: Basic128Rsa15's profile caps the asymmetric key at
// 2048 bits (uapolicy/policyBasic128Rsa15.go:73-74), and on a tree where
// the client picks the legacy policy - the interop row's known bug - it
// encrypts the password under that policy with this very certificate, so
// a longer server key would turn the row's red into an unscoreable run;
// Basic256Sha256 requires at least 2048 bits
// (uapolicy/policyBasic256Sha256.go:86), so 2048 is the one size that
// lets every arm run. It returns the key, the DER certificate, and "" -
// or, when the minting failed, the reason the run cannot be scored. A
// minting failure is a decline ("could not set the test up"): it comes
// from the environment's random source, not from gopcua.
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

// freePort asks the kernel for an unused TCP port. The in-tree server's
// EndPoint option wants a number, and a fixed one would collide with any
// other OPC UA work on the same machine - port 4840 above all, which other
// suites bind. It is the same shape as the server package's freePort
// (tests/server/harness.go): the shared contract in ../README.md would
// move a helper two packages call into tests/internal, and this one is
// duplicated instead because moving it means editing tests/server, which
// is not this package's to change; the first change that may touch both
// is the one that moves it. It returns "" or the reason no port could be
// reserved.
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
