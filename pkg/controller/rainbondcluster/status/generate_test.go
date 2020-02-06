package status

import (
	"fmt"
	"testing"

	rainbondv1alpha1 "github.com/GLYASAI/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestGenerateRainbondClusterPackageExtractedCondition(t *testing.T) {
	tests := []struct {
		name            string
		rainbondcluster *rainbondv1alpha1.RainbondCluster
		rainbondPackage *rainbondv1alpha1.RainbondPackage
		err             error
		want            rainbondv1alpha1.RainbondClusterCondition
	}{
		{
			name: "without installation package",
			rainbondcluster: &rainbondv1alpha1.RainbondCluster{
				Spec: rainbondv1alpha1.RainbondClusterSpec{
					InstallMode: rainbondv1alpha1.InstallationModeWithoutPackage,
				},
			},
			want: rainbondv1alpha1.RainbondClusterCondition{
				Type:   rainbondv1alpha1.PackageExtracted,
				Status: rainbondv1alpha1.ConditionTrue,
				Reason: string(rainbondv1alpha1.InstallationModeWithoutPackage),
			},
		},
		{
			name: "already extracted",
			rainbondcluster: &rainbondv1alpha1.RainbondCluster{
				Spec: rainbondv1alpha1.RainbondClusterSpec{},
				Status: &rainbondv1alpha1.RainbondClusterStatus{
					Conditions: []rainbondv1alpha1.RainbondClusterCondition{
						{
							Type:   rainbondv1alpha1.PackageExtracted,
							Status: rainbondv1alpha1.ConditionTrue,
						},
					},
				},
			},
			want: rainbondv1alpha1.RainbondClusterCondition{
				Type:   rainbondv1alpha1.PackageExtracted,
				Status: rainbondv1alpha1.ConditionTrue,
			},
		},
		{
			name:            "rainbondpackage waiting",
			rainbondcluster: &rainbondv1alpha1.RainbondCluster{},
			rainbondPackage: &rainbondv1alpha1.RainbondPackage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rainbondpackage",
					Namespace: "rbd-system",
				},
				Status: &rainbondv1alpha1.RainbondPackageStatus{
					Phase: rainbondv1alpha1.RainbondPackageWaiting,
				},
			},
			want: rainbondv1alpha1.RainbondClusterCondition{
				Type:   rainbondv1alpha1.PackageExtracted,
				Status: rainbondv1alpha1.ConditionFalse,
				Reason: fmt.Sprintf("RainbondPackage%s", string(rainbondv1alpha1.RainbondPackageWaiting)),
			},
		},
		{
			name:            "rainbondpackage extracting",
			rainbondcluster: &rainbondv1alpha1.RainbondCluster{},
			rainbondPackage: &rainbondv1alpha1.RainbondPackage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rainbondpackage",
					Namespace: "rbd-system",
				},
				Status: &rainbondv1alpha1.RainbondPackageStatus{
					Phase: rainbondv1alpha1.RainbondPackageExtracting,
				},
			},
			want: rainbondv1alpha1.RainbondClusterCondition{
				Type:   rainbondv1alpha1.PackageExtracted,
				Status: rainbondv1alpha1.ConditionFalse,
				Reason: fmt.Sprintf("RainbondPackage%s", string(rainbondv1alpha1.RainbondPackageExtracting)),
			},
		},
		{
			name:            "rainbondpackage loading",
			rainbondcluster: &rainbondv1alpha1.RainbondCluster{},
			rainbondPackage: &rainbondv1alpha1.RainbondPackage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rainbondpackage",
					Namespace: "rbd-system",
				},
				Status: &rainbondv1alpha1.RainbondPackageStatus{
					Phase: rainbondv1alpha1.RainbondPackageLoading,
				},
			},
			want: rainbondv1alpha1.RainbondClusterCondition{
				Type:   rainbondv1alpha1.PackageExtracted,
				Status: rainbondv1alpha1.ConditionTrue,
			},
		},
	}

	scheme := runtime.NewScheme()
	_ = rainbondv1alpha1.AddToScheme(scheme)
	for i := range tests {
		tc := tests[i]
		t.Run(tc.name, func(t *testing.T) {
			s := Status{
				client: fake.NewFakeClientWithScheme(scheme),
			}
			if tc.rainbondPackage != nil {
				s.client = fake.NewFakeClientWithScheme(scheme, tc.rainbondPackage)
			}

			got := s.GenerateRainbondClusterPackageExtractedCondition(tc.rainbondcluster)
			assert.Equal(t, tc.want.Type, got.Type)
			assert.Equal(t, tc.want.Status, got.Status)
			assert.Equal(t, tc.want.Reason, got.Reason)
		})
	}
}
