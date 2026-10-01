package spectest

import (
	"testing"

	"github.com/gopcua/opcua/ua"
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
