package matrix

import (
	"slices"
	"testing"
	"time"

	"github.com/gopcua/opcua/tests/spec/internal/fault"
	"github.com/gopcua/opcua/tests/spec/internal/harness"
	"github.com/gopcua/opcua/tests/spec/internal/message"
	"github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	idleValue1   int32 = 301
	idleValue2   int32 = 302
	idleValue3   int32 = 303
	idleSentinel int32 = 399
)

// standInReceivedSentinel stands in for a scenario invariant.
var standInReceivedSentinel = Invariant{
	Name:  "StandInReceivedSentinel",
	Phase: BeforeClose,
	Assert: func(observed Observed) {
		Expect(observed.Sentinel).NotTo(BeNil(), "the workload staged no sentinel")
		Expect(slices.Contains(observed.Received, observed.Sentinel.Value)).To(BeTrue(),
			"the client did not receive the sentinel value %d", observed.Sentinel.Value)
	},
}

// standInReceivedEveryValue stands in for a scenario invariant: the
// client received each of the values the workload answered, at least
// once each.
var standInReceivedEveryValue = Invariant{
	Name:  "StandInReceivedEveryValue",
	Phase: AfterClose,
	Assert: func(observed Observed) {
		for _, value := range []int32{idleValue1, idleValue2, idleValue3} {
			Expect(slices.Contains(observed.Received, value)).To(BeTrue(),
				"the client did not receive the value %d", value)
		}
	},
}

// idle is the scenario of the runner's self-spec: a workload that
// answers its values around the fault and then the sentinel, with no
// rules.
var idle = Scenario{
	Clause:  "P4-0",
	Name:    "Idle",
	Ordinal: 1,
	Sends:   []message.Message{message.Publish},
	Workload: func(env *harness.Environment, f fault.Fault, _ int32) Outcome {
		sub := env.Subscription()
		env.Server.WaitHeldPublish().Answer(sub, idleValue1)
		env.Server.WaitHeldPublish().Answer(sub, idleValue2)
		injected := f.Inject(env)
		env.Server.WaitHeldPublish().Answer(sub, idleValue3)
		faultEnd := time.Now()
		answeredAt := time.Now()
		env.Server.WaitHeldPublish().Answer(sub, idleSentinel)
		return Outcome{
			Injected:   injected,
			FaultEnd:   faultEnd,
			Sentinel:   idleSentinel,
			AnsweredAt: answeredAt,
		}
	},
	Invariants: []Invariant{standInReceivedSentinel, standInReceivedEveryValue},
}

// idleFaults are the faults the self-spec crosses idle with.
var idleFaults = []fault.Fault{mustFault("Server/Pause"), mustFault("Server/DuplicateSequence")}

// mustFault returns the catalogue's fault of that name, panicking when
// the catalogue holds none: it runs at registration, where an unknown
// name is a programming mistake.
func mustFault(name string) fault.Fault {
	f, ok := fault.Named(name)
	if !ok {
		panic("no fault named " + name)
	}
	return f
}

var _ = ginkgo.Describe("P4-0", func() {
	ginkgo.DescribeTableSubtree(idle.Name, func(f fault.Fault) {
		obs := Run(idle, f)
		obs.BeforeCloseInvariants()
		obs.AfterCloseInvariants()
	}, Entries(idle, idleFaults...))
})

// TestSelfSpec runs the runner's self-spec standalone, so the runner's
// own registration and checks are exercised like a clause file's. The
// nested failure-continuation run (SPECTEST_FAILFIRST) registers its
// own scenario on top.
func TestSelfSpec(t *testing.T) {
	suiteConfig, reporterConfig := ginkgo.GinkgoConfiguration()
	suiteConfig.FailOnEmpty = true
	RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "matrix self-spec", suiteConfig, reporterConfig)
}
