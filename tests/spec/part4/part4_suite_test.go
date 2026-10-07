package part4

import (
	"flag"
	"fmt"
	"os"
	"testing"

	"github.com/gopcua/opcua/tests/spec/internal/spectest"
	"github.com/gopcua/opcua/tests/spec/internal/suitegate"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func resolveFilters(visit func(func(*flag.Flag)), labelFlag string, focusFlag []string, getenv func(string) string) (labelFilter string, focus []string, labelSource string, focusSource string) {
	var labelPassed, focusPassed bool
	visit(func(passed *flag.Flag) {
		switch passed.Name {
		case "ginkgo.label-filter":
			labelPassed = true
		case "ginkgo.focus":
			focusPassed = true
		}
	})
	labelEnv := getenv("SPECTEST_LABEL_FILTER")
	focusEnv := getenv("SPECTEST_FOCUS")
	labelFilter = spectest.LabelFilter(labelPassed, labelFlag, labelEnv)
	focus = spectest.FocusFilter(focusPassed, focusFlag, focusEnv)
	labelSource = "default"
	if labelPassed {
		labelSource = "flag"
	} else if labelEnv != "" {
		labelSource = "SPECTEST_LABEL_FILTER"
	}
	focusSource = "default"
	if focusPassed {
		focusSource = "flag"
	} else if focusEnv != "" {
		focusSource = "SPECTEST_FOCUS"
	}
	return labelFilter, focus, labelSource, focusSource
}

func TestPart4(t *testing.T) {
	suiteConfig, reporterConfig := GinkgoConfiguration()
	labelFilter, focus, labelSource, focusSource := resolveFilters(flag.Visit, suiteConfig.LabelFilter, suiteConfig.FocusStrings, os.Getenv)
	fmt.Printf("part4: label filter %q from %s; focus %v from %s\n", labelFilter, labelSource, focus, focusSource)
	suiteConfig.LabelFilter = labelFilter
	suiteConfig.FocusStrings = focus
	suiteConfig.FailOnEmpty = true
	RegisterFailHandler(Fail)
	RunSpecs(t, "OPC UA Part 4", suiteConfig, reporterConfig)
}

// TestMain runs the package's tests only when the spec suite runs, so
// a plain `go test ./...` skips the package visibly instead of paying
// for its specs.
func TestMain(m *testing.M) {
	if !suitegate.Enabled() {
		fmt.Println(suitegate.SkipReason())
		os.Exit(0)
	}
	os.Exit(m.Run())
}
