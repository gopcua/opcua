package spectest

import (
	"context"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/tests/spec/message"
	"github.com/gopcua/opcua/ua"
	"github.com/gopcua/opcua/uacp"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const valueCutTarget int32 = 7004

// cutAtDriver says how one spec makes the client send the message the
// entry's cut fires on: setup runs before the cut is armed, fire after.
type cutAtDriver struct {
	setup func(env *Environment)
	fire  func(env *Environment)
}

func cutAtDriverFor(msg message.Message) cutAtDriver {
	switch msg {
	case message.HEL, message.OpenSecureChannel, message.ActivateSession, message.Read:
		return cutAtDriver{fire: func(env *Environment) { env.Relay.Cut() }}
	case message.CreateSession:
		return cutAtDriver{
			setup: func(env *Environment) {
				env.Relay.RedirectTo(env.StartServer().Address())
			},
			fire: func(env *Environment) { env.Relay.Cut() },
		}
	case message.TransferSubscriptions:
		return cutAtDriver{
			setup: func(env *Environment) {
				second := env.StartServer()
				second.QueueTransferRefusal(ua.StatusBadSubscriptionIDInvalid)
				env.Relay.RedirectTo(second.Address())
			},
			fire: func(env *Environment) { env.Relay.Cut() },
		}
	case message.Republish:
		return cutAtDriver{
			setup: func(env *Environment) {
				last := env.LastSequenceNumber()
				second := env.StartServer()
				moved := second.QueueTransferSuccess(last, last+1)
				moved.Retain(last+1, valueCutTarget)
				env.Relay.RedirectTo(second.Address())
			},
			fire: func(env *Environment) { env.Relay.Cut() },
		}
	case message.CreateSubscription:
		return cutAtDriver{fire: func(env *Environment) {
			notifications := make(chan *opcua.PublishNotificationData, notificationBuffer)
			ctx, cancel := context.WithTimeout(context.Background(), specWait)
			defer cancel()
			_, _ = env.Client.Subscribe(ctx, &opcua.SubscriptionParameters{
				Interval:          defaultPublishingInterval,
				LifetimeCount:     lifetimeCount,
				MaxKeepAliveCount: maxKeepAliveCount,
			}, notifications)
		}}
	case message.CreateMonitoredItems:
		return cutAtDriver{fire: func(env *Environment) {
			notifications := make(chan *opcua.PublishNotificationData, notificationBuffer)
			ctx, cancel := context.WithTimeout(context.Background(), specWait)
			defer cancel()
			subscription, subscribeErr := env.Client.Subscribe(ctx, &opcua.SubscriptionParameters{
				Interval:          defaultPublishingInterval,
				LifetimeCount:     lifetimeCount,
				MaxKeepAliveCount: maxKeepAliveCount,
			}, notifications)
			if subscribeErr == nil {
				_, _ = subscription.Monitor(ctx, ua.TimestampsToReturnBoth,
					opcua.NewMonitoredItemCreateRequestWithDefaults(env.Server.node, ua.AttributeIDValue, monitorClientHandle))
			}
		}}
	case message.DeleteSubscriptions:
		return cutAtDriver{fire: func(env *Environment) {
			ctx, cancel := context.WithTimeout(context.Background(), specWait)
			defer cancel()
			_ = env.ClientSubscription().Cancel(ctx)
		}}
	case message.Publish:
		var held HeldPublish
		return cutAtDriver{
			setup: func(env *Environment) {
				held = env.Server.WaitHeldPublish()
			},
			fire: func(env *Environment) {
				held.Answer(env.Subscription(), valueCutTarget)
			},
		}
	case message.CloseSession, message.CloseSecureChannel:
		return cutAtDriver{fire: func(env *Environment) {
			ctx, cancel := context.WithTimeout(context.Background(), specWait)
			defer cancel()
			_ = env.Client.Close(ctx)
		}}
	}
	Fail("no driver for message " + msg.String())
	return cutAtDriver{}
}

var _ = DescribeTable("Relay CutAt on every message",
	func(msg message.Message, moment Moment) {
		var opts []Option
		if msg == message.CloseSession || msg == message.CloseSecureChannel {
			// The driver closes the client while the cut has severed its
			// connection; a redialling client would race Close's c.conn read
			// against Dial's write (issue #883), and nothing these entries
			// assert needs a reconnect.
			opts = append(opts, WithClientOptions(opcua.AutoReconnect(false)))
		}
		env := Start(GinkgoT(), opts...)
		driver := cutAtDriverFor(msg)
		if driver.setup != nil {
			driver.setup(env)
		}
		env.Relay.CutAt(moment, msg)
		m := env.Mark()
		driver.fire(env)

		Eventually(func(g Gomega) {
			g.Expect(env.Relay.ArmedCuts()).To(BeEmpty(),
				"the cut armed for %s at %s never fired: %v", msg, momentName(moment), env.Relay.ArmedCuts())
		}, specWait).Should(Succeed())

		if msg == message.HEL {
			assertHelCut(env, m, moment)
			return
		}
		switch moment {
		case BeforeRequestReachesServer:
			assertRequestDroppedCut(env, m, msg)
		case ResponseNeverReachesClient:
			assertResponseDroppedCut(env, m, msg)
		case AfterResponseReachesClient:
			assertResponseDeliveredCut(env, m, msg)
		}
	},
	Entry("HEL before the request reaches the server", message.HEL, BeforeRequestReachesServer),
	Entry("HEL with the response never reaching the client", message.HEL, ResponseNeverReachesClient),
	Entry("HEL after the response reaches the client", message.HEL, AfterResponseReachesClient),
	Entry("OpenSecureChannel before the request reaches the server", message.OpenSecureChannel, BeforeRequestReachesServer),
	Entry("OpenSecureChannel with the response never reaching the client", message.OpenSecureChannel, ResponseNeverReachesClient),
	Entry("OpenSecureChannel after the response reaches the client", message.OpenSecureChannel, AfterResponseReachesClient),
	Entry("CreateSession before the request reaches the server", message.CreateSession, BeforeRequestReachesServer),
	Entry("CreateSession with the response never reaching the client", message.CreateSession, ResponseNeverReachesClient),
	Entry("CreateSession after the response reaches the client", message.CreateSession, AfterResponseReachesClient),
	Entry("ActivateSession before the request reaches the server", message.ActivateSession, BeforeRequestReachesServer),
	Entry("ActivateSession with the response never reaching the client", message.ActivateSession, ResponseNeverReachesClient),
	Entry("ActivateSession after the response reaches the client", message.ActivateSession, AfterResponseReachesClient),
	Entry("CloseSession before the request reaches the server", message.CloseSession, BeforeRequestReachesServer),
	Entry("CloseSession with the response never reaching the client", message.CloseSession, ResponseNeverReachesClient),
	Entry("CloseSession after the response reaches the client", message.CloseSession, AfterResponseReachesClient),
	Entry("Read before the request reaches the server", message.Read, BeforeRequestReachesServer),
	Entry("Read with the response never reaching the client", message.Read, ResponseNeverReachesClient),
	Entry("Read after the response reaches the client", message.Read, AfterResponseReachesClient),
	Entry("CreateSubscription before the request reaches the server", message.CreateSubscription, BeforeRequestReachesServer),
	Entry("CreateSubscription with the response never reaching the client", message.CreateSubscription, ResponseNeverReachesClient),
	Entry("CreateSubscription after the response reaches the client", message.CreateSubscription, AfterResponseReachesClient),
	Entry("CreateMonitoredItems before the request reaches the server", message.CreateMonitoredItems, BeforeRequestReachesServer),
	Entry("CreateMonitoredItems with the response never reaching the client", message.CreateMonitoredItems, ResponseNeverReachesClient),
	Entry("CreateMonitoredItems after the response reaches the client", message.CreateMonitoredItems, AfterResponseReachesClient),
	Entry("DeleteSubscriptions before the request reaches the server", message.DeleteSubscriptions, BeforeRequestReachesServer),
	Entry("DeleteSubscriptions with the response never reaching the client", message.DeleteSubscriptions, ResponseNeverReachesClient),
	Entry("DeleteSubscriptions after the response reaches the client", message.DeleteSubscriptions, AfterResponseReachesClient),
	Entry("Publish before the request reaches the server", message.Publish, BeforeRequestReachesServer),
	Entry("Publish with the response never reaching the client", message.Publish, ResponseNeverReachesClient),
	Entry("Publish after the response reaches the client", message.Publish, AfterResponseReachesClient),
	Entry("Republish before the request reaches the server", message.Republish, BeforeRequestReachesServer),
	Entry("Republish with the response never reaching the client", message.Republish, ResponseNeverReachesClient),
	Entry("Republish after the response reaches the client", message.Republish, AfterResponseReachesClient),
	Entry("TransferSubscriptions before the request reaches the server", message.TransferSubscriptions, BeforeRequestReachesServer),
	Entry("TransferSubscriptions with the response never reaching the client", message.TransferSubscriptions, ResponseNeverReachesClient),
	Entry("TransferSubscriptions after the response reaches the client", message.TransferSubscriptions, AfterResponseReachesClient),
	Entry("CloseSecureChannel before the request reaches the server", message.CloseSecureChannel, BeforeRequestReachesServer),
)

func momentName(moment Moment) string {
	switch moment {
	case BeforeRequestReachesServer:
		return "BeforeRequestReachesServer"
	case ResponseNeverReachesClient:
		return "ResponseNeverReachesClient"
	case AfterResponseReachesClient:
		return "AfterResponseReachesClient"
	}
	return "unknown"
}

func messageOfResponse(service any) (message.Message, bool) {
	switch service.(type) {
	case *ua.OpenSecureChannelResponse:
		return message.OpenSecureChannel, true
	case *ua.CreateSessionResponse:
		return message.CreateSession, true
	case *ua.ActivateSessionResponse:
		return message.ActivateSession, true
	case *ua.CloseSessionResponse:
		return message.CloseSession, true
	case *ua.ReadResponse:
		return message.Read, true
	case *ua.CreateSubscriptionResponse:
		return message.CreateSubscription, true
	case *ua.CreateMonitoredItemsResponse:
		return message.CreateMonitoredItems, true
	case *ua.DeleteSubscriptionsResponse:
		return message.DeleteSubscriptions, true
	case *ua.PublishResponse:
		return message.Publish, true
	case *ua.RepublishResponse:
		return message.Republish, true
	case *ua.TransferSubscriptionsResponse:
		return message.TransferSubscriptions, true
	}
	return 0, false
}

func cutAtRequestsSince(env *Environment, m Mark, message message.Message) []ServiceRecord[ua.Request] {
	var matched []ServiceRecord[ua.Request]
	for _, record := range env.Recorder.RequestsSince(m) {
		decoded, ok := record.Message()
		if !ok {
			continue
		}
		if named, isNamed := messageOfRequest(decoded); isNamed && named == message {
			matched = append(matched, record)
		}
	}
	return matched
}

func cutAtResponsesSince(env *Environment, m Mark, message message.Message) []ServiceRecord[ua.Response] {
	var matched []ServiceRecord[ua.Response]
	for _, record := range env.Recorder.ResponsesSince(m) {
		decoded, ok := record.Message()
		if !ok {
			continue
		}
		if named, isNamed := messageOfResponse(decoded); isNamed && named == message {
			matched = append(matched, record)
		}
	}
	return matched
}

func transportsOfTypeSince(env *Environment, m Mark, typ TransportType) []TransportRecord {
	var matched []TransportRecord
	for _, record := range env.Recorder.Transport() {
		if record.Order > m.order && record.Type == typ {
			matched = append(matched, record)
		}
	}
	return matched
}

func cutAtPairedRequest(env *Environment, connection int, requestID uint32, message message.Message) (ServiceRecord[ua.Request], bool) {
	for _, record := range env.Recorder.Requests() {
		if record.Connection != connection || record.RequestID != requestID {
			continue
		}
		decoded, ok := record.Message()
		if !ok {
			continue
		}
		if named, isNamed := messageOfRequest(decoded); isNamed && named == message {
			return record, true
		}
	}
	return ServiceRecord[ua.Request]{}, false
}

func helRecordOnConnection(env *Environment, connection int) (TransportRecord, bool) {
	for _, record := range env.Recorder.Transport() {
		if record.Connection == connection && record.Type == HEL {
			return record, true
		}
	}
	return TransportRecord{}, false
}

func ackRecordsOnConnection(env *Environment, connection int) []TransportRecord {
	var matched []TransportRecord
	for _, record := range env.Recorder.Transport() {
		if record.Connection == connection && record.Type == ACK {
			matched = append(matched, record)
		}
	}
	return matched
}

func assertRequestDroppedCut(env *Environment, m Mark, message message.Message) {
	var target ServiceRecord[ua.Request]
	Eventually(func(g Gomega) {
		var dropped []ServiceRecord[ua.Request]
		for _, record := range cutAtRequestsSince(env, m, message) {
			if record.Fate == Dropped {
				dropped = append(dropped, record)
			}
		}
		g.Expect(dropped).To(HaveLen(1),
			"%s requests with Fate Dropped since the mark: %d, want exactly the one the cut dropped", message, len(dropped))
		target = dropped[0]
	}, specWait).Should(Succeed())

	Expect(env.Recorder.ConnectionStateOf(target.Connection)).To(Equal(Closed),
		"the relay connection %d the dropped %s request rode on is not closed", target.Connection, message)
	Expect(responseOf(env.Recorder.Responses(), target.Connection, target.RequestID)).To(BeNil(),
		"a response with the dropped %s request's id %d exists although the request never reached the server", message, target.RequestID)
	decoded, ok := target.Message()
	Expect(ok).To(BeTrue(),
		"the dropped %s record does not yield its decoded message", message)
	named, isNamed := messageOfRequest(decoded)
	Expect(isNamed && named == message).To(BeTrue(),
		"the dropped record decodes to %T, want the %s request", decoded, message)
}

func assertResponseDroppedCut(env *Environment, m Mark, message message.Message) {
	var target ServiceRecord[ua.Response]
	Eventually(func(g Gomega) {
		var dropped []ServiceRecord[ua.Response]
		for _, record := range cutAtResponsesSince(env, m, message) {
			if record.Fate == Dropped {
				dropped = append(dropped, record)
			}
		}
		g.Expect(dropped).To(HaveLen(1),
			"%s responses with Fate Dropped since the mark: %d, want exactly the one the cut dropped", message, len(dropped))
		target = dropped[0]
	}, specWait).Should(Succeed())

	Expect(env.Recorder.ConnectionStateOf(target.Connection)).To(Equal(Closed),
		"the relay connection %d the dropped %s response rode on is not closed", target.Connection, message)
	request, found := cutAtPairedRequest(env, target.Connection, target.RequestID, message)
	Expect(found).To(BeTrue(),
		"no recorded %s request pairs with the dropped response (connection %d, request id %d)", message, target.Connection, target.RequestID)
	Expect(request.Fate).To(Equal(Forwarded),
		"the %s request the dropped response answers is recorded %s, want Forwarded", message, request.Fate)
	decoded, ok := target.Message()
	Expect(ok).To(BeTrue(),
		"the dropped %s response record does not yield its decoded message", message)
	named, isNamed := messageOfResponse(decoded)
	Expect(isNamed && named == message).To(BeTrue(),
		"the dropped record decodes to %T, want the %s response", decoded, message)
}

func assertResponseDeliveredCut(env *Environment, m Mark, message message.Message) {
	var target ServiceRecord[ua.Response]
	Eventually(func(g Gomega) {
		var delivered []ServiceRecord[ua.Response]
		for _, record := range cutAtResponsesSince(env, m, message) {
			if record.Fate == Forwarded && env.Recorder.ConnectionStateOf(record.Connection) == Closed {
				delivered = append(delivered, record)
			}
		}
		g.Expect(delivered).To(HaveLen(1),
			"%s responses with Fate Forwarded on a closed connection since the mark: %d, want exactly the one the cut delivered before closing", message, len(delivered))
		target = delivered[0]
	}, specWait).Should(Succeed())

	Expect(env.Recorder.ConnectionStateOf(target.Connection)).To(Equal(Closed),
		"the relay connection %d the delivered %s response rode on is not closed", target.Connection, message)
	request, found := cutAtPairedRequest(env, target.Connection, target.RequestID, message)
	Expect(found).To(BeTrue(),
		"no recorded %s request pairs with the delivered response (connection %d, request id %d)", message, target.Connection, target.RequestID)
	Expect(request.Fate).To(Equal(Forwarded),
		"the %s request the delivered response answers is recorded %s, want Forwarded", message, request.Fate)
	decoded, ok := target.Message()
	Expect(ok).To(BeTrue(),
		"the delivered %s response record does not yield its decoded message", message)
	named, isNamed := messageOfResponse(decoded)
	Expect(isNamed && named == message).To(BeTrue(),
		"the delivered record decodes to %T, want the %s response", decoded, message)
}

func assertHelCut(env *Environment, m Mark, moment Moment) {
	switch moment {
	case BeforeRequestReachesServer:
		var target TransportRecord
		Eventually(func(g Gomega) {
			var dropped []TransportRecord
			for _, record := range transportsOfTypeSince(env, m, HEL) {
				if record.Fate == Dropped {
					dropped = append(dropped, record)
				}
			}
			g.Expect(dropped).To(HaveLen(1),
				"HEL transport records with Fate Dropped since the mark: %d, want exactly the one the cut dropped", len(dropped))
			target = dropped[0]
		}, specWait).Should(Succeed())
		Expect(env.Recorder.ConnectionStateOf(target.Connection)).To(Equal(Closed),
			"the relay connection %d the dropped HEL rode on is not closed", target.Connection)
		Expect(ackRecordsOnConnection(env, target.Connection)).To(BeEmpty(),
			"connection %d carries an ACK although its HEL was dropped before it reached the server", target.Connection)
		Expect(target.Type).To(Equal(HEL),
			"the dropped record is a %v transport record, want a HEL", target.Type)
	case ResponseNeverReachesClient:
		var target TransportRecord
		Eventually(func(g Gomega) {
			var dropped []TransportRecord
			for _, record := range transportsOfTypeSince(env, m, ACK) {
				if record.Fate == Dropped {
					dropped = append(dropped, record)
				}
			}
			g.Expect(dropped).To(HaveLen(1),
				"ACK transport records with Fate Dropped since the mark: %d, want exactly the one the cut dropped", len(dropped))
			target = dropped[0]
		}, specWait).Should(Succeed())
		Expect(env.Recorder.ConnectionStateOf(target.Connection)).To(Equal(Closed),
			"the relay connection %d the dropped ACK rode on is not closed", target.Connection)
		hel, found := helRecordOnConnection(env, target.Connection)
		Expect(found).To(BeTrue(),
			"connection %d carries no HEL although its ACK was recorded", target.Connection)
		Expect(hel.Fate).To(Equal(Forwarded),
			"the HEL whose ACK the cut dropped is recorded %s, want Forwarded", hel.Fate)
		Expect(target.Type).To(Equal(ACK),
			"the dropped record is a %v transport record, want an ACK", target.Type)
	case AfterResponseReachesClient:
		var target TransportRecord
		Eventually(func(g Gomega) {
			var delivered []TransportRecord
			for _, record := range transportsOfTypeSince(env, m, ACK) {
				if record.Fate == Forwarded && env.Recorder.ConnectionStateOf(record.Connection) == Closed {
					delivered = append(delivered, record)
				}
			}
			g.Expect(delivered).To(HaveLen(1),
				"ACK transport records with Fate Forwarded on a closed connection since the mark: %d, want exactly the one the cut delivered before closing", len(delivered))
			target = delivered[0]
		}, specWait).Should(Succeed())
		Expect(env.Recorder.ConnectionStateOf(target.Connection)).To(Equal(Closed),
			"the relay connection %d the delivered ACK rode on is not closed", target.Connection)
		hel, found := helRecordOnConnection(env, target.Connection)
		Expect(found).To(BeTrue(),
			"connection %d carries no HEL although its ACK was recorded", target.Connection)
		Expect(hel.Fate).To(Equal(Forwarded),
			"the HEL whose ACK the cut delivered is recorded %s, want Forwarded", hel.Fate)
		Expect(target.Type).To(Equal(ACK),
			"the delivered record is a %v transport record, want an ACK", target.Type)
	}
}

var _ = Describe("Relay CutAt transport pairing", func() {
	It("fires a HEL response cut on the ACK of the connection that carried the HEL", func() {
		relay, recorder, _ := newInjectedRelay()
		relay.CutAt(AfterResponseReachesClient, message.HEL)
		finished := false

		hello, err := helloWire()
		Expect(err).NotTo(HaveOccurred(), "encoding the HEL failed")
		recorder.observe(0, clientToServer, hello)
		ack, err := ackWire()
		Expect(err).NotTo(HaveOccurred(), "encoding the ACK failed")
		var written [][]byte
		Expect(recorder.forward(0, serverToClient, ack, func(message []byte) error {
			written = append(written, message)
			return nil
		}, func(armedCut) error {
			finished = true
			return nil
		})).To(Succeed(), "observing the ACK failed")

		Expect(finished).To(BeTrue(),
			"the HEL response cut did not fire on the ACK answering the HEL of its own connection")
		Expect(written).To(Equal([][]byte{ack}),
			"the relay did not write the ACK the cut fired on")
		Expect(relay.ArmedCuts()).To(BeEmpty(),
			"ArmedCuts still lists the cut after it fired")
		var ackRecords []TransportRecord
		for _, record := range recorder.Transport() {
			if record.Type == ACK {
				ackRecords = append(ackRecords, record)
			}
		}
		Expect(ackRecords).To(HaveLen(1),
			"the ACK the cut fired on left %d transport records, want 1", len(ackRecords))
		Expect(ackRecords[0].Fate).To(Equal(Forwarded),
			"the ACK the cut fired on is not recorded as Forwarded")
	})

	It("does not fire a HEL response cut on an ACK of a connection that never carried a HEL", func() {
		relay, recorder, _ := newInjectedRelay()
		relay.CutAt(AfterResponseReachesClient, message.HEL)
		finished := false

		hello, err := helloWire()
		Expect(err).NotTo(HaveOccurred(), "encoding the HEL failed")
		recorder.observe(0, clientToServer, hello)
		ack, err := ackWire()
		Expect(err).NotTo(HaveOccurred(), "encoding the ACK failed")
		var written [][]byte
		Expect(recorder.forward(1, serverToClient, ack, func(message []byte) error {
			written = append(written, message)
			return nil
		}, func(armedCut) error {
			finished = true
			return nil
		})).To(Succeed(), "observing the ACK failed")

		Expect(finished).To(BeFalse(),
			"the HEL response cut fired on an ACK of a connection that never carried a HEL")
		Expect(written).To(Equal([][]byte{ack}),
			"the relay did not write the ACK it must not claim")
		Expect(relay.ArmedCuts()).To(Equal([]string{
			"after a HEL response reaches the client",
		}), "ArmedCuts no longer lists the cut although no HEL paired with the ACK on its connection")
	})
})

func helloWire() ([]byte, error) {
	hello := &uacp.Hello{
		Version:        0,
		ReceiveBufSize: 65535,
		SendBufSize:    65535,
		MaxMessageSize: 16777216,
		MaxChunkCount:  4096,
		EndpointURL:    "opc.tcp://spectest:1",
	}
	body, err := hello.Encode()
	if err != nil {
		return nil, err
	}
	return transportMessage(uacp.MessageTypeHello, body)
}

func ackWire() ([]byte, error) {
	ack := &uacp.Acknowledge{
		Version:        0,
		ReceiveBufSize: 65535,
		SendBufSize:    65535,
		MaxMessageSize: 16777216,
		MaxChunkCount:  4096,
	}
	body, err := ack.Encode()
	if err != nil {
		return nil, err
	}
	return transportMessage(uacp.MessageTypeAcknowledge, body)
}
