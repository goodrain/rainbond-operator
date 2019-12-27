package rainbond

import (
	rainbondv1alpha1 "github.com/GLYASAI/rainbond-operator/pkg/apis/rainbond/v1alpha1"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var monitorName = "rbd-monitor"

func daemonSetForRainbondMonitor(r *rainbondv1alpha1.Rainbond) interface{} {
	labels := labelsForRainbond(monitorName) // TODO: only on rainbond
	ds := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      monitorName,
			Namespace: r.Namespace, // TODO: can use custom namespace?
			Labels:    labels,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:   monitorName,
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            monitorName,
							Image:           "rainbond/rbd-monitor:" + r.Spec.Version,
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
								"--etcd-endpoints=http://rbd-etcd.rbd-system.svc.cluster.local:2379",
								"--advertise-addr=$(POD_IP):9999",
								"--alertmanager-address=$(POD_IP):9093",
								"--web.listen-address=$(POD_IP):9999",
								"--storage.tsdb.path=/prometheusdata",
								"--storage.tsdb.no-lockfile",
								"--storage.tsdb.retention=7d",
								"--log.level=debug",
							},
						},
					},
				},
			},
		},
	}

	return ds
}
