package usecase_test

import (
	"github.com/goodrain/rainbond-operator/pkg/openapi/upgrade/usecase"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestVersions(t *testing.T) {
	upgradeUcase := usecase.NewUpgradeUsecase("./testdata/version")
	versions, err := upgradeUcase.Versions()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		t.FailNow()
	}

	wantVersions := []string{"v5.2.1", "v5.2.2", "v5.2.999"}
	assert.ElementsMatch(t, wantVersions, versions)
}
