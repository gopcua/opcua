package spectest

import (
	"testing"

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

func TestServiceMatches(t *testing.T) {
	cases := []struct {
		service Service
		message any
		want    bool
	}{
		{Republish, &ua.RepublishRequest{}, true},
		{Republish, &ua.ReadRequest{}, false},
		{Read, &ua.ReadRequest{}, true},
		{Read, &ua.RepublishRequest{}, false},
	}
	for _, c := range cases {
		if got := c.service.matches(c.message); got != c.want {
			t.Errorf("%s.matches(%T) = %v, want %v", c.service.name(), c.message, got, c.want)
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

func TestNotificationsReturnsThePublishNotificationsOnly(t *testing.T) {
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
	recorder.appendService(1, 0, serverToClient, 10, Forwarded, first)
	recorder.appendService(2, 0, serverToClient, 11, Forwarded, republish)
	recorder.appendService(3, 1, serverToClient, 12, Forwarded, second)

	notifications := recorder.Notifications()
	if len(notifications) != 2 {
		t.Fatalf("Notifications returned %d notifications, want the two from the Publish responses", len(notifications))
	}
	if notifications[0].Order != 1 || notifications[0].Connection != 0 || notifications[0].SubscriptionID != 4 || notifications[0].SequenceNumber != 2 || notifications[0].Value != 7001 {
		t.Errorf("Notifications[0] = %+v, want order 1, connection 0, subscription 4, sequence number 2, value 7001", notifications[0])
	}
	if notifications[1].Order != 3 || notifications[1].Connection != 1 || notifications[1].SubscriptionID != 4 || notifications[1].SequenceNumber != 3 || notifications[1].Value != 7002 {
		t.Errorf("Notifications[1] = %+v, want order 3, connection 1, subscription 4, sequence number 3, value 7002", notifications[1])
	}
}
