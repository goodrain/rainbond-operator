package rainbond

import (
	rainbondv1alpha1 "github.com/GLYASAI/rainbond-operator/pkg/apis/rainbond/v1alpha1"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var rbdAPIName = "rbd-api"

func daemonSetForRainbondAPI(r *rainbondv1alpha1.Rainbond) interface{} {
	labels := labelsForRainbond(rbdAPIName) // TODO: only on rainbond
	ds := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rbdAPIName,
			Namespace: r.Namespace, // TODO: can use custom namespace?
			Labels:    labels,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:   rbdAPIName,
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
							Name:            rbdAPIName,
							Image:           "rainbond/rbd-api:" + r.Spec.Version,
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
									Name:  "EX_DOMAIN",
									Value: "foobar.grapps.cn", // TODO: huangrh
								},
							},
							Args: []string{ // TODO: huangrh
								"--api-addr-ssl=0.0.0.0:8443",
								"--api-addr=$(POD_IP):8888",
								"--log-level=debug",
								"--mysql=root:rainbond@tcp(rbd-db-mysql.rbd-system.svc.cluster.local:3306)/region",
								"--api-ssl-enable=true",
								"--api-ssl-certfile=/etc/goodrain/region.goodrain.me/ssl/server.pem",
								"--api-ssl-keyfile=/etc/goodrain/region.goodrain.me/ssl/server.key.pem",
								"--client-ca-file=/etc/goodrain/region.goodrain.me/ssl/ca.pem",
								"--etcd=http://rbd-etcd.rbd-system.svc.cluster.local:2379",
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "ssl",
									MountPath: "/etc/goodrain/region.goodrain.me/ssl",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "ssl",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: "rbd-api-ssl", // TODO: check this secret before create rbd-api
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
