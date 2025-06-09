package handler

import (
	"context"
	"os"

	"github.com/goodrain/rainbond-operator/util/constants"
	"github.com/goodrain/rainbond-operator/util/rbdutil"
	"k8s.io/apimachinery/pkg/api/resource"

	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/api/v1alpha1"
	"github.com/goodrain/rainbond-operator/util/commonutil"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// MonitorName name for rbd-monitor.
var MonitorName = "rbd-monitor"

type monitor struct {
	ctx        context.Context
	client     client.Client
	etcdSecret *corev1.Secret

	component *rainbondv1alpha1.RbdComponent
	cluster   *rainbondv1alpha1.RainbondCluster
	labels    map[string]string

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
		storageRequest: getStorageRequest("MONITOR_DATA_STORAGE_REQUEST", 1),
	}
}

func (m *monitor) Before() error {
	return setStorageCassName(m.ctx, m.client, m.component.Namespace, m)
}

func (m *monitor) Resources() []client.Object {
	return []client.Object{
		m.configmap(),
		m.statefulset(),
		m.serviceForMonitor(),
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

	resources = mergeResources(resources, m.component.Spec.Resources)

	vms := append(m.component.Spec.VolumeMounts, []corev1.VolumeMount{
		{
			Name:      claimName,
			MountPath: "/prometheusdata",
		},
		{
			Name:      "prom-config",
			MountPath: "/etc/prometheus/prometheus.yml",
			SubPath:   "prometheus.yml",
		},
		{
			Name:      "rules-config",
			MountPath: "/etc/prometheus/rules.yml",
			SubPath:   "rules.yml",
		},
	}...)

	vs := append(m.component.Spec.Volumes, []corev1.Volume{
		{
			Name: "prom-config",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					DefaultMode: commonutil.Int32(420),
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "prometheus-config",
					},
				},
			},
		},
		{
			Name: "rules-config",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					DefaultMode: commonutil.Int32(420),
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "prometheus-config",
					},
				},
			},
		},
	}...)

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
					TerminationGracePeriodSeconds: commonutil.Int64(0),
					ServiceAccountName:            rbdutil.GetenvDefault("SERVICE_ACCOUNT_NAME", "rainbond-operator"),
					Containers: []corev1.Container{
						{
							Image:           m.component.Spec.Image,
							ImagePullPolicy: m.component.ImagePullPolicy(),
							Name:            MonitorName,
							VolumeMounts:    vms,
							Resources:       resources,
							LivenessProbe: &corev1.Probe{
								Handler: corev1.Handler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/-/healthy",
										Port: intstr.FromInt(9090),
									},
								},
								InitialDelaySeconds: 30,
								TimeoutSeconds:      2,
								PeriodSeconds:       10,
								SuccessThreshold:    1,
								FailureThreshold:    3,
							},
							ReadinessProbe: &corev1.Probe{
								Handler: corev1.Handler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/-/healthy",
										Port: intstr.FromInt(9090),
									},
								},
								InitialDelaySeconds: 30,
								TimeoutSeconds:      2,
								PeriodSeconds:       10,
								SuccessThreshold:    1,
								FailureThreshold:    3,
							},
						},
					},
					Volumes:  vs,
					Affinity: m.component.Spec.Affinity,
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
						IntVal: 9090,
					},
				},
			},
			Selector: m.labels,
		},
	}

	return svc
}

// configmap 配置文件
func (m *monitor) configmap() client.Object {
	prometheus, _ := os.ReadFile("/config/prom/prometheus.yml")
	rules, _ := os.ReadFile("/config/prom/rules.yml")
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "prometheus-config",
			Namespace: rbdutil.GetenvDefault("RBD_NAMESPACE", constants.Namespace),
		},
		Data: map[string]string{
			"prometheus.yml": string(prometheus),
			"rules.yml":      string(rules),
		},
	}
}
