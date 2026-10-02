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
	got, msg := Verdict([]types.Report{{SpecReports: specs}}, log)
	if got != wantExit {
		t.Errorf("%s: Verdict exit = %d, want %d (message: %q)", name, got, wantExit, msg)
	}
	for _, want := range contains {
		if !strings.Contains(msg, want) {
			t.Errorf("%s: message %q does not contain %q", name, msg, want)
		}
	}
	if namesSpec != nil {
		if want := namesSpec.FullText(); !strings.Contains(msg, want) {
			t.Errorf("%s: message %q does not name the spec %q", name, msg, want)
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

	t.Run("unreadable log warns and the report alone decides", func(t *testing.T) {
		reportPath := writeFile("report.json",
			marshalReports(knownDefectIt(types.SpecStateFailed, plainDefectMsg)))
		logPath := filepath.Join(dir, "absent.log")
		exit, _, stderr := runGated(t, []string{"knowndefectgate", reportPath, logPath})
		if exit != 0 {
			t.Errorf("exit = %d, want 0: the labelled spec still fails, so the defect is present", exit)
		}
		for _, want := range []string{"absent.log", "rules were skipped"} {
			if !strings.Contains(stderr, want) {
				t.Errorf("stderr %q does not contain %q", stderr, want)
			}
		}
	})
}
