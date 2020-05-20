package rainbondpackage

import (
	"context"
	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	"testing"
)

var pkgHandle *pkg

func init() {
	pkgHandle, _ = newpkg(context.Background(), nil, &rainbondv1alpha1.RainbondPackage{
		Spec: rainbondv1alpha1.RainbondPackageSpec{
			PkgPath: "/tmp/rainbond.tar",
		},
		Status: initPackageStatus(rainbondv1alpha1.Waiting),
	}, log)
	pkgHandle.setCluster(&rainbondv1alpha1.RainbondCluster{
		Spec: rainbondv1alpha1.RainbondClusterSpec{
			ConfigCompleted: true,
			InstallPackageConfig: rainbondv1alpha1.InstallPackageConfig{
				URL: "https://rainbond-pkg.oss-cn-shanghai.aliyuncs.com/offline/5.2/rainbond.images.2020-02-07-5.2-dev.tgz",
				MD5: "f8989cabef3cff564d63a9ee445c24e26ff44af96a647b3f4c980e0e780b8032",
			},
		},
	})
}

func TestParseImageName(t *testing.T) {
	tests := []struct {
		name string
		str  string
		want string
	}{
		{
			name: "ok",
			str:  "Loaded image: rainbond/rbd-api:V5.2-dev",
			want: "rainbond/rbd-api:V5.2-dev",
		},
	}

	for idx := range tests {
		tc := tests[idx]
		t.Run(tc.name, func(t *testing.T) {
			got := parseImageName(tc.str)
			if got == "" {
				t.Error("parse image name failure")
				return
			}
			if tc.want != got {
				t.Errorf("want %s, but got %s", tc.want, got)
			}
		})
	}
}
