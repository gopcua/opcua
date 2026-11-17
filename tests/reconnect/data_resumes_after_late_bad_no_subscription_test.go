//go:build conformance

// A client whose application deletes its only subscription and creates a
// replacement keeps supplying Publish requests, so the replacement receives
// notifications. Conformance units 744/2284, Subscription Client Basic: the
// client shall "Supply a sufficient number of Publish requests to the Server
// so that Notifications can be sent whenever a publish timer expires."
// OPC 10000-4 §5.14.8.1: when the last Subscription of a Session is deleted,
// "all Publish requests still queued for that Session are de-queued and shall
// be returned with Bad_NoSubscription". The de-queued answers are the
// clause's own doing and not the application's, so a client that stops
// publishing on the back of one abandons the replacement it has just built.
//
// The neighbouring rows.
// data_resumes_on_recreated_subscription_test.go scores a replacement the
// client built after a drop and a Republish fault, not after the
// application's own delete; data_resumes_after_reconnect_test.go scores a
// subscription that survived the drop and needed no replacement at all.
// Two other rows count the Publish the client sent, as this one does:
// publish_resumes_when_republish_loop_ends_test.go counts them after a
// Republish loop ends, and data_resumes_on_recreated_subscription_test.go
// counts them after the drop, in newSubscriptionAfterDrop and
// publishOpportunitySetupFailed; this row counts them after the fault the
// server was holding was answered.
//
// The scripted server is the default one with the HeldPublishOnCommand
// option: the server holds the stream's second Publish, the scenario
// deletes the only subscription and creates and monitors a replacement, and
// only then commands the held Publishes' answer, a ServiceFault carrying
// Bad_NoSubscription for every Publish the server holds. The scenario is
// this row's own, because runScenario's shape is the drop's. See harness.go.
//
// The row asserts the outcome: a notification the server created after the
// replacement existed reached the application's own callback within
// noSubscriptionBound. It does not ask how the client supplies Publishes -
// one at a time, a counter, a chain - only that some Publish follows the
// fault. The order is constructed rather than hoped for, and it is a legal
// server order: §5.14.8.1 sets no point in time for the de-queued answers,
// so the late order is one a conforming server may produce. The
// natural order - the answer commanded right after Cancel and before
// Subscribe - is not scored: it shows the defect only by chance, so the
// answer is commanded after the replacement exists. The arm
// keys on an absence, no notification for the replacement, because a parked
// loop leaves no presence on the wire, and the setup checks make sure the
// client reached the stimulus first; ../README.md's paragraph on keying an
// arm on an absence is the rule this follows.
//
// The reference stacks, as read and not run.
// Eclipse Milo at 5fe65addec870a49ff24b419cb811d094a919f81 passes by design:
// PublishingManager.java lines 1167-1182 refill after a Bad_NoSubscription
// when the subscription generation moved since the request was sent, and
// the comment there describes this case.
// open62541 at 1deeb501d5173e337b9bf16d94598d8164b123ab passes:
// src/client/ua_client_subscriptions.c decrements the outstanding count for
// every Publish response (line 1709), returns early on Bad_NoSubscription
// (lines 1737-1744), and the response callback then refills while any
// subscription exists (lines 1852-1853, 1913-1923).
// node-opcua at f666591c04a7f4322f0c43c45cf2f32dc670ccd7 passes: registering
// the replacement refills the pipeline (client_publish_engine.ts lines
// 141-176), so Publishes for the replacement are already outstanding when
// the late Bad_NoSubscription stops the old chain (lines 311-322).
// UA-.NETStandard at a132e6f1ee36590dba690d9afd00250c9893c73b fails at its
// default MinPublishRequestCount of 1: src/Opc.Ua.Client/Session/
// Subscription/ClassicSubscriptionEngine.cs `NotifySubscriptionsChanged`
// calls `QueueBeginPublish` (lines 184-187), which sends nothing while the
// held request fills the one slot (lines 981-1000, 1067-1120), and a
// Bad_NoSubscription answer returns without queueing another (lines
// 637-650).
// asyncua (github.com/FreeOpcUa/opcua-asyncio) at
// e506dc6bc63c13176a9ede78b905ca9ca0a2523f fails:
// asyncua/client/ua_session.py lines 370-372 restart the publish loop only
// when its task has finished, and lines 460-462 end the loop on
// Bad_NoSubscription.
// That is not an exception in the sense of ../README.md's question 6: the
// clause is the client's, Milo documents the case and fixed it, and two
// stacks failing it is two stacks with the defect gopcua carried until
// https://github.com/gopcua/opcua/pull/957, which the row names so a reader
// knows it is not a fixture artifact.

package reconnect

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/ua"
)

// noSubscriptionBound is how long the row waits, after the server has
// answered the held Publish, for a notification the server created for the
// replacement subscription. An unpaused client sends its next Publish
// immediately and the server paces every answer publishPacing, so the
// honest wait is one pacing plus scheduling; the bound is fifteen pacings,
// which covers a loaded machine without giving a parked loop anywhere to
// hide: a parked loop sends nothing however long the row waits.
const noSubscriptionBound = 3 * time.Second

func TestDataResumesAfterLateBadNoSubscription(t *testing.T) {
	// gopcua did not do this until
	// https://github.com/gopcua/opcua/pull/957 merged; it passes since.

	run := runLateBadNoSubscriptionScenario()

	if why := setupFailed(run.Post); why != "" {
		t.Fatalf("could not set the test up: %s", why)
	}
	// A Subscribe or a Monitor that failed after the delete is gopcua's own
	// behaviour against a server that grants both, so it is scored rather
	// than declined; the wire check below would otherwise read the missing
	// requests as the fixture's doing.
	if run.ClientErr != "" {
		t.Errorf("gopcua's own behaviour made this run unscoreable: %s", run.ClientErr)
		return
	}
	if why := deleteThenRecreateSetupFailed(run); why != "" {
		t.Fatalf("could not set the test up: %s", why)
	}

	fresh := freshDeliveriesAfterFault(run)
	published := publishesAfterFault(run)

	switch {
	case len(fresh) > 0:
		t.Logf("notifications %v, created for the replacement, reached the application's own "+
			"callback after the held Publish was answered with Bad_NoSubscription (the client "+
			"sent %d Publish after the fault was commanded)", fresh, published)
	default:
		t.Errorf("the server answered the held Publish with Bad_NoSubscription after the "+
			"replacement's CreateMonitoredItems was answered, and the client sent %d Publish "+
			"after the fault was commanded, but no notification created for the replacement ever "+
			"reached the application within %s (%d reached it in all, none above sequence %d)",
			published, noSubscriptionBound, len(run.Post.Deliveries), run.Faulted.LastSeq)
	}
}

// lateFaultRun is one run of this row's scenario. HeldRequests is how
// many requests the server had recorded while the stream was live and one
// Publish was held, so Faulted's requests past it are the delete and the
// replacement the scenario commands next. Faulted is the wire and the
// deliveries frozen the moment the fault was commanded, and Post the same
// frozen after the bound. Post.HarnessErr is a stage the scenario could
// not reach, the fixture's or the environment's doing, and ClientErr is
// gopcua's own behaviour making the run unscoreable. The two are never
// both set, because the first one to happen ends the run.
type lateFaultRun struct {
	HeldRequests int
	Faulted      runRecord
	Post         runRecord
	ClientErr    string
}

// deleteThenRecreateSetupFailed reports why the wire does not show the order
// this row constructs, or "" if it does. The fault is commanded after
// Monitor on the replacement returned, so a DeleteSubscriptions, then a
// CreateSubscription, then a CreateMonitoredItems, all before the fault was
// commanded, is what the scenario's own calls put on the wire; anything else
// means the fault was not answered after a replacement existed, and the
// absence the arm keys on would then say nothing about the client. It reads
// the run record rather than the Go return values alone, which is why it
// runs after the calls have all returned.
func deleteThenRecreateSetupFailed(run lateFaultRun) string {
	var seen []string
	step := 0
	for _, r := range run.Faulted.Requests[run.HeldRequests:] {
		seen = append(seen, r.Service)
		switch {
		case step == 0 && r.Service == "DeleteSubscriptionsRequest":
			step = 1
		case step == 1 && r.Service == "CreateSubscriptionRequest":
			step = 2
		case step == 2 && r.Service == "CreateMonitoredItemsRequest":
			step = 3
		}
	}
	if step != 3 {
		return fmt.Sprintf("the wire does not show a DeleteSubscriptions, then a "+
			"CreateSubscription, then a CreateMonitoredItems, all before the fault was "+
			"commanded, so the Bad_NoSubscription did not arrive after a replacement existed "+
			"(the requests between the held Publish and the fault were %v)", seen)
	}
	return ""
}

// publishesAfterFault counts the Publish requests the client sent after the
// fault was commanded. Faulted is frozen at that moment, so everything past
// its request log is the client's own doing after the de-queued answer
// reached it.
func publishesAfterFault(run lateFaultRun) int {
	n := 0
	for _, r := range run.Post.Requests[len(run.Faulted.Requests):] {
		if r.Service == "PublishRequest" {
			n++
		}
	}
	return n
}

// freshDeliveriesAfterFault is the sequence numbers the application's own
// callback received that the server created after the fault was commanded.
// The server creates no notification between the replacement's
// CreateMonitoredItems and the fault - the one Publish gopcua keeps
// outstanding is the held one, and no other arrives in that window - and
// after the fault it creates them only in answer to a
// Publish, on the one subscription anything is monitored on, which is the
// replacement, so a delivery above Faulted's LastSeq is data the
// replacement carried.
func freshDeliveriesAfterFault(run lateFaultRun) []uint32 {
	var out []uint32
	for _, d := range run.Post.Deliveries {
		if d.Seq > run.Faulted.LastSeq {
			out = append(out, d.Seq)
		}
	}
	return out
}

// runLateBadNoSubscriptionScenario drives the HeldPublishOnCommand server.
// It waits until the stream is live and one Publish is held, deletes the
// only subscription, creates and monitors the replacement, and only then
// commands the held Publish's Bad_NoSubscription answer, which makes the
// client's own ordering - the replacement's Subscribe resuming the publish
// loop before the fault is processed - certain rather than raced for.
func runLateBadNoSubscriptionScenario() lateFaultRun {
	var run lateFaultRun

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	srv, err := newServer(ctx, runOptions{HeldPublishOnCommand: true})
	if err != nil {
		run.Post.HarnessErr = fmt.Sprintf("the scripted server could not start: %v", err)
		return run
	}
	defer func() { srv.ln.Close() }()

	c, err := opcua.NewClient(srv.endpoint,
		opcua.SecurityMode(ua.MessageSecurityModeNone),
		opcua.AutoReconnect(true),
		opcua.ReconnectInterval(100*time.Millisecond),
		opcua.RequestTimeout(3*time.Second),
		opcua.DialTimeout(3*time.Second),
	)
	if err != nil {
		run.Post.HarnessErr = fmt.Sprintf("constructing the client: %v", err)
		return run
	}
	if err := c.Connect(ctx); err != nil {
		run.Post.HarnessErr = fmt.Sprintf("the initial connect failed: %v", err)
		return run
	}
	defer func() {
		closeCtx, closeCancel := context.WithTimeout(context.Background(), 3*time.Second)
		_ = c.Close(closeCtx)
		closeCancel()
	}()

	notifs := make(chan *opcua.PublishNotificationData, 32)
	params := &opcua.SubscriptionParameters{Interval: 200 * time.Millisecond}
	sub, err := c.Subscribe(ctx, params, notifs)
	if err != nil {
		run.Post.HarnessErr = fmt.Sprintf("subscribing: %v", err)
		return run
	}
	go func() {
		for n := range notifs {
			dcn, ok := n.Value.(*ua.DataChangeNotification)
			if !ok || len(dcn.MonitoredItems) == 0 {
				continue
			}
			if seq, ok := dcn.MonitoredItems[0].Value.Value.Value().(int32); ok {
				srv.rec.recordDelivery(uint32(seq))
			}
		}
	}()

	item := opcua.NewMonitoredItemCreateRequestWithDefaults(monitoredNode, ua.AttributeIDValue, 1)
	if _, err := sub.Monitor(ctx, ua.TimestampsToReturnBoth, item); err != nil {
		run.Post.HarnessErr = fmt.Sprintf("monitoring the item: %v", err)
		return run
	}

	// Wait until the stream is live and one Publish is held: the publish
	// loop is then blocked on that Publish, so nothing the client sends can
	// disturb the order the scenario is about to command.
	if why := waitOn("no notification reached the application before the delete",
		func() bool { return srv.rec.deliveryCount() > 0 }); why != "" {
		run.Post.HarnessErr = why
		return run
	}
	if why := waitOn("the client never left a publish outstanding for the server to hold",
		func() bool { return srv.rec.heldPublishSeq() != 0 }); why != "" {
		run.Post.HarnessErr = why
		return run
	}

	var held runRecord
	srv.snapshot(&held)
	run.HeldRequests = len(held.Requests)

	if err := sub.Cancel(ctx); err != nil {
		run.Post.HarnessErr = fmt.Sprintf("the delete of the subscription failed: %v", err)
		return run
	}
	sub2, err := c.Subscribe(ctx, params, notifs)
	if err != nil {
		run.ClientErr = "subscribing to the replacement: " + err.Error()
		return run
	}
	if _, err := sub2.Monitor(ctx, ua.TimestampsToReturnBoth, item); err != nil {
		run.ClientErr = "monitoring the item on the replacement: " + err.Error()
		return run
	}

	// The freeze the count and the deliveries are read from: everything
	// past it happened after the fault was commanded.
	srv.snapshot(&run.Faulted)

	if !srv.answerHeldPublish(ua.StatusBadNoSubscription) {
		run.Post.HarnessErr = "the scripted server was not holding the Publish to answer"
		return run
	}

	deadline := time.After(noSubscriptionBound)
	for {
		srv.snapshot(&run.Post)
		if len(freshDeliveriesAfterFault(run)) > 0 {
			break
		}
		select {
		case <-deadline:
			// the bound is up: Post is what the client did with it
			return run
		case <-time.After(20 * time.Millisecond):
		}
	}
	// A grace period, so the Publishes the client sent while the row was
	// noticing the delivery are recorded in Post too.
	time.Sleep(300 * time.Millisecond)
	srv.snapshot(&run.Post)
	return run
}

// waitOn polls ok until it holds and returns "" then, or the reason when the
// deadline passes first.
func waitOn(reason string, ok func() bool) string {
	deadline := time.After(10 * time.Second)
	for !ok() {
		select {
		case <-deadline:
			return reason
		case <-time.After(10 * time.Millisecond):
		}
	}
	return ""
}
