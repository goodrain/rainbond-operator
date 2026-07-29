package handler

import (
	"strings"
	"testing"

	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestLocalPathHelperPodIsSystemClusterCritical(t *testing.T) {
	t.Parallel()

	component := &rainbondv1alpha1.RbdComponent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      LocalPathName,
			Namespace: "rbd-system",
		},
	}
	handler := &localPath{component: component}
	configMap, ok := handler.configMap().(*corev1.ConfigMap)
	if !ok {
		t.Fatalf("expected *corev1.ConfigMap, got %T", handler.configMap())
	}
	if got := configMap.Data["helperPod.yaml"]; !strings.Contains(got, "priorityClassName: system-cluster-critical") {
		t.Fatalf("expected helper pod to use system-cluster-critical, got:\n%s", got)
	}
	if got := configMap.Data["helperPod.yaml"]; !strings.Contains(got, "ephemeral-storage: 256Mi") {
		t.Fatalf("expected helper pod ephemeral-storage request 256Mi, got:\n%s", got)
	}
}
