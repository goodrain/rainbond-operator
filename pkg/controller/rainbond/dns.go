package rainbond

import (
	rainbondv1alpha1 "github.com/GLYASAI/rainbond-operator/pkg/apis/rainbond/v1alpha1"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var rbdDNSName = "rbd-dns"

func daemonSetForRainbondDNS(r *rainbondv1alpha1.Rainbond) interface{} {
	labels := labelsForRainbond(rbdDNSName) // TODO: only on rainbond
	ds := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rbdDNSName,
			Namespace: r.Namespace, // TODO: can use custom namespace?
			Labels:    labels,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:   rbdDNSName,
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
							Name:            rbdDNSName,
							Image:           "rainbond/rbd-dns",
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
								"--kubecfg-file=/opt/rainbond/etc/kubernetes/kubecfg/admin.kubeconfig",
								"--v=2",
								"--healthz-port=8089",
								"--dns-bind-address=$(HOST_IP)",
								"--nameservers=202.106.0.22,1.2.4.8",
								"--recoders=goodrain.me=192.168.2.63,*.goodrain.me=192.168.2.63,rainbond.kubernetes.apiserver=192.168.2.63",
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "kubecfg",
									MountPath: "/opt/rainbond/etc/kubernetes/kubecfg",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "kubecfg",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: "kubecfg",
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
