//go:build integration
// +build integration

package uatest

import (
	"context"
	"testing"

	"github.com/gopcua/opcua"
)

func TestUnsetUserIdentityTokenConnect(t *testing.T) {
	ctx := context.Background()

	srv := NewServer("unset_useridentitytoken_server.py")
	defer srv.Close()

	c, err := opcua.NewClient(srv.Endpoint, srv.Opts...)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Connect(ctx); err != nil {
		t.Fatal(err)
	}
	defer c.Close(ctx)
}
