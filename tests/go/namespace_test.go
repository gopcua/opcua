//go:build integration
// +build integration

package uatest2

import (
	"context"
	"testing"
	"time"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/ua"
	"github.com/stretchr/testify/require"
)

func TestNamespace(t *testing.T) {
	ctx := context.Background()

	srv := startServer()
	defer srv.Close()

	c, err := opcua.NewClient("opc.tcp://localhost:4840", opcua.SecurityMode(ua.MessageSecurityModeNone))
	require.NoError(t, err, "NewClient failed")

	err = c.Connect(ctx)
	require.NoError(t, err, "Connect failed")
	defer c.Close(ctx)

	time.Sleep(2 * time.Second)

	t.Run("NamespaceArray", func(t *testing.T) {
		got, err := c.NamespaceArray(ctx)
		require.NoError(t, err, "NamespaceArray failed")

		want := []string{
			"http://opcfoundation.org/UA/",
			"NodeNamespace",
			"http://gopcua.com/",
		}
		require.Equal(t, want, got, "NamespaceArray not equal")
	})
	t.Run("FindNamespace", func(t *testing.T) {
		ns, err := c.FindNamespace(ctx, "http://gopcua.com/")
		require.NoError(t, err, "FindNamespace failed")
		require.Equal(t, uint16(2), ns, "namespace id not equal")
	})
	t.Run("UpdateNamespaces", func(t *testing.T) {
		err := c.UpdateNamespaces(ctx)
		require.NoError(t, err, "UpdateNamespaces failed")
	})
}
