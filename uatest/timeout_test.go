//go:build integration
// +build integration

package uatest

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/errors"
)

const (
	// this _must_ be a "host" that will silently eat SYNs (no RSTs)
	// 203.0.113.0/24 is IETF TEST-NET-3
	tcpNoRstTestServer   = "opc.tcp://203.0.113.0:4840"
	forceTimeoutDuration = time.Second * 5
)

func TestClientTimeoutViaOptions(t *testing.T) {
	c, err := opcua.NewClient(tcpNoRstTestServer, opcua.DialTimeout(forceTimeoutDuration))
	if err != nil {
		t.Fatal(err)
	}

	connectAndValidate(t, c, context.Background(), forceTimeoutDuration)
}

func TestClientTimeoutViaContext(t *testing.T) {
	c, err := opcua.NewClient(tcpNoRstTestServer)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), forceTimeoutDuration)
	defer cancel()

	connectAndValidate(t, c, ctx, forceTimeoutDuration)
}

func connectAndValidate(t *testing.T, c *opcua.Client, ctx context.Context, d time.Duration) {
	start := time.Now()

	err := c.Connect(ctx)
	if err == nil {
		t.Fatal("err should not be nil")
	}

	elapsed := time.Since(start)

	var oe *net.OpError
	switch {
	case errors.As(err, &oe) && !oe.Timeout():
		t.Fatalf("got %#v, wanted net.timeoutError", oe.Unwrap())
	case errors.As(err, &oe):
		// ignore
	default:
		t.Fatalf("got %T, wanted %T", err, &net.OpError{})
	}

	pct := 0.05

	if !within(elapsed, d, pct) {
		t.Fatalf("took %s, expected %s +/- %v%%", elapsed, d, pct*100)
	}
}

func within(x, y time.Duration, pct float64) bool {
	if pct > 1 || pct < 0 {
		panic("invalid pct")
	}
	p := float64(x) * pct
	return float64(y) >= (float64(x)-p) && float64(y) <= (float64(x)+p)
}
