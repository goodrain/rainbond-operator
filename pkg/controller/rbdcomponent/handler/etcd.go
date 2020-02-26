package handler

import (
	"context"
	"fmt"
	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	"github.com/goodrain/rainbond-operator/pkg/util/commonutil"
	"github.com/goodrain/rainbond-operator/pkg/util/k8sutil"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// EtcdName name for rbd-etcd.
var EtcdName = "rbd-etcd"

type etcd struct {
	component *rainbondv1alpha1.RbdComponent
	cluster   *rainbondv1alpha1.RainbondCluster
	labels    map[string]string
}

// NewETCD creates a new rbd-etcd handler.
func NewETCD(ctx context.Context, client client.Client, component *rainbondv1alpha1.RbdComponent, cluster *rainbondv1alpha1.RainbondCluster) ComponentHandler {
	labels := LabelsForRainbondComponent(component)
	labels["etcd_node"] = EtcdName
	return &etcd{
		component: component,
		cluster:   cluster,
		labels:    labels,
	}
}

func (e *etcd) Before() error {
	if e.cluster.Spec.EtcdConfig != nil {
		return NewIgnoreError(fmt.Sprintf("specified etcd configuration"))
	}

	return nil
}

func (e *etcd) Resources() []interface{} {
	return []interface{}{
		e.statefulsetForEtcd(),
		e.serviceForEtcd(),
	}
}

func (e *etcd) After() error {
	return nil
}

func (e *etcd) statefulsetForEtcd() interface{} {
	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      EtcdName,
			Namespace: e.component.Namespace,
			Labels:    e.labels,
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas:    commonutil.Int32(1),
			ServiceName: EtcdName,
			Selector: &metav1.LabelSelector{
				MatchLabels: e.labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:      EtcdName,
					Namespace: e.component.Namespace,
					Labels:    e.labels,
				},
				Spec: corev1.PodSpec{
					TerminationGracePeriodSeconds: commonutil.Int64(0),
					NodeSelector:                  e.cluster.Status.FirstMasterNodeLabel(),
					Tolerations: []corev1.Toleration{
						{
							Key:    e.cluster.Status.MasterRoleLabel,
							Effect: corev1.TaintEffectNoSchedule,
						},
					},
					Containers: []corev1.Container{
						{
							Name:            EtcdName,
							Image:           e.component.Spec.Image,
							ImagePullPolicy: e.component.ImagePullPolicy(),
							Command: []string{
								"/usr/local/bin/etcd",
								"--name",
								EtcdName,
								"--initial-advertise-peer-urls",
								fmt.Sprintf("http://%s:2380", EtcdName),
								"--listen-peer-urls",
								"http://0.0.0.0:2380",
								"--listen-client-urls",
								"http://0.0.0.0:2379",
								"--advertise-client-urls",
								fmt.Sprintf("http://%s:2379", EtcdName),
								"--initial-cluster",
								fmt.Sprintf("%s=http://%s:2380", EtcdName, EtcdName),
								"--initial-cluster-state",
								"new",
							},
							Ports: []corev1.ContainerPort{
								{
									Name:          "client",
									ContainerPort: 2379,
								},
								{
									Name:          "server",
									ContainerPort: 2380,
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "etcd-data",
									MountPath: "/var/lib/etcd",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "etcd-data",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/opt/rainbond/data/etcd",
									Type: k8sutil.HostPath(corev1.HostPathDirectoryOrCreate),
								},
							},
						},
					},
				},
			},
		},
	}
	return sts
}

func (e *etcd) serviceForEtcd() interface{} {
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      EtcdName,
			Namespace: e.component.Namespace,
			Labels:    e.labels,
		},
		Spec: corev1.ServiceSpec{
			ClusterIP: "None",
			Ports: []corev1.ServicePort{
				{
					Name: "client",
					Port: 2379,
				},
				{
					Name: "server",
					Port: 2380,
				},
			},
			Selector: e.labels,
		},
	}

	return svc
}
