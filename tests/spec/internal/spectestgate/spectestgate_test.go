package spectestgate_test

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
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
	cmd := exec.Command("go", "test", "-count=1", "-json", "./tests/spec/internal/matrix/")
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "SPECTEST=1")
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("go test -json ./tests/spec/internal/matrix/ with SPECTEST=1 failed: %v", err)
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
		t.Fatalf("with SPECTEST=1, no test of the matrix package ran, want the suite gate open")
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

// TestInternalCitesNoClause asserts no non-test code under
// tests/spec/internal/ cites a clause of the standard: internal/ makes
// the suite happen and records what happened, while the clause files
// under tests/spec/part4/ say what must hold, so a clause label or an
// import of a part package in internal/ would put Part 4 content back
// into the engine. Comments may cite the standard.
func TestInternalCitesNoClause(t *testing.T) {
	modRoot, err := exec.Command("go", "list", "-m", "-f", "{{.Dir}}").Output()
	if err != nil {
		t.Fatalf("go list -m failed: %v", err)
	}
	internal := filepath.Join(strings.TrimSpace(string(modRoot)), "tests", "spec", "internal")

	clause := regexp.MustCompile(`^"P[0-9]+-`)
	fset := token.NewFileSet()
	parsed := 0
	err = filepath.WalkDir(internal, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return err
		}
		parsed++
		for _, imported := range file.Imports {
			if strings.Contains(strings.Trim(imported.Path.Value, `"`), "/tests/spec/part") {
				t.Errorf("%s imports %s, a part package: what a clause demands belongs in its clause file, not in internal/", path, imported.Path.Value)
			}
		}
		ast.Inspect(file, func(node ast.Node) bool {
			literal, isString := node.(*ast.BasicLit)
			if !isString || literal.Kind != token.STRING {
				return true
			}
			if clause.MatchString(literal.Value) {
				t.Errorf("%s holds the clause label %s: a clause label belongs in the clause files under tests/spec/part, not in internal/", path, literal.Value)
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatalf("walking %s failed: %v", internal, err)
	}
	if parsed == 0 {
		t.Fatalf("parsed no non-test .go file under %s, so the check matched nothing", internal)
	}
}
