package main

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestStartupTriggerReconcilesClustersAndComponents(t *testing.T) {
	t.Setenv("OPERATOR_VERSION", "test-version")

	scheme := runtime.NewScheme()
	if err := rainbondv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("add Rainbond scheme: %v", err)
	}
	cluster := &rainbondv1alpha1.RainbondCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "rainbondcluster",
			Namespace:   "rbd-system",
			Annotations: map[string]string{"keep": "cluster"},
		},
	}
	api := &rainbondv1alpha1.RbdComponent{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "rbd-api",
			Namespace:   "rbd-system",
			Annotations: map[string]string{"keep": "api"},
		},
	}
	appUI := &rainbondv1alpha1.RbdComponent{
		ObjectMeta: metav1.ObjectMeta{Name: "rbd-app-ui", Namespace: "rbd-system"},
	}
	k8sClient := fake.NewFakeClientWithScheme(scheme, cluster, api, appUI)
	trigger := &rainbondClusterReconcileTrigger{Client: k8sClient, Log: logr.Discard()}

	if err := trigger.Start(context.Background()); err != nil {
		t.Fatalf("run startup trigger: %v", err)
	}

	assertOperatorVersionAnnotation(t, k8sClient, client.ObjectKeyFromObject(cluster), &rainbondv1alpha1.RainbondCluster{}, "cluster")
	assertOperatorVersionAnnotation(t, k8sClient, client.ObjectKeyFromObject(api), &rainbondv1alpha1.RbdComponent{}, "api")
	assertOperatorVersionAnnotation(t, k8sClient, client.ObjectKeyFromObject(appUI), &rainbondv1alpha1.RbdComponent{}, "")
}

func TestStartupTriggerSkipsObjectsAlreadyReconciledByVersion(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := rainbondv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("add Rainbond scheme: %v", err)
	}
	component := &rainbondv1alpha1.RbdComponent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rbd-api",
			Namespace: "rbd-system",
			Annotations: map[string]string{
				operatorVersionAnnotation:    "test-version",
				operatorUpdateTimeAnnotation: "preserved-time",
			},
		},
	}
	k8sClient := fake.NewFakeClientWithScheme(scheme, component)
	trigger := &rainbondClusterReconcileTrigger{Client: k8sClient, Log: logr.Discard()}

	trigger.triggerObjectReconcile(context.Background(), component, "test-version", "RbdComponent")

	updated := &rainbondv1alpha1.RbdComponent{}
	if err := k8sClient.Get(context.Background(), client.ObjectKeyFromObject(component), updated); err != nil {
		t.Fatalf("get component: %v", err)
	}
	if got := updated.Annotations[operatorUpdateTimeAnnotation]; got != "preserved-time" {
		t.Fatalf("expected update time to remain unchanged, got %q", got)
	}
}

func assertOperatorVersionAnnotation(t *testing.T, k8sClient client.Client, key client.ObjectKey, object client.Object, preserved string) {
	t.Helper()

	if err := k8sClient.Get(context.Background(), key, object); err != nil {
		t.Fatalf("get %s: %v", key, err)
	}
	annotations := object.GetAnnotations()
	if got := annotations[operatorVersionAnnotation]; got != "test-version" {
		t.Fatalf("expected operator version annotation on %s, got %q", key, got)
	}
	if annotations[operatorUpdateTimeAnnotation] == "" {
		t.Fatalf("expected operator update time annotation on %s", key)
	}
	if preserved != "" && annotations["keep"] != preserved {
		t.Fatalf("expected existing annotation on %s to be preserved", key)
	}
}
