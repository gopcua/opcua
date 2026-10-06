package spectest

import (
	"io"
	"net"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/gopcua/opcua/tests/spec/message"
	"github.com/gopcua/opcua/ua"
	"github.com/gopcua/opcua/uacp"
)

func TestServiceRecordMessagePerFate(t *testing.T) {
	cases := []struct {
		fate   Fate
		name   string
		wantOK bool
	}{
		{Forwarded, "Forwarded", true},
		{Dropped, "Dropped", true},
		{Truncated, "Truncated", false},
		{Aborted, "Aborted", false},
	}
	for _, c := range cases {
		record := ServiceRecord[ua.Request]{Fate: c.fate, message: &ua.ReadRequest{}}
		message, ok := record.Message()
		if ok != c.wantOK {
			t.Errorf("a record with Fate %s yields Message() ok %v, want %v", c.name, ok, c.wantOK)
		}
		if c.wantOK && message == nil {
			t.Errorf("a record with Fate %s yields Message() with no message, want the decoded one", c.name)
		}
		if !c.wantOK && message != nil {
			t.Errorf("a record with Fate %s yields Message() with a message, want none", c.name)
		}
	}
}

func TestMessageOfRequest(t *testing.T) {
	cases := []struct {
		service any
		want    message.Message
		named   bool
	}{
		{&ua.RepublishRequest{}, message.Republish, true},
		{&ua.ReadRequest{}, message.Read, true},
		{&ua.PublishRequest{}, message.Publish, true},
		{&ua.OpenSecureChannelRequest{}, message.OpenSecureChannel, true},
		{&ua.CloseSecureChannelRequest{}, message.CloseSecureChannel, true},
		{&ua.CreateSessionRequest{}, message.CreateSession, true},
		{&ua.ActivateSessionRequest{}, message.ActivateSession, true},
		{&ua.CloseSessionRequest{}, message.CloseSession, true},
		{&ua.CreateSubscriptionRequest{}, message.CreateSubscription, true},
		{&ua.CreateMonitoredItemsRequest{}, message.CreateMonitoredItems, true},
		{&ua.DeleteSubscriptionsRequest{}, message.DeleteSubscriptions, true},
		{&ua.TransferSubscriptionsRequest{}, message.TransferSubscriptions, true},
		{&ua.ReadResponse{}, 0, false},
		{nil, 0, false},
	}
	for _, c := range cases {
		got, named := messageOfRequest(c.service)
		if named != c.named || got != c.want {
			t.Errorf("messageOfRequest(%T) = (%v, %v), want (%v, %v)", c.service, got, named, c.want, c.named)
		}
	}
}

func TestTransportRecordsAServerError(t *testing.T) {
	fake := &fakeT{}
	recorder := newRecorder(fake)
	errorMessage := &uacp.Error{ErrorCode: uint32(ua.StatusBadNotConnected), Reason: "spectest error"}
	body, err := errorMessage.Encode()
	if err != nil {
		t.Fatalf("encoding the ERR body failed: %v", err)
	}
	wire, err := transportMessage(uacp.MessageTypeError, body)
	if err != nil {
		t.Fatalf("encoding the ERR transport message failed: %v", err)
	}

	recorder.observe(0, serverToClient, wire)

	records := recorder.Transport()
	if len(records) != 1 {
		t.Fatalf("Transport returned %d records for one ERR message, want 1", len(records))
	}
	if records[0].Type != ERR {
		t.Fatalf("the ERR message was recorded as type %v, want ERR", records[0].Type)
	}
	if records[0].Err == nil {
		t.Fatalf("the ERR record carries no error")
	}
	if records[0].Err.Status != ua.StatusBadNotConnected {
		t.Fatalf("the ERR record carries status %v, want Bad_NotConnected", records[0].Err.Status)
	}
	if records[0].Err.Reason != "spectest error" {
		t.Fatalf("the ERR record carries reason %q, want the written reason", records[0].Err.Reason)
	}
	for _, cleanup := range fake.cleanups {
		cleanup()
	}
}

func TestNotificationsReturnsPublishAndRepublishNotificationsInOrder(t *testing.T) {
	fake := &fakeT{}
	recorder := newRecorder(fake)
	first := &ua.PublishResponse{
		SubscriptionID:      4,
		NotificationMessage: dataChangeNotificationMessage(2, 1, 7001),
	}
	republish := &ua.RepublishResponse{
		NotificationMessage: dataChangeNotificationMessage(1, 1, 9001),
	}
	second := &ua.PublishResponse{
		SubscriptionID:      4,
		NotificationMessage: dataChangeNotificationMessage(3, 1, 7002),
	}
	recorder.appendService(0, 0, clientToServer, 11, Forwarded, &ua.RepublishRequest{SubscriptionID: 4, RetransmitSequenceNumber: 1})
	recorder.appendService(1, 0, serverToClient, 10, Forwarded, first)
	recorder.appendService(2, 0, serverToClient, 11, Forwarded, republish)
	recorder.appendService(3, 1, serverToClient, 12, Forwarded, second)

	notifications := recorder.Notifications()
	if len(notifications) != 3 {
		t.Fatalf("Notifications returned %d notifications, want one per forwarded answer", len(notifications))
	}
	if notifications[0].Order != 1 || notifications[0].Connection != 0 || notifications[0].SubscriptionID != 4 || notifications[0].SequenceNumber != 2 || notifications[0].Value != 7001 || notifications[0].Republished {
		t.Errorf("Notifications[0] = %+v, want order 1, connection 0, subscription 4, sequence number 2, value 7001, not republished", notifications[0])
	}
	if notifications[1].Order != 2 || notifications[1].Connection != 0 || notifications[1].SubscriptionID != 4 || notifications[1].SequenceNumber != 1 || notifications[1].Value != 9001 || !notifications[1].Republished {
		t.Errorf("Notifications[1] = %+v, want order 2, connection 0, subscription 4 from the Republish request, sequence number 1, value 9001, republished", notifications[1])
	}
	if notifications[2].Order != 3 || notifications[2].Connection != 1 || notifications[2].SubscriptionID != 4 || notifications[2].SequenceNumber != 3 || notifications[2].Value != 7002 || notifications[2].Republished {
		t.Errorf("Notifications[2] = %+v, want order 3, connection 1, subscription 4, sequence number 3, value 7002, not republished", notifications[2])
	}

	env := &Environment{Recorder: recorder}
	if env.LastSequenceNumber() != 3 {
		t.Errorf("LastSequenceNumber = %d, want 3, the highest delivered sequence number", env.LastSequenceNumber())
	}
}

func TestLastSequenceNumberRaisesARecordedDecodeError(t *testing.T) {
	fake := &fakeT{}
	recorder := newRecorder(fake)
	wire, err := messageChunk(uacp.ChunkTypeFinal, 42, []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff})
	if err != nil {
		t.Fatalf("encoding the undecodable message failed: %v", err)
	}
	recorder.observe(0, clientToServer, wire)
	env := &Environment{t: fake, Recorder: recorder}

	if !fatalPanics(func() { env.LastSequenceNumber() }) {
		t.Fatalf("LastSequenceNumber returned without failing although the recorder stored a decode error")
	}
	if len(fake.fatals) != 1 || !strings.HasPrefix(fake.fatals[0], "spectest: connection 0, client-to-server direction, byte offset 0:") {
		t.Fatalf("LastSequenceNumber failed with %q, want the recorded decode error's message, not a sequence fault", fake.fatals)
	}
	for _, cleanup := range fake.cleanups {
		cleanup()
	}
}

func TestRepublishAnswersPairByConnectionAndTransportRequestID(t *testing.T) {
	fake := &fakeT{}
	recorder := newRecorder(fake)
	recorder.appendService(1, 0, clientToServer, 21, Forwarded, &ua.RepublishRequest{SubscriptionID: 7, RetransmitSequenceNumber: 1})
	recorder.appendService(2, 1, clientToServer, 21, Forwarded, &ua.RepublishRequest{SubscriptionID: 8, RetransmitSequenceNumber: 1})
	recorder.appendService(3, 0, serverToClient, 21, Forwarded, &ua.RepublishResponse{
		NotificationMessage: dataChangeNotificationMessage(1, monitorClientHandle, 7001),
	})
	recorder.appendService(4, 1, serverToClient, 21, Forwarded, &ua.RepublishResponse{
		NotificationMessage: dataChangeNotificationMessage(1, monitorClientHandle, 7002),
	})

	notifications := recorder.Notifications()
	if len(notifications) != 2 {
		t.Fatalf("Notifications returned %d notifications, want one per connection's Republish answer", len(notifications))
	}
	if notifications[0].SubscriptionID != 7 {
		t.Errorf("the Republish answer on connection 0 carries subscription %d, want 7, the id its own connection's request named", notifications[0].SubscriptionID)
	}
	if notifications[1].SubscriptionID != 8 {
		t.Errorf("the Republish answer on connection 1 carries subscription %d, want 8, the id its own connection's request named", notifications[1].SubscriptionID)
	}
	for _, cleanup := range fake.cleanups {
		cleanup()
	}
}

func TestRepublishAnswerPairsByTransportRequestIDNotRequestHandle(t *testing.T) {
	fake := &fakeT{}
	recorder := newRecorder(fake)
	request, err := republishRequestWireWithHandle(21, 99)
	if err != nil {
		t.Fatalf("encoding the Republish with a service-header handle of 99 failed: %v", err)
	}
	recorder.observe(0, clientToServer, request)
	response, err := republishResponseWire(21)
	if err != nil {
		t.Fatalf("encoding the Republish response failed: %v", err)
	}
	recorder.observe(0, serverToClient, response)

	notifications := recorder.Notifications()
	if len(notifications) != 1 {
		t.Fatalf("Notifications returned %d notifications, want the republished one", len(notifications))
	}
	if notifications[0].SubscriptionID != 1 || !notifications[0].Republished {
		t.Fatalf("the republished notification is %+v, want the subscription id 1 of the request whose transport request id 21 the response answered, republished", notifications[0])
	}
	for _, cleanup := range fake.cleanups {
		cleanup()
	}
}

func TestRepublishedNotificationCarriesTheRepublishRequestsSubscriptionID(t *testing.T) {
	fake := &fakeT{}
	recorder := newRecorder(fake)
	recorder.appendService(1, 0, clientToServer, 21, Forwarded, &ua.RepublishRequest{SubscriptionID: 42, RetransmitSequenceNumber: 6})
	recorder.appendService(2, 0, serverToClient, 21, Forwarded, &ua.RepublishResponse{
		NotificationMessage: dataChangeNotificationMessage(6, 1, 7001),
	})

	notifications := recorder.Notifications()
	if len(notifications) != 1 {
		t.Fatalf("Notifications returned %d notifications, want the republished one", len(notifications))
	}
	if notifications[0].SubscriptionID != 42 || !notifications[0].Republished || notifications[0].SequenceNumber != 6 {
		t.Fatalf("the republished notification is %+v, want the subscription id 42 of its Republish request, republished, sequence number 6", notifications[0])
	}
	for _, cleanup := range fake.cleanups {
		cleanup()
	}
}

func TestLastSequenceNumberCountsRepublishedNotifications(t *testing.T) {
	fake := &fakeT{}
	recorder := newRecorder(fake)
	recorder.appendService(1, 0, clientToServer, 21, Forwarded, &ua.RepublishRequest{SubscriptionID: 4, RetransmitSequenceNumber: 7})
	recorder.appendService(2, 0, serverToClient, 10, Forwarded, &ua.PublishResponse{
		SubscriptionID:      4,
		NotificationMessage: dataChangeNotificationMessage(2, 1, 7001),
	})
	recorder.appendService(3, 0, serverToClient, 21, Forwarded, &ua.RepublishResponse{
		NotificationMessage: dataChangeNotificationMessage(7, 1, 7002),
	})

	env := &Environment{Recorder: recorder}
	if sequenceNumber := env.LastSequenceNumber(); sequenceNumber != 7 {
		t.Fatalf("LastSequenceNumber = %d, want 7, the highest sequence number a Republish delivered", sequenceNumber)
	}
	for _, cleanup := range fake.cleanups {
		cleanup()
	}
}

func TestArmedCutsListsACutUntilItsPositionIsMarked(t *testing.T) {
	ft := &fakeT{}
	var listedOnCut []string
	var relay *Relay
	var recorder *Recorder
	relay, recorder = newRelay(ft, "opc.tcp://127.0.0.1:1", func() {
		listedOnCut = relay.ArmedCuts()
	})
	relay.CutAt(BeforeRequestReachesServer, message.Republish)

	wire, err := republishRequestWire(21)
	if err != nil {
		t.Fatalf("encoding the Republish failed: %v", err)
	}
	if forwardErr := recorder.forward(0, clientToServer, wire, func([]byte) error { return nil }, func(armedCut) error { return nil }); forwardErr != nil {
		t.Fatalf("observing the Republish the cut fired on failed: %v", forwardErr)
	}

	if len(listedOnCut) != 1 || listedOnCut[0] != "before a Republish request reaches the server" {
		t.Fatalf("while onCut ran, ArmedCuts listed %q, want the firing cut still listed", listedOnCut)
	}
	if cuts := relay.ArmedCuts(); len(cuts) != 0 {
		t.Fatalf("after the cut marked its position, ArmedCuts still lists %q", cuts)
	}
	for _, cleanup := range ft.cleanups {
		cleanup()
	}
}

func TestASecondMatchingMessageDoesNotClaimACutWhileItFires(t *testing.T) {
	ft := &fakeT{}
	relay, _ := newRelay(ft, "opc.tcp://127.0.0.1:1", nil)
	relay.CutAt(BeforeRequestReachesServer, message.Read)
	matches := func(c armedCut) bool { return c.moment == BeforeRequestReachesServer && c.message == message.Read }

	if _, fired := relay.takeArmed(matches); !fired {
		t.Fatalf("the first matching message claimed no cut")
	}
	if _, fired := relay.takeArmed(matches); fired {
		t.Fatalf("a second matching message claimed the cut while it was already firing")
	}
	if cuts := relay.ArmedCuts(); len(cuts) != 1 || cuts[0] != "before a Read request reaches the server" {
		t.Fatalf("while the cut fires, ArmedCuts lists %q, want the firing cut alone", cuts)
	}
	for _, cleanup := range ft.cleanups {
		cleanup()
	}
}

func TestDisarmingOneOfTwoIdenticalFiringCutsLeavesTheOtherAbleToFire(t *testing.T) {
	ft := &fakeT{}
	relay, _ := newRelay(ft, "opc.tcp://127.0.0.1:1", nil)
	relay.CutAt(BeforeRequestReachesServer, message.Read)
	relay.CutAt(BeforeRequestReachesServer, message.Read)
	matches := func(c armedCut) bool { return c.moment == BeforeRequestReachesServer && c.message == message.Read }

	first, fired := relay.takeArmed(matches)
	if !fired {
		t.Fatalf("the first matching message claimed no cut")
	}
	second, fired := relay.takeArmed(matches)
	if !fired {
		t.Fatalf("a matching message on a second connection claimed no second identical cut")
	}
	relay.disarm(second)
	relay.release(first)
	reclaimed, fired := relay.takeArmed(matches)
	if !fired {
		t.Fatalf("disarm removed a cut other than the one it was given, so the released first cut can no longer fire although the second cut already finished")
	}
	relay.disarm(reclaimed)
	if cuts := relay.ArmedCuts(); len(cuts) != 0 {
		t.Fatalf("ArmedCuts lists %q after both identical cuts fired", cuts)
	}
	for _, cleanup := range ft.cleanups {
		cleanup()
	}
}

type drainProbeConn struct {
	halfClosed  chan struct{}
	readUnblock chan struct{}
}

func (c drainProbeConn) Read([]byte) (int, error) {
	<-c.readUnblock
	return 0, io.EOF
}

func (c drainProbeConn) Write(b []byte) (int, error)      { return len(b), nil }
func (c drainProbeConn) Close() error                     { return nil }
func (c drainProbeConn) LocalAddr() net.Addr              { return nil }
func (c drainProbeConn) RemoteAddr() net.Addr             { return nil }
func (c drainProbeConn) SetDeadline(time.Time) error      { return nil }
func (c drainProbeConn) SetReadDeadline(time.Time) error  { return nil }
func (c drainProbeConn) SetWriteDeadline(time.Time) error { return nil }

func (c drainProbeConn) CloseWrite() error {
	select {
	case <-c.halfClosed:
	default:
		close(c.halfClosed)
	}
	return nil
}

func TestCutAfterResponseDisarmsBeforeDrainingTheClient(t *testing.T) {
	ft := &fakeT{}
	var listedOnCut []string
	client := drainProbeConn{halfClosed: make(chan struct{}), readUnblock: make(chan struct{})}
	server := drainProbeConn{halfClosed: make(chan struct{}), readUnblock: make(chan struct{})}
	var relay *Relay
	relay, recorder := newRelay(ft, "opc.tcp://127.0.0.1:1", func() {
		listedOnCut = relay.ArmedCuts()
	})
	relay.CutAt(AfterResponseReachesClient, message.Read)

	staged, err := readRequestWire(9)
	if err != nil {
		t.Fatalf("encoding the request failed: %v", err)
	}
	recorder.observe(0, clientToServer, staged)
	response, err := readResponseWire(9)
	if err != nil {
		t.Fatalf("encoding the response failed: %v", err)
	}
	done := make(chan error, 1)
	go func() {
		done <- recorder.forward(0, serverToClient, response, func([]byte) error { return nil }, func(claim armedCut) error {
			return relay.cutAfterResponse(0, client, server, claim)
		})
	}()

	<-client.halfClosed
	if len(listedOnCut) != 1 || listedOnCut[0] != "after a Read response reaches the client" {
		t.Fatalf("while onCut ran, ArmedCuts listed %q, want the firing cut still listed", listedOnCut)
	}
	if cuts := relay.ArmedCuts(); len(cuts) != 0 {
		t.Fatalf("while the after-response cut still drains the client, ArmedCuts lists %q, want the cut gone before the drain", cuts)
	}
	close(client.readUnblock)
	if forwardErr := <-done; forwardErr != nil {
		t.Fatalf("the after-response cut failed: %v", forwardErr)
	}
	for _, cleanup := range ft.cleanups {
		cleanup()
	}
}

type closeRecordingConn struct {
	onClose      func()
	onCloseWrite func()
}

func (c closeRecordingConn) Read([]byte) (int, error)         { return 0, io.EOF }
func (c closeRecordingConn) Write(b []byte) (int, error)      { return len(b), nil }
func (c closeRecordingConn) Close() error                     { c.onClose(); return nil }
func (c closeRecordingConn) CloseWrite() error                { c.onCloseWrite(); return nil }
func (c closeRecordingConn) LocalAddr() net.Addr              { return nil }
func (c closeRecordingConn) RemoteAddr() net.Addr             { return nil }
func (c closeRecordingConn) SetDeadline(time.Time) error      { return nil }
func (c closeRecordingConn) SetReadDeadline(time.Time) error  { return nil }
func (c closeRecordingConn) SetWriteDeadline(time.Time) error { return nil }

func TestCutMarksTheCutPositionBeforeClosingSockets(t *testing.T) {
	fake := &fakeT{}
	var order []string
	client := closeRecordingConn{onClose: func() { order = append(order, "client closed") }}
	server := closeRecordingConn{onClose: func() { order = append(order, "server closed") }}
	relay, _ := newRelay(fake, "127.0.0.1:1", func() { order = append(order, "cut position marked") })
	relay.mu.Lock()
	relay.connections = []relayConn{{index: 0, client: client, server: server}}
	relay.mu.Unlock()

	relay.Cut()

	want := []string{"cut position marked", "client closed", "server closed"}
	if !slices.Equal(order, want) {
		t.Errorf("Cut touched sockets before recording the cut position: %v, want %v", order, want)
	}
}

func TestCloseConnectionMarksTheCutPositionBeforeClosingSockets(t *testing.T) {
	fake := &fakeT{}
	var order []string
	client := closeRecordingConn{onClose: func() { order = append(order, "client closed") }}
	server := closeRecordingConn{onClose: func() { order = append(order, "server closed") }}
	relay, _ := newRelay(fake, "127.0.0.1:1", func() { order = append(order, "cut position marked") })
	relay.mu.Lock()
	relay.connections = []relayConn{{index: 0, client: client, server: server}}
	relay.mu.Unlock()

	relay.closeConnection(0, armedCut{})

	want := []string{"cut position marked", "client closed", "server closed"}
	if !slices.Equal(order, want) {
		t.Errorf("closeConnection touched sockets before recording the cut position: %v, want %v", order, want)
	}
}

func TestCutAfterResponseMarksTheCutPositionBeforeHalfClosing(t *testing.T) {
	fake := &fakeT{}
	var order []string
	client := closeRecordingConn{
		onClose:      func() { order = append(order, "client closed") },
		onCloseWrite: func() { order = append(order, "client half closed") },
	}
	server := closeRecordingConn{onClose: func() { order = append(order, "server closed") }}
	relay, _ := newRelay(fake, "127.0.0.1:1", func() { order = append(order, "cut position marked") })

	if err := relay.cutAfterResponse(0, client, server, armedCut{}); err != nil {
		t.Fatalf("cutAfterResponse with recording connections failed: %v", err)
	}

	want := []string{"cut position marked", "client half closed", "server closed", "client closed"}
	if !slices.Equal(order, want) {
		t.Errorf("the after-response cut touched sockets before recording the cut position: %v, want %v", order, want)
	}
}
