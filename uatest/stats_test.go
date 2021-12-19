//go:build integration
// +build integration

package uatest

import (
	"context"
	"expvar"
	"testing"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/stats"
	"github.com/pascaldekloe/goe/verify"
)

func newExpVarInt(i int64) *expvar.Int {
	v := &expvar.Int{}
	v.Set(i)
	return v
}

func TestStats(t *testing.T) {
	stats.Reset()

	srv := NewServer("rw_server.py")
	defer srv.Close()

	c := opcua.NewClient(srv.Endpoint, srv.Opts...)
	if err := c.Connect(context.Background()); err != nil {
		t.Fatal(err)
	}

	c.Close()

	expected := map[string]*expvar.Int{
		"Dial":             newExpVarInt(1),
		"ActivateSession":  newExpVarInt(1),
		"NamespaceArray":   newExpVarInt(1),
		"UpdateNamespaces": newExpVarInt(1),
		"NodesToRead":      newExpVarInt(1),
		"Read":             newExpVarInt(1),
		"Send":             newExpVarInt(2),
		"Close":            newExpVarInt(1),
		"CloseSession":     newExpVarInt(2),
	}

	for k, ev := range expected {
		v := stats.Client().Get(k)
		verify.Values(t, k, v, ev)
	}
}
