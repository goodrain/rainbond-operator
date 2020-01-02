package rbdcomponent

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	rainbondv1alpha1 "github.com/GLYASAI/rainbond-operator/pkg/apis/rainbond/v1alpha1"
)

var rbdEtcdName = "rbd-etcd"

func podForEtcd0(rc *rainbondv1alpha1.RbdComponent) interface{} {
	ds := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "etcd0",
			Namespace: rc.Namespace, // TODO: can use custom namespace?
			Labels: map[string]string{
				"app":       "etcd",
				"etcd_node": "etcd0",
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:            "etcd0",
					Image:           "quay.io/coreos/etcd:latest",
					ImagePullPolicy: corev1.PullIfNotPresent,
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
				},
			},
		},
	}

	return ds
}

func serviceForEtcd0(rc *rainbondv1alpha1.RbdComponent) interface{} {
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "etcd0",
			Namespace: rc.Namespace,
			Labels: map[string]string{
				"etcd_node": "etcd0",
			}, // TODO
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
			}, // TODO
		},
	}

	return svc
}
