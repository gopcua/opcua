package matrix

import (
	"github.com/gopcua/opcua/tests/spec/internal/harness"
	"github.com/gopcua/opcua/ua"
)

// Rule is one Part 4 behaviour the specs assert: a stable Name the
// known-defect table cites, the Clause label and Keyword of its
// sentence in the standard, and the Check that asserts it against the
// context the caller observed.
type Rule struct {
	Name    string
	Clause  string
	Keyword string
	Check   func(c Context)
}

// Context is what a rule's Check needs: the environment the scenario
// ran in, the Mark taken before the fault, the last sequence number
// the client delivered, the subscription the client held before the
// fault and the one it recreated when it did, the server the client
// should end on, how a transfer answer counts as refused when the
// scenario scripted one, and the value the scenario answered on the new
// subscription.
type Context struct {
	Env             *harness.Environment
	Mark            harness.Mark
	LastSeq         uint32
	Sub             harness.Subscription
	Recreated       harness.Subscription
	Server          *harness.ScriptedServer
	TransferRefusal func(harness.ServiceRecord[ua.Response]) bool
	Value           int32
	// CyclesCompleted and CyclesWanted say how many cancel-then-subscribe
	// cycles the workload drove and how many it wanted to complete; a
	// client that stops sending Publish requests (#895) completes fewer
	// of them.
	CyclesCompleted int
	CyclesWanted    int
	// HeldOrder is the recorded Order of the Publish request the workload
	// held past the client's publish timeout; HeldAnswerOrder is the
	// recorded Order of the response it then sent to it, or 0 when it
	// never answered it.
	HeldOrder       int
	HeldAnswerOrder int
}
