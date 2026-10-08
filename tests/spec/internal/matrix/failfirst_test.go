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
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

const (
	failFirstValue1   int32 = 351
	failFirstValue2   int32 = 352
	failFirstValue3   int32 = 353
	failFirstSentinel int32 = 399
)

// failFirstInvariants are the stand-ins the failure-continuation case
// asserts in place of the subscription invariants, which live in part4
// and cannot be imported here. Every stand-in reads the after-close
// snapshot, so HaveFired, which the matrix adds between the rules and
// the after-close invariants, is the case's first check and fails
// while the stand-ins behind it pass. failFirst's doc says why the
// fault never fires.
var failFirstInvariants = []Invariant{
	{Name: "StandInReceivedAnswers", Phase: AfterClose, Assert: func(observed Observed) {
		gomega.Expect(observed.Received).To(gomega.ContainElements(failFirstValue1, failFirstValue2, failFirstValue3, failFirstSentinel), "the stand-in client missed an answered value")
	}},
	{Name: "StandInEachValueOnce", Phase: AfterClose, Assert: func(observed Observed) {
		counts := map[int32]int{}
		for _, value := range observed.Received {
			counts[value]++
		}
		for value, count := range counts {
			gomega.Expect(count).To(gomega.Equal(1), "the stand-in client received %d %d times", value, count)
		}
	}},
	{Name: "StandInOnlyProduced", Phase: AfterClose, Assert: func(observed Observed) {
		produced := map[int32]bool{}
		for _, entry := range observed.Produced {
			produced[entry.Value] = true
		}
		for _, value := range observed.Received {
			gomega.Expect(produced[value]).To(gomega.BeTrue(), "the stand-in client received %d, which no server produced", value)
		}
	}},
	{Name: "StandInOneServer", Phase: AfterClose, Assert: func(observed Observed) {
		gomega.Expect(observed.Servers).To(gomega.HaveLen(1), "the stand-in environment created the wrong server count")
	}},
	{Name: "StandInSentinelReceived", Phase: AfterClose, Assert: func(observed Observed) {
		gomega.Expect(observed.Sentinel).NotTo(gomega.BeNil(), "the stand-in client staged no sentinel")
		gomega.Expect(observed.Sentinel.ReceivedAt).NotTo(gomega.BeZero(), "the stand-in client never received the sentinel")
	}},
	{Name: "StandInSessionsClosed", Phase: AfterClose, Assert: func(observed Observed) {
		for _, server := range observed.Servers {
			gomega.Expect(server.KnownSessions-server.ClosingAttempted).To(gomega.BeZero(), "the stand-in server %d still holds a session", server.Index)
		}
	}},
}

// failFirst is the self-spec of the case runner's failure
// continuation: one case whose first check fails while every later
// check passes, because the fault arms on a message the workload's
// client never sends.
var failFirst = Scenario{
	Clause:  "P4-COF",
	Name:    "Idle",
	Ordinal: 2,
	Sends:   []message.Message{message.Read},
	Workload: func(env *harness.Environment, f fault.Fault, _ int32) Outcome {
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
	},
	Invariants: failFirstInvariants,
}

// The failure-continuation suite registers only in the nested run, so
// neither the plain self-spec bootstrap nor the failure matrix run
// carries its deliberately failing case.
func init() {
	if os.Getenv("SPECTEST_FAILFIRST") != "" {
		ginkgo.Describe(failFirst.Clause, func() {
			ginkgo.DescribeTableSubtree(failFirst.Name, func(f fault.Fault) {
				obs := Run(failFirst, f)
				obs.BeforeCloseInvariants()
				obs.AfterCloseInvariants()
			}, Entries(failFirst, faultNamed("RequestLost/Read")))
		})
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
