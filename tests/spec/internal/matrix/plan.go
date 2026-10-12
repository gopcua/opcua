// Package matrix runs the failure matrix: a clause file declares each
// scenario as a value and registers it, and the matrix plans one case
// per fault with its checks, then runs a real client through each case
// and asserts its checks against what it observed. A rule is a Rule
// value in the clause file of the clause it cites. The workload returns
// the rule's facts as an untyped value, and the clause package's Check
// asserts them to its own type. An invariant is an Invariant value in
// the clause package's invariants file, part4/invariants_test.go. The
// matrix adds the HaveFired check to every applicable case itself. The
// self-spec in this package registers stand-in scenarios and exercises
// the runner's own registration and checks. No code under internal/
// asserts what a clause demands.
package matrix

import (
	"time"

	"github.com/gopcua/opcua/tests/spec/internal/fault"
	"github.com/gopcua/opcua/tests/spec/internal/harness"
)

// Outcome is what a scenario's Workload observed: the armed fault, the
// fault's end event, the sentinel value with the moment its answer
// was sent, and the rules' facts.
type Outcome struct {
	Injected   *harness.Injected
	FaultEnd   time.Time
	Sentinel   int32
	AnsweredAt time.Time
	Rules      any
}

// Phase picks the snapshot an invariant asserts on. A rule check
// carries a phase too, but nothing reads it at run time: a rule reads
// the recorder through the workload's facts, after the client
// closed (Observation.Facts).
type Phase int

// The zero Phase is invalid; no check carries it.
const (
	// BeforeClose: the invariant asserts on the snapshot taken while
	// the client is still connected.
	BeforeClose Phase = iota + 1
	// AfterClose: the invariant asserts on the snapshot taken after
	// the client closed.
	AfterClose
)

// Check is one assertion a case runs: an invariant's or a rule's name
// and its phase (see Phase for what the phase does).
type Check struct {
	Name  string
	Phase Phase
}

// Case is one scenario × fault combination the plan decided:
// its path, its skip reason when the fault does not apply, its checks,
// and the first number of its value block.
type Case struct {
	Path   []string
	Skip   *fault.Reason
	Checks []Check
	Block  int32
}
