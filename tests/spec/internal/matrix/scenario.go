package matrix

import (
	"fmt"
	"reflect"
	"slices"
	"sync"

	"github.com/gopcua/opcua/tests/spec/internal/fault"
	"github.com/gopcua/opcua/tests/spec/internal/harness"
	"github.com/gopcua/opcua/tests/spec/internal/message"
	"github.com/onsi/ginkgo/v2"
)

// Scenario is one workload a clause file crosses with the fault
// catalogue. A clause file declares it as a value and registers it
// with ginkgo.DescribeTableSubtree, Entries and Run, and writes the
// case's check Its with the Observation that Run returns.
type Scenario struct {
	// Clause is the text of the clause container and the first part
	// of every spec path, "P4-6.7" for example. A spec path is the
	// chain of ginkgo container texts that locates one spec:
	// clause, scenario, fault, then the check's own name as the
	// leaf.
	Clause string
	// Name is the text of the scenario container.
	Name string
	// Ordinal numbers the scenario's value blocks, counting from 1.
	// Two scenarios of one test binary never share an ordinal.
	Ordinal int
	// Sends lists the messages a correct client sends after the arm
	// point. A fault on any other message does not apply.
	Sends []message.Message
	// Options returns the harness options one case needs, none when it
	// is nil. block is the first number of the case's value block.
	Options func(f fault.Fault, block int32) []harness.Option
	// Workload drives the client through one case and injects the
	// fault at its arm point.
	Workload func(env *harness.Environment, f fault.Fault, block int32) Outcome
	// Rules returns the rules the clause prescribes under the fault.
	// None means the case asserts the invariants only.
	Rules func(f fault.Fault) []Rule
	// Invariants are the checks the workload owes under every fault,
	// because of what kind of workload it is.
	Invariants []Invariant
}

// Invariant is one property a kind of workload owes under every fault:
// its name, the snapshot it reads and the assertion on that snapshot.
type Invariant struct {
	Name   string
	Phase  Phase
	Assert func(observed Observed)
}

var (
	registeredMu sync.Mutex
	// ordinalOwner maps each ordinal to the clause and name of the
	// scenario that holds it, and ordinalOf the reverse.
	ordinalOwner = make(map[int]string)
	ordinalOf    = make(map[string]int)
)

// register records the scenario's ordinal and name for this test
// binary. It returns an error when another scenario holds the ordinal
// or another scenario of the clause has the name.
func register(s Scenario) error {
	registeredMu.Lock()
	defer registeredMu.Unlock()
	id := s.Clause + "/" + s.Name
	if owner, taken := ordinalOwner[s.Ordinal]; taken && owner != id {
		return fmt.Errorf("scenario %s has ordinal %d, which %s already holds", id, s.Ordinal, owner)
	}
	if ordinal, named := ordinalOf[id]; named && ordinal != s.Ordinal {
		return fmt.Errorf("two scenarios are named %s, with ordinals %d and %d", id, ordinal, s.Ordinal)
	}
	ordinalOwner[s.Ordinal] = id
	ordinalOf[id] = s.Ordinal
	return nil
}

// Entries returns one ginkgo.Entry per fault for the scenario's
// ginkgo.DescribeTableSubtree. The entry's text is the fault's name.
// An entry whose fault applies is Ordered and ContinueOnFailure, so
// its checks share one run of the workload, and one failing check
// retires only itself. It panics when the scenario's cases cannot be
// planned, when another scenario already holds its ordinal, or when
// another scenario of the clause has its name.
func Entries(s Scenario, faults ...fault.Fault) []ginkgo.TableEntry {
	if _, err := s.Cases(faults); err != nil {
		panic(err)
	}
	if err := register(s); err != nil {
		panic(err)
	}
	var entries []ginkgo.TableEntry
	for _, f := range faults {
		if f.Available(s.Sends) != nil {
			entries = append(entries, ginkgo.Entry(f.Name(), f))
			continue
		}
		entries = append(entries, ginkgo.Entry(f.Name(), f, ginkgo.Ordered, ginkgo.ContinueOnFailure))
	}
	return entries
}

// Cases plans the scenario over the faults: one case per fault, each
// with its value block and its checks in the order they run. It
// returns an error when the ordinal is not positive or a rule the
// scenario prescribes is malformed or shares an invariant's name.
func (s Scenario) Cases(faults []fault.Fault) ([]Case, error) {
	if s.Ordinal < 1 {
		return nil, fmt.Errorf("scenario %s/%s has ordinal %d; ordinals count from 1", s.Clause, s.Name, s.Ordinal)
	}
	if err := s.validateRules(faults); err != nil {
		return nil, err
	}
	var cases []Case
	for _, f := range faults {
		cases = append(cases, s.caseOf(f))
	}
	return cases, nil
}

// validateRules rejects the rules the scenario prescribes under the
// faults. A rule needs a name of its own and a Check, because a
// case's checks are asserted by name: a rule that shares an
// invariant's name plans two checks under one name, and a name two
// rules share leaves the case unable to pick either. The
// shared-name rejection compares the rules' Check functions by code
// pointer, so two closures built from one function literal count as
// one rule.
func (s Scenario) validateRules(faults []fault.Fault) error {
	if s.Rules == nil {
		return nil
	}
	checkOf := map[string]uintptr{}
	for _, f := range faults {
		for _, rule := range s.rulesUnder(f) {
			if rule.Name == "" {
				return fmt.Errorf("scenario %s/%s prescribes a rule with an empty Name under %s", s.Clause, s.Name, f.Name())
			}
			if rule.Check == nil {
				return fmt.Errorf("the rule %s of scenario %s/%s under %s has no Check", rule.Name, s.Clause, s.Name, f.Name())
			}
			if rule.Name == haveFiredInvariant.Name || slices.ContainsFunc(s.Invariants, func(invariant Invariant) bool { return invariant.Name == rule.Name }) {
				return fmt.Errorf("the rule %s of scenario %s/%s under %s shares an invariant's name, so the case plans two checks under one name", rule.Name, s.Clause, s.Name, f.Name())
			}
			check := reflect.ValueOf(rule.Check).Pointer()
			if previous, seen := checkOf[rule.Name]; seen && previous != check {
				return fmt.Errorf("the rules %s of scenario %s/%s under %s share one name and differ in Check", rule.Name, s.Clause, s.Name, f.Name())
			}
			checkOf[rule.Name] = check
		}
	}
	return nil
}

// caseOf plans one case: its path, its value block, and either the
// reason the fault does not apply or its checks. The invariants read
// before the client closes come first, then the rules, then the
// HaveFired check, then the invariants read after it closes.
func (s Scenario) caseOf(f fault.Fault) Case {
	c := Case{Path: []string{s.Clause, s.Name, f.Name()}, Block: s.block(f)}
	if reason := f.Available(s.Sends); reason != nil {
		c.Skip = reason
		return c
	}
	invariantsIn := func(phase Phase) {
		for _, invariant := range s.Invariants {
			if invariant.Phase == phase {
				c.Checks = append(c.Checks, Check{Name: invariant.Name, Phase: phase})
			}
		}
	}
	invariantsIn(BeforeClose)
	for _, rule := range s.rulesUnder(f) {
		c.Checks = append(c.Checks, Check{Name: rule.Name, Phase: BeforeClose})
	}
	c.Checks = append(c.Checks, Check{Name: haveFiredInvariant.Name, Phase: haveFiredInvariant.Phase})
	invariantsIn(AfterClose)
	return c
}

// rulesUnder returns the rules the clause prescribes under the fault,
// none when the scenario declares no Rules.
func (s Scenario) rulesUnder(f fault.Fault) []Rule {
	if s.Rules == nil {
		return nil
	}
	return s.Rules(f)
}

// block returns the first number of the case's value block: 1000 times
// the case's ordinal among every scenario × fault pair, so every value
// a case answers is its block plus an offset below 1000, and no two
// cases share a value.
func (s Scenario) block(f fault.Fault) int32 {
	faultOrdinal := slices.IndexFunc(fault.AllFaults, func(candidate fault.Fault) bool {
		return candidate.Name() == f.Name()
	})
	return int32(1000 * ((s.Ordinal-1)*len(fault.AllFaults) + faultOrdinal + 1))
}
