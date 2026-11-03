package handler

import (
	"bytes"
	"context"
	"os"
	"reflect"
	"strings"
	"testing"

	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"
)

func TestMonitorPrometheusConfigCollectsAPIServerRequests(t *testing.T) {
	data, err := os.ReadFile("../../config/prom/prometheus.yml")
	if err != nil {
		t.Fatal(err)
	}
	type relabelConfig struct {
		SourceLabels []string `json:"source_labels"`
		Regex        string   `json:"regex"`
		Action       string   `json:"action"`
	}
	var config struct {
		Jobs []struct {
			Name           string `json:"job_name"`
			Scheme         string `json:"scheme"`
			MetricsPath    string `json:"metrics_path"`
			ScrapeInterval string `json:"scrape_interval"`
			ScrapeTimeout  string `json:"scrape_timeout"`
			TokenFile      string `json:"bearer_token_file"`
			TLS            struct {
				CAFile             string `json:"ca_file"`
				ServerName         string `json:"server_name"`
				InsecureSkipVerify bool   `json:"insecure_skip_verify"`
			} `json:"tls_config"`
			Discovery []struct {
				Role       string `json:"role"`
				Namespaces struct {
					Names []string `json:"names"`
				} `json:"namespaces"`
			} `json:"kubernetes_sd_configs"`
			StaticConfigs  []interface{}   `json:"static_configs"`
			Relabels       []relabelConfig `json:"relabel_configs"`
			MetricRelabels []relabelConfig `json:"metric_relabel_configs"`
		} `json:"scrape_configs"`
	}
	if err := yamlutil.NewYAMLOrJSONDecoder(bytes.NewReader(data), 4096).Decode(&config); err != nil {
		t.Fatal(err)
	}
	matched := 0
	for _, job := range config.Jobs {
		if job.Name != "kubernetes-apiservers" {
			continue
		}
		matched++
		if job.Scheme != "https" || job.MetricsPath != "/metrics" {
			t.Fatalf("API server scrape must use HTTPS /metrics: %+v", job)
		}
		if job.ScrapeInterval != "30s" || job.ScrapeTimeout != "10s" {
			t.Fatal("API server scrape must have a bounded timeout and an explicit interval")
		}
		if job.TokenFile != "/var/run/secrets/kubernetes.io/serviceaccount/token" ||
			job.TLS.CAFile != "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt" ||
			job.TLS.ServerName != "kubernetes.default.svc" || job.TLS.InsecureSkipVerify {
			t.Fatal("API server scrape must use the mounted service-account credentials and verified TLS")
		}
		if len(job.Discovery) != 1 || job.Discovery[0].Role != "endpoints" ||
			!reflect.DeepEqual(job.Discovery[0].Namespaces.Names, []string{"default"}) || len(job.StaticConfigs) != 0 {
			t.Fatal("discover each API server endpoint independently instead of scraping a shared load balancer")
		}
		wantTargets := []relabelConfig{{
			SourceLabels: []string{"__meta_kubernetes_namespace", "__meta_kubernetes_service_name", "__meta_kubernetes_endpoint_port_name"},
			Regex:        "default;kubernetes;https",
			Action:       "keep",
		}}
		if !reflect.DeepEqual(job.Relabels, wantTargets) {
			t.Fatalf("API server discovery must select only default/kubernetes:https, got %+v", job.Relabels)
		}
		wantMetrics := []relabelConfig{{SourceLabels: []string{"__name__"}, Regex: "apiserver_request_total", Action: "keep"}}
		if !reflect.DeepEqual(job.MetricRelabels, wantMetrics) {
			t.Fatalf("retain the request counter used by the shipped dashboard without collecting unrelated series: %+v", job.MetricRelabels)
		}
	}
	if matched != 1 {
		t.Fatalf("expected exactly one default API server scrape job, got %d", matched)
	}
}

func TestMonitorPrometheusConfigIncludesPluginDiscoveryJobs(t *testing.T) {
	prometheusConfig, err := os.ReadFile("../../config/prom/prometheus.yml")
	if err != nil {
		t.Fatalf("read monitor Prometheus config: %v", err)
	}

	config := string(prometheusConfig)
	var parsed struct {
		ScrapeConfigs []struct {
			JobName       string `json:"job_name"`
			StaticConfigs []struct {
				Targets []string `json:"targets"`
			} `json:"static_configs"`
			KubernetesSDConfigs []struct {
				Role string `json:"role"`
			} `json:"kubernetes_sd_configs"`
		} `json:"scrape_configs"`
	}
	decoder := yamlutil.NewYAMLOrJSONDecoder(bytes.NewReader(prometheusConfig), 4096)
	if err := decoder.Decode(&parsed); err != nil {
		t.Fatalf("parse monitor Prometheus config: %v", err)
	}

	jobs := make(map[string]struct{}, len(parsed.ScrapeConfigs))
	var hamiDevicePluginTargets []string
	var hamiDevicePluginDiscoveryRoles []string
	for _, job := range parsed.ScrapeConfigs {
		jobs[job.JobName] = struct{}{}
		if job.JobName == "hami-device-plugin" {
			for _, staticConfig := range job.StaticConfigs {
				hamiDevicePluginTargets = append(hamiDevicePluginTargets, staticConfig.Targets...)
			}
			for _, discoveryConfig := range job.KubernetesSDConfigs {
				hamiDevicePluginDiscoveryRoles = append(hamiDevicePluginDiscoveryRoles, discoveryConfig.Role)
			}
		}
	}
	legacyVMManagedBlock := strings.Join([]string{
		"  # BEGIN rainbond-vm managed",
		"  - job_name: kubevirt-vm",
		"    honor_labels: true",
		"    scrape_interval: 15s",
		"    scrape_timeout: 10s",
		"    metrics_path: /metrics",
		"    scheme: https",
		"    kubernetes_sd_configs:",
		"    - role: endpoints",
		"    bearer_token_file: /var/run/secrets/kubernetes.io/serviceaccount/token",
		"    tls_config:",
		"      ca_file: /var/run/secrets/kubernetes.io/serviceaccount/ca.crt",
		"      insecure_skip_verify: true",
		"    relabel_configs:",
		"    - source_labels: [__meta_kubernetes_namespace]",
		"      regex: rbd-plugins",
		"      action: keep",
		"    - source_labels: [__meta_kubernetes_service_label_prometheus_kubevirt_io]",
		"      regex: true",
		"      action: keep",
		"    - source_labels: [__meta_kubernetes_endpoint_port_name]",
		"      regex: metrics",
		"      action: keep",
		"  # END rainbond-vm managed",
	}, "\n")
	if !strings.Contains(config, legacyVMManagedBlock) {
		t.Fatal("expected kubevirt-vm job to exactly match the legacy VM managed block")
	}
	for _, expected := range []string{
		"__meta_kubernetes_service_label_prometheus_kubevirt_io",
		"job_name: hami-device-plugin",
		"regex: hami-device-plugin-monitor",
		"__meta_kubernetes_endpoint_port_name",
		"regex: monitorport",
		"__meta_kubernetes_endpoint_node_name",
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
	for _, jobName := range []string{"hami-device-plugin", "kubevirt-vm", "gpu-observer"} {
		if _, found := jobs[jobName]; !found {
			t.Fatalf("expected parsed monitor Prometheus config to contain job %q", jobName)
		}
	}
	if len(hamiDevicePluginTargets) != 0 {
		t.Fatalf("expected hami-device-plugin to avoid static targets, got %v", hamiDevicePluginTargets)
	}
	if len(hamiDevicePluginDiscoveryRoles) != 1 || hamiDevicePluginDiscoveryRoles[0] != "endpoints" {
		t.Fatalf("expected hami-device-plugin to discover Service endpoints, got %v", hamiDevicePluginDiscoveryRoles)
	}
	if count := strings.Count(config, "job_name: kubevirt-vm"); count != 1 {
		t.Fatalf("expected exactly one operator-managed kubevirt-vm job, got %d", count)
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
