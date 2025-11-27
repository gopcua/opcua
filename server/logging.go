package server

import (
	"context"

	"github.com/gopcua/opcua/ua"
	"github.com/gopcua/opcua/ualog"
)

func logServiceRequest(ctx context.Context, req ua.Request) {
	ualog.DebugFunc(ctx, "handling service request", func() []ualog.Attr { return []ualog.Attr{ualog.Any("request", req)} })
}

func newServiceLogAttributeCreatorForSet(set string) func(string) ualog.Attr {
	return func(service string) ualog.Attr {
		return ualog.GroupAttrs("service", ualog.String("set", set), ualog.String("name", service))
	}
}
