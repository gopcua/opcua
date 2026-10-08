package matrix

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/gopcua/opcua/tests/spec/internal/fault"
	"github.com/gopcua/opcua/tests/spec/internal/harness"
	"github.com/gopcua/opcua/tests/spec/internal/message"
	"github.com/gopcua/opcua/tests/spec/internal/rules"
	"github.com/gopcua/opcua/tests/spec/internal/specrun"
)

const (
	failFirstValue1   int32 = 351
	failFirstValue2   int32 = 352
	failFirstValue3   int32 = 353
	failFirstSentinel int32 = 399
)

// failFirstSuite is the self-spec of the case runner's failure
// continuation: one case whose first check fails while every later
// check passes, because the fault arms on a message the workload's
// client never sends.
type failFirstSuite struct{}

func (failFirstSuite) Clause() string { return "P4-COF" }

func (failFirstSuite) Scenarios() []SuiteScenario {
	return []SuiteScenario{failFirstScenario{}}
}

func (failFirstSuite) Rules(Category) []rules.Rule { return nil }

type failFirstScenario struct{}

func (failFirstScenario) Name() string { return "Idle" }

func (failFirstScenario) Sends() []message.Message {
	return []message.Message{message.Read}
}

func (failFirstScenario) Options(fault.Fault) []harness.Option { return nil }

func (failFirstScenario) Category(fault.Fault) Category { return Unspecified }

func (failFirstScenario) Run(env *harness.Environment, f fault.Fault) Outcome {
	sub := env.Subscription()
	env.Server.WaitHeldPublish().Answer(sub, failFirstValue1)
	env.Server.WaitHeldPublish().Answer(sub, failFirstValue2)
	injected := f.Inject(env)
	env.Server.WaitHeldPublish().Answer(sub, failFirstValue3)
	faultEnd := time.Now()
	answeredAt := time.Now()
	env.Server.WaitHeldPublish().Answer(sub, failFirstSentinel)
	return Outcome{
		Injected:   injected,
		FaultEnd:   faultEnd,
		Sentinel:   failFirstSentinel,
		AnsweredAt: answeredAt,
	}
}

// The failure-continuation suite registers only in the nested run, so
// neither the plain self-spec bootstrap nor the failure matrix run
// carries its deliberately failing case.
func init() {
	if os.Getenv("SPECTEST_FAILFIRST") != "" {
		RegisterSuites([]Suite{failFirstSuite{}}, nil, faultNamed("RequestLost/Read"))
	}
}

// TestContinueOnFailure asserts the case container runs every check of
// a case whose first check fails: one failing check retires only
// itself, never the checks behind it. It spawns a nested run of the
// self-spec bootstrap with the failure-continuation suite registered,
// reads that run's report and counts the case's check states.
func TestContinueOnFailure(t *testing.T) {
	if os.Getenv("SPECTEST_FAILFIRST") != "" {
		t.Skip("the nested run of this test would recurse")
		return
	}
	if specrun.MatrixRuns() {
		t.Skip("the matrix run covers the case container without this probe")
		return
	}
	report := filepath.Join(t.TempDir(), "report.json")
	cmd := exec.Command("go", "test", "-count=1", "-run", "TestSelfSpec", ".", "-ginkgo.json-report="+report)
	cmd.Env = append(os.Environ(), "SPECTEST_FAILFIRST=1")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("the nested self-spec run passed although its first check fails on purpose; output:\n%s", out)
	}
	counts := parseFailFirstChecks(t, report)
	if counts["failed"] != 1 {
		t.Fatalf("the failure-continuation case reports %d failed checks, want exactly the first one; counts: %v; output:\n%s", counts["failed"], counts, out)
	}
	if counts["passed"] != 6 {
		t.Fatalf("the failure-continuation case reports %d passed checks, want the six behind the failing first one; counts: %v; output:\n%s", counts["passed"], counts, out)
	}
	if counts["skipped"] != 0 {
		t.Fatalf("the failure-continuation case reports %d skipped checks, want none; counts: %v; output:\n%s", counts["skipped"], counts, out)
	}
}

// parseFailFirstChecks reads the nested run's report and counts the
// check states of the failure-continuation suite's specs. ginkgo
// writes one report object per RunSpecs call.
func parseFailFirstChecks(t *testing.T, report string) map[string]int {
	t.Helper()
	raw, err := os.ReadFile(report)
	if err != nil {
		t.Fatalf("reading the nested run's report failed: %v", err)
	}
	var runs []struct {
		SpecReports []struct {
			ContainerHierarchyTexts []string
			LeafNodeText            string
			State                   string
		}
	}
	if err := json.Unmarshal(raw, &runs); err != nil {
		t.Fatalf("parsing the nested run's report failed: %v", err)
	}
	counts := map[string]int{}
	for _, run := range runs {
		for _, spec := range run.SpecReports {
			if len(spec.ContainerHierarchyTexts) == 0 || spec.ContainerHierarchyTexts[0] != "P4-COF" {
				continue
			}
			if spec.LeafNodeText == "" {
				continue
			}
			counts[spec.State]++
		}
	}
	return counts
}
