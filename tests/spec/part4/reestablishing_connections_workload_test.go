package part4

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/tests/spec/internal/fault"
	"github.com/gopcua/opcua/tests/spec/internal/harness"
	"github.com/gopcua/opcua/ua"
)

func faultByName(t *testing.T, name string) fault.Fault {
	t.Helper()
	for _, f := range fault.AllFaults {
		if f.Name() == name {
			return f
		}
	}
	t.Fatalf("no fault named %s", name)
	return nil
}

// TestBreaksTransport pins which faults are their own transport loss:
// a stalled link goes silent instead of closing, and a Publish fault
// that cuts the connection itself fires its cut on the arm exchange,
// so the workload cuts for no other fault than those.
func TestBreaksTransport(t *testing.T) {
	cases := map[string]bool{
		"Link/Stall":                                  false,
		"Link/ClosedOnAccept":                         true,
		"Link/HELUnanswered":                          true,
		"Server/Pause":                                true,
		"Server/DuplicateSequence":                    true,
		"Consumer/Slow":                               true,
		"RequestLost/Publish":                         false,
		"RequestLost/Read":                            true,
		"ResponseLost/Publish":                        false,
		"ResponseLost/ActivateSession":                true,
		"CutAfterResponse/Publish":                    false,
		"CutAfterResponse/CloseSession":               true,
		"DelayBelowTimeout/Publish":                   true,
		"DelayAboveTimeout/ActivateSession":           true,
		"Overload/Publish/Bad_TooManyPublishRequests": true,
	}
	for name, want := range cases {
		if got := breaksTransport(faultByName(t, name)); got != want {
			t.Errorf("breaksTransport(%s) = %v, want %v", name, got, want)
		}
	}
}

// TestConsumerBurst pins how many values the workload answers right
// after arming: only the slow consumer needs a burst, large enough to
// fill its notification buffer.
func TestConsumerBurst(t *testing.T) {
	for name, want := range map[string]int{
		"Consumer/Slow":                          8,
		"Link/Stall":                             0,
		"Server/Pause":                           0,
		"RequestLost/Publish":                    0,
		"Overload/Publish/Bad_TooManyOperations": 0,
	} {
		if got := consumerBurst(faultByName(t, name)); got != want {
			t.Errorf("consumerBurst(%s) = %d, want %d", name, got, want)
		}
	}
}

// TestResponseLostPublishDropsTheArmAnswer drives the SessionSurvives
// case of ResponseLost/Publish and asserts the armed cut drops the
// answer of the arm exchange, not an answer the client received before
// the arm: a cut that can only fire on a response that raced the arm
// reads as never fired when the race goes the other way, so the
// workload must give the fault a response that exists only because it
// armed first.
func TestResponseLostPublishDropsTheArmAnswer(t *testing.T) {
	f := faultByName(t, "ResponseLost/Publish")
	cases, err := sessionSurvives.Cases(fault.AllFaults)
	if err != nil {
		t.Fatalf("planning the SessionSurvives cases returned an error: %v", err)
	}
	block := cases[slices.Index(fault.AllFaults, f)].Block

	opts := append(sessionSurvives.Options(f, block), f.Options()...)
	env := harness.New(t, opts...)
	outcome := sessionSurvives.Workload(env, f, block)

	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) && !slices.Contains(env.Received(), outcome.Sentinel) {
		time.Sleep(50 * time.Millisecond)
	}
	for time.Now().Before(deadline) {
		states := env.States()
		if len(states) > 0 && states[len(states)-1] == opcua.Connected {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := env.Client.Close(ctx); err != nil {
		t.Fatalf("closing the client failed: %v", err)
	}

	var dropped []int32
	for _, record := range env.Recorder.Responses() {
		if record.Fate != harness.Dropped {
			continue
		}
		decoded, ok := record.Message()
		if !ok {
			continue
		}
		response, isPublish := decoded.(*ua.PublishResponse)
		if !isPublish {
			continue
		}
		if value, carries := publishValue(response); carries {
			dropped = append(dropped, value)
		}
	}
	if len(dropped) == 0 {
		t.Fatalf("the recorder saw no dropped Publish response, so the armed cut never fired on one")
	}
	for _, value := range dropped {
		if slices.Contains(env.Received(), value) {
			t.Errorf("a dropped Publish response carries %d, a value the client received before the arm", value)
		}
	}
	if !slices.Contains(dropped, reestablishingValuesOf(block).vArm) {
		t.Errorf("no dropped Publish response carries the arm answer %d; dropped values: %v", reestablishingValuesOf(block).vArm, dropped)
	}
}

// publishValue returns the int32 the notification of a Publish
// response carries, and whether it carries one.
func publishValue(response *ua.PublishResponse) (int32, bool) {
	message := response.NotificationMessage
	if message == nil {
		return 0, false
	}
	for _, data := range message.NotificationData {
		if data == nil || data.Value == nil {
			continue
		}
		change, isDataChange := data.Value.(*ua.DataChangeNotification)
		if !isDataChange || len(change.MonitoredItems) == 0 || change.MonitoredItems[0].Value == nil {
			continue
		}
		value := change.MonitoredItems[0].Value.Value.Value()
		if number, isInt32 := value.(int32); isInt32 {
			return number, true
		}
	}
	return 0, false
}
