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

	want := map[string]*expvar.Int{
		"Dial":             newExpVarInt(1),
		"ActivateSession":  newExpVarInt(1),
		"NamespaceArray":   newExpVarInt(1),
		"UpdateNamespaces": newExpVarInt(1),
		"NodesToRead":      newExpVarInt(1),
		"Read":             newExpVarInt(1),
		"Send":             newExpVarInt(2),
		"Close":            newExpVarInt(1),
		"CloseSession":     newExpVarInt(2),
		"SecureChannel":    newExpVarInt(2),
		"Session":          newExpVarInt(4),
		"State":            newExpVarInt(0),
	}

	got := map[string]expvar.Var{}
	stats.Client().Do(func(kv expvar.KeyValue) { got[kv.Key] = kv.Value })
	for k := range got {
		if _, ok := want[k]; !ok {
			t.Fatalf("got unexpected key %q", k)
		}
	}
	for k := range want {
		if _, ok := got[k]; !ok {
			t.Fatalf("missing expected key %q", k)
		}
	}

	for k, ev := range want {
		v := stats.Client().Get(k)
		if !verify.Values(t, "", v, ev) {
			t.Errorf("got %s for %q, want %s", v.String(), k, ev.String())
		}
	}
}
