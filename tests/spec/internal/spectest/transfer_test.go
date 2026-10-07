package spectest

import (
	"context"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/ua"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	valueAfterReconnect int32 = 9101
	valueRetained       int32 = 9102
)

var _ = Describe("ScriptedServer transfer, delete and redirect", func() {
	It("answers the old session's ActivateSession with Bad_SessionIdInvalid and accepts a fresh session after RedirectTo", func() {
		env := Start(GinkgoT())
		second := env.StartServer()
		env.Relay.RedirectTo(second.Address())
		m := env.Mark()

		env.Relay.Cut()
		env.WaitUntilReconnected()

		Eventually(func() bool {
			activateAnsweredInvalid := false
			sawCreateSession := false
			for _, request := range env.Recorder.RequestsSince(m) {
				message, forwarded := request.Message()
				if !forwarded {
					continue
				}
				if _, isActivate := message.(*ua.ActivateSessionRequest); !isActivate {
					continue
				}
				for _, response := range env.Recorder.ResponsesSince(m) {
					answer, answerForwarded := response.Message()
					if !answerForwarded || response.RequestID != request.RequestID {
						continue
					}
					if header := answer.Header(); header != nil && header.ServiceResult == ua.StatusBadSessionIDInvalid {
						activateAnsweredInvalid = true
					}
				}
			}
			for _, record := range env.Recorder.ResponsesSince(m) {
				message, forwarded := record.Message()
				if !forwarded {
					continue
				}
				if _, isCreate := message.(*ua.CreateSessionResponse); isCreate {
					sawCreateSession = true
				}
			}
			return activateAnsweredInvalid && sawCreateSession
		}).WithTimeout(specWait).Should(BeTrue(),
			"after the redirect the old session's ActivateSession was never answered Bad_SessionIdInvalid followed by a CreateSession")

		created := second.WaitCreatedSubscription(m)
		var monitoredStatus *ua.StatusCode
		for _, record := range env.Recorder.ResponsesSince(m) {
			message, forwarded := record.Message()
			if !forwarded {
				continue
			}
			if response, isMonitor := message.(*ua.CreateMonitoredItemsResponse); isMonitor {
				status := response.ResponseHeader.ServiceResult
				monitoredStatus = &status
			}
		}
		Expect(monitoredStatus).NotTo(BeNil(),
			"the client created no monitored item on the second server")
		Expect(*monitoredStatus).To(Equal(ua.StatusOK),
			"the CreateMonitoredItems on the second server was not answered Good")

		second.WaitHeldPublish().Answer(created, valueAfterReconnect)
		Eventually(func() []int32 { return env.ReceivedSince(m) }).WithTimeout(specWait).
			Should(Equal([]int32{valueAfterReconnect}),
				"the client did not deliver the value answered on the second server's subscription")
		Expect(env.Server.UnusedScripts()).To(BeEmpty(),
			"the first server holds no unused script: %v", env.Server.UnusedScripts())
		Expect(second.UnusedScripts()).To(BeEmpty(),
			"the second server holds no unused script: %v", second.UnusedScripts())
	})

	It("answers a queued transfer refusal with the scripted status per id", func() {
		env := Start(GinkgoT())
		env.Server.QueueTransferRefusal(ua.StatusBadUserAccessDenied)

		response, err := sendTransfer(env.Client, []uint32{1, 2})

		Expect(err).NotTo(HaveOccurred(), "the TransferSubscriptions failed: %v", err)
		Expect(response.Results).To(HaveLen(2),
			"the refusal must answer one TransferResult per requested id")
		for i, result := range response.Results {
			Expect(result.StatusCode).To(Equal(ua.StatusBadUserAccessDenied),
				"transfer result %d carries %v, want the scripted refusal status", i, result.StatusCode)
		}
		Expect(env.Server.UnusedScripts()).To(BeEmpty(),
			"the queued refusal was consumed, so no script stays unused: %v", env.Server.UnusedScripts())
	})

	It("answers an unqueued TransferSubscriptions with Bad_ServiceUnsupported", func() {
		env := Start(GinkgoT())
		m := env.Mark()

		_, err := sendTransfer(env.Client, []uint32{1})
		Expect(err).To(MatchError(ua.StatusBadServiceUnsupported),
			"with nothing queued, TransferSubscriptions must get the scripted server's default answer")

		var transferRequestID uint32
		sawRequest := false
		for _, record := range env.Recorder.RequestsSince(m) {
			message, forwarded := record.Message()
			if !forwarded {
				continue
			}
			if _, isTransfer := message.(*ua.TransferSubscriptionsRequest); isTransfer {
				transferRequestID = record.RequestID
				sawRequest = true
			}
		}
		Expect(sawRequest).To(BeTrue(), "the recorder saw no TransferSubscriptions request")
		var fault *ua.ServiceFault
		for _, record := range env.Recorder.ResponsesSince(m) {
			if record.RequestID != transferRequestID {
				continue
			}
			message, forwarded := record.Message()
			if !forwarded {
				continue
			}
			if serviceFault, isFault := message.(*ua.ServiceFault); isFault {
				fault = serviceFault
			}
		}
		Expect(fault).NotTo(BeNil(),
			"the unqueued TransferSubscriptions was not answered with a ServiceFault")
		Expect(fault.ResponseHeader.ServiceResult).To(Equal(ua.StatusBadServiceUnsupported),
			"the recorded ServiceFault carries %v, want Bad_ServiceUnsupported", fault.ResponseHeader.ServiceResult)
	})

	It("lists an unused queued transfer answer in UnusedScripts", func() {
		env := Start(GinkgoT())
		env.Server.QueueTransferRefusal(ua.StatusBadUserAccessDenied)

		Expect(env.Server.UnusedScripts()).To(ConsistOf("queued transfer answer never used"),
			"the unused queued transfer answer must appear in UnusedScripts")
	})

	It("answers DeleteSubscriptions Good for a live id and Bad_SubscriptionIDInvalid for an unknown one, and never reuses an id", func() {
		env := Start(GinkgoT())
		live := env.Subscription()

		deleteResponse, err := sendDelete(env.Client, []uint32{live.ID(), live.ID() + 100})
		Expect(err).NotTo(HaveOccurred(), "the DeleteSubscriptions failed: %v", err)
		Expect(deleteResponse.Results).To(Equal([]ua.StatusCode{ua.StatusOK, ua.StatusBadSubscriptionIDInvalid}),
			"DeleteSubscriptions must answer Good for the live id and Bad_SubscriptionIDInvalid for the unknown one")

		seen := map[uint32]bool{live.ID(): true}
		for i := 0; i < 100; i++ {
			created, createErr := sendCreate(env.Client)
			Expect(createErr).NotTo(HaveOccurred(), "CreateSubscription %d failed: %v", i, createErr)
			Expect(seen[created.SubscriptionID]).To(BeFalse(),
				"the server reused subscription id %d after %d create-delete cycles", created.SubscriptionID, i)
			seen[created.SubscriptionID] = true

			_, deleteErr := sendDelete(env.Client, []uint32{created.SubscriptionID})
			Expect(deleteErr).NotTo(HaveOccurred(), "DeleteSubscriptions %d failed: %v", i, deleteErr)
		}
		Expect(env.Server.UnusedScripts()).To(BeEmpty(),
			"the create-delete cycles used no scripts: %v", env.Server.UnusedScripts())
	})
})

func sendTransfer(client *opcua.Client, subscriptionIDs []uint32) (*ua.TransferSubscriptionsResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), specWait)
	defer cancel()
	request := &ua.TransferSubscriptionsRequest{SubscriptionIDs: subscriptionIDs, SendInitialValues: false}
	var response *ua.TransferSubscriptionsResponse
	err := client.Send(ctx, request, func(v ua.Response) error {
		if transfer, isTransfer := v.(*ua.TransferSubscriptionsResponse); isTransfer {
			response = transfer
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if response == nil {
		return nil, fmt.Errorf("the TransferSubscriptions got no TransferSubscriptionsResponse")
	}
	return response, nil
}

func sendDelete(client *opcua.Client, subscriptionIDs []uint32) (*ua.DeleteSubscriptionsResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), specWait)
	defer cancel()
	request := &ua.DeleteSubscriptionsRequest{SubscriptionIDs: subscriptionIDs}
	var response *ua.DeleteSubscriptionsResponse
	err := client.Send(ctx, request, func(v ua.Response) error {
		if deleted, isDelete := v.(*ua.DeleteSubscriptionsResponse); isDelete {
			response = deleted
		}
		return nil
	})
	return response, err
}

func sendCreate(client *opcua.Client) (*ua.CreateSubscriptionResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), specWait)
	defer cancel()
	request := &ua.CreateSubscriptionRequest{
		RequestedPublishingInterval: 100,
		RequestedLifetimeCount:      lifetimeCount,
		RequestedMaxKeepAliveCount:  maxKeepAliveCount,
	}
	var response *ua.CreateSubscriptionResponse
	err := client.Send(ctx, request, func(v ua.Response) error {
		if created, isCreate := v.(*ua.CreateSubscriptionResponse); isCreate {
			response = created
		}
		return nil
	})
	return response, err
}

func sendMonitor(client *opcua.Client, node *ua.NodeID, subscriptionID uint32) (*ua.CreateMonitoredItemsResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), specWait)
	defer cancel()
	request := &ua.CreateMonitoredItemsRequest{
		SubscriptionID: subscriptionID,
		ItemsToCreate: []*ua.MonitoredItemCreateRequest{
			opcua.NewMonitoredItemCreateRequestWithDefaults(node, ua.AttributeIDValue, monitorClientHandle),
		},
	}
	var response *ua.CreateMonitoredItemsResponse
	err := client.Send(ctx, request, func(v ua.Response) error {
		if monitored, isMonitor := v.(*ua.CreateMonitoredItemsResponse); isMonitor {
			response = monitored
		}
		return nil
	})
	return response, err
}

func monitorHandleOf(message *ua.NotificationMessage) uint32 {
	for _, data := range message.NotificationData {
		if data == nil || data.Value == nil {
			continue
		}
		if change, isDataChange := data.Value.(*ua.DataChangeNotification); isDataChange && len(change.MonitoredItems) > 0 {
			return change.MonitoredItems[0].ClientHandle
		}
	}
	return 0
}

func startTransferServer(t *testing.T) (*ScriptedServer, *opcua.Client, func()) {
	t.Helper()
	ft := &fakeT{}
	srv := newScriptedServer(ft)
	relay, recorder := newRelay(ft, srv.Address(), nil)
	srv.recorder = recorder
	client, err := opcua.NewClient("opc.tcp://"+relay.address(),
		opcua.SecurityMode(ua.MessageSecurityModeNone),
		opcua.AutoReconnect(false),
		opcua.RequestTimeout(clientRequestTimeout))
	if err != nil {
		t.Fatalf("creating the client failed: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), startTimeout)
	if err := client.Connect(ctx); err != nil {
		cancel()
		t.Fatalf("the client never connected through the relay to the fresh server: %v", err)
	}
	cancel()
	closeAll := func() {
		closeCtx, closeCancel := context.WithTimeout(context.Background(), teardownWait)
		defer closeCancel()
		_ = client.Close(closeCtx)
		srv.close()
		for _, cleanup := range ft.cleanups {
			cleanup()
		}
	}
	return srv, client, closeAll
}

func prepareTransferOnFreshServer(t *testing.T, client *opcua.Client, srv *ScriptedServer) {
	t.Helper()
	created, err := sendCreate(client)
	if err != nil {
		t.Fatalf("the CreateSubscription on the fresh server failed: %v", err)
	}
	monitored, err := sendMonitor(client, srv.node, created.SubscriptionID)
	if err != nil {
		t.Fatalf("the CreateMonitoredItems on the fresh server failed: %v", err)
	}
	if len(monitored.Results) != 1 || monitored.Results[0].StatusCode != ua.StatusOK {
		t.Fatalf("the CreateMonitoredItems answered %+v, want one Good result", monitored.Results)
	}
	deleted, err := sendDelete(client, []uint32{created.SubscriptionID})
	if err != nil {
		t.Fatalf("the DeleteSubscriptions on the fresh server failed: %v", err)
	}
	if !slices.Equal(deleted.Results, []ua.StatusCode{ua.StatusOK}) {
		t.Fatalf("the DeleteSubscriptions answered %v, want [Good]", deleted.Results)
	}
}

func holdRawPublish(client *opcua.Client, srv *ScriptedServer) (HeldPublish, func() error) {
	ctx, cancel := context.WithTimeout(context.Background(), specWait)
	done := make(chan error, 1)
	go func() {
		done <- client.Send(ctx, &ua.PublishRequest{
			RequestHeader:                &ua.RequestHeader{TimeoutHint: uint32(specWait.Milliseconds())},
			SubscriptionAcknowledgements: []*ua.SubscriptionAcknowledgement{},
		}, func(ua.Response) error { return nil })
	}()
	held := srv.WaitHeldPublish()
	wait := func() error {
		defer cancel()
		return <-done
	}
	return held, wait
}

func TestWaitCreatedSubscriptionReturnsTheFirstSubscriptionAfterTheMark(t *testing.T) {
	env := Start(t)
	before, err := sendCreate(env.Client)
	if err != nil {
		t.Fatalf("the CreateSubscription before the mark failed: %v", err)
	}
	m := env.Mark()
	firstAfter, err := sendCreate(env.Client)
	if err != nil {
		t.Fatalf("the first CreateSubscription after the mark failed: %v", err)
	}
	secondAfter, err := sendCreate(env.Client)
	if err != nil {
		t.Fatalf("the second CreateSubscription after the mark failed: %v", err)
	}

	waited := env.Server.WaitCreatedSubscription(m)

	if waited.ID() != firstAfter.SubscriptionID {
		t.Fatalf("WaitCreatedSubscription returned subscription %d, want %d, the first subscription created after the mark; a subscription created before the mark, such as %d or %d, must not be returned", waited.ID(), firstAfter.SubscriptionID, env.Subscription().ID(), before.SubscriptionID)
	}
	_ = secondAfter
}

func TestWaitCreatedSubscriptionSkipsTheOtherServersSubscriptions(t *testing.T) {
	env := Start(t)
	second := env.StartServer()
	m := env.Mark()
	_, err := sendCreate(env.Client)
	if err != nil {
		t.Fatalf("the CreateSubscription on the first server failed: %v", err)
	}

	second.heldWait = 300 * time.Millisecond
	fake := &fakeT{}
	second.t = fake
	if !fatalPanics(func() { second.WaitCreatedSubscription(m) }) {
		t.Fatalf("WaitCreatedSubscription returned a subscription created on another server")
	}
	if len(fake.fatals) != 1 || fake.fatals[0] != "client created no subscription" {
		t.Fatalf("WaitCreatedSubscription failed with %q, want the plain no-subscription failure for a server that saw no subscription of its own", fake.fatals)
	}
}

func TestUnusedScriptsListsStagedRetainsOnThePendingTransfer(t *testing.T) {
	env := Start(t)
	moved := env.Server.QueueTransferSuccess(5, 6)
	moved.Retain(6, valueRetained)

	scripts := env.Server.UnusedScripts()
	if !slices.Contains(scripts, "queued transfer answer never used") {
		t.Fatalf("UnusedScripts %v lists no unused queued transfer answer", scripts)
	}
	if !slices.Contains(scripts, "staged retained 6 for the pending transfer") {
		t.Fatalf("UnusedScripts %v lists no retained message staged on the pending transfer handle", scripts)
	}
}

func TestTransferCounterStartsAboveTheAvailableNumbers(t *testing.T) {
	srv, client, closeAll := startTransferServer(t)
	defer closeAll()
	prepareTransferOnFreshServer(t, client, srv)

	moved := srv.QueueTransferSuccess(5, 6)
	transfer, err := sendTransfer(client, []uint32{1})
	if err != nil {
		t.Fatalf("the TransferSubscriptions on the fresh server failed: %v", err)
	}
	if len(transfer.Results) != 1 || transfer.Results[0].StatusCode != ua.StatusOK {
		t.Fatalf("the transfer answered %+v, want one Good result", transfer.Results)
	}
	if moved.ID() != 1 {
		t.Fatalf("the transferred handle names subscription %d, want 1, the id the transfer created", moved.ID())
	}

	held, wait := holdRawPublish(client, srv)
	held.Answer(moved, 9400)
	if publishErr := wait(); publishErr != nil {
		t.Fatalf("the raw Publish request failed after the Answer: %v", publishErr)
	}
	answered := srv.recorder.answeredPublishes()
	if len(answered) == 0 {
		t.Fatalf("the recorder saw no answered Publish response on the fresh server")
	}
	last := answered[len(answered)-1]
	if last.sequenceNumber != 7 || last.value != 9400 {
		t.Fatalf("the first Answer after the transfer used sequence %d, want 7 (max(available)+1)", last.sequenceNumber)
	}
}

func TestTransferSuccessAnswersFromTheQueue(t *testing.T) {
	srv, client, closeAll := startTransferServer(t)
	defer closeAll()
	prepareTransferOnFreshServer(t, client, srv)

	moved := srv.QueueTransferSuccess(5, 6)
	moved.Retain(6, valueRetained)
	transfer, err := sendTransfer(client, []uint32{1})
	if err != nil {
		t.Fatalf("the TransferSubscriptions on the fresh server failed: %v", err)
	}
	if len(transfer.Results) != 1 || transfer.Results[0].StatusCode != ua.StatusOK {
		t.Fatalf("the transfer answered %+v, want one Good result", transfer.Results)
	}
	if got := transfer.Results[0].AvailableSequenceNumbers; !slices.Equal(got, []uint32{5, 6}) {
		t.Fatalf("the transfer result carries available sequence numbers %v, want [5 6]", got)
	}
	if moved.ID() != 1 {
		t.Fatalf("the transferred handle names subscription %d, want 1, the id the transfer created", moved.ID())
	}

	held, wait := holdRawPublish(client, srv)

	republishCtx, republishCancel := context.WithTimeout(context.Background(), specWait)
	defer republishCancel()
	republishRequest := &ua.RepublishRequest{SubscriptionID: 1, RetransmitSequenceNumber: 6}
	var republishResponse ua.Response
	republishErr := client.Send(republishCtx, republishRequest, func(v ua.Response) error {
		republishResponse = v
		return nil
	})
	if republishErr != nil {
		t.Fatalf("the Republish for the transferred id failed: %v", republishErr)
	}
	republish, isRepublish := republishResponse.(*ua.RepublishResponse)
	if !isRepublish {
		t.Fatalf("the Republish answered with a %T, want a RepublishResponse", republishResponse)
	}
	value, carries := dataChangeValue(republish.NotificationMessage)
	if !carries || value != any(valueRetained) {
		t.Fatalf("the Republish for (1, 6) answered with %v, want the retained value %d", value, valueRetained)
	}
	if handle := monitorHandleOf(republish.NotificationMessage); handle != monitorClientHandle {
		t.Fatalf("the Republish for (1, 6) answered with client handle %d, want the recorded handle %d", handle, monitorClientHandle)
	}

	held.Answer(moved, 9300)
	if publishErr := wait(); publishErr != nil {
		t.Fatalf("the raw Publish request failed after the Answer: %v", publishErr)
	}
	answered := srv.recorder.answeredPublishes()
	if len(answered) == 0 {
		t.Fatalf("the recorder saw no answered Publish response on the fresh server")
	}
	last := answered[len(answered)-1]
	if last.sequenceNumber != 7 || last.value != 9300 {
		t.Fatalf("the Answer after the transfer used sequence %d, want 7 (max(available)+1)", last.sequenceNumber)
	}
	if scripts := srv.UnusedScripts(); len(scripts) != 0 {
		t.Fatalf("the transfer answer and the retained message were used, so UnusedScripts must be empty: %v", scripts)
	}
}

func TestTransferSuccessAnswersTwoIdsWithALiveSubscriptionForEach(t *testing.T) {
	srv, client, closeAll := startTransferServer(t)
	defer closeAll()
	prepareTransferOnFreshServer(t, client, srv)

	moved := srv.QueueTransferSuccess(5, 6)
	moved.Retain(6, valueRetained)
	transfer, err := sendTransfer(client, []uint32{1, 2})
	if err != nil {
		t.Fatalf("the two-id TransferSubscriptions failed: %v", err)
	}
	if len(transfer.Results) != 2 {
		t.Fatalf("the transfer answered %d results, want one per requested id", len(transfer.Results))
	}
	for i, result := range transfer.Results {
		if result.StatusCode != ua.StatusOK {
			t.Fatalf("transfer result %d carries %v, want Good", i, result.StatusCode)
		}
		if !slices.Equal(result.AvailableSequenceNumbers, []uint32{5, 6}) {
			t.Fatalf("transfer result %d carries available sequence numbers %v, want [5 6]", i, result.AvailableSequenceNumbers)
		}
	}
	if moved.ID() != 1 {
		t.Fatalf("the transferred handle names subscription %d, want 1: the deferred subscription binds to the first id", moved.ID())
	}

	first, err := sendRepublishOn(client, 1, 6)
	if err != nil {
		t.Fatalf("the Republish for the first transferred id failed: %v", err)
	}
	if first.NotificationMessage == nil || first.NotificationMessage.SequenceNumber != 6 {
		t.Fatalf("the Republish for (1, 6) answered with %+v, want the staged sequence number 6", first.NotificationMessage)
	}
	second, err := sendRepublishOn(client, 2, 6)
	if err == nil {
		t.Fatalf("the Republish for the second id answered with %+v, want Bad_MessageNotAvailable: the second id got a fresh harness subscription with an empty queue", second.NotificationMessage)
	}
	if err == nil || err.Error() != ua.StatusBadMessageNotAvailable.Error() {
		t.Fatalf("the Republish for the second id failed with %v, want Bad_MessageNotAvailable", err)
	}
	if scripts := srv.UnusedScripts(); len(scripts) != 0 {
		t.Fatalf("the two-id transfer left unused scripts: %v", scripts)
	}
}

func sendRepublishOn(client *opcua.Client, subscriptionID, sequenceNumber uint32) (*ua.RepublishResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), specWait)
	defer cancel()
	request := &ua.RepublishRequest{SubscriptionID: subscriptionID, RetransmitSequenceNumber: sequenceNumber}
	var response ua.Response
	err := client.Send(ctx, request, func(v ua.Response) error {
		response = v
		return nil
	})
	if err != nil {
		return nil, err
	}
	republish, isRepublish := response.(*ua.RepublishResponse)
	if !isRepublish {
		return nil, fmt.Errorf("the Republish for (%d, %d) answered with a %T, want a RepublishResponse", subscriptionID, sequenceNumber, response)
	}
	return republish, nil
}
