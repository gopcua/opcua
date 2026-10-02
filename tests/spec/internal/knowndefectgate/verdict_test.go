package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/onsi/ginkgo/v2/types"
)

var allSpecStates = []types.SpecState{
	types.SpecStateInvalid,
	types.SpecState(1),
	types.SpecStatePending,
	types.SpecStateSkipped,
	types.SpecStatePassed,
	types.SpecStateFailed,
	types.SpecStateAborted,
	types.SpecStatePanicked,
	types.SpecStateInterrupted,
	types.SpecStateTimedout,
}

const plainDefectMsg = "expected the client to send a Republish request, got none"
const passesNowMsg = "passes now; remove its known-defect label"
const harnessFailureMsg = "harness failure, not a defect"

func knownDefectIt(state types.SpecState, failure string) types.SpecReport {
	return types.SpecReport{
		ContainerHierarchyTexts: []string{"when the session survives a transport loss"},
		LeafNodeText:            "calls Republish from the next expected sequence number, incrementing",
		LeafNodeLabels:          []string{"known-defect"},
		LeafNodeType:            types.NodeTypeIt,
		State:                   state,
		Failure:                 types.Failure{Message: failure},
	}
}

func knownDefectBeforeEach(state types.SpecState, failure string) types.SpecReport {
	s := knownDefectIt(state, failure)
	s.Failure.FailureNodeType = types.NodeTypeBeforeEach
	return s
}

func withAdditional(s types.SpecReport, msgs ...string) types.SpecReport {
	for _, m := range msgs {
		s.AdditionalFailures = append(s.AdditionalFailures, types.AdditionalFailure{
			State:   types.SpecStateFailed,
			Failure: types.Failure{Message: m},
		})
	}
	return s
}

func assertVerdict(t *testing.T, name string, specs []types.SpecReport, log string, wantExit int, contains []string, namesSpec *types.SpecReport) {
	t.Helper()
	got, message := Verdict([]types.Report{{SpecReports: specs}}, log)
	if got != wantExit {
		t.Errorf("%s: Verdict exit = %d, want %d (message: %q)", name, got, wantExit, message)
	}
	for _, want := range contains {
		if !strings.Contains(message, want) {
			t.Errorf("%s: message %q does not contain %q", name, message, want)
		}
	}
	if namesSpec != nil {
		if want := namesSpec.FullText(); !strings.Contains(message, want) {
			t.Errorf("%s: message %q does not name the spec %q", name, message, want)
		}
	}
}

func TestVerdict(t *testing.T) {
	stateWant := map[types.SpecState]struct {
		wantExit int
		contains []string
		pair     bool
	}{
		types.SpecStateInvalid:     {wantExit: 1, contains: []string{"verdict unknown"}, pair: true},
		types.SpecState(1):         {wantExit: 1, contains: []string{"unknown spec state"}, pair: true},
		types.SpecStatePending:     {wantExit: 0, contains: []string{"pending, not run"}, pair: true},
		types.SpecStateSkipped:     {wantExit: 0, contains: []string{"skipped, not run"}, pair: true},
		types.SpecStatePassed:      {wantExit: 1, contains: []string{passesNowMsg}},
		types.SpecStateFailed:      {wantExit: 0, contains: []string{"failed, defect still present"}},
		types.SpecStateAborted:     {wantExit: 1, contains: []string{"verdict unknown"}},
		types.SpecStatePanicked:    {wantExit: 1, contains: []string{"verdict unknown", "check whether"}},
		types.SpecStateInterrupted: {wantExit: 1, contains: []string{"verdict unknown"}},
		types.SpecStateTimedout:    {wantExit: 1, contains: []string{"verdict unknown", "check whether"}},
	}
	checked := map[types.SpecState]bool{}
	for i := uint(0); i < 32; i++ {
		state := types.SpecState(1 << i)
		if state.String() == types.SpecStateInvalid.String() {
			continue
		}
		if _, ok := stateWant[state]; !ok {
			t.Fatalf("no rule recorded for SpecState %d", uint(state))
		}
		checked[state] = true
	}
	for state := range stateWant {
		if state == types.SpecStateInvalid {
			continue
		}
		if !checked[state] {
			t.Fatalf("SpecState %d was not verified by the bit loop", uint(state))
		}
	}

	for _, state := range allSpecStates {
		want, ok := stateWant[state]
		if !ok {
			t.Fatalf("no rule recorded for SpecState %d", uint(state))
		}
		spec := knownDefectIt(state, plainDefectMsg)
		specs := []types.SpecReport{spec}
		if want.pair {
			specs = append(specs, knownDefectIt(types.SpecStateFailed, plainDefectMsg))
		}
		var namesSpec *types.SpecReport
		switch state {
		case types.SpecStatePassed, types.SpecStateAborted, types.SpecStateInterrupted, types.SpecStateInvalid, types.SpecStatePanicked, types.SpecStateTimedout:
			namesSpec = &spec
		}
		assertVerdict(t, fmt.Sprintf("state %d", uint(state)), specs, "", want.wantExit, want.contains, namesSpec)
	}

	unknownPair := []types.SpecReport{
		knownDefectIt(types.SpecState(1<<9), plainDefectMsg),
		knownDefectIt(types.SpecStateFailed, plainDefectMsg),
	}
	assertVerdict(t, "unnamed spec state bit", unknownPair, "", 1, []string{"unknown spec state"}, &unknownPair[0])

	cases := []struct {
		name      string
		spec      types.SpecReport
		log       string
		wantExit  int
		contains  []string
		namesSpec bool
	}{
		{
			name:      "known-defect failed in an It with a spectest message",
			spec:      knownDefectIt(types.SpecStateFailed, "spectest: undecodable message at offset 3"),
			wantExit:  1,
			contains:  []string{harnessFailureMsg},
			namesSpec: true,
		},
		{
			name:     "known-defect failed in a BeforeEach with a plain message",
			spec:     knownDefectBeforeEach(types.SpecStateFailed, "client never entered Reconnecting within 15s of the cut"),
			wantExit: 0,
		},
		{
			name: "racy on the container, passed",
			spec: types.SpecReport{
				ContainerHierarchyTexts:  []string{"when the session survives a transport loss"},
				ContainerHierarchyLabels: [][]string{{"known-defect", "racy"}},
				LeafNodeText:             "keeps publishing when a pause and a resume arrive together",
				LeafNodeType:             types.NodeTypeIt,
				State:                    types.SpecStatePassed,
			},
			wantExit:  0,
			namesSpec: true,
		},
		{
			name: "known-defect only on the container, passed",
			spec: types.SpecReport{
				ContainerHierarchyTexts:  []string{"when the session survives a transport loss"},
				ContainerHierarchyLabels: [][]string{{"known-defect"}},
				LeafNodeText:             "sends no Publish until Republish has answered Bad_MessageNotAvailable",
				LeafNodeType:             types.NodeTypeIt,
				State:                    types.SpecStatePassed,
			},
			wantExit:  1,
			contains:  []string{passesNowMsg},
			namesSpec: true,
		},
		{
			name: "a known-defect It fails on main with a plain message",
			spec: types.SpecReport{
				ContainerHierarchyTexts: []string{"when Republish answers Bad_SubscriptionIdInvalid"},
				LeafNodeText:            "resumes publishing with the new subscription",
				LeafNodeLabels:          []string{"known-defect"},
				LeafNodeType:            types.NodeTypeIt,
				State:                   types.SpecStateFailed,
				Failure:                 types.Failure{Message: "client sent no Publish request"},
			},
			wantExit: 0,
		},
		{
			name:      "spectest message only in an AdditionalFailures entry",
			spec:      withAdditional(knownDefectIt(types.SpecStateFailed, plainDefectMsg), "spectest: answering a held Publish whose connection has closed"),
			wantExit:  1,
			contains:  []string{harnessFailureMsg},
			namesSpec: true,
		},
		{
			name:     "log contains a data race report",
			spec:     knownDefectIt(types.SpecStateFailed, plainDefectMsg),
			log:      "=== RUN   TestX\nWARNING: DATA RACE\nRead at 0x000000000000\n",
			wantExit: 1,
			contains: []string{"DATA RACE"},
		},
		{
			name:     "log contains the go test timeout panic",
			spec:     knownDefectIt(types.SpecStateFailed, plainDefectMsg),
			log:      "panic: test timed out after 30m0s\nrunning tests:\n",
			wantExit: 1,
			contains: []string{"test timed out"},
		},
		{
			name:     "all known-defect specs skipped",
			spec:     knownDefectIt(types.SpecStateSkipped, ""),
			wantExit: 1,
			contains: []string{"none of them ran"},
		},
		{
			name: "no known-defect specs in the report",
			spec: types.SpecReport{
				ContainerHierarchyTexts: []string{"when the session is gone"},
				LeafNodeText:            "creates a new session only after ActivateSession fails",
				LeafNodeType:            types.NodeTypeIt,
				State:                   types.SpecStatePassed,
			},
			wantExit: 0,
			contains: []string{"no spec in the report carries the known-defect label"},
		},
	}
	for _, c := range cases {
		var namesSpec *types.SpecReport
		if c.namesSpec {
			namesSpec = &c.spec
		}
		assertVerdict(t, c.name, []types.SpecReport{c.spec}, c.log, c.wantExit, c.contains, namesSpec)
	}
}

func runGated(t *testing.T, args []string) (exit int, stdout, stderr string) {
	t.Helper()
	dir := t.TempDir()
	stdoutPath := filepath.Join(dir, "stdout")
	stderrPath := filepath.Join(dir, "stderr")
	outFile, err := os.Create(stdoutPath)
	if err != nil {
		t.Fatal(err)
	}
	errFile, err := os.Create(stderrPath)
	if err != nil {
		_ = outFile.Close()
		t.Fatal(err)
	}
	oldStdout, oldStderr := os.Stdout, os.Stderr
	os.Stdout, os.Stderr = outFile, errFile
	exit = run(args)
	os.Stdout, os.Stderr = oldStdout, oldStderr
	if err := outFile.Close(); err != nil {
		t.Fatal(err)
	}
	if err := errFile.Close(); err != nil {
		t.Fatal(err)
	}
	read := func(p string) string {
		data, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}
	return exit, read(stdoutPath), read(stderrPath)
}

func TestRun(t *testing.T) {
	dir := t.TempDir()
	writeFile := func(name string, data []byte) string {
		t.Helper()
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, data, 0o600); err != nil {
			t.Fatal(err)
		}
		return p
	}
	marshalReports := func(specs ...types.SpecReport) []byte {
		data, err := json.Marshal([]types.Report{{SpecReports: specs}})
		if err != nil {
			t.Fatal(err)
		}
		return data
	}

	t.Run("report and log flow through to the exit code and message", func(t *testing.T) {
		reportPath := writeFile("report.json",
			marshalReports(knownDefectIt(types.SpecStateFailed, "spectest: undecodable message at offset 3")))
		logPath := writeFile("run.log", []byte("=== RUN TestX\n"))
		exit, stdout, stderr := runGated(t, []string{"knowndefectgate", reportPath, logPath})
		if exit != 1 {
			t.Errorf("exit = %d, want 1", exit)
		}
		if !strings.Contains(stdout, harnessFailureMsg) {
			t.Errorf("stdout %q does not contain %q", stdout, harnessFailureMsg)
		}
		if stderr != "" {
			t.Errorf("stderr %q, want empty", stderr)
		}
	})

	t.Run("wrong argument count fails the gate", func(t *testing.T) {
		exit, _, stderr := runGated(t, []string{"knowndefectgate"})
		if exit != 1 {
			t.Errorf("exit = %d, want 1", exit)
		}
		if !strings.Contains(stderr, "usage: knowndefectgate") {
			t.Errorf("stderr %q does not contain the usage line", stderr)
		}
	})

	t.Run("truncated report JSON fails the gate", func(t *testing.T) {
		reportPath := writeFile("truncated.json", []byte(`[{"SpecReports":`))
		if got := run([]string{"knowndefectgate", reportPath, writeFile("run.log", nil)}); got != 1 {
			t.Errorf("exit = %d, want 1", got)
		}
	})

	t.Run("empty report file fails the gate", func(t *testing.T) {
		reportPath := writeFile("empty.json", nil)
		if got := run([]string{"knowndefectgate", reportPath, writeFile("run.log", nil)}); got != 1 {
			t.Errorf("exit = %d, want 1", got)
		}
	})

	t.Run("missing report file fails the gate", func(t *testing.T) {
		reportPath := filepath.Join(dir, "absent.json")
		exit, _, stderr := runGated(t, []string{"knowndefectgate", reportPath, writeFile("run.log", nil)})
		if exit != 1 {
			t.Errorf("exit = %d, want 1", exit)
		}
		if !strings.Contains(stderr, "absent.json") {
			t.Errorf("stderr %q does not name the missing report", stderr)
		}
	})

	for _, c := range []struct {
		name string
		spec types.SpecReport
	}{
		{name: "an unreadable log fails the gate although the labelled spec still fails", spec: knownDefectIt(types.SpecStateFailed, plainDefectMsg)},
		{name: "an unreadable log fails the gate for a declared data-race spec", spec: dataRaceIt(types.SpecStatePassed)},
	} {
		t.Run(c.name, func(t *testing.T) {
			reportPath := writeFile("report.json", marshalReports(c.spec))
			logPath := filepath.Join(dir, "absent.log")
			exit, stdout, stderr := runGated(t, []string{"knowndefectgate", reportPath, logPath})
			if exit != 1 {
				t.Errorf("exit = %d, want 1: without the log the data-race and timeout-panic rules did not run", exit)
			}
			if want := "the log was not read, so the data-race and timeout-panic rules did not run\n"; stdout != want {
				t.Errorf("stdout %q, want exactly %q", stdout, want)
			}
			if !strings.Contains(stderr, "absent.log") {
				t.Errorf("stderr %q does not name the unreadable log", stderr)
			}
		})
	}
}

func TestVerdictCountsAnInvalidSpecAsRan(t *testing.T) {
	spec := knownDefectIt(types.SpecStateInvalid, plainDefectMsg)
	exit, message := Verdict([]types.Report{{SpecReports: []types.SpecReport{spec}}}, "")
	if exit != 1 {
		t.Errorf("Verdict exit = %d, want 1 for an invalid known-defect spec", exit)
	}
	if strings.Contains(message, "none of them ran") {
		t.Errorf("message %q also claims none of the known-defect specs ran", message)
	}
}

func dataRaceIt(state types.SpecState) types.SpecReport {
	return types.SpecReport{
		ContainerHierarchyTexts: []string{"when the client is closed while it re-dials"},
		LeafNodeText:            "does not race Close against the reconnect Dial",
		LeafNodeLabels:          []string{"known-defect"},
		LeafNodeType:            types.NodeTypeIt,
		State:                   state,
		ReportEntries: []types.ReportEntry{{
			Name:  "data-race",
			Value: types.WrapEntryValue([]string{"(*Client).Close", "(*Client).Dial"}),
		}},
	}
}

func raceLogBlock(frames ...string) string {
	var log strings.Builder
	log.WriteString("==================\n")
	log.WriteString("WARNING: DATA RACE\n")
	for i, frame := range frames {
		fmt.Fprintf(&log, "  %s\n", frame)
		if i < len(frames)-1 {
			log.WriteString("\n")
		}
	}
	log.WriteString("==================\n")
	return log.String()
}

func dataRaceItWithEntries(state types.SpecState, entries ...types.ReportEntry) types.SpecReport {
	spec := dataRaceIt(state)
	spec.ReportEntries = entries
	return spec
}

func dataRaceItFailing(state types.SpecState, failure string) types.SpecReport {
	spec := dataRaceIt(state)
	spec.Failure = types.Failure{Message: failure}
	return spec
}

func TestVerdictReadsARehydratedDataRaceDeclaration(t *testing.T) {
	encoded, err := json.Marshal([]types.Report{{SpecReports: []types.SpecReport{dataRaceIt(types.SpecStatePassed)}}})
	if err != nil {
		t.Fatal(err)
	}
	var rehydrated []types.Report
	if err := json.Unmarshal(encoded, &rehydrated); err != nil {
		t.Fatal(err)
	}
	log := raceLogBlock("github.com/gopcua/opcua.(*Client).Close()", "github.com/gopcua/opcua.(*Client).Dial()")
	exit, message := Verdict(rehydrated, log)
	if exit != 0 {
		t.Errorf("Verdict exit = %d, want 0, the rehydrated declaration must read as declared (message: %q)", exit, message)
	}
	if want := dataRaceIt(types.SpecStatePassed).FullText() + ": data race still present"; !strings.Contains(message, want) {
		t.Errorf("message %q does not carry the data-race verdict for the rehydrated declaration, want %q", message, want)
	}
}

func TestVerdictScansEverySpecInTheReport(t *testing.T) {
	unlabelled := types.SpecReport{
		ContainerHierarchyTexts: []string{"when the session is gone"},
		LeafNodeText:            "creates a new session only after ActivateSession fails",
		LeafNodeType:            types.NodeTypeIt,
		State:                   types.SpecStatePassed,
	}
	failing := knownDefectIt(types.SpecStateFailed, plainDefectMsg)

	t.Run("an unlabelled spec first does not stop the scan", func(t *testing.T) {
		exit, message := Verdict([]types.Report{{SpecReports: []types.SpecReport{unlabelled, failing}}}, "")
		if exit != 0 {
			t.Errorf("Verdict exit = %d, want 0, the failing known-defect spec is the defect", exit)
		}
		if want := failing.FullText() + ": failed, defect still present"; !strings.Contains(message, want) {
			t.Errorf("message %q does not report the labelled spec after the unlabelled one, want %q", message, want)
		}
	})

	t.Run("a data-race verdict does not stop the scan", func(t *testing.T) {
		racing := dataRaceIt(types.SpecStatePassed)
		log := raceLogBlock("github.com/gopcua/opcua.(*Client).Close()", "github.com/gopcua/opcua.(*Client).Dial()")
		exit, message := Verdict([]types.Report{{SpecReports: []types.SpecReport{racing, failing}}}, log)
		if exit != 0 {
			t.Errorf("Verdict exit = %d, want 0, both verdicts are the defect present", exit)
		}
		if !strings.Contains(message, racing.FullText()+": data race still present") {
			t.Errorf("message %q does not carry the data-race verdict", message)
		}
		if !strings.Contains(message, failing.FullText()+": failed, defect still present") {
			t.Errorf("message %q does not carry the failing spec's verdict after the data-race one", message)
		}
	})

	t.Run("a malformed data-race declaration does not stop the scan", func(t *testing.T) {
		malformed := dataRaceItWithEntries(types.SpecStatePassed,
			types.ReportEntry{Name: "data-race", Value: types.WrapEntryValue(42)})
		exit, message := Verdict([]types.Report{{SpecReports: []types.SpecReport{malformed, failing}}}, "")
		if exit != 1 {
			t.Errorf("Verdict exit = %d, want 1, the malformed declaration fails the gate", exit)
		}
		if !strings.Contains(message, malformed.FullText()+": data-race entry is not a list of function names") {
			t.Errorf("message %q does not carry the malformed-declaration failure", message)
		}
		if !strings.Contains(message, failing.FullText()+": failed, defect still present") {
			t.Errorf("message %q does not carry the failing spec's verdict after the malformed one", message)
		}
	})
}

func TestVerdictDataRaceRule(t *testing.T) {
	closing := "github.com/gopcua/opcua.(*Client).Close()"
	dialing := "github.com/gopcua/opcua.(*Client).Dial()"
	other := "github.com/gopcua/opcua.(*Client).Connect()"

	cases := []struct {
		name        string
		specs       []types.SpecReport
		log         string
		wantExit    int
		contains    []string
		notContains []string
		namesSpec   bool
	}{
		{
			name:      "a block naming both declared functions is the defect still present",
			specs:     []types.SpecReport{dataRaceIt(types.SpecStatePassed)},
			log:       raceLogBlock(closing, dialing),
			wantExit:  0,
			contains:  []string{"data race still present"},
			namesSpec: true,
		},
		{
			name:      "every block naming both declared functions is exempt, however many fire",
			specs:     []types.SpecReport{dataRaceIt(types.SpecStatePassed)},
			log:       raceLogBlock(closing, dialing) + raceLogBlock(closing, dialing) + raceLogBlock(closing, dialing),
			wantExit:  0,
			contains:  []string{"data race still present"},
			namesSpec: true,
		},
		{
			name:      "no block in the log means the label must go",
			specs:     []types.SpecReport{dataRaceIt(types.SpecStatePassed)},
			log:       "=== RUN TestPart4\n",
			wantExit:  1,
			contains:  []string{passesNowMsg},
			namesSpec: true,
		},
		{
			name:      "a block naming only one declared function counts as a data race and the spec as passing",
			specs:     []types.SpecReport{dataRaceIt(types.SpecStatePassed)},
			log:       raceLogBlock(closing, other),
			wantExit:  1,
			contains:  []string{"DATA RACE", passesNowMsg},
			namesSpec: true,
		},
		{
			name:     "no data-race spec ran, so the block is today's plain data race",
			specs:    []types.SpecReport{knownDefectIt(types.SpecStateFailed, plainDefectMsg)},
			log:      raceLogBlock(closing, dialing),
			wantExit: 1,
			contains: []string{"DATA RACE"},
		},
		{
			name: "an earlier entry with another name does not hide the data-race entry",
			specs: []types.SpecReport{dataRaceItWithEntries(types.SpecStatePassed,
				types.ReportEntry{Name: "teardown-note", Value: types.WrapEntryValue("closed")},
				types.ReportEntry{Name: "data-race", Value: types.WrapEntryValue([]string{"(*Client).Close", "(*Client).Dial"})},
			)},
			log:       raceLogBlock(closing, dialing),
			wantExit:  0,
			contains:  []string{"data race still present"},
			namesSpec: true,
		},
		{
			name: "a data-race entry whose value holds a non-string fails the gate",
			specs: []types.SpecReport{dataRaceItWithEntries(types.SpecStatePassed,
				types.ReportEntry{Name: "data-race", Value: types.WrapEntryValue([]any{"(*Client).Close", 42})},
			)},
			log:       raceLogBlock(closing, dialing),
			wantExit:  1,
			contains:  []string{"data-race entry is not a list of function names"},
			namesSpec: true,
		},
		{
			name: "a data-race entry holding an empty list fails the gate",
			specs: []types.SpecReport{dataRaceItWithEntries(types.SpecStatePassed,
				types.ReportEntry{Name: "data-race", Value: types.WrapEntryValue([]string{})},
			)},
			log:       "",
			wantExit:  1,
			contains:  []string{"data-race entry is not a list of function names"},
			namesSpec: true,
		},
		{
			name: "a later malformed data-race entry fails the gate even after a valid one",
			specs: []types.SpecReport{dataRaceItWithEntries(types.SpecStatePassed,
				types.ReportEntry{Name: "data-race", Value: types.WrapEntryValue([]string{"(*Client).Close", "(*Client).Dial"})},
				types.ReportEntry{Name: "data-race", Value: types.WrapEntryValue(42)},
			)},
			log:       raceLogBlock(closing, dialing),
			wantExit:  1,
			contains:  []string{"data-race entry is not a list of function names"},
			namesSpec: true,
		},
		{
			name: "a second header ends the open block, so two blocks are read",
			specs: []types.SpecReport{dataRaceItWithEntries(types.SpecStatePassed,
				types.ReportEntry{Name: "data-race", Value: types.WrapEntryValue([]string{"(*Client).Close"})},
			)},
			log:       "WARNING: DATA RACE\n" + closing + "\nWARNING: DATA RACE\n" + other + "\n==================\n",
			wantExit:  1,
			contains:  []string{"data race still present", "DATA RACE in " + other},
			namesSpec: true,
		},
		{
			name:     "a log that ends inside an undeclared block still reports the race",
			specs:    []types.SpecReport{knownDefectIt(types.SpecStateFailed, plainDefectMsg)},
			log:      "WARNING: DATA RACE\n" + closing + "\n",
			wantExit: 1,
			contains: []string{"DATA RACE in " + closing},
		},
		{
			name:      "a declared spec that failed with a spectest message is a harness failure and its block is reported",
			specs:     []types.SpecReport{dataRaceItFailing(types.SpecStateFailed, "spectest: the harness broke")},
			log:       raceLogBlock(closing, dialing),
			wantExit:  1,
			contains:  []string{"harness failure", "DATA RACE in"},
			namesSpec: true,
		},
		{
			name:      "a declared spec that was skipped did not run and its block is unmatched",
			specs:     []types.SpecReport{dataRaceIt(types.SpecStateSkipped)},
			log:       raceLogBlock(closing, dialing),
			wantExit:  1,
			contains:  []string{"not run", "none of them ran", "DATA RACE in"},
			namesSpec: true,
		},
		{
			name:      "a declared spec that panicked is a verdict unknown",
			specs:     []types.SpecReport{dataRaceIt(types.SpecStatePanicked)},
			log:       raceLogBlock(closing, dialing),
			wantExit:  1,
			contains:  []string{"verdict unknown", "check whether"},
			namesSpec: true,
		},
		{
			name:      "a declared spec that failed with a plain message asserts nothing, so its verdict is unknown",
			specs:     []types.SpecReport{dataRaceItFailing(types.SpecStateFailed, "the client never connected through the relay: dial refused")},
			log:       "",
			wantExit:  1,
			contains:  []string{": failed, but it declares a data race and asserts nothing; verdict unknown"},
			namesSpec: true,
		},
		{
			name: "a data-race entry naming the empty function fails the gate",
			specs: []types.SpecReport{dataRaceItWithEntries(types.SpecStatePassed,
				types.ReportEntry{Name: "data-race", Value: types.WrapEntryValue([]string{""})},
			)},
			log:       raceLogBlock(other),
			wantExit:  1,
			contains:  []string{"data-race entry is not a list of function names"},
			namesSpec: true,
		},
		{
			name: "a data-race entry naming the empty function in the rehydrated form fails the gate",
			specs: []types.SpecReport{dataRaceItWithEntries(types.SpecStatePassed,
				types.ReportEntry{Name: "data-race", Value: types.WrapEntryValue([]any{""})},
			)},
			log:       raceLogBlock(other),
			wantExit:  1,
			contains:  []string{"data-race entry is not a list of function names"},
			namesSpec: true,
		},
		{
			name:        "a second, undeclared race block fails the run",
			specs:       []types.SpecReport{dataRaceIt(types.SpecStatePassed)},
			log:         raceLogBlock(closing, dialing) + raceLogBlock(other),
			wantExit:    1,
			contains:    []string{"data race still present", "DATA RACE"},
			notContains: []string{passesNowMsg},
			namesSpec:   true,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			exit, message := Verdict([]types.Report{{SpecReports: c.specs}}, c.log)
			if exit != c.wantExit {
				t.Errorf("Verdict exit = %d, want %d (message: %q)", exit, c.wantExit, message)
			}
			for _, want := range c.contains {
				if !strings.Contains(message, want) {
					t.Errorf("message %q does not contain %q", message, want)
				}
			}
			for _, unwanted := range c.notContains {
				if strings.Contains(message, unwanted) {
					t.Errorf("message %q must not contain %q", message, unwanted)
				}
			}
			if c.namesSpec {
				if want := c.specs[0].FullText(); !strings.Contains(message, want) {
					t.Errorf("message %q does not name the spec %q", message, want)
				}
			}
		})
	}
}
