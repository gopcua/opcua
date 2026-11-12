package userstore_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"fmt"
	"math/big"
	"net"
	"net/url"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/contrib/userstore"
	"github.com/gopcua/opcua/id"
	"github.com/gopcua/opcua/server"
	"github.com/gopcua/opcua/ua"
)

func freePort(t *testing.T) int {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

func genCert(t *testing.T, uri string) ([]byte, *rsa.PrivateKey) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(time.Now().UnixNano()),
		Subject:               pkix.Name{CommonName: uri},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment | x509.KeyUsageDataEncipherment | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{"localhost"},
	}
	u, err := url.Parse(uri)
	require.NoError(t, err)
	tmpl.URIs = []*url.URL{u}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	require.NoError(t, err)
	return der, key
}

type e2e struct {
	addr     string
	ep       *ua.EndpointDescription
	cliCert  []byte
	cliKey   *rsa.PrivateKey
	usersYML string
	store    *userstore.Store
}

func startServer(t *testing.T, wrap func(server.UserNameAuthenticator) server.UserNameAuthenticator) *e2e {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "users.yaml")
	h, err := userstore.HashPassword("s3cret", bcrypt.MinCost)
	require.NoError(t, err)
	require.NoError(t, userstore.Update(path, func(f *userstore.File) error {
		f.Users["alice"] = &userstore.User{Password: h, Roles: []string{"operator"}}
		return nil
	}))

	store, err := userstore.Open(path, userstore.Options{ReloadInterval: 20 * time.Millisecond, Logf: t.Logf})
	require.NoError(t, err)
	t.Cleanup(func() { store.Close() })

	auth := store.UserNameAuthenticator()
	if wrap != nil {
		auth = wrap(auth)
	}

	port := freePort(t)
	srvCert, srvKey := genCert(t, "urn:gopcua:userstore:server")
	cliCert, cliKey := genCert(t, "urn:gopcua:userstore:client")
	srv := server.New(
		server.EndPoint("localhost", port),
		server.PrivateKey(srvKey),
		server.Certificate(srvCert),
		server.EnableSecurity("Basic256Sha256", ua.MessageSecurityModeSignAndEncrypt),
		server.UserNameAuth(auth),
		server.X509Auth(store.X509Authenticator()),
	)
	ns := server.NewNodeNameSpace(srv, "e2e")
	srv.AddNamespace(ns)
	root, _ := srv.Namespace(0)
	root.Objects().AddRef(ns.Objects(), id.HasComponent, true)
	n := ns.AddNewVariableStringNode("v", int32(7))
	ns.Objects().AddRef(n, id.HasComponent, true)
	require.NoError(t, srv.Start(context.Background()))
	t.Cleanup(func() { srv.Close() })

	addr := fmt.Sprintf("opc.tcp://localhost:%d", port)
	var eps []*ua.EndpointDescription
	require.Eventually(t, func() bool {
		eps, err = opcua.GetEndpoints(context.Background(), addr)
		return err == nil
	}, 10*time.Second, 50*time.Millisecond)
	var ep *ua.EndpointDescription
	for _, e := range eps {
		if e.SecurityMode == ua.MessageSecurityModeSignAndEncrypt {
			ep = e
		}
	}
	require.NotNil(t, ep)
	return &e2e{addr: addr, ep: ep, cliCert: cliCert, cliKey: cliKey, usersYML: path, store: store}
}

func (e *e2e) login(t *testing.T, user, pass string) error {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	c, err := opcua.NewClient(e.addr,
		opcua.SecurityFromEndpoint(e.ep, ua.UserTokenTypeUserName),
		opcua.Certificate(e.cliCert),
		opcua.PrivateKey(e.cliKey),
		opcua.AuthUsername(user, pass),
	)
	if err != nil {
		return err
	}
	if err := c.Connect(ctx); err != nil {
		return err
	}
	defer c.Close(ctx)
	_, err = c.Node(ua.NewStringNodeID(1, "v")).Value(ctx)
	return err
}

func TestServerWithUsersFile(t *testing.T) {
	e := startServer(t, nil)

	require.NoError(t, e.login(t, "alice", "s3cret"))
	require.ErrorIs(t, e.login(t, "alice", "wrong"), ua.StatusBadUserAccessDenied)
	require.ErrorIs(t, e.login(t, "bob", "pw"), ua.StatusBadUserAccessDenied)

	// add bob while the server is running
	h, err := userstore.HashPassword("pw", bcrypt.MinCost)
	require.NoError(t, err)
	require.NoError(t, userstore.Update(e.usersYML, func(f *userstore.File) error {
		f.Users["bob"] = &userstore.User{Password: h}
		return nil
	}))
	require.Eventually(t, func() bool { return e.store.User("bob") != nil }, 3*time.Second, 20*time.Millisecond)
	require.NoError(t, e.login(t, "bob", "pw"))

	// disable alice: her next login fails, even though it was cached
	require.NoError(t, userstore.Update(e.usersYML, func(f *userstore.File) error {
		f.Users["alice"].Disabled = true
		return nil
	}))
	require.Eventually(t, func() bool { return e.store.User("alice").Disabled }, 3*time.Second, 20*time.Millisecond)
	require.ErrorIs(t, e.login(t, "alice", "s3cret"), ua.StatusBadUserAccessDenied)
}

// TestSlowLoginDoesNotBlockServer checks that a slow authenticator (bcrypt at
// a high cost, LDAP, ...) does not stall the requests of other clients.
func TestSlowLoginDoesNotBlockServer(t *testing.T) {
	release := make(chan struct{})
	var slowCalls atomic.Int32
	e := startServer(t, func(next server.UserNameAuthenticator) server.UserNameAuthenticator {
		return func(user, pass string) error {
			if user == "slow" {
				slowCalls.Add(1)
				<-release
				return errors.New("denied")
			}
			return next(user, pass)
		}
	})
	defer func() {
		select {
		case <-release:
		default:
			close(release)
		}
	}()

	slowDone := make(chan error, 1)
	go func() { slowDone <- e.login(t, "slow", "x") }()
	require.Eventually(t, func() bool { return slowCalls.Load() == 1 }, 5*time.Second, 10*time.Millisecond)

	// the slow login is stuck in the authenticator; others must still work
	start := time.Now()
	require.NoError(t, e.login(t, "alice", "s3cret"))
	require.Less(t, time.Since(start), 5*time.Second)

	close(release)
	require.ErrorIs(t, <-slowDone, ua.StatusBadUserAccessDenied)
}
