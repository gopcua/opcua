package matrix_test

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/tests/spec/internal/fault"
	"github.com/gopcua/opcua/tests/spec/internal/harness"
	"github.com/gopcua/opcua/tests/spec/internal/matrix"
	"github.com/gopcua/opcua/tests/spec/internal/matrix/suites"
	"github.com/gopcua/opcua/tests/spec/internal/specrun"
	"github.com/gopcua/opcua/ua"
)

// TestResponseLostPublishDropsTheArmAnswer drives the SessionSurvives
// case of ResponseLost/Publish and asserts the armed cut drops the
// answer of the arm exchange, not an answer the client received before
// the arm: a cut that can only fire on a response that raced the arm
// reads as never fired when the race goes the other way, so the
// workload must give the fault a response that exists only because it
// armed first.
func TestResponseLostPublishDropsTheArmAnswer(t *testing.T) {
	if specrun.MatrixRuns() {
		t.Skip("the matrix run drives this case itself")
	}
	var scenario matrix.SuiteScenario
	for _, s := range suites.P4_06_07().Scenarios() {
		if s.Name() == "SessionSurvives" {
			scenario = s
		}
	}
	if scenario == nil {
		t.Fatalf("the §6.7 suite has no SessionSurvives scenario")
	}
	var f fault.Fault
	for _, candidate := range fault.AllFaults {
		if candidate.Name() == "ResponseLost/Publish" {
			f = candidate
		}
	}
	if f == nil {
		t.Fatalf("the fault catalogue has no ResponseLost/Publish")
	}

	opts := append(scenario.Options(f), f.Options()...)
	env := harness.New(t, opts...)
	outcome := scenario.Run(env, f)

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
	if !slices.Contains(dropped, suites.ArmValueOf(scenario, f)) {
		t.Errorf("no dropped Publish response carries the arm answer %d; dropped values: %v", suites.ArmValueOf(scenario, f), dropped)
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
