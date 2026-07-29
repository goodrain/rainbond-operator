package componentmgr

import (
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestRbdDBStatefulSetCanUpdatePodTemplate(t *testing.T) {
	t.Parallel()

	statefulSet := &appsv1.StatefulSet{ObjectMeta: metav1.ObjectMeta{Name: "rbd-db"}}
	if !objectCanUpdate(statefulSet) {
		t.Fatal("expected rbd-db StatefulSet pod template to be updateable")
	}
}

func TestRbdDBNonStatefulResourcesRemainProtectedFromUpdate(t *testing.T) {
	t.Parallel()

	service := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "rbd-db"}}
	if objectCanUpdate(service) {
		t.Fatal("expected non-StatefulSet rbd-db resources to remain protected from update")
	}
}

func TestUpdateRuntimeObjectPreservesStatefulSetImmutableFields(t *testing.T) {
	t.Parallel()

	old := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "rbd-db",
			ResourceVersion: "42",
		},
		Spec: appsv1.StatefulSetSpec{
			ServiceName:         "existing-service",
			PodManagementPolicy: appsv1.ParallelPodManagement,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"name": "existing"},
			},
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
				{ObjectMeta: metav1.ObjectMeta{Name: "existing-data"}},
			},
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{PriorityClassName: "old-priority"},
			},
		},
	}
	newStatefulSet := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{Name: "rbd-db"},
		Spec: appsv1.StatefulSetSpec{
			ServiceName:         "new-service",
			PodManagementPolicy: appsv1.OrderedReadyPodManagement,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"name": "new"},
			},
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
				{ObjectMeta: metav1.ObjectMeta{Name: "new-data"}},
			},
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{PriorityClassName: "system-cluster-critical"},
			},
		},
	}

	mgr := &RbdcomponentMgr{}
	got := mgr.updateRuntimeObject(old, newStatefulSet).(*appsv1.StatefulSet)
	if got.ResourceVersion != old.ResourceVersion {
		t.Fatalf("expected resource version %q, got %q", old.ResourceVersion, got.ResourceVersion)
	}
	if got.Spec.ServiceName != old.Spec.ServiceName {
		t.Fatalf("expected serviceName %q, got %q", old.Spec.ServiceName, got.Spec.ServiceName)
	}
	if got.Spec.PodManagementPolicy != old.Spec.PodManagementPolicy {
		t.Fatalf("expected podManagementPolicy %q, got %q", old.Spec.PodManagementPolicy, got.Spec.PodManagementPolicy)
	}
	if got.Spec.Selector.MatchLabels["name"] != "existing" {
		t.Fatalf("expected existing selector, got %v", got.Spec.Selector.MatchLabels)
	}
	if got.Spec.VolumeClaimTemplates[0].Name != "existing-data" {
		t.Fatalf("expected existing volume claim template, got %q", got.Spec.VolumeClaimTemplates[0].Name)
	}
	if got.Spec.Template.Spec.PriorityClassName != "system-cluster-critical" {
		t.Fatalf("expected updated pod template, got priority class %q", got.Spec.Template.Spec.PriorityClassName)
	}
}
