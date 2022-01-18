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
	srv := NewServer("rw_server.py")
	defer srv.Close()

	c := opcua.NewClient(srv.Endpoint, srv.Opts...)
	if err := c.Connect(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	t.Run("NamespaceArray", func(t *testing.T) {
		got, err := c.NamespaceArray()
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
		ns, err := c.FindNamespace("http://gopcua.com/")
		if err != nil {
			t.Fatal(err)
		}
		if got, want := ns, uint16(2); got != want {
			t.Fatalf("got namespace id %d want %d", got, want)
		}
	})
	t.Run("UpdateNamespaces", func(t *testing.T) {
		err := c.UpdateNamespaces()
		if err != nil {
			t.Fatal(err)
		}
	})
}
