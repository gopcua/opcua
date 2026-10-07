package suites

import (
	"testing"

	"github.com/gopcua/opcua/tests/spec/faults"
)

func faultByName(t *testing.T, name string) faults.Fault {
	t.Helper()
	for _, fault := range faults.AllFaults {
		if fault.Name() == name {
			return fault
		}
	}
	t.Fatalf("no fault named %s", name)
	return nil
}

// TestBreaksTransport pins which faults are their own transport loss:
// a stalled link goes silent instead of closing, so the workload cuts
// for no other fault than itself.
func TestBreaksTransport(t *testing.T) {
	cases := map[string]bool{
		"Link/Stall":                                  false,
		"Link/ClosedOnAccept":                         true,
		"Link/HELUnanswered":                          true,
		"Server/Pause":                                true,
		"Server/DuplicateSequence":                    true,
		"Consumer/Slow":                               true,
		"RequestLost/Publish":                         true,
		"ResponseLost/ActivateSession":                true,
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

// TestCaseValuesNeverOverlap pins that no two cases of the suite share
// a value: every case derives its block — first value, answered
// values, retained value, sentinel and consumer burst alike — from its
// own scenario and fault, so a received value matches the notification
// that carried it by value and never a value another case answered.
func TestCaseValuesNeverOverlap(t *testing.T) {
	seen := map[int32]string{}
	for _, scenario := range P4_06_07().Scenarios() {
		for _, f := range faults.AllFaults {
			owner := scenario.Name() + "/" + f.Name()
			for _, value := range caseValuesOf(scenario, f) {
				if previous, taken := seen[value]; taken {
					t.Errorf("value %d belongs to %s and %s", value, previous, owner)
				}
				seen[value] = owner
			}
		}
	}
	if len(seen) < 100 {
		t.Fatalf("collected %d distinct values over every case, want at least one per case and value", len(seen))
	}
}
