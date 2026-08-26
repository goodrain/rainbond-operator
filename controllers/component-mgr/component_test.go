package componentmgr

import (
	"context"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	"github.com/goodrain/rainbond-operator/controllers/handler"
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

func TestStatefulSetNeedsUpdateDetectsMonitorConfigChecksumChange(t *testing.T) {
	t.Parallel()

	old := &appsv1.StatefulSet{
		Spec: appsv1.StatefulSetSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{"rainbond.io/monitor-config-checksum": "before"}},
				Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "rbd-monitor", Image: "monitor:v1"}}},
			},
		},
	}
	desired := old.DeepCopy()
	desired.Spec.Template.Annotations["rainbond.io/monitor-config-checksum"] = "after"

	if !statefulSetNeedsUpdate(old, desired) {
		t.Fatal("expected a changed monitor config checksum to roll the StatefulSet")
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

func TestResourceNeedsUpdateNormalizesMonitorConfigMapAndSkipsEquivalentService(t *testing.T) {
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
			Annotations: map[string]string{
				handler.MonitorConfigMapOwnerAnnotation: handler.MonitorName,
			},
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
	if _, found := normalizedConfigMap.Data["plugin-rules"]; found {
		t.Fatal("expected monitor ConfigMap to discard the obsolete plugin-owned key")
	}
	if !resourceNeedsUpdate(currentConfigMap, normalizedConfigMap) {
		t.Fatal("expected monitor ConfigMap with obsolete plugin-owned key to update")
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

func TestUpdateRuntimeObjectUpdatesChangedMonitorConfigMapData(t *testing.T) {
	t.Parallel()

	mgr := &RbdcomponentMgr{log: logr.Discard()}
	current := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "prometheus-config",
			Namespace:       "rbd-system",
			ResourceVersion: "42",
		},
		Data: map[string]string{
			"prometheus.yml": "operator scrape configuration",
			"rules.yml":      "operator rules",
		},
	}
	desired := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      current.Name,
			Namespace: current.Namespace,
		},
		Data: map[string]string{
			"prometheus.yml": "operator scrape configuration with kubevirt",
			"rules.yml":      "operator rules",
		},
	}

	normalized := mgr.updateRuntimeObject(current, desired).(*corev1.ConfigMap)
	if normalized.ResourceVersion != current.ResourceVersion {
		t.Fatalf("expected resource version %q, got %q", current.ResourceVersion, normalized.ResourceVersion)
	}
	if normalized.Data["prometheus.yml"] != desired.Data["prometheus.yml"] {
		t.Fatalf("expected desired monitor ConfigMap data, got %q", normalized.Data["prometheus.yml"])
	}
	if !resourceNeedsUpdate(current, normalized) {
		t.Fatal("expected a changed operator-managed monitor ConfigMap to update")
	}

	equivalent := current.DeepCopy()
	if resourceNeedsUpdate(current, equivalent) {
		t.Fatal("expected equivalent monitor ConfigMap data to skip update")
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

func TestMonitorConfigDriftTriggersOneConfigMapAndStatefulSetUpdate(t *testing.T) {
	t.Parallel()

	currentConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:            handler.MonitorConfigMapName,
			Namespace:       "rbd-system",
			ResourceVersion: "42",
		},
		Data: map[string]string{
			"prometheus.yml": strings.Join([]string{
				"scrape_configs:",
				"  - job_name: kubevirt-vm",
				"  # BEGIN rainbond-vm managed",
				"  - job_name: kubevirt-vm",
				"  # END rainbond-vm managed",
			}, "\n"),
			"rules.yml":    "rules: []",
			"plugin-rules": "obsolete",
		},
	}
	desiredConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      currentConfigMap.Name,
			Namespace: currentConfigMap.Namespace,
			Annotations: map[string]string{
				handler.MonitorConfigMapOwnerAnnotation: handler.MonitorName,
			},
		},
		Data: map[string]string{
			"prometheus.yml": strings.Join([]string{
				"scrape_configs:",
				"  # BEGIN rainbond-vm managed",
				"  - job_name: kubevirt-vm",
				"  # END rainbond-vm managed",
			}, "\n"),
			"rules.yml": "rules: []",
		},
	}
	currentStatefulSet := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{Name: handler.MonitorName, Namespace: "rbd-system", ResourceVersion: "42"},
		Spec: appsv1.StatefulSetSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{"rainbond.io/monitor-config-checksum": "stale"}},
				Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: handler.MonitorName, Image: "monitor:v1"}}},
			},
		},
	}
	desiredStatefulSet := currentStatefulSet.DeepCopy()
	desiredStatefulSet.ResourceVersion = ""
	desiredStatefulSet.Spec.Template.Annotations["rainbond.io/monitor-config-checksum"] = "current"

	trackingClient := &monitorUpdateCountingClient{
		configMap:   currentConfigMap,
		statefulSet: currentStatefulSet,
	}
	mgr := &RbdcomponentMgr{ctx: context.Background(), client: trackingClient, log: logr.Discard()}
	for _, resource := range []client.Object{desiredConfigMap.DeepCopy(), desiredStatefulSet.DeepCopy()} {
		if _, err := mgr.UpdateOrCreateResource(resource); err != nil {
			t.Fatalf("reconcile monitor resource: %v", err)
		}
	}
	if trackingClient.configMapUpdates != 1 {
		t.Fatalf("expected one monitor ConfigMap update, got %d", trackingClient.configMapUpdates)
	}
	if trackingClient.statefulSetUpdates != 1 {
		t.Fatalf("expected one monitor StatefulSet update, got %d", trackingClient.statefulSetUpdates)
	}
	if count := strings.Count(trackingClient.configMap.Data["prometheus.yml"], "job_name: kubevirt-vm"); count != 1 {
		t.Fatalf("expected monitor ConfigMap to normalize duplicate kubevirt jobs, got %d", count)
	}

	for _, resource := range []client.Object{desiredConfigMap.DeepCopy(), desiredStatefulSet.DeepCopy()} {
		if _, err := mgr.UpdateOrCreateResource(resource); err != nil {
			t.Fatalf("reconcile unchanged monitor resource: %v", err)
		}
	}
	if trackingClient.configMapUpdates != 1 || trackingClient.statefulSetUpdates != 1 {
		t.Fatalf("expected no extra updates after convergence, got ConfigMap=%d StatefulSet=%d", trackingClient.configMapUpdates, trackingClient.statefulSetUpdates)
	}
}

type updateCountingClient struct {
	client.Client
	statefulSet *appsv1.StatefulSet
	updateCalls int
}

type monitorUpdateCountingClient struct {
	client.Client
	configMap          *corev1.ConfigMap
	statefulSet        *appsv1.StatefulSet
	configMapUpdates   int
	statefulSetUpdates int
}

func (c *monitorUpdateCountingClient) Get(_ context.Context, _ client.ObjectKey, obj client.Object) error {
	switch target := obj.(type) {
	case *corev1.ConfigMap:
		c.configMap.DeepCopyInto(target)
	case *appsv1.StatefulSet:
		c.statefulSet.DeepCopyInto(target)
	default:
		panic("unexpected monitor resource type")
	}
	return nil
}

func (c *monitorUpdateCountingClient) Update(_ context.Context, obj client.Object, _ ...client.UpdateOption) error {
	switch updated := obj.(type) {
	case *corev1.ConfigMap:
		c.configMap = updated.DeepCopy()
		c.configMapUpdates++
	case *appsv1.StatefulSet:
		c.statefulSet = updated.DeepCopy()
		c.statefulSetUpdates++
	default:
		panic("unexpected monitor resource type")
	}
	return nil
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
