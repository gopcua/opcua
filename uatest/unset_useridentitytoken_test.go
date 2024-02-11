//go:build integration
// +build integration

package uatest

import (
	"context"
	"log"
	"testing"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/ua"
)

func TestUnsetUserIdentityTokenConnect(t *testing.T) {
	ctx := context.Background()

	srv := NewServer("unset_useridentitytoken_server.py")
	defer srv.Close()

	endpoints, err := opcua.GetEndpoints(ctx, *&srv.Endpoint)
	if err != nil {
		log.Fatal(err)
	}

	ep := opcua.SelectEndpoint(endpoints, "None", ua.MessageSecurityModeFromString("None"))

	opts := []opcua.Option{
		opcua.SecurityPolicy("None"),
		opcua.SecurityModeString("None"),
		opcua.AuthUsername("user", "pass"),
		opcua.SecurityFromEndpoint(ep, ua.UserTokenTypeUserName),
	}

	c, err := opcua.NewClient(ep.EndpointURL, opts...)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Connect(ctx); err != nil {
		t.Fatal(err)
	}
	defer c.Close(ctx)
}
