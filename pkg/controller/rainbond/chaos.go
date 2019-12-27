package rainbond

import (
	rainbondv1alpha1 "github.com/GLYASAI/rainbond-operator/pkg/apis/rainbond/v1alpha1"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var rbdChaosName = "rbd-chaos"

func daemonSetForRainbondChaos(r *rainbondv1alpha1.Rainbond) interface{} {
	labels := labelsForRainbond(rbdChaosName) // TODO: only on rainbond
	ds := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rbdChaosName,
			Namespace: r.Namespace, // TODO: can use custom namespace?
			Labels:    labels,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:   rbdChaosName,
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					HostNetwork: true,
					DNSPolicy:   corev1.DNSClusterFirstWithHostNet,
					Tolerations: []corev1.Toleration{
						{
							Key:    "node-role.kubernetes.io/master",
							Effect: corev1.TaintEffectNoSchedule,
						},
					},
					Containers: []corev1.Container{
						{
							Name:            rbdChaosName,
							Image:           "rainbond/rbd-chaos:" + r.Spec.Version,
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
								{
									Name:  "SOURCE_DIR",
									Value: "/cache/source",
								},
								{
									Name:  "CACHE_DIR",
									Value: "/cache",
								},
							},
							Args: []string{ // TODO: huangrh
								"--etcd-endpoints=http://rbd-etcd.rbd-system.svc.cluster.local:2379",
								"--hostIP=$(POD_IP)",
								"--log-level=debug",
								"--mysql=root:rainbond@tcp(rbd-db-mysql.rbd-system.svc.cluster.local:3306)/region",
							},

							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "grdata",
									MountPath: "/grdata",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "grdata",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: "grdata",
								},
							},
						},
					},
				},
			},
		},
	}

	return ds
}
