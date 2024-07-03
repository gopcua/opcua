//go:build integration
// +build integration

package uatest2

import (
	"context"
	"testing"
	"time"

	"github.com/pascaldekloe/goe/verify"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/ua"
)

func TestNamespace(t *testing.T) {
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
	defer c.Close(ctx)

	time.Sleep(2 * time.Second)

	t.Run("NamespaceArray", func(t *testing.T) {
		got, err := c.NamespaceArray(ctx)
		if err != nil {
			t.Fatal(err)
		}
		want := []string{
			"http://opcfoundation.org/UA/",
			"NodeNamespace",
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
