package fault

import (
	"testing"

	"github.com/gopcua/opcua/tests/spec/internal/message"
)

func TestAvailable(t *testing.T) {
	cases := []struct {
		fault   string
		sends   []message.Message
		wantNil bool
	}{
		{"RequestLost/HEL", []message.Message{message.Publish}, false},
		{"RequestLost/HEL", []message.Message{message.HEL, message.Publish}, true},
		{"Overload/Publish/Bad_TooManyPublishRequests", []message.Message{message.Read}, false},
		{"Overload/Publish/Bad_TooManyPublishRequests", []message.Message{message.Publish}, true},
		{"Link/ClosedOnAccept", []message.Message{message.Publish}, false},
		{"Link/ListenerClosed", []message.Message{message.Publish}, false},
		{"Link/HELUnanswered", []message.Message{message.Publish}, false},
		{"Link/ClosedOnAccept", []message.Message{message.HEL}, true},
		{"Link/ListenerClosed", []message.Message{message.HEL}, true},
		{"Link/HELUnanswered", []message.Message{message.HEL}, true},
		{"Link/Stall", []message.Message{}, true},
		{"Server/Pause", []message.Message{}, true},
		{"Consumer/Slow", []message.Message{}, true},
		{"Server/DuplicateSequence", []message.Message{message.Read}, false},
		{"Server/SkippedSequence", []message.Message{message.Read}, false},
		{"Server/DuplicateSequence", []message.Message{message.Publish}, true},
		{"Server/SkippedSequence", []message.Message{message.Publish}, true},
		{"Control/None", []message.Message{}, true},
	}
	byName := faultsByName()
	for _, c := range cases {
		f, ok := byName[c.fault]
		if !ok {
			t.Errorf("no fault named %s", c.fault)
			continue
		}
		reason := f.Available(c.sends)
		if c.wantNil {
			if reason != nil {
				t.Errorf("%s.Available(%v) = %q, want nil", c.fault, c.sends, reason.Text)
			}
			continue
		}
		if reason == nil {
			t.Errorf("%s.Available(%v) = nil, want a reason", c.fault, c.sends)
		}
	}
}

func faultsByName() map[string]Fault {
	byName := make(map[string]Fault)
	for _, f := range AllFaults {
		byName[f.Name()] = f
	}
	return byName
}
