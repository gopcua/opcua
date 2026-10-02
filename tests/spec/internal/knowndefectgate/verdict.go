package main

import (
	"fmt"
	"slices"
	"strings"

	"github.com/onsi/ginkgo/v2/types"
)

// Verdict reports whether the known-defect labelled specs in the Ginkgo
// reports still fail as their labels claim, and whether the log shows a
// data race no spec declared. A spec that records a report entry named
// data-race takes its verdict from the log instead of its state: a
// WARNING: DATA RACE block naming every declared function is the defect
// still present, and no such block means the label must go. A failure
// message starting "spectest:" is a harness failure, not the defect.
func Verdict(reports []types.Report, log string) (exit int, message string) {
	var lines []string
	fail := func(format string, args ...any) {
		exit = 1
		lines = append(lines, fmt.Sprintf(format, args...))
	}
	blocks := raceBlocks(log)
	matched := make([]bool, len(blocks))
	knownDefectSeen := false
	ran := false
	for _, report := range reports {
		for _, spec := range report.SpecReports {
			if !slices.Contains(spec.Labels(), "known-defect") {
				continue
			}
			knownDefectSeen = true
			name := spec.FullText()
			if functions, declared := dataRaceFunctions(spec); declared {
				ran = true
				found := false
				for i, block := range blocks {
					if blockNamesEvery(block, functions) {
						matched[i] = true
						found = true
					}
				}
				if found {
					lines = append(lines, name+": data race still present")
				} else {
					fail("%s: passes now; remove its known-defect label", name)
				}
				continue
			}
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
	for i := range blocks {
		if matched[i] {
			continue
		}
		fail("log: WARNING: DATA RACE")
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

func dataRaceFunctions(spec types.SpecReport) ([]string, bool) {
	for _, entry := range spec.ReportEntries {
		if entry.Name != "data-race" {
			continue
		}
		switch value := entry.Value.GetRawValue().(type) {
		case []string:
			if len(value) == 0 {
				return nil, false
			}
			return value, true
		case []any:
			functions := make([]string, 0, len(value))
			for _, item := range value {
				function, isString := item.(string)
				if !isString {
					return nil, false
				}
				functions = append(functions, function)
			}
			if len(functions) == 0 {
				return nil, false
			}
			return functions, true
		}
	}
	return nil, false
}

func raceBlocks(log string) []string {
	var blocks []string
	var block []string
	inBlock := false
	for _, line := range strings.Split(log, "\n") {
		if strings.HasPrefix(line, "WARNING: DATA RACE") {
			inBlock = true
			block = nil
			continue
		}
		if !inBlock {
			continue
		}
		if isDelimiter(line) {
			blocks = append(blocks, strings.Join(block, "\n"))
			block = nil
			inBlock = false
			continue
		}
		block = append(block, line)
	}
	if inBlock {
		blocks = append(blocks, strings.Join(block, "\n"))
	}
	return blocks
}

func isDelimiter(line string) bool {
	return line != "" && strings.Trim(line, "=") == ""
}

func blockNamesEvery(block string, functions []string) bool {
	for _, function := range functions {
		if !strings.Contains(block, function+"(") {
			return false
		}
	}
	return true
}
