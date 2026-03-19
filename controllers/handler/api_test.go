package handler

import (
	"context"
	"strings"
	"testing"

	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestSecretAndConfigMapForAPIRegeneratesWhenRegionConfigMissing(t *testing.T) {
	t.Parallel()

	component := &rainbondv1alpha1.RbdComponent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      APIName,
			Namespace: "rbd-system",
		},
	}
	cluster := &rainbondv1alpha1.RainbondCluster{
		Spec: rainbondv1alpha1.RainbondClusterSpec{
			GatewayIngressIPs: []string{"1.2.3.4"},
		},
	}
	availableIPs := strings.ReplaceAll(strings.Join(cluster.GatewayIngressIPs(), "-"), ".", "_")

	k8sClient := &staticClient{
		scheme:  runtime.NewScheme(),
		objects: map[client.ObjectKey]client.Object{},
	}
	k8sClient.objects[client.ObjectKey{Name: apiServerSecretName, Namespace: component.Namespace}] = &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      apiServerSecretName,
			Namespace: component.Namespace,
			Labels: map[string]string{
				"availableips": availableIPs,
			},
		},
	}
	k8sClient.objects[client.ObjectKey{Name: apiClientSecretName, Namespace: component.Namespace}] = &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      apiClientSecretName,
			Namespace: component.Namespace,
			Labels: map[string]string{
				"availableips": availableIPs,
			},
		},
	}

	handler := &api{
		ctx:       context.Background(),
		client:    k8sClient,
		component: component,
		cluster:   cluster,
		labels:    LabelsForRainbondComponent(component),
	}

	resources := handler.secretAndConfigMapForAPI()
	if len(resources) == 0 {
		t.Fatalf("expected api bootstrap resources to be regenerated when region-config is missing")
	}
	if !containsObject(resources, "region-config") {
		t.Fatalf("expected regenerated resources to include region-config")
	}
}

func TestAPIDeploymentConfiguresStartupProbeForSlowBoot(t *testing.T) {
	t.Setenv("IS_SQLLITE", "true")

	component := &rainbondv1alpha1.RbdComponent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      APIName,
			Namespace: "rbd-system",
		},
		Spec: rainbondv1alpha1.RbdComponentSpec{
			Image: "example.com/rbd-api:test",
		},
	}
	cluster := &rainbondv1alpha1.RainbondCluster{
		Spec: rainbondv1alpha1.RainbondClusterSpec{
			SuffixHTTPHost: "example.com",
		},
	}
	handler := &api{
		ctx:       context.Background(),
		component: component,
		cluster:   cluster,
		labels:    LabelsForRainbondComponent(component),
	}

	deployment, ok := handler.deployment().(*appsv1.Deployment)
	if !ok {
		t.Fatalf("expected *appsv1.Deployment, got %T", handler.deployment())
	}
	if len(deployment.Spec.Template.Spec.Containers) != 1 {
		t.Fatalf("expected one container, got %d", len(deployment.Spec.Template.Spec.Containers))
	}

	container := deployment.Spec.Template.Spec.Containers[0]
	if container.StartupProbe == nil {
		t.Fatalf("expected startupProbe to be configured")
	}
	if container.StartupProbe.HTTPGet == nil {
		t.Fatalf("expected startupProbe to use HTTP GET")
	}
	if got := container.StartupProbe.HTTPGet.Path; got != "/v2/health" {
		t.Fatalf("expected startupProbe path /v2/health, got %q", got)
	}
	if got := container.StartupProbe.HTTPGet.Port.IntValue(); got != 8888 {
		t.Fatalf("expected startupProbe port 8888, got %d", got)
	}
	if container.ReadinessProbe == nil || container.ReadinessProbe.HTTPGet == nil {
		t.Fatalf("expected readinessProbe HTTP GET to remain configured")
	}
	if got := container.ReadinessProbe.HTTPGet.Path; got != "/v2/health" {
		t.Fatalf("expected readinessProbe path /v2/health, got %q", got)
	}
	if container.LivenessProbe == nil || container.LivenessProbe.HTTPGet == nil {
		t.Fatalf("expected livenessProbe HTTP GET to remain configured")
	}
	if got := container.LivenessProbe.HTTPGet.Path; got != "/healthz" {
		t.Fatalf("expected livenessProbe path /healthz, got %q", got)
	}
}

func containsObject(objects []client.Object, name string) bool {
	for _, object := range objects {
		if object != nil && object.GetName() == name {
			return true
		}
	}
	return false
}

type staticClient struct {
	scheme  *runtime.Scheme
	objects map[client.ObjectKey]client.Object
}

func (s *staticClient) Get(_ context.Context, key client.ObjectKey, obj client.Object) error {
	stored, ok := s.objects[key]
	if !ok {
		return apierrors.NewNotFound(schema.GroupResource{Resource: "objects"}, key.Name)
	}

	switch out := obj.(type) {
	case *corev1.Secret:
		secret, ok := stored.(*corev1.Secret)
		if !ok {
			return apierrors.NewBadRequest("stored object is not a secret")
		}
		secret.DeepCopyInto(out)
		return nil
	case *corev1.ConfigMap:
		configMap, ok := stored.(*corev1.ConfigMap)
		if !ok {
			return apierrors.NewBadRequest("stored object is not a configmap")
		}
		configMap.DeepCopyInto(out)
		return nil
	default:
		return apierrors.NewBadRequest("unsupported object type")
	}
}

func (s *staticClient) List(context.Context, client.ObjectList, ...client.ListOption) error {
	panic("unexpected List call in test")
}

func (s *staticClient) Create(context.Context, client.Object, ...client.CreateOption) error {
	panic("unexpected Create call in test")
}

func (s *staticClient) Delete(context.Context, client.Object, ...client.DeleteOption) error {
	panic("unexpected Delete call in test")
}

func (s *staticClient) Update(context.Context, client.Object, ...client.UpdateOption) error {
	panic("unexpected Update call in test")
}

func (s *staticClient) Patch(context.Context, client.Object, client.Patch, ...client.PatchOption) error {
	panic("unexpected Patch call in test")
}

func (s *staticClient) DeleteAllOf(context.Context, client.Object, ...client.DeleteAllOfOption) error {
	panic("unexpected DeleteAllOf call in test")
}

func (s *staticClient) Status() client.StatusWriter {
	return staticStatusWriter{}
}

func (s *staticClient) Scheme() *runtime.Scheme {
	return s.scheme
}

func (s *staticClient) RESTMapper() meta.RESTMapper {
	return nil
}

type staticStatusWriter struct{}

func (staticStatusWriter) Update(context.Context, client.Object, ...client.UpdateOption) error {
	panic("unexpected Status().Update call in test")
}

func (staticStatusWriter) Patch(context.Context, client.Object, client.Patch, ...client.PatchOption) error {
	panic("unexpected Status().Patch call in test")
}
