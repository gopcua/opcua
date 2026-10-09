package part4

import (
	"flag"
	"slices"
	"testing"
)

func TestResolveFilters(t *testing.T) {
	tests := []struct {
		passedFlags     []string
		focusFlag       []string
		focusEnv        string
		wantFocus       []string
		wantFocusSource string
	}{
		{
			passedFlags:     []string{"ginkgo.focus"},
			focusFlag:       []string{"a", "b"},
			focusEnv:        "c",
			wantFocus:       []string{"a", "b"},
			wantFocusSource: "flag",
		},
		{
			focusFlag:       []string{"preset"},
			focusEnv:        "f",
			wantFocus:       []string{"f"},
			wantFocusSource: "SPECTEST_FOCUS",
		},
		{
			focusFlag:       []string{"preset"},
			wantFocus:       nil,
			wantFocusSource: "default",
		},
	}
	for _, test := range tests {
		visit := func(fn func(*flag.Flag)) {
			for _, name := range test.passedFlags {
				fn(&flag.Flag{Name: name})
			}
		}
		getenv := func(name string) string {
			if name == "SPECTEST_FOCUS" {
				return test.focusEnv
			}
			return ""
		}
		focus, focusSource := resolveFilters(visit, test.focusFlag, getenv)
		if !slices.Equal(focus, test.wantFocus) {
			t.Errorf("resolveFilters(passedFlags=%v, focusFlag=%v, focusEnv=%q) focus = %v, want %v", test.passedFlags, test.focusFlag, test.focusEnv, focus, test.wantFocus)
		}
		if focusSource != test.wantFocusSource {
			t.Errorf("resolveFilters(passedFlags=%v, focusFlag=%v, focusEnv=%q) focus source = %q, want %q", test.passedFlags, test.focusFlag, test.focusEnv, focusSource, test.wantFocusSource)
		}
	}
}
