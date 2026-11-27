//go:build conformance

// DeleteMonitoredItems that names an item of a subscription the same
// client has just deleted answers Bad_MonitoredItemIdInvalid for that
// id, the server keeps answering both the client that walked the
// deletion through and a second client that never held the subscription,
// and CreateSubscription asked for a publishing interval of 0 answers
// with the fastest interval the server supports - 1 ms - and delivers
// notifications on it.
//
// The clauses: OPC 10000-4 §5.13.6.2 - a monitored item id that names no
// monitored item answers Bad_MonitoredItemIdInvalid for that id while
// the rest of the request is answered - and the item ids of a deleted
// subscription name nothing, because deleting a subscription deletes its
// monitored items (OPC 10000-4 §5.14.8.2). OPC 10000-4 §5.14.2.2 - a
// requested publishing interval of 0 is revised to the fastest interval
// the server supports, and the revised value is the interval the
// subscription runs at. Both readings are taken from the Part 4 tables
// and are not checkable in this tree; the first author with the text in
// front of them should confirm or correct them, the way README.md asks
// for its OPC 10000-3 §5.6.2 8.58 citation.
//
// Neighbouring rows: setvalue_on_a_subscribed_tag_returns_test.go,
// setattribute_on_a_subscribed_tag_returns_test.go and
// read_while_writing_is_race_free_test.go drive one live subscription
// through the client's own Subscribe, where this row drives the
// subscription services as raw requests, because Client.Subscribe
// cannot ask for what this row scores: SubscriptionParameters.setDefaults
// replaces a zero interval with the default (subscription.go:384), so a
// CreateSubscription with RequestedPublishingInterval 0 has to be sent
// by hand. client_keeps_working_across_token_renewals_test.go is the
// other row that scores the pair's survival - across token renewals,
// where this one scores the server's survival of its own deletion path.
// On the tree without the server changes this row guards, the sequence
// crashed the server twice over: CreateSubscription with an interval of
// 0 handed the subscription's ticker a zero interval, which
// time.NewTicker panics on (server/subscription_service.go:386), and
// DeleteMonitoredItems fell through its unknown-id branch to a nil
// dereference of the very item it had just reported invalid
// (server/monitored_item_service.go). The server package's unit tests
// hold the handlers' answers for sessions and ids; what no unit test
// holds is the sequence over the wire with a real session, which is
// what crashed.
//
// The row runs against the in-tree gopcua server holding one
// MapNamespace and a real gopcua client connected over the wire - the
// shared fixture in harness.go - and speaks CreateSubscription,
// CreateMonitoredItems, Publish, DeleteSubscriptions and
// DeleteMonitoredItems to that server as raw service calls through
// Client.Send, which fills in the request header and the session's
// authentication token (uasc/secure_channel_instance.go:65); why the
// in-tree server and not a scripted one is in harness.go and README.md.
// The second client is the row's own, connected to the harness's
// endpoint. See harness.go.
//
// The row asserts that the revised interval is 1 ms; that the raw
// subscription delivers a data change for every node it monitors and
// then the value the row writes; that both ids of the DeleteMonitoredItems
// answer - the deleted subscription's item and an id no subscription
// ever owned - are Bad_MonitoredItemIdInvalid; and that a Read from the
// first client and one from the second each return the written value.
// The deletion is scored by polling: the server purges a deleted
// subscription's items on a goroutine after the DeleteSubscriptions
// response, so the first DeleteMonitoredItems may still find the item,
// which is early rather than wrong, and the poll's deadline is respond -
// a map delete and a channel close, not a timing assumption. The row
// deliberately scores no bytes: every value it reads crossed the wire
// through two gopcua codecs, so the checks score delivery and not
// bytes, and the written literal is the same outside-gopcua evidence
// the write-half rows stand on; no row anywhere holds the literal bytes
// it could name, for the reason README.md gives.
//
// No reference stack has been read for this behaviour; what one would
// mean here - another server's source - is in README.md. The instrument
// the verdict rests on is the pair's liveness: the crash this row exists
// to catch kills the process, and no answer at all is what a crash looks
// like from a client.

package server

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/ua"
)

const (
	// tag2 and tag3 are two more keys in the map namespace, so that the
	// row monitors a few nodes and not only the fixture's one.
	tag2 = "Tag2"
	tag3 = "Tag3"

	// neverAssigned is a monitored item id no subscription ever owned: the
	// server hands item ids out starting at one
	// (server/monitored_item_service.go:143), so no item is ever zero.
	neverAssigned = uint32(0)
)

func TestDeleteMonitoredItemsOnADeletedSubscriptionAnswers(t *testing.T) {
	h, teardown, why := startServerAndClient()
	defer teardown()
	if why != "" {
		t.Fatalf("could not set the test up: %s", why)
	}

	ctx := context.Background()

	// The literal comes from outside gopcua; the header says what the
	// checks it feeds score.
	const written = int32(-4242)

	h.ns.SetValue(tag2, int32(11))
	h.ns.SetValue(tag3, int32(12))

	// The subscription is asked for a publishing interval of 0: the
	// server must revise that, not run it.
	sub, why := createSubscription(ctx, h.client)
	if why != "" {
		t.Fatalf("could not set the test up: %s", why)
	}
	if sub.revised != 1 {
		t.Errorf("CreateSubscription asked for a publishing interval of 0 ms "+
			"revised it to %g ms, not 1", sub.revised)
	}

	// The handles are the keys' indices plus one, so the tag the row
	// writes is handle 1.
	keys := []string{tag, tag2, tag3}
	itemIDs, why := monitorNodes(ctx, h.client, sub, h.ns.ID(), keys)
	if why != "" {
		t.Fatalf("could not set the test up: %s", why)
	}

	changes, why := publishDataChanges(ctx, h.client, sub, []uint32{1, 2, 3})
	if why != "" {
		t.Errorf("the subscription created with a publishing interval of 0 ms "+
			"did not deliver its initial changes: %s", why)
		return
	}

	// A value written after the first changes must be delivered too: an
	// interval revised from 0 that delivers only the initial changes has
	// not been fixed.
	h.ns.SetValue(tag, written)
	changes, why = publishDataChanges(ctx, h.client, sub, []uint32{1})
	if why != "" {
		t.Errorf("the subscription created with a publishing interval of 0 ms "+
			"did not deliver the written value: %s", why)
		return
	}
	if got := changes[1].Value.Value(); got != written {
		t.Errorf("the subscription was notified of %v (%T), but %v (%T) was written",
			got, got, written, written)
		return
	}

	if why := deleteSubscription(ctx, h.client, sub); why != "" {
		t.Fatalf("could not set the test up: %s", why)
	}

	// The verdict the row exists for: the deleted subscription's item and
	// an id no subscription ever owned are both reported invalid, and the
	// server is still standing for the reads after it.
	ids := []uint32{itemIDs[0], neverAssigned}
	results, why := deleteMonitoredItemsUntilInvalid(ctx, h.client, sub, ids)
	if why != "" {
		t.Errorf("DeleteMonitoredItems on the deleted subscription's ids: %s", why)
	} else {
		t.Logf("DeleteMonitoredItems answered %v for ids %v", results, ids)
	}

	if v, err := h.client.Node(ua.NewStringNodeID(h.ns.ID(), tag)).Value(ctx); err != nil {
		t.Errorf("the server did not answer the same client's Read after the deletion: %v", err)
	} else if got := v.Value(); got != written {
		t.Errorf("the same client's Read returned %v (%T), not %v", got, got, written)
	}

	second, secondTeardown, why := connectSecondClient(h.endpoint)
	if why != "" {
		t.Errorf("the server did not accept a second client after the deletion: %s", why)
		return
	}
	defer secondTeardown()
	if v, err := second.Node(ua.NewStringNodeID(h.ns.ID(), tag)).Value(ctx); err != nil {
		t.Errorf("the server did not answer the second client's Read after the deletion: %v", err)
	} else if got := v.Value(); got != written {
		t.Errorf("the second client's Read returned %v (%T), not %v", got, got, written)
	}
}

// rawSub is the row's subscription, created by the CreateSubscription
// service directly rather than by Client.Subscribe, for the reason the
// header gives.
type rawSub struct {
	id      uint32
	revised float64
}

// sendService sends req over the client's secure channel and returns the
// answer as R. The requests carry no header of their own: Client.Send
// injects the session's authentication token and uasc fills in the rest
// of the request header (uasc/secure_channel_instance.go:65). It returns
// the response and "" - or the zero response and the reason no answer of
// the right type arrived.
func sendService[R ua.Response](ctx context.Context, c *opcua.Client, req ua.Request) (R, string) {
	var resp R
	err := c.Send(ctx, req, func(v ua.Response) error {
		r, ok := v.(R)
		if !ok {
			return fmt.Errorf("the answer was a %T, not a %T", v, resp)
		}
		resp = r
		return nil
	})
	if err != nil {
		return resp, fmt.Sprintf("%T: %v", req, err)
	}
	return resp, ""
}

// createSubscription sends CreateSubscription with a requested publishing
// interval of 0. The counts are generous enough that the subscription
// does not time itself out between the row's publishes. It returns the
// subscription and "" - or the reason the server refused it.
func createSubscription(ctx context.Context, c *opcua.Client) (rawSub, string) {
	res, why := sendService[*ua.CreateSubscriptionResponse](ctx, c, &ua.CreateSubscriptionRequest{
		RequestedPublishingInterval: 0,
		RequestedLifetimeCount:      1000,
		RequestedMaxKeepAliveCount:  20,
		PublishingEnabled:           true,
	})
	if why != "" {
		return rawSub{}, why
	}
	if res.ResponseHeader.ServiceResult != ua.StatusOK {
		return rawSub{}, fmt.Sprintf("CreateSubscription was refused: %v",
			res.ResponseHeader.ServiceResult)
	}
	return rawSub{id: res.SubscriptionID, revised: res.RevisedPublishingInterval}, ""
}

// monitorNodes registers a monitored item on each key through the real
// CreateMonitoredItems service, with the key's index plus one as the
// client handle, so a notification can be told apart by handle. It
// returns the monitored item ids, in the keys' order, and "" - or the
// reason the server refused an item.
func monitorNodes(ctx context.Context, c *opcua.Client, sub rawSub, ns uint16, keys []string) ([]uint32, string) {
	items := make([]*ua.MonitoredItemCreateRequest, len(keys))
	for i, key := range keys {
		items[i] = opcua.NewMonitoredItemCreateRequestWithDefaults(
			ua.NewStringNodeID(ns, key), ua.AttributeIDValue, uint32(i+1))
	}
	res, why := sendService[*ua.CreateMonitoredItemsResponse](ctx, c, &ua.CreateMonitoredItemsRequest{
		SubscriptionID:     sub.id,
		TimestampsToReturn: ua.TimestampsToReturnBoth,
		ItemsToCreate:      items,
	})
	if why != "" {
		return nil, why
	}
	ids := make([]uint32, len(res.Results))
	for i, r := range res.Results {
		if r.StatusCode != ua.StatusOK {
			return nil, fmt.Sprintf("monitor %s was refused: %v", keys[i], r.StatusCode)
		}
		ids[i] = r.MonitoredItemID
	}
	return ids, ""
}

// publishDataChanges sends Publish requests on sub until every handle in
// want has been seen in a data change, and returns the newest value each
// handle delivered. The server answers a Publish whose subscription
// queue is empty with a keep-alive, and an item's first change may share
// a data change with the others or miss it, so both are retried, not
// scored. The deadline is respond, enforced on the context so that a
// wedged server cannot outlive it inside one Send. It returns the values
// and "" - or the reason the changes did not arrive.
func publishDataChanges(ctx context.Context, c *opcua.Client, sub rawSub, want []uint32) (map[uint32]*ua.DataValue, string) {
	ctx, cancel := context.WithTimeout(ctx, respond)
	defer cancel()

	got := map[uint32]*ua.DataValue{}
	for {
		res, why := sendService[*ua.PublishResponse](ctx, c, &ua.PublishRequest{
			SubscriptionAcknowledgements: []*ua.SubscriptionAcknowledgement{},
		})
		if why != "" {
			return nil, why
		}
		if res.SubscriptionID != sub.id {
			return nil, fmt.Sprintf("a publish on subscription %d was answered for subscription %d",
				sub.id, res.SubscriptionID)
		}
		for _, item := range dataChangeItems(res.NotificationMessage) {
			if item.Value == nil {
				return nil, fmt.Sprintf("a notification for handle %d carried no value", item.ClientHandle)
			}
			if item.Value.Status != ua.StatusOK {
				return nil, fmt.Sprintf("a notification for handle %d carried status %v, not Good",
					item.ClientHandle, item.Value.Status)
			}
			got[item.ClientHandle] = item.Value
		}
		seen := true
		for _, handle := range want {
			if got[handle] == nil {
				seen = false
			}
		}
		if seen {
			return got, ""
		}
		if ctx.Err() != nil {
			return nil, fmt.Sprintf("no data change for handles %v arrived on subscription %d within %v",
				want, sub.id, respond)
		}
	}
}

// dataChangeItems returns the monitored-item notifications msg carries
// if it is a data change, and nothing if it is a keep-alive or another
// notification kind.
func dataChangeItems(msg *ua.NotificationMessage) []*ua.MonitoredItemNotification {
	var items []*ua.MonitoredItemNotification
	for _, data := range msg.NotificationData {
		if data == nil || data.Value == nil {
			continue
		}
		if dcn, ok := data.Value.(*ua.DataChangeNotification); ok {
			items = append(items, dcn.MonitoredItems...)
		}
	}
	return items
}

// deleteSubscription removes sub through the real DeleteSubscriptions
// service. It returns "" - or the reason the server refused it.
func deleteSubscription(ctx context.Context, c *opcua.Client, sub rawSub) string {
	res, why := sendService[*ua.DeleteSubscriptionsResponse](ctx, c, &ua.DeleteSubscriptionsRequest{
		SubscriptionIDs: []uint32{sub.id},
	})
	if why != "" {
		return why
	}
	if len(res.Results) != 1 || res.Results[0] != ua.StatusOK {
		return fmt.Sprintf("deleting subscription %d was refused: %v", sub.id, res.Results)
	}
	return ""
}

// deleteMonitoredItemsUntilInvalid sends DeleteMonitoredItems for ids on
// sub until every id answers Bad_MonitoredItemIdInvalid, and returns the
// final results and "" - or the reason not every id did within respond.
// The server purges a deleted subscription's items on a goroutine after
// the DeleteSubscriptions response, so an id may still be found on the
// first calls; that is early rather than wrong, and the call is retried.
// A map delete and a channel close - respond is not a timing assumption
// for it.
func deleteMonitoredItemsUntilInvalid(ctx context.Context, c *opcua.Client, sub rawSub, ids []uint32) ([]ua.StatusCode, string) {
	deadline := time.After(respond)
	for {
		res, why := sendService[*ua.DeleteMonitoredItemsResponse](ctx, c, &ua.DeleteMonitoredItemsRequest{
			SubscriptionID:   sub.id,
			MonitoredItemIDs: ids,
		})
		if why != "" {
			return nil, why
		}
		invalid := true
		for _, r := range res.Results {
			if r != ua.StatusBadMonitoredItemIDInvalid {
				invalid = false
			}
		}
		if invalid {
			return res.Results, ""
		}
		select {
		case <-deadline:
			return nil, fmt.Sprintf("ids %v were not all reported invalid within %v; "+
				"the last answer was %v", ids, respond, res.Results)
		case <-time.After(time.Millisecond):
		}
	}
}

// connectSecondClient connects a client of the row's own to endpoint,
// mirroring the fixture's own client. It returns the client, a teardown
// for everything that came up, and "" - or the reason no connection
// could be made.
func connectSecondClient(endpoint string) (*opcua.Client, func(), string) {
	ctx := context.Background()
	second, err := opcua.NewClient(endpoint, opcua.SecurityMode(ua.MessageSecurityModeNone))
	if err != nil {
		return nil, func() {}, fmt.Sprintf("new client: %v", err)
	}
	if err := second.Connect(ctx); err != nil {
		_ = second.Close(ctx)
		return nil, func() {}, fmt.Sprintf("connect to %s: %v", endpoint, err)
	}
	return second, func() { _ = second.Close(ctx) }, ""
}
