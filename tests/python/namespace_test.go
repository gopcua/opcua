//go:build integration

package uatest

import (
	"context"
	"testing"

	"github.com/gopcua/opcua"
	"github.com/stretchr/testify/require"
)

func TestNamespace(t *testing.T) {
	ctx := context.Background()

	srv := NewPythonServer("rw_server.py")
	defer srv.Close()

	c, err := opcua.NewClient(srv.Endpoint, srv.Opts...)
	require.NoError(t, err, "NewClient failed")

	err = c.Connect(ctx)
	require.NoError(t, err, "Connect failed")
	defer c.Close(ctx)

	t.Run("NamespaceArray", func(t *testing.T) {
		got, err := c.NamespaceArray(ctx)
		require.NoError(t, err, "NamespaceArray failed")

		want := []string{
			"http://opcfoundation.org/UA/",
			"urn:freeopcua:python:server",
			"http://gopcua.com/",
		}
		require.Equal(t, want, got)
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
