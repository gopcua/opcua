package spectest

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestSpectest(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "spectest")
}
