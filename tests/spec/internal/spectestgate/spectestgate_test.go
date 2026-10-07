package spectestgate_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// TestSuiteSkipsByDefault asserts the whole tests/spec tree skips unless
// SPECTEST=1 is set: with the variable unset, `go test ./tests/spec/...`
// runs no test at all — every package's TestMain prints the skip reason
// and exits — and with the variable set, the tests run.
func TestSuiteSkipsByDefault(t *testing.T) {
	modRoot, err := exec.Command("go", "list", "-m", "-f", "{{.Dir}}").Output()
	if err != nil {
		t.Fatalf("go list -m failed: %v", err)
	}
	root := strings.TrimSpace(string(modRoot))

	listCmd := exec.Command("go", "list", "-f", "{{if .TestGoFiles}}{{.ImportPath}}{{end}}", "./tests/spec/...")
	listCmd.Dir = root
	list, err := listCmd.Output()
	if err != nil {
		t.Fatalf("go list ./tests/spec/... failed: %v", err)
	}
	packages := strings.Fields(string(list))
	if len(packages) == 0 {
		t.Fatalf("go list named no test package")
	}

	jsonTest := func(env []string) (ran, skippedReason int, err error) {
		cmd := exec.Command("go", "test", "-count=1", "-json", "./tests/spec/...")
		cmd.Dir = root
		cmd.Env = env
		output, err := cmd.Output()
		if err != nil {
			return 0, 0, err
		}
		decoder := json.NewDecoder(strings.NewReader(string(output)))
		for {
			var event struct {
				Package string
				Test    string
				Action  string
				Output  string
			}
			if err := decoder.Decode(&event); err != nil {
				break
			}
			if event.Test != "" {
				if event.Action == "run" {
					ran++
				}
				continue
			}
			if event.Action == "output" && strings.Contains(event.Output, "run the spec suite") {
				skippedReason++
			}
		}
		return ran, skippedReason, nil
	}

	// With SPECTEST unset, no test runs, and every package prints the
	// skip reason.
	ran, printed, err := jsonTest(spectestUnset(os.Environ()))
	if err != nil {
		t.Fatalf("go test -json ./tests/spec/... with SPECTEST unset failed: %v", err)
	}
	if ran != 0 {
		t.Fatalf("with SPECTEST unset, %d test(s) ran, want none: a package's TestMain ran its tests although the suite gate was closed", ran)
	}
	if printed < len(packages) {
		t.Fatalf("with SPECTEST unset, %d of %d test package(s) printed the skip reason, want every package's TestMain to print it", printed, len(packages))
	}

	// With SPECTEST=1, the tests run.
	cmd := exec.Command("go", "test", "-count=1", "-json", "./tests/spec/rules/")
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "SPECTEST=1")
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("go test -json ./tests/spec/rules/ with SPECTEST=1 failed: %v", err)
	}
	ran = 0
	decoder := json.NewDecoder(strings.NewReader(string(output)))
	for {
		var event struct {
			Test   string
			Action string
		}
		if err := decoder.Decode(&event); err != nil {
			break
		}
		if event.Test != "" && event.Action == "run" {
			ran++
		}
	}
	if ran == 0 {
		t.Fatalf("with SPECTEST=1, no test of the rules package ran, want the suite gate open")
	}
}

// spectestUnset strips every SPECTEST* variable from the environment a
// subprocess runs with.
func spectestUnset(environ []string) []string {
	var kept []string
	for _, entry := range environ {
		if strings.HasPrefix(entry, "SPECTEST=") {
			continue
		}
		kept = append(kept, entry)
	}
	return kept
}
