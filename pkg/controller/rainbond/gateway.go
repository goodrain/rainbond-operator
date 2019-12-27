package rainbond

import (
	rainbondv1alpha1 "github.com/GLYASAI/rainbond-operator/pkg/apis/rainbond/v1alpha1"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var gatewayName = "rbd-gateway"

func daemonSetForRainbondGateway(r *rainbondv1alpha1.Rainbond) *appsv1.DaemonSet {
	labels := labelsForRainbond(gatewayName) // TODO: only on rainbond
	ds := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      gatewayName,
			Namespace: r.Namespace, // TODO: can use custom namespace?
			Labels:    labels,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:   gatewayName,
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
					NodeSelector: map[string]string{
						"node-role.kubernetes.io/master": "",  // TODO: not serious
					},
					Containers: []corev1.Container{
						{
							Name:            gatewayName,
							Image:           "rainbond/rbd-gateway:" + r.Spec.Version,
							ImagePullPolicy: corev1.PullIfNotPresent, // TODO: custom
							Args: []string{ // TODO: huangrh
								"--log-level=debug",
								"--error-log=/dev/stderr error",
								"--enable-kubeapi=false",
								"--etcd-endpoints=http://rbd-etcd.rbd-system.svc.cluster.local:2379",
								"--enable-lang-grme=false",
								"--enable-mvn-grme=false",
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
									SecretName: "kubecfg",  // TODO: check before creating
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
