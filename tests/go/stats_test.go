package uatest2

import (
	"context"
	"expvar"
	"testing"

	"github.com/pascaldekloe/goe/verify"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/stats"
	"github.com/gopcua/opcua/ua"
)

func newExpVarInt(i int64) *expvar.Int {
	v := &expvar.Int{}
	v.Set(i)
	return v
}

func TestStats(t *testing.T) {
	stats.Reset()

	ctx := context.Background()

	srv := startServer()
	defer srv.Close()

	c, err := opcua.NewClient("opc.tcp://localhost:4840", opcua.SecurityMode(ua.MessageSecurityModeNone))
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Connect(ctx); err != nil {
		t.Fatal(err)
	}
	c.Close(ctx)

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
		"SecureChannel":    newExpVarInt(3),
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
