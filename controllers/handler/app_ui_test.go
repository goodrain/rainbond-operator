package handler

import (
	"context"
	"testing"

	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func newAppUIHandlerForTest(k8sClient client.Client) *appui {
	component := &rainbondv1alpha1.RbdComponent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      AppUIName,
			Namespace: "rbd-system",
		},
		Spec: rainbondv1alpha1.RbdComponentSpec{
			Image: "example.com/rbd-app-ui:test",
		},
	}
	cluster := &rainbondv1alpha1.RainbondCluster{
		Spec: rainbondv1alpha1.RainbondClusterSpec{
			SuffixHTTPHost: "example.com",
			ImageHub:       &rainbondv1alpha1.ImageHub{Domain: "example.com"},
		},
	}
	return &appui{
		ctx:       context.Background(),
		client:    k8sClient,
		component: component,
		cluster:   cluster,
		labels:    LabelsForRainbondComponent(component),
	}
}

func findEnv(envs []corev1.EnvVar, name string) (corev1.EnvVar, bool) {
	for _, e := range envs {
		if e.Name == name {
			return e, true
		}
	}
	return corev1.EnvVar{}, false
}

func TestAppUIDeploymentDisablesDefaultMarketForOfflineInstall(t *testing.T) {
	t.Setenv("IS_SQLLITE", "true")

	handler := newAppUIHandlerForTest(nil)
	handler.cluster.Spec.InstallMode = rainbondv1alpha1.InstallationModeOffline
	handler.component.Spec.Env = []corev1.EnvVar{
		{Name: "DISABLE_DEFAULT_APP_MARKET", Value: "false"},
	}

	deployment := handler.deploymentForAppUI().(*appsv1.Deployment)
	env, ok := findEnv(deployment.Spec.Template.Spec.Containers[0].Env, "DISABLE_DEFAULT_APP_MARKET")
	if !ok {
		t.Fatalf("expected DISABLE_DEFAULT_APP_MARKET env to be present")
	}
	if got := env.Value; got != "true" {
		t.Fatalf("expected DISABLE_DEFAULT_APP_MARKET=true, got %q", got)
	}
}

// SECRET_KEY must be injected from the persistent Secret via secretKeyRef,
// never as an inline value derived from volatile host hardware info.
func TestAppUIDeploymentInjectsSecretKeyFromSecretRef(t *testing.T) {
	t.Setenv("IS_SQLLITE", "true")

	handler := newAppUIHandlerForTest(nil)

	deployment, ok := handler.deploymentForAppUI().(*appsv1.Deployment)
	if !ok {
		t.Fatalf("expected *appsv1.Deployment, got %T", handler.deploymentForAppUI())
	}
	if len(deployment.Spec.Template.Spec.Containers) == 0 {
		t.Fatalf("expected at least one container")
	}

	env, ok := findEnv(deployment.Spec.Template.Spec.Containers[0].Env, "SECRET_KEY")
	if !ok {
		t.Fatalf("expected SECRET_KEY env to be present")
	}
	if env.Value != "" {
		t.Fatalf("SECRET_KEY must not be an inline value, got %q", env.Value)
	}
	if env.ValueFrom == nil || env.ValueFrom.SecretKeyRef == nil {
		t.Fatalf("SECRET_KEY must be sourced from a secretKeyRef")
	}
	if got := env.ValueFrom.SecretKeyRef.Name; got != appUISecretName {
		t.Fatalf("expected secretKeyRef name %q, got %q", appUISecretName, got)
	}
	if got := env.ValueFrom.SecretKeyRef.Key; got != appUISecretKey {
		t.Fatalf("expected secretKeyRef key %q, got %q", appUISecretKey, got)
	}
}

// When no Secret exists yet, the operator generates a strong random key.
func TestAppUISecretGeneratesRandomKeyWhenAbsent(t *testing.T) {
	k8sClient := &staticClient{
		scheme:  runtime.NewScheme(),
		objects: map[client.ObjectKey]client.Object{},
	}
	handler := newAppUIHandlerForTest(k8sClient)

	obj := handler.secretForAppUI()
	secret, ok := obj.(*corev1.Secret)
	if !ok {
		t.Fatalf("expected *corev1.Secret, got %T", obj)
	}
	if secret.Name != appUISecretName {
		t.Fatalf("expected secret name %q, got %q", appUISecretName, secret.Name)
	}
	key := secret.Data[appUISecretKey]
	if len(key) == 0 {
		t.Fatalf("expected a generated SECRET_KEY")
	}
	// 32 random bytes hex-encoded => 64 chars.
	if len(key) != 64 {
		t.Fatalf("expected 64-char hex key, got %d chars", len(key))
	}
}

// The existing key MUST be preserved across reconciles/upgrades, otherwise
// every reconcile rotates SECRET_KEY and logs every user out.
func TestAppUISecretReusesExistingKey(t *testing.T) {
	const preserved = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	k8sClient := &staticClient{
		scheme:  runtime.NewScheme(),
		objects: map[client.ObjectKey]client.Object{},
	}
	k8sClient.objects[client.ObjectKey{Name: appUISecretName, Namespace: "rbd-system"}] = &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: appUISecretName, Namespace: "rbd-system"},
		Data:       map[string][]byte{appUISecretKey: []byte(preserved)},
	}
	handler := newAppUIHandlerForTest(k8sClient)

	obj := handler.secretForAppUI()
	secret, ok := obj.(*corev1.Secret)
	if !ok {
		t.Fatalf("expected *corev1.Secret, got %T", obj)
	}
	if got := string(secret.Data[appUISecretKey]); got != preserved {
		t.Fatalf("expected existing SECRET_KEY to be reused, got %q", got)
	}
}

// Two independent generations must differ (real randomness, not a constant).
func TestRandomSecretKeyIsUnique(t *testing.T) {
	a := randomSecretKey()
	b := randomSecretKey()
	if a == "" || b == "" {
		t.Fatalf("expected non-empty keys")
	}
	if a == b {
		t.Fatalf("expected two random keys to differ, both were %q", a)
	}
}
