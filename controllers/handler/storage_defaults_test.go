package handler

import (
	"context"
	"testing"

	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestDefaultResourcesIncludeEphemeralStorageRequest(t *testing.T) {
	t.Parallel()

	resources := setDefaultResources(corev1.ResourceRequirements{})
	if got := resources.Requests[corev1.ResourceEphemeralStorage]; got.String() != "256Mi" {
		t.Fatalf("expected default ephemeral-storage request 256Mi, got %s", got.String())
	}
}

func TestDefaultResourcesPreserveConfiguredEphemeralStorageRequest(t *testing.T) {
	t.Parallel()

	resources := setDefaultResources(corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceEphemeralStorage: resource.MustParse("2Gi"),
		},
	})
	if got := resources.Requests[corev1.ResourceEphemeralStorage]; got.String() != "2Gi" {
		t.Fatalf("expected configured ephemeral-storage request 2Gi, got %s", got.String())
	}
}

func TestComponentPVCStorageRequestDefaults(t *testing.T) {
	t.Setenv("DB_DATA_STORAGE_REQUEST", "")
	t.Setenv("APP_UI_DATA_STORAGE_REQUEST", "")
	t.Setenv("HUB_DATA_STORAGE_REQUEST", "")
	t.Setenv("MINIO_DATA_STORAGE_REQUEST", "")
	t.Setenv("MONITOR_DATA_STORAGE_REQUEST", "")

	component := &rainbondv1alpha1.RbdComponent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-component",
			Namespace: "rbd-system",
		},
	}
	cluster := &rainbondv1alpha1.RainbondCluster{}
	ctx := context.Background()

	tests := []struct {
		name string
		got  int64
		want int64
	}{
		{name: "db", got: NewDB(ctx, nil, component, cluster).(*db).storageRequest, want: 5},
		{name: "app ui", got: NewAppUI(ctx, nil, component, cluster).(*appui).storageRequest, want: 5},
		{name: "hub", got: NewHub(ctx, nil, component, cluster).(*hub).storageRequest, want: 200},
		{name: "minio", got: NewMinIO(ctx, nil, component, cluster).(*minIO).storageRequest, want: 100},
		{name: "monitor", got: NewMonitor(ctx, nil, component, cluster).(*monitor).storageRequest, want: 100},
	}

	for _, tt := range tests {
		if tt.got != tt.want {
			t.Fatalf("%s default storage request = %dGi, want %dGi", tt.name, tt.got, tt.want)
		}
	}
}
