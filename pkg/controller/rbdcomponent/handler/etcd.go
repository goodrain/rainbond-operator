package handler

import (
	"context"
	"github.com/GLYASAI/rainbond-operator/pkg/util/commonutil"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	rainbondv1alpha1 "github.com/GLYASAI/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	"github.com/GLYASAI/rainbond-operator/pkg/util/k8sutil"
)

var EtcdName = "rbd-etcd"

type etcd struct {
	component *rainbondv1alpha1.RbdComponent
	cluster   *rainbondv1alpha1.RainbondCluster
}

func NewETCD(ctx context.Context, client client.Client, component *rainbondv1alpha1.RbdComponent, cluster *rainbondv1alpha1.RainbondCluster) ComponentHandler {
	return &etcd{
		component: component,
		cluster:   cluster,
	}
}

func (e *etcd) Before() error {
	// No prerequisites, if no gateway-installed node is specified, install on all nodes that meet the conditions
	return nil
}

func (e *etcd) Resources() []interface{} {
	return []interface{}{
		e.podForEtcd0(),
		e.serviceForEtcd0(),
	}
}

func (e *etcd) After() error {
	return nil
}

func (e *etcd) podForEtcd0() interface{} {
	po := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "etcd0",
			Namespace: e.component.Namespace,
			Labels: map[string]string{
				"app":       "etcd",
				"etcd_node": "etcd0",
			},
		},
		Spec: corev1.PodSpec{
			TerminationGracePeriodSeconds: commonutil.Int64(0),
			Containers: []corev1.Container{
				{
					Name:            "etcd0",
					Image:           e.component.Spec.Image,
					ImagePullPolicy: e.component.ImagePullPolicy(),
					Command: []string{
						"/usr/local/bin/etcd",
						"--name",
						"etcd0",
						"--initial-advertise-peer-urls",
						"http://etcd0:2380",
						"--listen-peer-urls",
						"http://0.0.0.0:2380",
						"--listen-client-urls",
						"http://0.0.0.0:2379",
						"--advertise-client-urls",
						"http://etcd0:2379",
						"--initial-cluster",
						"etcd0=http://etcd0:2380",
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
	}

	return po
}

func (e *etcd) serviceForEtcd0() interface{} {
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "etcd0",
			Namespace: e.component.Namespace,
			Labels: map[string]string{
				"etcd_node": "etcd0",
			},
		},
		Spec: corev1.ServiceSpec{
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
			Selector: map[string]string{
				"etcd_node": "etcd0",
			},
		},
	}

	return svc
}
