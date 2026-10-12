package specrun

import (
	"slices"
	"testing"
)

func TestFocusFilterPrecedence(t *testing.T) {
	focusCases := []struct {
		flagPassed bool
		given      []string
		env        string
		want       []string
	}{
		{true, []string{"a"}, "b", []string{"a"}},
		{true, []string{"a", "b"}, "c", []string{"a", "b"}},
		{true, []string{""}, "y", []string{""}},
		{false, []string{"preset"}, "y", []string{"y"}},
		{false, nil, "b", []string{"b"}},
		{false, nil, "", nil},
	}
	for _, c := range focusCases {
		got := FocusFilter(c.flagPassed, c.given, c.env)
		if c.want == nil {
			if got != nil {
				t.Errorf("FocusFilter(flagPassed=%v, given=%v, env=%q) = %v, want nil", c.flagPassed, c.given, c.env, got)
			}
			continue
		}
		if !slices.Equal(got, c.want) {
			t.Errorf("FocusFilter(flagPassed=%v, given=%v, env=%q) = %v, want %v", c.flagPassed, c.given, c.env, got, c.want)
		}
	}
}
