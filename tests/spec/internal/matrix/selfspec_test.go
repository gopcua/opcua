package matrix

import (
	"flag"
	"os"
	"testing"
	"time"

	"github.com/gopcua/opcua/tests/spec/internal/fault"
	"github.com/gopcua/opcua/tests/spec/internal/harness"
	"github.com/gopcua/opcua/tests/spec/internal/message"
	"github.com/gopcua/opcua/tests/spec/internal/specrun"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
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
	Invariants: SubscriptionInvariants,
	// The DuplicateSequence case's delivery check: the client delivers
	// a notification whose sequence number it already received, which
	// the invariant now catches.
	KnownDefects: []KnownDefect{{
		Issue:   Unfiled("duplicate-sequence", "client delivers a notification whose sequence number it already received (S9)"),
		Check:   "DeliverEachValueOnce",
		Applies: FaultsNamed("Server/DuplicateSequence"),
	}},
}

// idleFaults are the faults the self-spec crosses idle with.
var idleFaults = []fault.Fault{faultNamed("Server/Pause"), faultNamed("Server/DuplicateSequence")}

var _ = Describe("P4-0", func() {
	DescribeTableSubtree(idle.Name, func(f fault.Fault) {
		obs := Run(idle, f)
		obs.BeforeCloseInvariants()
		obs.AfterCloseInvariants()
	}, Entries(idle, idleFaults...))
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

// TestSelfSpecDefectKeepsTheCaseRunning asserts the labelled case is
// neither skipped nor emptied: a known defect may retire one check of
// a case, never the case itself.
func TestSelfSpecDefectKeepsTheCaseRunning(t *testing.T) {
	cases, err := idle.Cases(idleFaults)
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

// TestSelfSpec runs the registered self-spec scenarios standalone: the
// label filter resolves through specrun.LabelFilter, so the default
// run excludes the labelled known-defect checks.
func TestSelfSpec(t *testing.T) {
	suiteConfig, reporterConfig := GinkgoConfiguration()
	var labelPassed bool
	flag.Visit(func(passed *flag.Flag) {
		if passed.Name == "ginkgo.label-filter" {
			labelPassed = true
		}
	})
	suiteConfig.LabelFilter = specrun.LabelFilter(labelPassed, suiteConfig.LabelFilter, os.Getenv("SPECTEST_LABEL_FILTER"))
	RegisterFailHandler(Fail)
	RunSpecs(t, "matrix self-spec", suiteConfig, reporterConfig)
}
