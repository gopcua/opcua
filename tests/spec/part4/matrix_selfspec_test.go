package part4

import (
	"time"

	"github.com/gopcua/opcua/tests/spec/internal/fault"
	"github.com/gopcua/opcua/tests/spec/internal/harness"
	"github.com/gopcua/opcua/tests/spec/internal/matrix"
	"github.com/gopcua/opcua/tests/spec/internal/message"
	"github.com/onsi/ginkgo/v2"
)

const (
	idleValue1   int32 = 301
	idleValue2   int32 = 302
	idleValue3   int32 = 303
	idleSentinel int32 = 399
)

// idle is the scenario of the runner's self-spec: a workload that
// answers three values around the fault and then the sentinel, with no
// rules.
var idle = matrix.Scenario{
	Clause:  "P4-0",
	Name:    "Idle",
	Ordinal: 6,
	Sends:   []message.Message{message.Publish},
	Workload: func(env *harness.Environment, f fault.Fault, _ int32) matrix.Outcome {
		sub := env.Subscription()
		env.Server.WaitHeldPublish().Answer(sub, idleValue1)
		env.Server.WaitHeldPublish().Answer(sub, idleValue2)
		injected := f.Inject(env)
		env.Server.WaitHeldPublish().Answer(sub, idleValue3)
		faultEnd := time.Now()
		answeredAt := time.Now()
		env.Server.WaitHeldPublish().Answer(sub, idleSentinel)
		return matrix.Outcome{
			Injected:   injected,
			FaultEnd:   faultEnd,
			Sentinel:   idleSentinel,
			AnsweredAt: answeredAt,
		}
	},
	Invariants: subscriptionInvariants,
}

// idleFaults are the faults the self-spec crosses idle with.
var idleFaults = []fault.Fault{faultNamed("Server/Pause"), faultNamed("Server/DuplicateSequence")}

var _ = ginkgo.Describe("P4-0", func() {
	ginkgo.DescribeTableSubtree(idle.Name, func(f fault.Fault) {
		obs := matrix.Run(idle, f)
		obs.BeforeCloseInvariants()
		obs.AfterCloseInvariants()
	}, matrix.Entries(idle, idleFaults...))
})

// faultNamed returns the catalogue's fault of that name.
func faultNamed(name string) fault.Fault {
	for _, f := range fault.AllFaults {
		if f.Name() == name {
			return f
		}
	}
	panic("no fault named " + name)
}
