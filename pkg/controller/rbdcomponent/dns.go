package rbdcomponent

import (
	rainbondv1alpha1 "github.com/GLYASAI/rainbond-operator/pkg/apis/rainbond/v1alpha1"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var rbdDNSName = "rbd-dns"

func resourcesForDNS(r *rainbondv1alpha1.RbdComponent) []interface{} {
	return []interface{}{
		daemonSetForDNS(r),
	}
}

func daemonSetForDNS(r *rainbondv1alpha1.RbdComponent) interface{} {
	labels := r.Labels()
	ds := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rbdDNSName,
			Namespace: r.Namespace,
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
							Image:           "goodrain.me/rbd-dns:" + r.Spec.Version, // TODO: indicate image directly
							ImagePullPolicy: corev1.PullIfNotPresent,                 // TODO: custom
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
									Name: "HOST_IP",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "status.hostIP",
										},
									},
								},
								{
									Name:  "EX_DOMAIN",
									Value: "foobar.grapps.cn", // TODO: huangrh
								},
							},
							Args: []string{ // TODO: huangrh
								"--v=2",
								"--healthz-port=8089",
								"--dns-bind-address=$(POD_IP)",
								"--nameservers=202.106.0.22,1.2.4.8",
								"--recoders=goodrain.me=$(HOST_IP),*.goodrain.me=$(HOST_IP),rainbond.kubernetes.apiserver=$(HOST_IP)",
							},
						},
					},
				},
			},
		},
	}

	return ds
}
