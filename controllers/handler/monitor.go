package handler

import (
	"context"

	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/goodrain/rainbond-operator/util/probeutil"

	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/api/v1alpha1"
	"github.com/goodrain/rainbond-operator/util/commonutil"
	mv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// MonitorName name for rbd-monitor.
var MonitorName = "rbd-monitor"

type monitor struct {
	ctx              context.Context
	client           client.Client
	component        *rainbondv1alpha1.RbdComponent
	cluster          *rainbondv1alpha1.RainbondCluster
	labels           map[string]string
	pvcParametersRWO *pvcParameters
	storageRequest   int64
}

var _ ComponentHandler = &monitor{}
var _ StorageClassRWOer = &monitor{}

// NewMonitor returns a new rbd-monitor handler.
func NewMonitor(ctx context.Context, client client.Client, component *rainbondv1alpha1.RbdComponent, cluster *rainbondv1alpha1.RainbondCluster) ComponentHandler {
	return &monitor{
		ctx:            ctx,
		client:         client,
		component:      component,
		cluster:        cluster,
		labels:         LabelsForRainbondComponent(component),
		storageRequest: getStorageRequest("MONITOR_DATA_STORAGE_REQUEST", 21),
	}
}

func (m *monitor) Before() error {

	if err := setStorageCassName(m.ctx, m.client, m.component.Namespace, m); err != nil {
		return err
	}

	return nil
}

func (m *monitor) Resources() []client.Object {
	return []client.Object{
		m.statefulset(),
		m.serviceForMonitor(),
		m.serviceMonitorForMonitor(),
	}
}

func (m *monitor) After() error {
	return nil
}

func (m *monitor) ListPods() ([]corev1.Pod, error) {
	return listPods(m.ctx, m.client, m.component.Namespace, m.labels)
}

func (m *monitor) SetStorageClassNameRWO(pvcParameters *pvcParameters) {
	m.pvcParametersRWO = pvcParameters
}

func (m *monitor) statefulset() client.Object {
	claimName := "data" // unnecessary
	promDataPVC := createPersistentVolumeClaimRWO(m.component.Namespace, claimName, m.pvcParametersRWO, m.labels, m.storageRequest)

	args := []string{
		"--alertmanager-address=$(POD_IP):9093",
		"--storage.tsdb.path=/prometheusdata",
		"--storage.tsdb.no-lockfile",
		"--storage.tsdb.retention=7d",
	}
	volumeMounts := []corev1.VolumeMount{
		{
			Name:      claimName,
			MountPath: "/prometheusdata",
		},
	}
	var volumes []corev1.Volume
	env := []corev1.EnvVar{
		{
			Name: "POD_IP",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "status.podIP",
				},
			},
		},
	}

	resources := corev1.ResourceRequirements{
		Limits: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("2048Mi"),
			corev1.ResourceCPU:    resource.MustParse("1000m"),
		},
		Requests: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("512Mi"),
			corev1.ResourceCPU:    resource.MustParse("200m"),
		},
	}

	env = mergeEnvs(env, m.component.Spec.Env)
	resources = mergeResources(resources, m.component.Spec.Resources)
	args = mergeArgs(args, m.component.Spec.Args)
	volumeMounts = mergeVolumeMounts(volumeMounts, m.component.Spec.VolumeMounts)
	volumes = mergeVolumes(volumes, m.component.Spec.Volumes)

	// prepare probe
	readinessProbe := probeutil.MakeReadinessProbeHTTP("", "/monitor/health", 3329)
	ds := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      MonitorName,
			Namespace: m.component.Namespace,
			Labels:    m.labels,
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: m.component.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: m.labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:   MonitorName,
					Labels: m.labels,
				},
				Spec: corev1.PodSpec{
					ImagePullSecrets:              imagePullSecrets(m.component, m.cluster),
					TerminationGracePeriodSeconds: commonutil.Int64(30),
					ServiceAccountName:            "rainbond-operator",
					Containers: []corev1.Container{
						{
							Name:            MonitorName,
							Image:           m.component.Spec.Image,
							ImagePullPolicy: m.component.ImagePullPolicy(),
							Env:             env,
							Args:            args,
							VolumeMounts:    volumeMounts,
							ReadinessProbe:  readinessProbe,
							Resources:       resources,
						},
					},
					Volumes: volumes,
				},
			},
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{*promDataPVC},
		},
	}

	return ds
}

func (m *monitor) serviceForMonitor() client.Object {
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      MonitorName,
			Namespace: m.component.Namespace,
			Labels:    m.labels,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name: "http",
					Port: 9999,
					TargetPort: intstr.IntOrString{
						IntVal: 9999,
					},
				},
			},
			Selector: m.labels,
		},
	}

	return svc
}

func (m *monitor) serviceMonitorForMonitor() client.Object {
	return &mv1.ServiceMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Name:        MonitorName,
			Namespace:   m.component.Namespace,
			Labels:      m.labels,
			Annotations: map[string]string{"ignore_controller_update": "true"},
		},
		Spec: mv1.ServiceMonitorSpec{
			NamespaceSelector: mv1.NamespaceSelector{
				MatchNames: []string{m.component.Namespace},
			},
			Selector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"name": MonitorName,
				},
			},
			Endpoints: []mv1.Endpoint{
				{
					Port:          "http",
					Path:          "/metrics",
					Interval:      "1m",
					ScrapeTimeout: "30s",
				},
			},
			JobLabel: "name",
		},
	}
}
