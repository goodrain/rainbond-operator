package e2e

import (
	"github.com/goodrain/rainbond-operator/test/e2e/framework"
	"testing"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
)

// RunE2ETests checks configuration parameters (specified through flags) and then runs
// E2E tests using the Ginkgo runner.
func RunE2ETests(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)

	ginkgo.RunSpecs(t, "appstore e2e suite")
}

var _ = ginkgo.SynchronizedAfterSuite(func() {
	framework.Logf("Running AfterSuite actions")
}, func() {})
