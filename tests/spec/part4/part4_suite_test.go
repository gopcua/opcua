package part4

import (
	"flag"
	"fmt"
	"os"
	"testing"

	"github.com/gopcua/opcua/tests/spec/internal/specrun"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func resolveFilters(visit func(func(*flag.Flag)), focusFlag []string, getenv func(string) string) (focus []string, focusSource string) {
	var focusPassed bool
	visit(func(passed *flag.Flag) {
		if passed.Name == "ginkgo.focus" {
			focusPassed = true
		}
	})
	focusEnv := getenv("SPECTEST_FOCUS")
	focus = specrun.FocusFilter(focusPassed, focusFlag, focusEnv)
	focusSource = "default"
	if focusPassed {
		focusSource = "flag"
	} else if focusEnv != "" {
		focusSource = "SPECTEST_FOCUS"
	}
	return focus, focusSource
}

func TestPart4(t *testing.T) {
	suiteConfig, reporterConfig := GinkgoConfiguration()
	focus, focusSource := resolveFilters(flag.Visit, suiteConfig.FocusStrings, os.Getenv)
	fmt.Printf("part4: focus %v from %s\n", focus, focusSource)
	suiteConfig.FocusStrings = focus
	suiteConfig.FailOnEmpty = true
	RegisterFailHandler(Fail)
	RunSpecs(t, "OPC UA Part 4", suiteConfig, reporterConfig)
}

// TestMain runs the package's tests only when the spec suite runs, so
// a plain `go test ./...` skips the package visibly instead of paying
// for its specs.
func TestMain(m *testing.M) {
	specrun.Main(m)
}
