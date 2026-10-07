package matrix

import (
	"flag"
	"os"
	"testing"
	"time"

	"github.com/gopcua/opcua/tests/spec/internal/faults"
	"github.com/gopcua/opcua/tests/spec/internal/message"
	"github.com/gopcua/opcua/tests/spec/internal/rules"
	"github.com/gopcua/opcua/tests/spec/internal/spectest"
	"github.com/gopcua/opcua/tests/spec/internal/suitegate"
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

func (idleScenario) Options(faults.Fault) []spectest.Option { return nil }

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

// selfSpecDefects labels the DuplicateSequence case's delivery check:
// the client delivers a notification whose sequence number it already
// received, which the invariant now catches.
var selfSpecDefects = []KnownDefect{
	{
		Issue: Unfiled("duplicate-sequence", "client delivers a notification whose sequence number it already received (S9)"),
		Check: "DeliverEachValueOnce",
		Applies: func(scenario string, f faults.Fault) bool {
			return f.Name() == "Server/DuplicateSequence"
		},
	},
}

func init() {
	RegisterSuites([]Suite{idleSuite{}}, selfSpecDefects, faultNamed("Server/Pause"), faultNamed("Server/DuplicateSequence"))
}

// TestSelfSpecDefectKeepsTheCaseRunning asserts the labelled case is
// neither skipped nor emptied: a known defect may retire one check of
// a case, never the case itself.
func TestSelfSpecDefectKeepsTheCaseRunning(t *testing.T) {
	cases, err := Plan([]Suite{idleSuite{}}, []faults.Fault{faultNamed("Server/Pause"), faultNamed("Server/DuplicateSequence")}, selfSpecDefects)
	if err != nil {
		t.Fatalf("Plan over the self-spec returned an error: %v", err)
	}
	var duplicate *Case
	for i := range cases {
		if cases[i].Path[2] == "Server/DuplicateSequence" {
			duplicate = &cases[i]
		}
	}
	if duplicate == nil {
		t.Fatalf("Plan built no DuplicateSequence case; paths: %v", casePaths(cases))
	}
	if duplicate.Skip != nil {
		t.Fatalf("the DuplicateSequence case is skipped: %s", duplicate.Skip.Text)
	}
	var labelled, unlabelled int
	for _, check := range duplicate.Checks {
		if contains(check.Labels, "known-defect") {
			labelled++
			continue
		}
		unlabelled++
	}
	if labelled != 1 {
		t.Fatalf("the DuplicateSequence case carries %d labelled checks, want exactly the delivery check", labelled)
	}
	if unlabelled == 0 {
		t.Fatalf("the DuplicateSequence case carries no unlabelled check, so the default filter would run nothing of it")
	}
}

func casePaths(cases []Case) []string {
	var paths []string
	for _, c := range cases {
		paths = append(paths, c.Path[0]+"/"+c.Path[1]+"/"+c.Path[2])
	}
	return paths
}

// TestSelfSpec runs the registered suites standalone, focused by -run:
// the label filter resolves exactly as TestMatrix does, so the default
// run excludes the labelled known-defect checks. It skips whenever the
// full matrix test runs in the same binary: ginkgo fails a second
// RunSpecs call, and TestMatrix covers this suite inside itself.
func TestSelfSpec(t *testing.T) {
	if suitegate.MatrixRuns() {
		t.Skip("TestMatrix runs the self-spec suite inside the matrix; a binary may run RunSpecs once")
		return
	}
	suiteConfig, reporterConfig := GinkgoConfiguration()
	var labelPassed bool
	flag.Visit(func(passed *flag.Flag) {
		if passed.Name == "ginkgo.label-filter" {
			labelPassed = true
		}
	})
	suiteConfig.LabelFilter = spectest.LabelFilter(labelPassed, suiteConfig.LabelFilter, os.Getenv("SPECTEST_LABEL_FILTER"))
	RegisterFailHandler(Fail)
	RunSpecs(t, "matrix self-spec", suiteConfig, reporterConfig)
}
