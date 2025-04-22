package uatest2

import (
	"context"
	"expvar"
	"testing"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/stats"
	"github.com/gopcua/opcua/ua"
	"github.com/stretchr/testify/require"
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
	require.NoError(t, err, "NewClient failed")

	err = c.Connect(ctx)
	require.NoError(t, err, "Connect failed")
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
		require.Contains(t, want, k, "got unexpected key %q", k)
	}
	for k := range want {
		require.Contains(t, got, k, "missing expected key %q", k)
	}

	for k, ev := range want {
		v := stats.Client().Get(k)
		require.Equal(t, ev, v)
	}
}
