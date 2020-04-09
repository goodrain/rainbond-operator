package rainbondcluster

import (
	"github.com/bmizerany/assert"
	"github.com/goodrain/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	"testing"
)

func TestGenerateDockerConfig(t *testing.T) {
	want := "eyJhdXRocyI6eyJnb29kcmFpbi5tZSI6eyJhdXRoIjoiWVdSdGFXNDZaMjl2WkhKaGFXND0iLCJwYXNzd29yZCI6Imdvb2RyYWluIiwidXNlcm5hbWUiOiJhZG1pbiJ9fX0="

	cluster := &v1alpha1.RainbondCluster{
		Spec: v1alpha1.RainbondClusterSpec{
			ImageHub: &v1alpha1.ImageHub{
				Domain:   "goodrain.me",
				Username: "admin",
				Password: "goodrain",
			},
		},
	}
	mgr := rainbondClusteMgr{
		cluster: cluster,
	}
	got := mgr.generateDockerConfig()

	assert.Equal(t, want, got)
}
