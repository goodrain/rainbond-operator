package handler

import (
	"context"
	"strings"
	"testing"

	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/api/v1alpha1"
	v2 "github.com/goodrain/rainbond-operator/api/v2"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestAPIGatewayResourcesPreserveConfiguredEphemeralStorageRequest(t *testing.T) {
	t.Parallel()

	resources := setDefaultAPIGatewayResources(corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceEphemeralStorage: resource.MustParse("2Gi"),
		},
	})
	if got := resources.Requests[corev1.ResourceEphemeralStorage]; got.String() != "2Gi" {
		t.Fatalf("expected configured ephemeral-storage request 2Gi, got %s", got.String())
	}
}

func TestAPIGatewayDeploymentIsCriticalAndRunsOncePerGatewayNode(t *testing.T) {
	t.Parallel()

	component := &rainbondv1alpha1.RbdComponent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ApiGatewayName,
			Namespace: "rbd-system",
		},
		Spec: rainbondv1alpha1.RbdComponentSpec{
			Image: "example.com/apisix-ingress:1.8.4@registry.cn-hangzhou.aliyuncs.com/goodrain/apisix:3.14.1-debian",
		},
	}
	cluster := &rainbondv1alpha1.RainbondCluster{
		Spec: rainbondv1alpha1.RainbondClusterSpec{
			NodesForGateway: []*rainbondv1alpha1.K8sNode{
				{Name: "gateway-1"},
				{Name: "gateway-2"},
			},
		},
	}
	handler := &apigateway{
		ctx:       context.Background(),
		component: component,
		cluster:   cluster,
		labels:    LabelsForRainbondComponent(component),
	}

	deployment, ok := handler.deploy().(*appsv1.Deployment)
	if !ok {
		t.Fatalf("expected *appsv1.Deployment, got %T", handler.deploy())
	}
	podSpec := deployment.Spec.Template.Spec
	var ingressContainer, apisixContainer *corev1.Container
	for i := range podSpec.Containers {
		switch podSpec.Containers[i].Name {
		case "ingress-apisix":
			ingressContainer = &podSpec.Containers[i]
		case "apisix":
			apisixContainer = &podSpec.Containers[i]
		}
	}
	if ingressContainer == nil {
		t.Fatal("expected ingress-apisix container")
	}
	if got := ingressContainer.Image; got != "example.com/apisix-ingress:1.8.4" {
		t.Fatalf("expected APISIX ingress controller 1.8.4 image, got %q", got)
	}
	if !strings.Contains(strings.Join(ingressContainer.Command, " "), "--apisix-admin-api-version v3") {
		t.Fatalf("expected ingress controller to use APISIX Admin API v3, got %v", ingressContainer.Command)
	}
	if apisixContainer == nil {
		t.Fatal("expected apisix container")
	}
	if got := apisixContainer.Image; got != "registry.cn-hangzhou.aliyuncs.com/goodrain/apisix:3.14.1-debian" {
		t.Fatalf("expected APISIX 3.14.1 image, got %q", got)
	}
	if got := podSpec.PriorityClassName; got != "system-cluster-critical" {
		t.Fatalf("expected priority class system-cluster-critical, got %q", got)
	}
	if got := deployment.Spec.Strategy.Type; got != appsv1.RollingUpdateDeploymentStrategyType {
		t.Fatalf("expected rolling update strategy, got %q", got)
	}
	rollingUpdate := deployment.Spec.Strategy.RollingUpdate
	if rollingUpdate == nil || rollingUpdate.MaxUnavailable == nil || rollingUpdate.MaxUnavailable.IntValue() != 1 {
		t.Fatalf("expected maxUnavailable 1, got %#v", rollingUpdate)
	}
	if rollingUpdate.MaxSurge == nil || rollingUpdate.MaxSurge.IntValue() != 0 {
		t.Fatalf("expected maxSurge 0, got %#v", rollingUpdate.MaxSurge)
	}
	for _, container := range podSpec.Containers {
		if got := container.Resources.Requests[corev1.ResourceEphemeralStorage]; got.String() != "512Mi" {
			t.Fatalf("expected %s ephemeral-storage request 512Mi, got %s", container.Name, got.String())
		}
	}
	if podSpec.Affinity == nil || podSpec.Affinity.NodeAffinity == nil {
		t.Fatalf("expected gateway node affinity to remain configured")
	}
	if podSpec.Affinity.PodAntiAffinity == nil {
		t.Fatalf("expected pod anti-affinity to keep one gateway pod per node")
	}
	terms := podSpec.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution
	if len(terms) != 1 {
		t.Fatalf("expected one required pod anti-affinity term, got %d", len(terms))
	}
	if got := terms[0].TopologyKey; got != "kubernetes.io/hostname" {
		t.Fatalf("expected topology key kubernetes.io/hostname, got %q", got)
	}
	if terms[0].LabelSelector == nil {
		t.Fatalf("expected pod anti-affinity label selector")
	}
	if got := terms[0].LabelSelector.MatchLabels["name"]; got != ApiGatewayName {
		t.Fatalf("expected pod anti-affinity to select %s, got %q", ApiGatewayName, got)
	}
}

func TestMonitorGlobalRuleUsesOnlyBuiltInPrometheus(t *testing.T) {
	t.Parallel()

	rule := (&apigateway{}).monitorGlobalRule().(*v2.ApisixGlobalRule)
	if len(rule.Spec.Plugins) != 1 {
		t.Fatalf("expected one global plugin, got %d", len(rule.Spec.Plugins))
	}
	if got := rule.Spec.Plugins[0].Name; got != "prometheus" {
		t.Fatalf("expected prometheus global plugin, got %q", got)
	}
}

func TestAPIGatewayConfigUsesSameAdminKeyAndAPIVersionAsIngressController(t *testing.T) {
	t.Parallel()

	configMap := (&apigateway{}).configmap().(*corev1.ConfigMap)
	config := configMap.Data["config.yaml"]
	for _, expected := range []string{
		"admin_key_required: true",
		"admin_key:",
		"name: admin",
		"key: " + apisixAdminKey,
		"role: admin",
		"admin_api_version: v3",
	} {
		if !strings.Contains(config, expected) {
			t.Fatalf("expected APISIX config to contain %q, got:\n%s", expected, config)
		}
	}
}

func TestAPIGatewayResourcesMigratesAdminConfigWithoutReplacingCustomSettings(t *testing.T) {
	t.Parallel()

	component := &rainbondv1alpha1.RbdComponent{
		ObjectMeta: metav1.ObjectMeta{Name: ApiGatewayName, Namespace: "rbd-system"},
	}
	existing := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "apisix-gw-config.yaml", Namespace: component.Namespace},
		Data: map[string]string{
			"config.yaml": `deployment:
  admin:
    allow_admin:
      - 10.0.0.0/8
    admin_key:
      - name: viewer
        key: read-only
        role: viewer
      - name: admin
        key: outdated
        role: viewer
  etcd:
    host:
      - http://127.0.0.1:12379
custom_setting:
  preserved: true
`,
		},
	}
	k8sClient := &staticClient{
		scheme: runtime.NewScheme(),
		objects: map[client.ObjectKey]client.Object{
			{Name: existing.Name, Namespace: existing.Namespace}: existing,
		},
	}
	handler := &apigateway{
		ctx:       context.Background(),
		client:    k8sClient,
		component: component,
		cluster:   &rainbondv1alpha1.RainbondCluster{},
		labels:    LabelsForRainbondComponent(component),
	}

	var migrated *corev1.ConfigMap
	for _, object := range handler.Resources() {
		if configMap, ok := object.(*corev1.ConfigMap); ok && configMap.Name == existing.Name {
			migrated = configMap
			break
		}
	}
	if migrated == nil {
		t.Fatal("expected existing APISIX ConfigMap to be migrated")
	}
	config := migrated.Data["config.yaml"]
	for _, expected := range []string{
		"admin_key_required: true",
		"admin_api_version: v3",
		"key: " + apisixAdminKey,
		"key: read-only",
		"role: viewer",
		"10.0.0.0/8",
		"custom_setting:",
		"preserved: true",
	} {
		if !strings.Contains(config, expected) {
			t.Fatalf("expected migrated APISIX config to contain %q, got:\n%s", expected, config)
		}
	}
	changed, err := ensureAPISIXAdminConfig(migrated)
	if err != nil {
		t.Fatalf("expected migrated APISIX config to remain valid: %v", err)
	}
	if changed {
		t.Fatal("expected APISIX admin config migration to be idempotent")
	}
}
