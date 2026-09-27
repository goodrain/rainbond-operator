package handler

import (
	"context"
	"reflect"
	"strings"
	"testing"

	rbd "github.com/goodrain/rainbond-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func coordinatedHubFixture() *hub {
	component := &rbd.RbdComponent{ObjectMeta: metav1.ObjectMeta{Name: "rbd-hub", Namespace: "rbd-system"}, Spec: rbd.RbdComponentSpec{Image: "registry.example/native@sha256:" + strings.Repeat("a", 64)}}
	component.Spec.RegistryCoordination = &rbd.RegistryCoordinationSpec{
		Image:      "registry.example/cleanup@sha256:" + strings.Repeat("b", 64),
		ConsoleURL: "http://rbd-app-ui.rbd-system.svc.cluster.local:7070", AllowHTTP: true,
		EnterpriseID: "enterprise", RegionName: "rainbond", ControlSecret: "cleanup-control", PermitSecret: "cleanup-permit",
		StorageID: "store", Generation: "one", VolumeUID: strings.Repeat("c", 64), RegistryPath: "/var/lib/registry",
	}
	cluster := &rbd.RainbondCluster{Spec: rbd.RainbondClusterSpec{ImageHub: &rbd.ImageHub{Domain: "goodrain.me"}}}
	return &hub{ctx: context.Background(), component: component, cluster: cluster, labels: LabelsForRainbondComponent(component)}
}

func TestRegistryCoordinationRendersOnePersistentIngress(t *testing.T) {
	h := coordinatedHubFixture()
	original := h.component.DeepCopy()
	object := h.deployment()
	if object == nil {
		t.Fatal("valid configuration rejected")
	}
	d := object.(*appsv1.Deployment)
	if d.Spec.Strategy.Type != appsv1.RecreateDeploymentStrategyType || len(d.Spec.Template.Spec.Containers) != 2 || len(d.Spec.Template.Spec.InitContainers) != 1 {
		t.Fatal("missing coordinated rollout")
	}
	native, sidecar := d.Spec.Template.Spec.Containers[0], d.Spec.Template.Spec.Containers[1]
	env := map[string]string{}
	for _, e := range native.Env {
		env[e.Name] = e.Value
	}
	if env["REGISTRY_HTTP_ADDR"] != "127.0.0.1:5000" || env["REGISTRY_STORAGE_MAINTENANCE_UPLOADPURGING_ENABLED"] != "false" {
		t.Fatal("native bypass remains")
	}
	if sidecar.Name != "registry-coordinator" || len(sidecar.Command) != 1 || sidecar.Command[0] != "/registry-coordinator" {
		t.Fatal("wrong coordinator entrypoint")
	}
	args := strings.Join(sidecar.Args, " ")
	for _, want := range []string{"--console-system-identity=true", "--console-enterprise=enterprise", "--console-region=rainbond", "--credential-file=/control/key", "--permit-key-file=/permit/key"} {
		if !strings.Contains(args, want) {
			t.Fatal("missing bound identity", want)
		}
	}
	if sidecar.ReadinessProbe == nil || sidecar.ReadinessProbe.HTTPGet.Port.IntVal != 5001 || native.ReadinessProbe == nil || native.ReadinessProbe.HTTPGet.Port.IntVal != 5001 {
		t.Fatal("probe bypasses coordinated ingress")
	}
	for _, m := range sidecar.VolumeMounts {
		if m.MountPath == "/registry" && !m.ReadOnly {
			t.Fatal("sidecar can write registry data")
		}
	}
	if d.Spec.Template.Annotations["rainbond.io/registry-gc-executor"] != "v1" {
		t.Fatal("missing verified executor declaration")
	}
	svc := h.serviceForHub().(*corev1.Service)
	if svc.Spec.Ports[0].Port != 5000 || svc.Spec.Ports[0].TargetPort.IntVal != 5001 {
		t.Fatal("public service bypasses sidecar")
	}
	if !reflect.DeepEqual(original, h.component) {
		t.Fatal("render mutated source configuration")
	}
	h.component.Spec.RegistryCoordination = nil
	plain := h.deployment().(*appsv1.Deployment)
	if len(plain.Spec.Template.Spec.Containers) != 1 || len(plain.Spec.Template.Spec.InitContainers) != 0 || h.serviceForHub().(*corev1.Service).Spec.Ports[0].TargetPort.IntVal != 5000 {
		t.Fatal("changed default installation")
	}
}

func TestRegistryCoordinationRejectsUnsafeConfiguration(t *testing.T) {
	for _, variant := range []string{"mutable-image", "invalid-image-syntax", "protected-registry-image", "shared-secret", "missing-scope", "unexpected-origin", "native-bypass", "multiple-replicas", "shadow-volume"} {
		t.Run(variant, func(t *testing.T) {
			h := coordinatedHubFixture()
			c := h.component.Spec.RegistryCoordination
			switch variant {
			case "mutable-image":
				c.Image = "registry.example/cleanup:latest"
			case "invalid-image-syntax":
				c.Image = "http://registry.example/cleanup@sha256:" + strings.Repeat("b", 64)
			case "protected-registry-image":
				c.Image = "goodrain.me/cleanup@sha256:" + strings.Repeat("b", 64)
			case "shared-secret":
				c.PermitSecret = c.ControlSecret
			case "missing-scope":
				c.EnterpriseID = ""
			case "unexpected-origin":
				c.ConsoleURL = "http://example.test/arbitrary/path"
			case "native-bypass":
				h.component.Spec.Env = append(h.component.Spec.Env, corev1.EnvVar{Name: "REGISTRY_HTTP_ADDR", Value: ":5000"})
			case "multiple-replicas":
				n := int32(2)
				h.component.Spec.Replicas = &n
			case "shadow-volume":
				h.component.Spec.Volumes = append(h.component.Spec.Volumes, corev1.Volume{Name: "cleanup-control"})
			}
			if h.deployment() != nil {
				t.Fatal("unsafe configuration rendered")
			}
		})
	}
}

func TestClearingCoordinatorConfigurationCannotBypassExistingIngress(t *testing.T) {
	h := coordinatedHubFixture()
	existing := h.deployment().(*appsv1.Deployment)
	scheme := runtime.NewScheme()
	if err := appsv1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	h.client = fake.NewClientBuilder().WithScheme(scheme).WithObjects(existing).Build()
	h.component.Spec.RegistryCoordination = nil
	if err := h.protectRegistryCoordination(); err == nil {
		t.Fatal("clearing configuration bypassed existing coordination")
	}
	h.client = fake.NewClientBuilder().WithScheme(scheme).Build()
	if err := h.protectRegistryCoordination(); err != nil {
		t.Fatal("blocked default registry", err)
	}
}
