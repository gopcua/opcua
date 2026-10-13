package server

import (
	"context"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/ua"
)

// startAcceptTestServer starts a server on a free port and returns its endpoint
// once it answers.
func startAcceptTestServer(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	port := ln.Addr().(*net.TCPAddr).Port
	require.NoError(t, ln.Close())

	srv := New(EndPoint("127.0.0.1", port), EnableSecurity("None", ua.MessageSecurityModeNone))
	ctx, cancel := context.WithCancel(context.Background())
	require.NoError(t, srv.Start(ctx))
	t.Cleanup(func() {
		cancel()
		_ = srv.Close()
	})
	endpoint := srv.URLs()[0]
	requireAnswers(t, endpoint, 5*time.Second)
	return endpoint
}

// requireAnswers fails unless a GetEndpoints request succeeds within budget.
func requireAnswers(t *testing.T, endpoint string, budget time.Duration) {
	t.Helper()
	deadline := time.Now().Add(budget)
	for {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		_, err := opcua.GetEndpoints(ctx, endpoint)
		cancel()
		if err == nil {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("server at %s did not answer within %s: %v", endpoint, budget, err)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func hostPort(endpoint string) string {
	return strings.TrimSuffix(strings.TrimPrefix(endpoint, "opc.tcp://"), "/")
}

// A client reset before its Hello made the handshake fail with a *net.OpError,
// which the accept loop took for a closed listener: the server never accepted
// another connection.
func TestServer_KeepsAcceptingAfterAClientResetsBeforeHello(t *testing.T) {
	endpoint := startAcceptTestServer(t)

	c, err := net.Dial("tcp", hostPort(endpoint))
	require.NoError(t, err)
	require.NoError(t, c.(*net.TCPConn).SetLinger(0)) // close with RST
	require.NoError(t, c.Close())

	requireAnswers(t, endpoint, 3*time.Second)
}

// A client that connected and never sent Hello blocked the handshake, which
// ran inside the accept loop, so no other client could connect meanwhile.
func TestServer_AnswersOthersWhileAClientStaysSilent(t *testing.T) {
	endpoint := startAcceptTestServer(t)

	c, err := net.Dial("tcp", hostPort(endpoint))
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.Close() })
	time.Sleep(100 * time.Millisecond) // let the server accept it first

	requireAnswers(t, endpoint, 3*time.Second)
}
