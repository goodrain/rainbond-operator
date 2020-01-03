package rbdcomponent

import (
	rainbondv1alpha1 "github.com/GLYASAI/rainbond-operator/pkg/apis/rainbond/v1alpha1"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var mqName = "rbd-mq"

func daemonSetForRainbondMQ(r *rainbondv1alpha1.RbdComponent) interface{} {
	labels := labelsForRbdComponent(mqName) // TODO: only on rainbond
	ds := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mqName,
			Namespace: r.Namespace, // TODO: can use custom namespace?
			Labels:    labels,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:   mqName,
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            mqName,
							Image:           "rainbond/rbd-mq:" + r.Spec.Version,
							ImagePullPolicy: corev1.PullIfNotPresent, // TODO: custom
							Env: []corev1.EnvVar{
								{
									Name: "POD_IP",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "status.podIP",
										},
									},
								},
							},
							Args: []string{ // TODO: huangrh
								"--log-level=info",
								"--etcd-endpoints=http://rbd-etcd.rbd-system.svc.cluster.local:2379",
								"--hostIP=$(HOST_IP)",
							},
						},
					},
				},
			},
		},
	}

	return ds
}
