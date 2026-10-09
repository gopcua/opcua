package matrix

import (
	"fmt"

	"github.com/onsi/gomega/types"
)

// HaveFired says the fault the case armed fired.
func HaveFired() types.GomegaMatcher {
	return &haveFired{}
}

type haveFired struct{}

func (m *haveFired) Match(actual any) (bool, error) {
	observed, ok := actual.(Observed)
	if !ok {
		return false, fmt.Errorf("want an Observed, got %T", actual)
	}
	return observed.Fired, nil
}

func (m *haveFired) FailureMessage(actual any) string {
	return "Expected the fault to have fired, but it never did"
}

func (m *haveFired) NegatedFailureMessage(actual any) string {
	return "Expected the fault not to have fired, but it did"
}
