package server

import (
	"context"

	"github.com/gopcua/opcua/ua"
	"github.com/gopcua/opcua/ualog"
)

// logServiceRequest logs information about the incoming request at
// the DEBUG level. If the current logger's debug level is not enabled
// no log attribute will be allocated
func logServiceRequest(ctx context.Context, req ua.Request) {
	ualog.DebugFunc(ctx, "handling service request", func() []ualog.Attr { return []ualog.Attr{ualog.Any("request", req)} })
}

// newServiceLogAttributeCreatorForSet is a utility function for wrapping a
// service's set and name in an attribte group for logging
func newServiceLogAttributeCreatorForSet(set string) func(string) ualog.Attr {
	return func(service string) ualog.Attr {
		return ualog.GroupAttrs("service", ualog.String("set", set), ualog.String("name", service))
	}
}
