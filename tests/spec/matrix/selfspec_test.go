package matrix

import (
	"testing"
	"time"

	"github.com/gopcua/opcua/tests/spec/faults"
	"github.com/gopcua/opcua/tests/spec/message"
	"github.com/gopcua/opcua/tests/spec/rules"
	"github.com/gopcua/opcua/tests/spec/spectest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	idleValue1   int32 = 301
	idleValue2   int32 = 302
	idleValue3   int32 = 303
	idleSentinel int32 = 399
)

// idleSuite is the test-only suite of the runner's self-spec: one
// scenario whose workload answers three values around the fault and
// then the sentinel, with no rules.
type idleSuite struct{}

func (idleSuite) Clause() string { return "P4-0" }

func (idleSuite) Scenarios() []Scenario {
	return []Scenario{idleScenario{}}
}

func (idleSuite) Rules(Category) []rules.Rule { return nil }

type idleScenario struct{}

func (idleScenario) Name() string { return "Idle" }

func (idleScenario) Sends() []message.Message {
	return []message.Message{message.Publish}
}

func (idleScenario) Options() []spectest.Option { return nil }

func (idleScenario) Category(f faults.Fault) Category { return Unspecified }

func (idleScenario) Run(env *spectest.Environment, f faults.Fault) Outcome {
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
}

func init() {
	RegisterSuites([]Suite{idleSuite{}}, nil, faultNamed("Server/Pause"), faultNamed("Server/DuplicateSequence"))
}

func TestSelfSpec(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "matrix self-spec")
}
