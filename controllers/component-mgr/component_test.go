package componentmgr

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
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

func TestStatefulSetNeedsUpdateSkipsEquivalentDesiredState(t *testing.T) {
	t.Parallel()

	replicas := int32(1)
	old := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "rbd-monitor",
			Namespace:       "rbd-system",
			ResourceVersion: "42",
			Labels:          map[string]string{"name": "rbd-monitor"},
			OwnerReferences: []metav1.OwnerReference{{UID: "component-uid", Controller: boolPtr(true)}},
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas:             &replicas,
			ServiceName:          "rbd-monitor",
			PodManagementPolicy:  appsv1.OrderedReadyPodManagement,
			RevisionHistoryLimit: int32Ptr(10),
			UpdateStrategy: appsv1.StatefulSetUpdateStrategy{
				Type: appsv1.RollingUpdateStatefulSetStrategyType,
			},
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"name": "rbd-monitor"}},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"name": "rbd-monitor"}},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyAlways,
					DNSPolicy:     corev1.DNSClusterFirst,
					SchedulerName: "default-scheduler",
					Containers: []corev1.Container{{
						Name:                     "rbd-monitor",
						Image:                    "monitor:v1",
						TerminationMessagePath:   "/dev/termination-log",
						TerminationMessagePolicy: corev1.TerminationMessageReadFile,
					}},
				},
			},
		},
	}

	desired := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:            old.Name,
			Namespace:       old.Namespace,
			Labels:          map[string]string{"name": "rbd-monitor"},
			OwnerReferences: old.OwnerReferences,
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"name": "rbd-monitor"}},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"name": "rbd-monitor"}},
				Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "rbd-monitor", Image: "monitor:v1"}}},
			},
		},
	}

	mgr := &RbdcomponentMgr{}
	normalized := mgr.updateRuntimeObject(old, desired).(*appsv1.StatefulSet)
	if statefulSetNeedsUpdate(old, normalized) {
		t.Fatal("expected API-defaulted fields to be ignored when desired StatefulSet is unchanged")
	}
}

func TestStatefulSetNeedsUpdateDetectsPodTemplateChange(t *testing.T) {
	t.Parallel()

	old := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{Name: "rbd-monitor", ResourceVersion: "42"},
		Spec: appsv1.StatefulSetSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "rbd-monitor", Image: "monitor:v1"}}},
			},
		},
	}
	desired := old.DeepCopy()
	desired.Spec.Template.Spec.Containers[0].Image = "monitor:v2"

	if !statefulSetNeedsUpdate(old, desired) {
		t.Fatal("expected a pod template change to update the StatefulSet")
	}
}

func TestStatefulSetNeedsUpdateIgnoresUnmanagedSpecFields(t *testing.T) {
	t.Parallel()

	replicas := int32(1)
	old := &appsv1.StatefulSet{
		Spec: appsv1.StatefulSetSpec{
			Replicas:             &replicas,
			RevisionHistoryLimit: int32Ptr(30),
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "rbd-monitor", Image: "monitor:v1"}}},
			},
		},
	}
	desired := &appsv1.StatefulSet{
		Spec: appsv1.StatefulSetSpec{
			Replicas: &replicas,
			Template: old.Spec.Template,
		},
	}

	if statefulSetNeedsUpdate(old, desired) {
		t.Fatal("expected unmanaged StatefulSet fields to be ignored")
	}
}

func TestResourceNeedsUpdateSkipsEquivalentConfigMapAndService(t *testing.T) {
	t.Parallel()

	mgr := &RbdcomponentMgr{log: logr.Discard()}
	ownerReferences := []metav1.OwnerReference{{UID: "component-uid", Controller: boolPtr(true)}}
	currentConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "prometheus-config",
			Namespace:       "rbd-system",
			ResourceVersion: "42",
			OwnerReferences: ownerReferences,
		},
		Data: map[string]string{
			"prometheus.yml": "global: {}",
			"rules.yml":      "groups: []",
			"plugin-rules":   "groups: []",
		},
	}
	desiredConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:            currentConfigMap.Name,
			Namespace:       currentConfigMap.Namespace,
			OwnerReferences: ownerReferences,
		},
		Data: map[string]string{
			"prometheus.yml": "global: {}",
			"rules.yml":      "groups: []",
		},
	}
	normalizedConfigMap := mgr.updateRuntimeObject(currentConfigMap, desiredConfigMap).(*corev1.ConfigMap)
	if normalizedConfigMap.ResourceVersion != currentConfigMap.ResourceVersion {
		t.Fatalf("expected resource version %q, got %q", currentConfigMap.ResourceVersion, normalizedConfigMap.ResourceVersion)
	}
	if got := normalizedConfigMap.Data["plugin-rules"]; got != "groups: []" {
		t.Fatalf("expected plugin ConfigMap key to be preserved, got %q", got)
	}
	if resourceNeedsUpdate(currentConfigMap, normalizedConfigMap) {
		t.Fatal("expected ConfigMap with only plugin-owned keys to skip update")
	}
	normalizedConfigMap.Data["prometheus.yml"] = "global:\n  scrape_interval: 15s"
	if !resourceNeedsUpdate(currentConfigMap, normalizedConfigMap) {
		t.Fatal("expected a change to an operator-managed ConfigMap key to update")
	}

	currentService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "rbd-monitor",
			Namespace:       "rbd-system",
			ResourceVersion: "42",
			OwnerReferences: ownerReferences,
		},
		Spec: corev1.ServiceSpec{
			ClusterIP: "10.43.1.2",
			Ports:     []corev1.ServicePort{{Name: "http", Port: 9999, TargetPort: intstr.FromInt(9090), Protocol: corev1.ProtocolTCP}},
			Selector:  map[string]string{"name": "rbd-monitor"},
			Type:      corev1.ServiceTypeClusterIP,
		},
	}
	desiredService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:            currentService.Name,
			Namespace:       currentService.Namespace,
			OwnerReferences: ownerReferences,
		},
		Spec: corev1.ServiceSpec{
			Ports:    []corev1.ServicePort{{Name: "http", Port: 9999, TargetPort: intstr.FromInt(9090)}},
			Selector: map[string]string{"name": "rbd-monitor"},
		},
	}
	normalizedService := mgr.updateRuntimeObject(currentService, desiredService).(*corev1.Service)
	if resourceNeedsUpdate(currentService, normalizedService) {
		t.Fatal("expected equivalent Service to skip update")
	}
}

func TestUpdateOrCreateResourceSkipsEquivalentStatefulSet(t *testing.T) {
	t.Parallel()

	stored := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{Name: "rbd-monitor", Namespace: "rbd-system", ResourceVersion: "42"},
		Spec: appsv1.StatefulSetSpec{
			ServiceName:         "rbd-monitor",
			PodManagementPolicy: appsv1.OrderedReadyPodManagement,
			Selector:            &metav1.LabelSelector{MatchLabels: map[string]string{"name": "rbd-monitor"}},
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "rbd-monitor", Image: "monitor:v1"}}},
			},
		},
	}
	desired := stored.DeepCopy()
	desired.ResourceVersion = ""

	trackingClient := &updateCountingClient{statefulSet: stored}
	mgr := &RbdcomponentMgr{ctx: context.Background(), client: trackingClient, log: logr.Discard()}
	if _, err := mgr.UpdateOrCreateResource(desired); err != nil {
		t.Fatalf("reconcile resource: %v", err)
	}
	if trackingClient.updateCalls != 0 {
		t.Fatalf("expected no StatefulSet update, got %d", trackingClient.updateCalls)
	}
}

type updateCountingClient struct {
	client.Client
	statefulSet *appsv1.StatefulSet
	updateCalls int
}

func (c *updateCountingClient) Get(_ context.Context, _ client.ObjectKey, obj client.Object) error {
	c.statefulSet.DeepCopyInto(obj.(*appsv1.StatefulSet))
	return nil
}

func (c *updateCountingClient) Update(_ context.Context, _ client.Object, _ ...client.UpdateOption) error {
	c.updateCalls++
	return nil
}

func boolPtr(v bool) *bool {
	return &v
}

func int32Ptr(v int32) *int32 {
	return &v
}
