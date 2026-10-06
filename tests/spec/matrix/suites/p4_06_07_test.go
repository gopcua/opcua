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
