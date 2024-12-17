//go:build integration

package uatest

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/errors"
	"github.com/stretchr/testify/require"
)

const (
	// this _must_ be a "host" that will silently eat SYNs (no RSTs)
	// 203.0.113.0/24 is IETF TEST-NET-3
	tcpNoRstTestServer   = "opc.tcp://203.0.113.0:4840"
	forceTimeoutDuration = time.Second * 5
)

func TestClientTimeoutViaOptions(t *testing.T) {
	c, err := opcua.NewClient(tcpNoRstTestServer, opcua.DialTimeout(forceTimeoutDuration))
	require.NoError(t, err, "NewClient failed")
	connectAndValidate(t, c, context.Background(), forceTimeoutDuration)
}

func TestClientTimeoutViaContext(t *testing.T) {
	c, err := opcua.NewClient(tcpNoRstTestServer)
	require.NoError(t, err, "NewClient failed")

	ctx, cancel := context.WithTimeout(context.Background(), forceTimeoutDuration)
	defer cancel()

	connectAndValidate(t, c, ctx, forceTimeoutDuration)
}

func connectAndValidate(t *testing.T, c *opcua.Client, ctx context.Context, d time.Duration) {
	start := time.Now()

	err := c.Connect(ctx)
	require.Error(t, err, "Connect should fail")

	elapsed := time.Since(start)

	var oe *net.OpError
	switch {
	case errors.As(err, &oe) && !oe.Timeout():
		require.Fail(t, "got %#v, wanted net.timeoutError", oe.Unwrap())
	case errors.As(err, &oe):
		// ignore
	default:
		require.Fail(t, "got %T, wanted %T", err, &net.OpError{})
	}

	pct := 0.05

	if !within(elapsed, d, pct) {
		require.Fail(t, "took %s, expected %s +/- %v%%", elapsed, d, pct*100)
	}
}

func within(x, y time.Duration, pct float64) bool {
	if pct > 1 || pct < 0 {
		panic("invalid pct")
	}
	p := float64(x) * pct
	return float64(y) >= (float64(x)-p) && float64(y) <= (float64(x)+p)
}
