package handler

import (
	"context"
	"testing"

	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
			Image: "example.com/apisix-ingress:test@example.com/apisix:test",
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
