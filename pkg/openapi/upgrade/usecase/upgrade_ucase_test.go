package usecase_test

import (
	"testing"

	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	"github.com/goodrain/rainbond-operator/pkg/generated/clientset/versioned/fake"
	"github.com/goodrain/rainbond-operator/pkg/openapi/upgrade/usecase"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestVersions(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = rainbondv1alpha1.AddToScheme(scheme)

	namespace := "rbd-system"
	clusterName := "rainbondcluster"
	cluster := &rainbondv1alpha1.RainbondCluster{
		ObjectMeta: v1.ObjectMeta{
			Namespace: namespace,
			Name:      clusterName,
		},
		Spec: rainbondv1alpha1.RainbondClusterSpec{
			InstallVersion: "v5.2.0",
		},
	}
	invalidCluster := &rainbondv1alpha1.RainbondCluster{
		ObjectMeta: v1.ObjectMeta{
			Namespace: namespace,
			Name:      "invalidcluster",
		},
		Spec: rainbondv1alpha1.RainbondClusterSpec{
			InstallVersion: "v5.2.0-dev",
		},
	}
	rainbondClient := fake.NewSimpleClientset(cluster, invalidCluster)

	tests := []struct {
		name, namespace, clusterName string
		wantVersions                 []string
	}{
		{
			name:        "ClusterNotFound",
			namespace:   namespace,
			clusterName: "foobar",
		},
		{
			name:         "OK",
			namespace:    namespace,
			clusterName:  clusterName,
			wantVersions: []string{"v5.2.1", "v5.2.2", "v5.2.999"},
		},
		{
			name:        "InvalidCluster",
			namespace:   namespace,
			clusterName: "invalidcluster",
		},
	}

	for i := range tests {
		tc := tests[i]

		t.Run(tc.name, func(t *testing.T) {
			upgradeUcase := usecase.NewUpgradeUsecase(rainbondClient, "./testdata/version", tc.namespace, tc.clusterName)
			versions, err := upgradeUcase.Versions()
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				t.FailNow()
			}

			assert.ElementsMatch(t, tc.wantVersions, versions)
		})
	}
}
