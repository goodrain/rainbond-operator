package rbdcomponent

import (
	"os"
	"strings"
	"testing"

	"github.com/operator-framework/operator-sdk/pkg/log/zap"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

func TestExtractInstallationPackage(t *testing.T) {
	t.Run("rainbond package not exist", func(t *testing.T) {
		pkgPath := "foobar.tgz"
		dst := "/tmp"

		if err := extractInstallationPackage(pkgPath, dst); err != nil {
			if strings.Contains(err.Error(), "no such file or directory") {
				return
			}
			t.Error(err)
		}
	})

	t.Run("ok", func(t *testing.T) {
		if err := os.RemoveAll("/tmp/rainbond-pkg-V5.2-dev"); err != nil {
			t.Error(err)
			return
		}

		pkgPath := "/Users/abewang/Goodrain/rainbond-pkg-V5.2-dev.tgz"
		dst := "/tmp"

		if err := extractInstallationPackage(pkgPath, dst); err != nil {
			t.Error(err)
		}
	})
}

func TestLoadRainbondImages(t *testing.T) {
	loadRainbondImages("/tmp/rainbond-pkg-V5.2-dev")
}

func TestPushImages(t *testing.T) {
	logf.SetLogger(zap.Logger())

	if err := pushRainbondImages("/tmp/rainbond-pkg-V5.2-dev"); err != nil {
		t.Error(err)
	}
}
