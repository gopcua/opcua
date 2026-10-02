package main

import (
	"fmt"
	"slices"
	"strings"

	"github.com/onsi/ginkgo/v2/types"
)

// Verdict reports whether the known-defect labelled specs in the Ginkgo
// reports still fail as their labels claim: a spec counts when its own or
// its container labels include "known-defect", and "racy" is read the same
// way. The exit is 1 when a labelled spec now passes, when every labelled
// spec was skipped, when one holds a spec state the switch below counts
// as a verdict-unknown failure, or when the log shows a data race or a
// test timeout. A racy pass, the lines naming the specs that did not run, and
// the line saying that no spec in the report carries the label are warnings
// with exit 0. A failure message starting "spectest:" is a harness
// failure, not the defect.
func Verdict(reports []types.Report, log string) (exit int, message string) {
	var lines []string
	fail := func(format string, args ...any) {
		exit = 1
		lines = append(lines, fmt.Sprintf(format, args...))
	}
	knownDefectSeen := false
	ran := false
	for _, report := range reports {
		for _, spec := range report.SpecReports {
			if !slices.Contains(spec.Labels(), "known-defect") {
				continue
			}
			knownDefectSeen = true
			name := spec.FullText()
			switch spec.State {
			case types.SpecStatePassed:
				ran = true
				if slices.Contains(spec.Labels(), "racy") {
					lines = append(lines, name+": passes, but is racy, so the pass is not evidence the defect is fixed")
				} else {
					fail("%s: passes now; remove its known-defect label", name)
				}
			case types.SpecStateFailed:
				ran = true
				if harness, ok := spectestFailure(spec); ok {
					fail("%s: harness failure, not a defect: %s", name, harness)
					continue
				}
				lines = append(lines, name+": failed, defect still present")
			case types.SpecStatePanicked, types.SpecStateTimedout:
				ran = true
				fail("%s: run %s, verdict unknown; check whether the %s is the defect", name, spec.State, spec.State)
			case types.SpecStateAborted, types.SpecStateInterrupted:
				ran = true
				fail("%s: run %s, verdict unknown", name, spec.State)
			case types.SpecStatePending, types.SpecStateSkipped:
				lines = append(lines, fmt.Sprintf("%s: %s, not run", name, spec.State))
			case types.SpecStateInvalid:
				ran = true
				fail("%s: spec state %s, verdict unknown", name, spec.State)
			default:
				fail("%s: unknown spec state %d", name, uint(spec.State))
			}
		}
	}
	if line := lineContaining(log, "WARNING: DATA RACE"); line != "" {
		fail("log: %s", line)
	}
	if line := lineContaining(log, "panic: test timed out after"); line != "" {
		fail("log: %s", line)
	}
	if !knownDefectSeen {
		lines = append(lines, "no spec in the report carries the known-defect label")
	}
	if knownDefectSeen && !ran {
		fail("known-defect specs in the report, but none of them ran; was the label filter applied?")
	}
	return exit, strings.Join(lines, "\n")
}

func spectestFailure(spec types.SpecReport) (string, bool) {
	if strings.HasPrefix(spec.Failure.Message, "spectest:") {
		return spec.Failure.Message, true
	}
	for _, additional := range spec.AdditionalFailures {
		if strings.HasPrefix(additional.Failure.Message, "spectest:") {
			return additional.Failure.Message, true
		}
	}
	return "", false
}

func lineContaining(log, needle string) string {
	for _, line := range strings.Split(log, "\n") {
		if strings.Contains(line, needle) {
			return strings.TrimSpace(line)
		}
	}
	return ""
}
