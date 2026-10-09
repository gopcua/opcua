package part4

import (
	"flag"
	"slices"
	"testing"
)

func TestResolveFilters(t *testing.T) {
	tests := []struct {
		passedFlags     []string
		labelFlag       string
		focusFlag       []string
		labelEnv        string
		focusEnv        string
		wantLabel       string
		wantFocus       []string
		wantLabelSource string
		wantFocusSource string
	}{
		{
			passedFlags:     []string{"ginkgo.label-filter", "ginkgo.focus"},
			labelFlag:       "x",
			focusFlag:       []string{"a", "b"},
			labelEnv:        "y",
			focusEnv:        "c",
			wantLabel:       "x",
			wantFocus:       []string{"a", "b"},
			wantLabelSource: "flag",
			wantFocusSource: "flag",
		},
		{
			labelFlag:       "preset",
			focusFlag:       []string{"preset"},
			labelEnv:        "y",
			focusEnv:        "f",
			wantLabel:       "y",
			wantFocus:       []string{"f"},
			wantLabelSource: "SPECTEST_LABEL_FILTER",
			wantFocusSource: "SPECTEST_FOCUS",
		},
		{
			labelFlag:       "preset",
			focusFlag:       []string{"preset"},
			wantLabel:       "!known-defect",
			wantFocus:       nil,
			wantLabelSource: "default",
			wantFocusSource: "default",
		},
		{
			passedFlags:     []string{"ginkgo.focus"},
			labelFlag:       "preset",
			focusFlag:       []string{"focusflag"},
			labelEnv:        "labelenv",
			wantLabel:       "labelenv",
			wantFocus:       []string{"focusflag"},
			wantLabelSource: "SPECTEST_LABEL_FILTER",
			wantFocusSource: "flag",
		},
		{
			passedFlags:     []string{"ginkgo.label-filter"},
			labelFlag:       "labelflag",
			focusFlag:       []string{"preset"},
			focusEnv:        "focusenv",
			wantLabel:       "labelflag",
			wantFocus:       []string{"focusenv"},
			wantLabelSource: "flag",
			wantFocusSource: "SPECTEST_FOCUS",
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
		labelFilter, focus, labelSource, focusSource := resolveFilters(visit, test.labelFlag, test.focusFlag, getenv)
		if labelFilter != test.wantLabel {
			t.Errorf("resolveFilters(passedFlags=%v, labelFlag=%q, focusFlag=%v, labelEnv=%q, focusEnv=%q) label filter = %q, want %q", test.passedFlags, test.labelFlag, test.focusFlag, test.labelEnv, test.focusEnv, labelFilter, test.wantLabel)
		}
		if !slices.Equal(focus, test.wantFocus) {
			t.Errorf("resolveFilters(passedFlags=%v, labelFlag=%q, focusFlag=%v, labelEnv=%q, focusEnv=%q) focus = %v, want %v", test.passedFlags, test.labelFlag, test.focusFlag, test.labelEnv, test.focusEnv, focus, test.wantFocus)
		}
		if labelSource != test.wantLabelSource {
			t.Errorf("resolveFilters(passedFlags=%v, labelFlag=%q, focusFlag=%v, labelEnv=%q, focusEnv=%q) label source = %q, want %q", test.passedFlags, test.labelFlag, test.focusFlag, test.labelEnv, test.focusEnv, labelSource, test.wantLabelSource)
		}
		if focusSource != test.wantFocusSource {
			t.Errorf("resolveFilters(passedFlags=%v, labelFlag=%q, focusFlag=%v, labelEnv=%q, focusEnv=%q) focus source = %q, want %q", test.passedFlags, test.labelFlag, test.focusFlag, test.labelEnv, test.focusEnv, focusSource, test.wantFocusSource)
		}
	}
}
