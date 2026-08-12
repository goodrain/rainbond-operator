package handler

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"

	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"
)

func TestMonitorPrometheusConfigIncludesPluginDiscoveryJobs(t *testing.T) {
	prometheusConfig, err := os.ReadFile("../../config/prom/prometheus.yml")
	if err != nil {
		t.Fatalf("read monitor Prometheus config: %v", err)
	}

	config := string(prometheusConfig)
	var parsed struct {
		ScrapeConfigs []struct {
			JobName string `json:"job_name"`
		} `json:"scrape_configs"`
	}
	decoder := yamlutil.NewYAMLOrJSONDecoder(bytes.NewReader(prometheusConfig), 4096)
	if err := decoder.Decode(&parsed); err != nil {
		t.Fatalf("parse monitor Prometheus config: %v", err)
	}

	jobs := make(map[string]struct{}, len(parsed.ScrapeConfigs))
	for _, job := range parsed.ScrapeConfigs {
		jobs[job.JobName] = struct{}{}
	}
	for _, expected := range []string{
		"job_name: kubevirt-vm",
		"__meta_kubernetes_service_label_prometheus_kubevirt_io",
		"job_name: gpu-observer",
		"__meta_kubernetes_service_name",
		"regex: gpu-observer",
		"__meta_kubernetes_endpoint_port_name",
		"regex: metrics",
	} {
		if !strings.Contains(config, expected) {
			t.Fatalf("expected monitor Prometheus config to contain %q", expected)
		}
	}
	for _, jobName := range []string{"kubevirt-vm", "gpu-observer"} {
		if _, found := jobs[jobName]; !found {
			t.Fatalf("expected parsed monitor Prometheus config to contain job %q", jobName)
		}
	}
}

func TestMonitorConfigChecksumIsStableAndDetectsChanges(t *testing.T) {
	current := map[string]string{
		"prometheus.yml": "scrape_configs: []\n",
		"rules.yml":      "groups: []\n",
	}
	withDifferentMapOrder := map[string]string{
		"rules.yml":      "groups: []\n",
		"prometheus.yml": "scrape_configs: []\n",
	}
	changed := map[string]string{
		"prometheus.yml": "scrape_configs:\n- job_name: changed\n",
		"rules.yml":      "groups: []\n",
	}

	if got, want := monitorConfigChecksum(current), monitorConfigChecksum(withDifferentMapOrder); got != want {
		t.Fatalf("expected stable checksum for equivalent data, got %q and %q", got, want)
	}
	if monitorConfigChecksum(current) == monitorConfigChecksum(changed) {
		t.Fatal("expected checksum to change when monitor configuration changes")
	}
}

func TestMonitorStatefulSetUsesConfigMapChecksum(t *testing.T) {
	component := &rainbondv1alpha1.RbdComponent{
		ObjectMeta: metav1.ObjectMeta{Name: MonitorName, Namespace: "rbd-system"},
	}
	monitorHandler := NewMonitor(context.Background(), nil, component, &rainbondv1alpha1.RainbondCluster{}).(*monitor)
	monitorHandler.pvcParametersRWO = &pvcParameters{}

	configMap := monitorHandler.configmap().(*corev1.ConfigMap)
	statefulSet := monitorHandler.statefulset().(*appsv1.StatefulSet)
	if got, want := statefulSet.Spec.Template.Annotations[monitorConfigChecksumAnnotation], monitorConfigChecksum(configMap.Data); got != want {
		t.Fatalf("expected monitor Pod template checksum %q, got %q", want, got)
	}
}
