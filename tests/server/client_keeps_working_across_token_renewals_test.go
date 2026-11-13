//go:build conformance

// A client renews its security token before it expires, and the channel
// stays usable through the renewals and the expiry of the old tokens: no
// request the application sends in that time may fail.
//
// The clauses: OPC 10000-4 §5.6.2.2, revisedLifetime: "The revised
// lifetime shall be used by the Client to renew a SecureChannel before it
// expires even if the MessageSecurityMode is NONE." OPC 10000-4 §5.6.2.1:
// "Clients should request a new SecurityToken after 75 % of its lifetime
// has elapsed." and "Clients should accept Messages secured by an
// expired SecurityToken for up to 25 % of the token lifetime." OPC
// 10000-6 §6.7.4: "The Client shall continue to accept the old
// SecurityToken until it expires or it receives a Message from the
// Server secured with the new SecurityToken." Together they make a
// channel whose token is short a moving window of validity: the client
// owes a renewal before each expiry, owes the old token its grace while
// the new one takes over, and the request an application sends at any
// instant of that window must still be answered. A channel that drops
// responses once an old token expires - whatever it drops them for -
// breaks §6.7.4's continuation duty and every clause above with it.
//
// Neighbouring rows: read_while_writing_is_race_free_test.go,
// setvalue_on_a_subscribed_tag_returns_test.go and
// setattribute_on_a_subscribed_tag_returns_test.go score properties of
// the server's own call graph within one token's life, where this row
// scores the pair - client and server - across the life and death of
// several; ../securechannel's rows score the client's single-token
// obligations against a scripted peer, where this row runs both real
// halves, because the regression it exists to catch lives in the
// meeting: gopcua's server answers each renewal with the same TokenId
// (uasc/secure_channel.go:867, the response carries the token id
// the channel was created with, and a renewal does not change it), and
// whether the client's own expiry of the old token then takes the
// renewed one with it is a question no scripted peer answers - the
// answer is the real server's, and only a run against it can hear it.
// The suite had no such row: nothing ran a real gopcua client against a
// real gopcua server through a renewal and an expiry, which is how a
// fix that passed every other row and every unit test could still break
// the pair.
//
// The row runs against the in-tree gopcua server and a real gopcua
// client connected over the wire - the shared fixture's shape in
// harness.go, but started by the row's own runArm, because the shared
// startServerAndClient fixes a token lifetime this row must choose and
// a security policy it must vary: the server comes up on an ephemeral
// port via the harness's freePort, the client is launched with
// opcua.Lifetime and opcua.RequestTimeout options of the row's own.
// The row runs in three arms that differ only in the security policy
// and mode they speak: SecurityPolicy None, and Basic256Sha256 in mode
// Sign and in mode SignAndEncrypt - the certificates for the two signed
// arms are minted in memory, nothing on disk, by the row's own
// mintArmCert, per the securitypolicy package's mintCert precedent.
// See harness.go.
//
// The row asserts that every Read the arm's client sends in its window
// is answered - no error, a Good ServiceResult - and reports the moment
// into the window, the status and the error text of any that is not; a
// dropped response shows as the Read returning StatusBadTimeout once
// uasc's own bound - the row's RequestTimeout plus a 250ms leniency
// (uasc/secure_channel.go:993-994) - fires - a chunk no instance is
// found for reaches no handler and the request is never answered - or,
// on a tree that closes the channel on the failed lookup, as the closed
// channel's
// EOF, and the row reports the costume without ever keying its verdict
// on which one it sees. It deliberately asserts nothing about the
// values the reads carry: both halves are
// gopcua, so the row scores liveness of the pair and not bytes, and the
// client-side obligations behind it are held row by row in
// ../securechannel. It declines, and does not score, when the
// environment fails its half - no free port, a certificate that
// would not mint - and scores as gopcua's failure a pair whose own
// halves could not come up, whether the server would not start or the
// client would not connect, because both of them are the library
// under test. The row passes on
// main as it stands; it carries no knownBug constant, and a run that
// fails on an unfixed tree is a finding, not a decline.
//
// No reference stack has been read for this behaviour; the in-tree
// server is the reference the row runs against, and the measurement on
// main and under the expiry-regression mutant is the row's own.
package server

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/server"
	"github.com/gopcua/opcua/ua"
)

const (
	// tokenLifetime is the token lifetime the row's clients ask for, and
	// the in-tree server echoes it back unchanged
	// (handleOpenSecureChannelRequest answers RevisedLifetime with the
	// RequestedLifetime it got, uasc/secure_channel.go:869), so the
	// client's own scheduleRenewal and scheduleExpiration timers - 0.75
	// and 1.25 of the lifetime (uasc/secure_channel.go:903-904,
	// uasc/secure_channel.go:935-936) - decide the schedule the row must
	// outlast. Those lines convert the product with time.Second *
	// time.Duration(seconds * factor), and the float-to-Duration
	// conversion truncates to whole seconds: the renewal timer is 2s and
	// the expiry timer 3s for this lifetime, so renewals arrive at 2s,
	// 4s, 6s... and each token expires 3s after its receipt, at 3s, 5s,
	// 7s... The lifetime cannot be shorter: at 2s the renewal timer
	// truncates to 1s and the expiry timer to 2s, so every expiry lands
	// on the same instant as a renewal and the gap the row needs - an
	// expiry whose bucket wipe is not instantly repopulated by the next
	// renewal's answer - disappears; below 1.34s the renewal timer
	// collapses to zero and the client renews in a tight loop forever,
	// which is not the schedule the clause describes.
	tokenLifetime = 3000 * time.Millisecond

	// armWindow is how long each arm keeps issuing Reads once its
	// session is live. It spans the first two renewals (2s, 4s after the
	// first token's receipt) and the first two expiries (3s, 5s) and
	// runs past 5s, so the row asks for at least two of each and issues
	// a Read inside the second expiry's grace window too.
	armWindow = 5500 * time.Millisecond

	// readEvery is the interval between the arm's Reads: short enough
	// that no window in the schedule is longer than a handful of
	// intervals, so a channel that stops answering for one cannot slip
	// between two reads and pass.
	readEvery = 250 * time.Millisecond

	// connectBound bounds each arm's Connect: the server is loopback and
	// in-process, so this is not a timing assumption for a healthy pair
	// - it lets a slow machine finish and releases a pair that wedged.
	connectBound = 10 * time.Second

	// readTimeout is the opcua.RequestTimeout the row's clients get,
	// and the only per-read bound: uasc waits for each response for
	// the timeout plus a 250ms leniency
	// (uasc/secure_channel.go:993-994), so a read whose response was
	// dropped fails with StatusBadTimeout after 750ms instead of the
	// ten second default, and a run against a tree that drops
	// responses still reports its failures and ends. The row puts no
	// context bound of its own on a read: one that fired before uasc's
	// would score a response the library's own contract still permits.
	readTimeout = 500 * time.Millisecond
)

// TestClientKeepsWorkingAcrossTokenRenewals scores the row: every Read
// every arm's client sends in its window must be answered. Each arm
// runs in its own subtest, so its teardown runs - deferred, per the
// harness's convention - before the next arm starts. A failure
// line is the verdict, never a decline; the one t.Fatalf below is the
// row's decline, and runArm has already split the reason by which side
// caused it, per the header: the environment's half is a decline, and
// a pair whose own halves could not come up is gopcua's behaviour,
// scored with t.Errorf as its failure.
func TestClientKeepsWorkingAcrossTokenRenewals(t *testing.T) {
	arms := []struct {
		name   string
		policy string
		mode   ua.MessageSecurityMode
		secure bool
	}{
		{name: "SecurityPolicy_None", policy: ua.SecurityPolicyURINone, mode: ua.MessageSecurityModeNone},
		{name: "Basic256Sha256_Sign", policy: ua.SecurityPolicyURIBasic256Sha256, mode: ua.MessageSecurityModeSign, secure: true},
		{name: "Basic256Sha256_SignAndEncrypt", policy: ua.SecurityPolicyURIBasic256Sha256, mode: ua.MessageSecurityModeSignAndEncrypt, secure: true},
	}

	for _, arm := range arms {
		t.Run(arm.name, func(t *testing.T) {
			o, teardown := runArm(arm.policy, arm.mode, arm.secure)
			defer teardown()
			if o.setupWhy != "" {
				if o.setupFromGopcua {
					t.Errorf("gopcua's own behaviour made this run unscoreable: %s", o.setupWhy)
					return
				}
				t.Fatalf("could not set the test up: %s", o.setupWhy)
			}
			for _, r := range o.reads {
				if r.status == ua.StatusOK {
					continue
				}
				t.Errorf("the Read %v into the window failed with %s (0x%08X): %s; "+
					"a request the application sends while the channel renews its "+
					"token and expires the old one may not fail",
					r.at, r.status, uint32(r.status), r.errText)
			}
		})
	}
}

// readOutcome is one Read the arm's client sent: the moment into the
// arm's window it was sent at, the status it came back as - Good, or
// the ServiceResult the server answered with, or the StatusCode the
// error carried when no answer arrived at all - and the failure's
// own text, which the row reports beside the status and keys nothing
// on; the header names the costumes a dropped response shows under.
type readOutcome struct {
	at      time.Duration
	status  ua.StatusCode
	errText string
}

// armOutcome is what one arm's run left within the row's reach: the
// reads it sent with their outcomes, and the reason the run cannot be
// scored - "" when it can - with which side caused it, per the header's
// split: the environment's half is a decline, and a pair whose own
// halves could not come up is gopcua's behaviour, scored as its failure.
type armOutcome struct {
	reads           []readOutcome
	setupWhy        string
	setupFromGopcua bool
}

// runArm runs one arm end to end: it brings the in-tree server up on an
// ephemeral port - under the arm's security policy, minting the
// certificate pair in memory when the arm's policy is not None - and
// connects a real gopcua client to it over the wire, launched with the
// row's token lifetime and request timeout. It returns what the arm's
// run left to score and a teardown for everything that came up; it
// returns plain values, and the row turns a non-empty reason into the
// failure or the decline. A stage that fails leaves the harness partly
// up: teardown is non-nil for whatever came up, and the row still owns
// calling it.
func runArm(policy string, mode ua.MessageSecurityMode, secure bool) (o armOutcome, teardown func()) {
	o = armOutcome{reads: []readOutcome{}}
	teardown = func() {}

	port, why := freePort()
	if why != "" {
		o.setupWhy = why
		return o, teardown
	}

	opts := []server.Option{
		server.EnableSecurity("None", ua.MessageSecurityModeNone),
		server.EnableAuthMode(ua.UserTokenTypeAnonymous),
		server.EndPoint("127.0.0.1", port),
	}
	clientOpts := []opcua.Option{
		opcua.SecurityPolicy(policy),
		opcua.SecurityMode(mode),
		opcua.Lifetime(tokenLifetime),
		opcua.RequestTimeout(readTimeout),
	}
	if secure {
		serverKey, serverCert, why := mintArmCert("gopcua-conformance-renewal-server")
		if why != "" {
			o.setupWhy = why
			return o, teardown
		}
		clientKey, clientCert, why := mintArmCert("gopcua-conformance-renewal-client")
		if why != "" {
			o.setupWhy = why
			return o, teardown
		}
		opts = append(opts,
			server.Certificate(serverCert),
			server.PrivateKey(serverKey),
			server.EnableSecurity("Basic256Sha256", ua.MessageSecurityModeSign),
			server.EnableSecurity("Basic256Sha256", ua.MessageSecurityModeSignAndEncrypt),
		)
		clientOpts = append(clientOpts,
			opcua.PrivateKey(clientKey),
			opcua.Certificate(clientCert),
			opcua.RemoteCertificate(serverCert),
		)
	}

	srv := server.New(opts...)
	ns := server.NewMapNamespace(srv, nsName)

	ctx := context.Background()
	if err := srv.Start(ctx); err != nil {
		// The server is as much the library under test as the client, so
		// a start failure is gopcua's behaviour, per the header's split.
		o.setupWhy = fmt.Sprintf("server start: %v", err)
		o.setupFromGopcua = true
		return o, teardown
	}
	teardown = func() { _ = srv.Close() }

	// Seeded after srv.Start for the reason the shared fixture gives
	// (harness.go).
	ns.SetValue(tag, int32(0))

	endpoint := fmt.Sprintf("opc.tcp://127.0.0.1:%d", port)
	client, err := opcua.NewClient(endpoint, clientOpts...)
	if err != nil {
		o.setupWhy = fmt.Sprintf("build the client: %v", err)
		o.setupFromGopcua = true
		return o, teardown
	}
	teardown = func() {
		_ = client.Close(ctx)
		_ = srv.Close()
	}

	connectCtx, cancel := context.WithTimeout(context.Background(), connectBound)
	defer cancel()
	if err := client.Connect(connectCtx); err != nil {
		o.setupWhy = fmt.Sprintf("connect to %s: %v", endpoint, err)
		o.setupFromGopcua = true
		return o, teardown
	}

	start := time.Now()
	node := ua.NewStringNodeID(ns.ID(), tag)
	for time.Since(start) < armWindow {
		at := time.Since(start)
		// No context bound of the row's own: RequestTimeout plus uasc's
		// leniency (uasc/secure_channel.go:993-994) is the bound each
		// Read already carries, and a shorter one would score responses
		// the library's own contract permits.
		res, err := client.Read(context.Background(), &ua.ReadRequest{
			NodesToRead: []*ua.ReadValueID{{
				NodeID:      node,
				AttributeID: ua.AttributeIDValue,
			}},
		})
		r := readOutcomeOf(err, res)
		r.at = at
		o.reads = append(o.reads, r)
		time.Sleep(readEvery)
	}
	return o, teardown
}

// readOutcomeOf is one Read's outcome: the status it came back as -
// the ServiceResult the server answered with, or, when no answer
// arrived, the StatusCode the error carried, StatusBadTimeout once
// uasc's RequestTimeout-plus-leniency bound fires - and the failure's
// own text, the error the Read returned or the ServiceResult the
// server answered with. It returns a plain value; the row turns it
// into its score.
func readOutcomeOf(err error, res *ua.ReadResponse) readOutcome {
	if err != nil {
		status := ua.StatusBadUnexpectedError
		var code ua.StatusCode
		if errors.As(err, &code) {
			status = code
		}
		return readOutcome{status: status, errText: err.Error()}
	}
	if res == nil || res.ResponseHeader == nil {
		return readOutcome{
			status:  ua.StatusBadUnexpectedError,
			errText: "the Read returned neither a response nor an error",
		}
	}
	return readOutcome{
		status:  res.ResponseHeader.ServiceResult,
		errText: fmt.Sprintf("the server answered with %s", res.ResponseHeader.ServiceResult),
	}
}

// mintArmCert mints one self-signed RSA key and X.509 certificate in
// memory, for the common name it is given; the signed arms mint the
// server's and the client's pair with it, nothing on disk, and every
// run mints its own. It is the same shape as the securitypolicy
// package's mintCert, held here because no second row calls it. It
// returns the key, the DER certificate, and "" - or, when the minting
// failed, the reason the run cannot be scored: a minting failure comes
// from the environment's random source, not from gopcua, so it is a
// decline.
func mintArmCert(commonName string) (key *rsa.PrivateKey, cert []byte, why string) {
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
