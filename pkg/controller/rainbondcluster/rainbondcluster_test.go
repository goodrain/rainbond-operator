package rainbondcluster

import (
	"testing"

	"github.com/bmizerany/assert"
	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestGenerateDockerConfig(t *testing.T) {
	want := "eyJhdXRocyI6eyJnb29kcmFpbi5tZSI6eyJhdXRoIjoiWVdSdGFXNDZaMjl2WkhKaGFXND0iLCJwYXNzd29yZCI6Imdvb2RyYWluIiwidXNlcm5hbWUiOiJhZG1pbiJ9fX0="

	cluster := &rainbondv1alpha1.RainbondCluster{
		Spec: rainbondv1alpha1.RainbondClusterSpec{
			ImageHub: &rainbondv1alpha1.ImageHub{
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

	assert.Equal(t, want, string(got))
}

func TestCreateImagePullSecret(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = rainbondv1alpha1.AddToScheme(scheme)
	cli := fake.NewFakeClientWithScheme(scheme)

	cluster := &rainbondv1alpha1.RainbondCluster{
		Spec: rainbondv1alpha1.RainbondClusterSpec{
			ImageHub: &rainbondv1alpha1.ImageHub{
				Domain:   "goodrain.me",
				Username: "admin",
				Password: "e1f3872f",
			},
		},
	}
	mgr := rainbondClusteMgr{
		cluster: cluster,
		client:  cli,
	}

	err := mgr.createImagePullSecret()
	if err != nil {
		t.Errorf("create image pull secret: %v", err)
		t.FailNow()
	}
}
