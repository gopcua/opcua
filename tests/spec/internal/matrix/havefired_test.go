package matrix

import (
	"strings"
	"testing"

	"github.com/onsi/gomega/types"
)

func assertMatcher(t *testing.T, matcher types.GomegaMatcher, observed Observed, wantPass bool, wantSubstring string) {
	t.Helper()
	success, err := matcher.Match(observed)
	if err != nil {
		t.Fatalf("%T.Match returned an error: %v", matcher, err)
	}
	if success != wantPass {
		t.Fatalf("%T matched %v on %v, want %v", matcher, success, observed, wantPass)
	}
	if wantPass {
		return
	}
	message := matcher.FailureMessage(observed)
	if !strings.Contains(message, wantSubstring) {
		t.Fatalf("%T failed with %q, want it to name %q", matcher, message, wantSubstring)
	}
}

func TestHaveFired(t *testing.T) {
	assertMatcher(t, HaveFired(), Observed{Fired: true}, true, "")
	assertMatcher(t, HaveFired(), Observed{Fired: false}, false, "fired")
}
