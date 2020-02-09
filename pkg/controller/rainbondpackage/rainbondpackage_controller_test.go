package rainbondpackage

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"testing"

	"github.com/docker/docker/pkg/jsonmessage"

	"github.com/docker/docker/client"
	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/pkg/apis/rainbond/v1alpha1"
)

var pkgHandle *pkg

func init() {
	pkgHandle, _ = newpkg(context.Background(), nil, &rainbondv1alpha1.RainbondPackage{
		Spec: rainbondv1alpha1.RainbondPackageSpec{
			PkgPath: "/tmp/rainbond.tar",
		},
		Status: initPackageStatus(),
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
func TestImageLoad(t *testing.T) {
	cli, _ := client.NewClientWithOpts(client.FromEnv)
	cli.NegotiateAPIVersion(context.TODO())

	file, err := os.Open("/tmp/rainbond/rainbond/api.tgz")
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	res, err := cli.ImageLoad(ctx, file, true)
	if err != nil {
		t.Errorf("path: %s; failed to load image: %v", "/tmp/rainbond-operator.tgz", err)
		return
	}
	if res.Body != nil {
		defer res.Body.Close()
		dec := json.NewDecoder(res.Body)
		for {
			var jm jsonmessage.JSONMessage
			if err := dec.Decode(&jm); err != nil {
				if err == io.EOF {
					break
				}
				t.Fatal(err)
				return
			}
			if jm.Error != nil {
				t.Fatal(jm.Error)
				return
			}
			t.Logf("%s\n", jm.Stream)
		}
	}
}

func TestParseImageName(t *testing.T) {
	tests := []struct {
		name string
		str  string
		want string
	}{
		{
			name: "foo",
			str:  "{\"stream\":\"Loaded image: rainbond/rbd-api:V5.2-dev\\n\"}",
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

func TestDownloadPackage(t *testing.T) {
	if err := pkgHandle.donwnloadPackage(); err != nil {
		t.Fatal(err)
	}
}

func TestUnpack(t *testing.T) {
	if err := pkgHandle.untartar(); err != nil {
		t.Fatal(err)
	}
}

func TestImagesLoadAndPush(t *testing.T) {
	if err := pkgHandle.imagesLoadAndPush(); err != nil {
		t.Fatal(err)
	}
}
