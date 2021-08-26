package handler

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/api/v1alpha1"
	"github.com/goodrain/rainbond-operator/util/commonutil"
	"github.com/goodrain/rainbond-operator/util/constants"
	"github.com/goodrain/rainbond-operator/util/k8sutil"
	"github.com/goodrain/rainbond-operator/util/probeutil"
	"github.com/goodrain/rainbond-operator/util/rbdutil"
	mv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NodeName name for rbd-node-proxy
var NodeName = "rbd-node-proxy"

type node struct {
	ctx       context.Context
	client    client.Client
	log       logr.Logger
	labels    map[string]string
	cluster   *rainbondv1alpha1.RainbondCluster
	component *rainbondv1alpha1.RbdComponent
}

var _ ComponentHandler = &node{}
var _ ResourcesCreator = &node{}
var _ Replicaser = &node{}

// NewNode creates a new rbd-node handler.
func NewNode(ctx context.Context, client client.Client, component *rainbondv1alpha1.RbdComponent, cluster *rainbondv1alpha1.RainbondCluster) ComponentHandler {
	return &node{
		ctx:       ctx,
		client:    client,
		log:       log.WithValues("Name: %s", component.Name),
		component: component,
		cluster:   cluster,
		labels:    LabelsForRainbondComponent(component),
	}
}

func (n *node) Before() error {
	return nil
}

func (n *node) Resources() []client.Object {
	return []client.Object{
		n.daemonSetForRainbondNode(),
		n.serviceForNode(),
		n.serviceMonitorForNode(),
		n.prometheusRuleForNode(),
	}
}

func (n *node) After() error {
	return nil
}

func (n *node) ListPods() ([]corev1.Pod, error) {
	return listPods(n.ctx, n.client, n.component.Namespace, n.labels)
}

func (n *node) ResourcesCreateIfNotExists() []client.Object {
	return []client.Object{}
}
func (n *node) Replicas() *int32 {
	nodeList := &corev1.NodeList{}
	if err := n.client.List(n.ctx, nodeList); err != nil {
		n.log.V(6).Info(fmt.Sprintf("list nodes: %v", err))
		return nil
	}
	return commonutil.Int32(int32(len(nodeList.Items)))
}

func (n *node) daemonSetForRainbondNode() client.Object {
	volumeMounts := []corev1.VolumeMount{
		{
			Name:      "sys",
			MountPath: "/sys",
		},
		{
			Name:      "dockersock",
			MountPath: "/var/run/docker.sock",
		},
		{
			Name:      "docker", // for container logs, ubuntu
			MountPath: "/var/lib/docker",
		},
		{
			Name:      "vardocker", // for container logs, centos
			MountPath: "/var/docker/lib",
		},
		{
			Name:      "dockercert",
			MountPath: "/etc/docker/certs.d",
		},
		{
			Name:      "etc",
			MountPath: "/newetc",
		},
		{
			Name:      "grlocaldata",
			MountPath: "/grlocaldata",
		},
	}
	volumes := []corev1.Volume{
		{
			Name: "sys",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/sys",
					Type: k8sutil.HostPath(corev1.HostPathDirectory),
				},
			},
		},
		{
			Name: "docker",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/var/lib/docker",
					Type: k8sutil.HostPath(corev1.HostPathDirectoryOrCreate),
				},
			},
		},
		{
			Name: "vardocker",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/var/docker/lib",
					Type: k8sutil.HostPath(corev1.HostPathDirectoryOrCreate),
				},
			},
		},

		{
			Name: "dockercert",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/etc/docker/certs.d",
					Type: k8sutil.HostPath(corev1.HostPathDirectoryOrCreate),
				},
			},
		},
		{
			Name: "dockersock",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/var/run/docker.sock",
					Type: k8sutil.HostPath(corev1.HostPathSocket),
				},
			},
		},
		{
			Name: "etc",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/etc",
					Type: k8sutil.HostPath(corev1.HostPathDirectory),
				},
			},
		},
		{
			Name: "grlocaldata",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/grlocaldata",
					Type: k8sutil.HostPathDirectoryOrCreate(),
				},
			},
		},
	}
	args := []string{
		"--image-repo-host=" + rbdutil.GetImageRepository(n.cluster),
		"--rbd-ns=" + n.component.Namespace,
	}
	if n.cluster.Spec.GatewayVIP != "" {
		args = append(args, "--gateway-vip="+n.cluster.Spec.GatewayVIP)
	}
	volumeMounts = mergeVolumeMounts(volumeMounts, n.component.Spec.VolumeMounts)
	volumes = mergeVolumes(volumes, n.component.Spec.Volumes)

	envs := []corev1.EnvVar{
		{
			Name: "POD_IP",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "status.podIP",
				},
			},
		},
		{
			Name: "NODE_NAME",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "spec.nodeName",
				},
			},
		},
		{
			Name:  "RBD_NAMESPACE",
			Value: n.component.Namespace,
		},
	}
	if n.cluster.Spec.ImageHub == nil || n.cluster.Spec.ImageHub.Domain == constants.DefImageRepository {
		envs = append(envs, corev1.EnvVar{
			Name:  "RBD_DOCKER_SECRET",
			Value: hubImageRepository,
		})
	}
	envs = mergeEnvs(envs, n.component.Spec.Env)

	// prepare probe
	readinessProbe := probeutil.MakeReadinessProbeHTTP("", "/v2/ping", 6100)
	args = mergeArgs(args, n.component.Spec.Args)
	ds := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      NodeName,
			Namespace: n.component.Namespace,
			Labels:    n.labels,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: n.labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:   NodeName,
					Labels: n.labels,
				},
				Spec: corev1.PodSpec{
					ImagePullSecrets:              imagePullSecrets(n.component, n.cluster),
					TerminationGracePeriodSeconds: commonutil.Int64(0),
					ServiceAccountName:            "rainbond-operator",
					HostAliases:                   hostsAliases(n.cluster),
					HostPID:                       true,
					DNSPolicy:                     corev1.DNSClusterFirstWithHostNet,
					HostNetwork:                   true,
					Tolerations: []corev1.Toleration{
						{
							Operator: corev1.TolerationOpExists, // tolerate everything.
						},
					},
					Containers: []corev1.Container{
						{
							Name:            NodeName,
							Image:           n.component.Spec.Image,
							ImagePullPolicy: n.component.ImagePullPolicy(),
							Env:             envs,
							Args:            args,
							VolumeMounts:    volumeMounts,
							ReadinessProbe:  readinessProbe,
						},
					},
					Volumes: volumes,
				},
			},
		},
	}

	return ds
}

func (n *node) serviceForNode() client.Object {
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      NodeName,
			Namespace: n.component.Namespace,
			Labels:    n.labels,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name: "api",
					Port: 6100,
					TargetPort: intstr.IntOrString{
						IntVal: 6100,
					},
				},
			},
			Selector: n.labels,
		},
	}
	return svc
}

func (n *node) serviceMonitorForNode() client.Object {
	return &mv1.ServiceMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Name:        NodeName,
			Namespace:   n.component.Namespace,
			Labels:      n.labels,
			Annotations: map[string]string{"ignore_controller_update": "true"},
		},
		Spec: mv1.ServiceMonitorSpec{
			NamespaceSelector: mv1.NamespaceSelector{
				MatchNames: []string{n.component.Namespace},
			},
			Selector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"name": NodeName,
				},
			},
			Endpoints: []mv1.Endpoint{
				{
					Port:          "api",
					Path:          "/app/metrics",
					Interval:      "5s",
					ScrapeTimeout: "4s",
				},
				{
					Port:          "api",
					Path:          "/node/metrics",
					Interval:      "30s",
					ScrapeTimeout: "30s",
				},
			},
			JobLabel: "name",
		},
	}
}

func (n *node) prometheusRuleForNode() client.Object {
	region := n.cluster.Annotations["regionName"]
	if region == "" {
		region = "default"
	}
	alertName := n.cluster.Annotations["alertName"]
	if alertName == "" {
		alertName = "rainbond"
	}
	commonLables := map[string]string{
		"Alert":  alertName,
		"Region": region,
	}
	getseverityLables := func(severity string) map[string]string {
		commonLables["severity"] = severity
		return commonLables
	}
	return &mv1.PrometheusRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      NodeName,
			Namespace: n.component.Namespace,
			Labels:    n.labels,
		},
		Spec: mv1.PrometheusRuleSpec{
			Groups: []mv1.RuleGroup{
				{
					Name:     "node-default-rule",
					Interval: "20s",
					Rules: []mv1.Rule{
						{
							Alert:       "HighCpuUsageOnNode",
							Expr:        intstr.FromString("sum by(instance) (rate(process_cpu_seconds_total[5m])) * 100 > 70"),
							For:         "5m",
							Labels:      getseverityLables("warning"),
							Annotations: map[string]string{"description": "{{ $labels.instance }} is using a LOT of CPU. CPU usage is {{ humanize $value}}%.", "summary": "HIGH CPU USAGE WARNING ON '{{ $labels.instance }}'"},
						},
						{
							Alert:       "HighLoadOnNode",
							Expr:        intstr.FromString("sum by (instance, job) (node_load5) / count by (instance, job) (node_cpu_seconds_total{mode=\"idle\"}) > 0.95"),
							For:         "5m",
							Labels:      getseverityLables("warning"),
							Annotations: map[string]string{"description": "{{ $labels.instance }} has a high load average. Load Average 5m is {{ humanize $value}}.", "summary": "HIGH LOAD AVERAGE WARNING ON '{{ $labels.instance }}'"},
						},
						{
							Alert:       "InodeFreerateLow",
							Expr:        intstr.FromString("node_filesystem_files_free{fstype=~\"ext4|xfs\"} / node_filesystem_files{fstype=~\"ext4|xfs\"} < 0.3"),
							For:         "5m",
							Labels:      getseverityLables("warning"),
							Annotations: map[string]string{"description": "the inode free rate is low of node {{ $labels.instance }}, current value is {{ humanize $value}}."},
						},
						{
							Alert:       "HighRootdiskUsageOnNode",
							Expr:        intstr.FromString("(node_filesystem_size_bytes{mountpoint='/'} - node_filesystem_free_bytes{mountpoint='/'}) * 100 / node_filesystem_size_bytes{mountpoint='/'} > 85"),
							For:         "5m",
							Labels:      getseverityLables("warning"),
							Annotations: map[string]string{"description": "More than 85% of disk used. Disk usage {{ humanize $value }} mountpoint {{ $labels.mountpoint }}%.", "summary": "LOW DISK SPACE WARING:NODE '{{ $labels.instance }}"},
						},
						{
							Alert:       "HighDockerdiskUsageOnNode",
							Expr:        intstr.FromString("(node_filesystem_size_bytes{mountpoint='/var/lib/docker'} - node_filesystem_free_bytes{mountpoint='/var/lib/docker'}) * 100 / node_filesystem_size_bytes{mountpoint='/var/lib/docker'} > 85"),
							For:         "5m",
							Labels:      getseverityLables("warning"),
							Annotations: map[string]string{"description": "More than 85% of disk used. Disk usage {{ humanize $value }} mountpoint {{ $labels.mountpoint }}%.", "summary": "LOW DISK SPACE WARING:NODE '{{ $labels.instance }}"},
						},
						{
							Alert:       "HighMemoryUsageOnNode",
							Expr:        intstr.FromString("((node_memory_MemTotal_bytes - node_memory_MemAvailable_bytes) / node_memory_MemTotal_bytes) * 100 > 80"),
							For:         "5m",
							Labels:      getseverityLables("warning"),
							Annotations: map[string]string{"description": "{{ $labels.instance }} is using a LOT of MEMORY. MEMORY usage is over {{ humanize $value}}%.", "summary": "HIGH MEMORY USAGE WARNING TASK ON '{{ $labels.instance }}'"},
						},
					},
				},
			},
		},
	}
}
