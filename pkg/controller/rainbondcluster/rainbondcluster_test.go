package rainbondcluster

import (
	"gotest.tools/assert"
	"testing"

	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

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
		scheme:  scheme,
	}

	err := mgr.createImagePullSecret()
	if err != nil {
		t.Errorf("create image pull secret: %v", err)
		t.FailNow()
	}
}

func TestGenerateConditions(t *testing.T) {
	cluster := &rainbondv1alpha1.RainbondCluster{
		Spec: rainbondv1alpha1.RainbondClusterSpec{
			RegionDatabase:&rainbondv1alpha1.Database{
				Host:     "127.0.0.1",
				Port:     3306,
				Username: "foo",
				Password: "bar",
				Name:     "foobar",
			},
		},
		Status: &rainbondv1alpha1.RainbondClusterStatus{},
	}
	mgr := rainbondClusteMgr{
		cluster: cluster,
	}
	mgr.generateConditions()

	_, condition := cluster.Status.GetCondition(rainbondv1alpha1.RainbondClusterConditionTypeDatabaseRegion)
	assert.Equal(t, condition.Status, corev1.ConditionFalse)
}
