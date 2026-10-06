package faults

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestFaults(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "faults")
}
