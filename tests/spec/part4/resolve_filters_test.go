package part4

import (
	"flag"
	"slices"
	"testing"
)

func TestResolveFilters(t *testing.T) {
	tests := []struct {
		passedFlags []string
		labelFlag   string
		focusFlag   []string
		labelEnv    string
		focusEnv    string
		wantLabel   string
		wantFocus   []string
		wantSources string
	}{
		{
			passedFlags: []string{"ginkgo.label-filter", "ginkgo.focus"},
			labelFlag:   "x",
			focusFlag:   []string{"a", "b"},
			labelEnv:    "y",
			focusEnv:    "c",
			wantLabel:   "x",
			wantFocus:   []string{"a", "b"},
			wantSources: "flag; flag",
		},
		{
			labelFlag:   "preset",
			focusFlag:   []string{"preset"},
			labelEnv:    "y",
			focusEnv:    "f",
			wantLabel:   "y",
			wantFocus:   []string{"f"},
			wantSources: "SPECTEST_LABEL_FILTER; SPECTEST_FOCUS",
		},
		{
			labelFlag:   "preset",
			focusFlag:   []string{"preset"},
			wantLabel:   "!known-defect",
			wantFocus:   nil,
			wantSources: "default; default",
		},
	}
	for _, test := range tests {
		visit := func(fn func(*flag.Flag)) {
			for _, name := range test.passedFlags {
				fn(&flag.Flag{Name: name})
			}
		}
		getenv := func(name string) string {
			switch name {
			case "SPECTEST_LABEL_FILTER":
				return test.labelEnv
			case "SPECTEST_FOCUS":
				return test.focusEnv
			}
			return ""
		}
		labelFilter, focus, sources := resolveFilters(visit, test.labelFlag, test.focusFlag, getenv)
		if labelFilter != test.wantLabel {
			t.Errorf("resolveFilters(passedFlags=%v, labelFlag=%q, focusFlag=%v, labelEnv=%q, focusEnv=%q) label filter = %q, want %q", test.passedFlags, test.labelFlag, test.focusFlag, test.labelEnv, test.focusEnv, labelFilter, test.wantLabel)
		}
		if !slices.Equal(focus, test.wantFocus) {
			t.Errorf("resolveFilters(passedFlags=%v, labelFlag=%q, focusFlag=%v, labelEnv=%q, focusEnv=%q) focus = %v, want %v", test.passedFlags, test.labelFlag, test.focusFlag, test.labelEnv, test.focusEnv, focus, test.wantFocus)
		}
		if sources != test.wantSources {
			t.Errorf("resolveFilters(passedFlags=%v, labelFlag=%q, focusFlag=%v, labelEnv=%q, focusEnv=%q) sources = %q, want %q", test.passedFlags, test.labelFlag, test.focusFlag, test.labelEnv, test.focusEnv, sources, test.wantSources)
		}
	}
}
