//go:build conformance

// Package server holds the gopcua server tests: one behaviour per file,
// with the clause it checks in the file's opening comment. This file holds
// the fixture they all run against.
//
// The fixture is not a scripted server. A row here drives the in-tree
// gopcua server - production code - through its public API, with a real
// gopcua client connected over the wire, because the behaviours this
// package scores are properties of the server's own call graph: the
// re-entrancy of its notification path and the locking around its
// namespaces. A scripted server would score the script. Where a row needs
// the server to misbehave in a way the in-tree server cannot, that is a
// fixture gap, and it is named in README.md rather than worked around.
//
// Every helper here returns plain values. A helper does not call t.Errorf
// or t.Fatalf; a row's test function turns a non-empty reason string into
// the failure or the decline, and it, not the fixture, registers the
// teardown with defer. A stage that fails leaves the harness partly
// up: teardown is non-nil for whatever came up, and the row still owns
// calling it.
//
// The package name is server, as the directory is; the production package
// github.com/gopcua/opcua/server is a different import path and is spelled
// the same way inside these files.
package server

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/server"
	"github.com/gopcua/opcua/ua"
)

// respond is how long a call that does no I/O, or one loopback round
// trip, gets before it counts as wedged. SetValue writes one map entry
// and pushes one notification onto a buffered channel; a loopback Write
// is one request-response round trip; five seconds is not a timing
// assumption for either.
const respond = 5 * time.Second

const (
	// tag is the key in the map namespace every row operates on.
	tag = "Tag1"

	// nsName is the map namespace's name. Server.AddNamespace wires no
	// browse path to it from namespace 0, but the name still surfaces in
	// the namespace's BrowseName and DisplayName.
	nsName = "ConcurrentNS"
)

// harness is one server with one MapNamespace and one connected client.
type harness struct {
	ns     *server.MapNamespace
	client *opcua.Client

	// notify carries the subscription's notifications once subscribeTo has
	// run. The initial one is consumed there, so what remains is the
	// result of whatever the row did afterwards.
	notify chan *opcua.PublishNotificationData
}

// startServerAndClient brings up an in-tree server holding a MapNamespace
// with one tag, and a client connected to it over the wire. It returns the
// harness, a teardown for everything that came up, and "" - or, when a
// stage failed, the reason the run never reached the point a row is about.
//
// The tag is seeded only after srv.Start: SetValue notifies through
// Server.MonitoredItemService, which Start creates in initHandlers
// (server/service_handlers.go:78), so a SetValue before Start panics on a
// nil service. Nothing is subscribed at seed time, so that call cannot
// wedge whatever the locking looks like.
func startServerAndClient() (*harness, func(), string) {
	port, why := freePort()
	if why != "" {
		return nil, func() {}, why
	}

	srv := server.New(
		server.EnableSecurity("None", ua.MessageSecurityModeNone),
		server.EnableAuthMode(ua.UserTokenTypeAnonymous),
		server.EndPoint("127.0.0.1", port),
	)

	ns := server.NewMapNamespace(srv, nsName)

	ctx := context.Background()
	if err := srv.Start(ctx); err != nil {
		return nil, func() {}, fmt.Sprintf("server start: %v", err)
	}
	teardown := func() { _ = srv.Close() }

	ns.SetValue(tag, int32(0))

	endpoint := fmt.Sprintf("opc.tcp://127.0.0.1:%d", port)
	client, err := opcua.NewClient(endpoint, opcua.SecurityMode(ua.MessageSecurityModeNone))
	if err != nil {
		teardown()
		return nil, func() {}, fmt.Sprintf("new client: %v", err)
	}
	if err := client.Connect(ctx); err != nil {
		_ = client.Close(ctx)
		teardown()
		return nil, func() {}, fmt.Sprintf("connect to %s: %v", endpoint, err)
	}

	return &harness{ns: ns, client: client}, func() {
		_ = client.Close(ctx)
		teardown()
	}, ""
}

// subscribeTo registers a monitored item on key through the real
// CreateSubscription and CreateMonitoredItems services, so that
// MonitoredItemService.Nodes is populated the way a live server populates
// it: writing the service's map from the row would assert about the row's
// own setup at exactly the point where the write-half defect lives. It
// returns a teardown for the subscription and "" - or, when a stage
// failed, the reason the run never reached the point a row is about.
//
// subscribeTo does not return until the subscription's initial
// notification has arrived and been consumed. CreateMonitoredItems fires
// that one in its own goroutine
// (server/monitored_item_service.go:232: `go s.ChangeNotification(nodeid)`),
// and it reads the tag; returning before it lands would leave the row's
// write racing the fixture's own setup rather than whatever the row is
// here to test. Waiting for it also proves the subscription is live, so a
// write that returns later cannot have done so because nothing was
// listening.
func (h *harness) subscribeTo(key string) (func(), string) {
	ctx := context.Background()

	notify := make(chan *opcua.PublishNotificationData, 16)
	h.notify = notify
	sub, err := h.client.Subscribe(ctx, &opcua.SubscriptionParameters{
		Interval: 100 * time.Millisecond,
	}, notify)
	if err != nil {
		return nil, fmt.Sprintf("subscribe: %v", err)
	}
	teardown := func() { _ = sub.Cancel(ctx) }

	node := ua.NewStringNodeID(h.ns.ID(), key)
	res, err := sub.Monitor(ctx, ua.TimestampsToReturnBoth,
		opcua.NewMonitoredItemCreateRequestWithDefaults(node, ua.AttributeIDValue, 42))
	if err != nil {
		teardown()
		return nil, fmt.Sprintf("monitor %s: %v", node, err)
	}
	if len(res.Results) != 1 || res.Results[0].StatusCode != ua.StatusOK {
		teardown()
		return nil, fmt.Sprintf("monitor %s was refused: %+v", node, res.Results)
	}

	select {
	case <-notify:
	case <-time.After(respond):
		teardown()
		return nil, fmt.Sprintf("no notification arrived for %s within %v, "+
			"so the subscription is not delivering and the write path would test nothing",
			node, respond)
	}
	return teardown, ""
}

// expectNotification waits for one data-change notification and checks
// the value it carries. Only the subscribed tag is monitored, so any
// notification on the subscription is about it. It returns "" or the
// reason the notification does not carry want.
//
// The value it checks crossed the wire through two gopcua codecs, so it
// scores delivery and not bytes; the rows that call this say so in their
// headers, and README.md says what would let them score bytes.
func (h *harness) expectNotification(want int32) string {
	deadline := time.After(respond)
	for {
		select {
		case n := <-h.notify:
			if n.Error != nil {
				return fmt.Sprintf("notification carried an error: %v", n.Error)
			}
			dcn, ok := n.Value.(*ua.DataChangeNotification)
			if !ok {
				continue // a keep-alive or an event; not what this watches
			}
			for _, item := range dcn.MonitoredItems {
				if item.Value == nil || item.Value.Value == nil {
					return fmt.Sprintf("notification carried no value: %+v", item.Value)
				}
				if item.Value.Status != ua.StatusOK {
					return fmt.Sprintf("the notification carried status %v, not Good", item.Value.Status)
				}
				got := item.Value.Value.Value()
				if got != want {
					return fmt.Sprintf("the subscription was notified of "+
						"%v (%T), but %v (%T) was written", got, got, want, want)
				}
				return ""
			}
		case <-deadline:
			return fmt.Sprintf("the write returned but no notification carrying it "+
				"arrived within %v; a write that returns by not notifying has not fixed anything",
				respond)
		}
	}
}

// freePort asks the kernel for an unused TCP port. The in-tree server's
// EndPoint option wants a number, and a fixed one would collide with any
// other OPC UA work on the same machine. It returns "" or the reason no
// port could be reserved.
func freePort() (int, string) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, fmt.Sprintf("reserve port: %v", err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	if err := l.Close(); err != nil {
		return 0, fmt.Sprintf("release port: %v", err)
	}
	return port, ""
}
