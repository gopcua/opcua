// Package matrix runs the failure matrix: a clause file declares each
// scenario as a value and registers it, and the matrix plans one case
// per fault with its checks and known-defect labels, then runs a real
// client through each case and asserts the invariants and rules
// against what it observed.
package matrix

import (
	"maps"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/gopcua/opcua/tests/spec/internal/fault"
	"github.com/gopcua/opcua/tests/spec/internal/harness"
	"github.com/gopcua/opcua/tests/spec/internal/rules"
)

// Outcome is what a scenario's Workload observed: the armed fault, the
// fault's end event, the sentinel value with the moment its answer
// was sent, and the rules' context.
type Outcome struct {
	Injected   *harness.Injected
	FaultEnd   time.Time
	Sentinel   int32
	AnsweredAt time.Time
	Rules      rules.Context
}

// Phase says whether a check runs before or after the client closes.
type Phase int

// The zero Phase is invalid; no check carries it.
const (
	// BeforeClose: the check reads the first snapshot, taken while the
	// client is still connected.
	BeforeClose Phase = iota + 1
	// AfterClose: the check reads the second snapshot, taken after the
	// client closed.
	AfterClose
)

// Check is one assertion a case runs: an invariant's or a rule's
// name, the labels it carries and the phase it runs in.
type Check struct {
	Name   string
	Labels []string
	Phase  Phase
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

func withLabels(base []string, extra ...string) []string {
	return append(append([]string{}, base...), extra...)
}

func faultGroup(f fault.Fault) string {
	group := f.Name()
	if index := strings.Index(group, "/"); index >= 0 {
		group = group[:index]
	}
	switch group {
	case "RequestLost", "ResponseLost", "CutAfterResponse", "DelayBelowTimeout", "DelayAboveTimeout":
		return "message"
	}
	return strings.ToLower(group)
}

// faultTarget returns the message name of a message-targeting fault
// and whether it names one.
func faultTarget(f fault.Fault) (string, bool) {
	name := f.Name()
	parts := strings.Split(name, "/")
	switch parts[0] {
	case "Overload", "DelayBelowTimeout", "DelayAboveTimeout", "RequestLost", "ResponseLost", "CutAfterResponse":
		if len(parts) < 2 {
			return "", false
		}
		return parts[1], true
	}
	return "", false
}

var (
	unfiledMu    sync.Mutex
	unfiledTexts = make(map[string]string)
)

// slugPattern says which slugs Unfiled accepts.
var slugPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// Unfiled records a finding no issue names yet and returns the label
// its defect carries, so the label names the defect and survives
// moving it to another file. The PR body lists UnfiledDefects so each
// one gets an issue or an explanation.
func Unfiled(slug, text string) string {
	unfiledMu.Lock()
	defer unfiledMu.Unlock()
	if !slugPattern.MatchString(slug) {
		panic("Unfiled: slug " + slug + " does not match " + slugPattern.String())
	}
	label := "unfiled-" + slug
	if old, recorded := unfiledTexts[label]; recorded && old != text {
		panic("Unfiled: slug " + slug + " already records " + old + ", not " + text)
	}
	unfiledTexts[label] = text
	return label
}

// UnfiledDefects lists every text Unfiled recorded, each paired with
// the label its defect carries, sorted by label.
func UnfiledDefects() []string {
	unfiledMu.Lock()
	defer unfiledMu.Unlock()
	var texts []string
	for _, label := range slices.Sorted(maps.Keys(unfiledTexts)) {
		texts = append(texts, label+": "+unfiledTexts[label])
	}
	return texts
}
