package e2e

import (
	"testing"
)

func init() {
	testing.Init()
}
func TestE2E(t *testing.T) {
	RunE2ETests(t)
}
