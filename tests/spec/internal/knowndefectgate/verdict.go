package main

import (
	"fmt"
	"slices"
	"strings"

	"github.com/gopcua/opcua/tests/spec/internal/harnessfault"
	"github.com/onsi/ginkgo/v2/types"
)

// Verdict reports whether the known-defect labelled specs in the Ginkgo
// reports still fail as their labels claim, and whether the log shows a
// data race no spec declared. A spec that records a report entry named
// data-race and passed takes its verdict from the log: a
// WARNING: DATA RACE block naming every declared function is the defect
// still present, and no such block means the label must go. Such a spec
// asserts nothing, so its failure leaves the verdict unknown. A failure
// message starting with harnessfault.Prefix is a harness failure, not
// the defect.
// The exit is 1 whenever a rule fails; every other line the gate prints
// is information.
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
			functions, kind := dataRaceFunctions(spec)
			if kind == declarationMalformed {
				fail("%s: data-race entry is not a list of function names", name)
				continue
			}
			if kind == declarationValid && spec.State == types.SpecStatePassed {
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
				fail("%s: passes now; remove its known-defect label", name)
			case types.SpecStateFailed:
				ran = true
				if harness, ok := spectestFailure(spec); ok {
					fail("%s: harness failure, not a defect: %s", name, harness)
					continue
				}
				if kind == declarationValid {
					fail("%s: failed, but it declares a data race and asserts nothing; verdict unknown", name)
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
		fail("log: WARNING: DATA RACE in %s", firstFrame(blocks[i]))
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
	if strings.HasPrefix(spec.Failure.Message, harnessfault.Prefix) {
		return spec.Failure.Message, true
	}
	for _, additional := range spec.AdditionalFailures {
		if strings.HasPrefix(additional.Failure.Message, harnessfault.Prefix) {
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

const (
	declarationAbsent = iota
	declarationValid
	declarationMalformed
)

func dataRaceFunctions(spec types.SpecReport) (functions []string, kind int) {
	for _, entry := range spec.ReportEntries {
		if entry.Name != "data-race" {
			continue
		}
		switch value := entry.Value.GetRawValue().(type) {
		case []string:
			if len(value) == 0 || slices.Contains(value, "") {
				return nil, declarationMalformed
			}
			if functions == nil {
				functions = value
			}
		case []any:
			converted := make([]string, 0, len(value))
			for _, item := range value {
				function, isString := item.(string)
				if !isString || function == "" {
					return nil, declarationMalformed
				}
				converted = append(converted, function)
			}
			if len(converted) == 0 {
				return nil, declarationMalformed
			}
			if functions == nil {
				functions = converted
			}
		default:
			return nil, declarationMalformed
		}
	}
	if functions == nil {
		return nil, declarationAbsent
	}
	return functions, declarationValid
}

func raceBlocks(log string) []string {
	var blocks []string
	var block []string
	inBlock := false
	for _, line := range strings.Split(log, "\n") {
		if strings.HasPrefix(line, "WARNING: DATA RACE") {
			if inBlock {
				blocks = append(blocks, strings.Join(block, "\n"))
			}
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

func firstFrame(block string) string {
	for _, line := range strings.Split(block, "\n") {
		if strings.Contains(line, "/") && strings.Contains(line, "(") {
			return strings.TrimSpace(line)
		}
	}
	return "a block with no function frame"
}
