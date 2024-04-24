//go:build integration
// +build integration

package uatest

import (
	"context"
	"testing"

	"github.com/pascaldekloe/goe/verify"

	"github.com/gopcua/opcua"
)

func TestNamespace(t *testing.T) {
	ctx := context.Background()

	srv := NewPythonServer("rw_server.py")
	defer srv.Close()

	c, err := opcua.NewClient(srv.Endpoint, srv.Opts...)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Connect(ctx); err != nil {
		t.Fatal(err)
	}
	defer c.Close(ctx)

	t.Run("NamespaceArray", func(t *testing.T) {
		got, err := c.NamespaceArray(ctx)
		if err != nil {
			t.Fatal(err)
		}
		want := []string{
			"http://opcfoundation.org/UA/",
			"urn:freeopcua:python:server",
			"http://gopcua.com/",
		}
		verify.Values(t, "", got, want)
	})
	t.Run("FindNamespace", func(t *testing.T) {
		ns, err := c.FindNamespace(ctx, "http://gopcua.com/")
		if err != nil {
			t.Fatal(err)
		}
		if got, want := ns, uint16(2); got != want {
			t.Fatalf("got namespace id %d want %d", got, want)
		}
	})
	t.Run("UpdateNamespaces", func(t *testing.T) {
		err := c.UpdateNamespaces(ctx)
		if err != nil {
			t.Fatal(err)
		}
	})
}
