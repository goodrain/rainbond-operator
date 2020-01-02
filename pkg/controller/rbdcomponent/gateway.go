package rbdcomponent

import (
	rainbondv1alpha1 "github.com/GLYASAI/rainbond-operator/pkg/apis/rainbond/v1alpha1"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var gatewayName = "rbd-gateway"

func daemonSetForGateway(r *rainbondv1alpha1.RbdComponent) interface{} {
	labels := labelsForRbdComponent(gatewayName) // TODO: only on rainbond
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
					ServiceAccountName: "rainbond-operator",
					HostNetwork:        true,
					DNSPolicy:          corev1.DNSClusterFirstWithHostNet,
					Tolerations: []corev1.Toleration{
						{
							Key:    "node-role.kubernetes.io/master",
							Effect: corev1.TaintEffectNoSchedule,
						},
					},
					NodeSelector: map[string]string{
						"node-role.kubernetes.io/master": "", // TODO: not serious
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
								"--etcd-endpoints=http://etcd0.rbd-system.svc.cluster.local:2379",
								"--enable-lang-grme=false",
								"--enable-mvn-grme=false",
								"--enable-rbd-endpoints=false",
							},
						},
					},
				},
			},
		},
	}

	return ds
}
