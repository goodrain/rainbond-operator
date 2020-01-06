package rbdcomponent

import (
	rainbondv1alpha1 "github.com/GLYASAI/rainbond-operator/pkg/apis/rainbond/v1alpha1"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var rbdWorkerName = "rbd-worker"

func resourcesForWorker(r *rainbondv1alpha1.RbdComponent) []interface{} {
	return []interface{}{
		daemonSetForWorker(r),
	}
}

func daemonSetForWorker(r *rainbondv1alpha1.RbdComponent) interface{} {
	labels := r.Labels()
	ds := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rbdWorkerName,
			Namespace: r.Namespace, // TODO: can use custom namespace?
			Labels:    labels,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:   rbdWorkerName,
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            rbdWorkerName,
							Image:           "goodrain.me/rbd-worker:" + r.Spec.Version,
							ImagePullPolicy: corev1.PullIfNotPresent, // TODO: custom
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "kubecfg",
									MountPath: "/opt/rainbond/etc/kubernetes/kubecfg",
								},
							},
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
							},
							Args: []string{
								"--log-level=debug",
								"--host-ip=$(POD_IP)",
								"--etcd-endpoints=http://etcd0:2379",
								"--node-name=$(HOST_IP)",
								"--kube-config=/opt/rainbond/etc/kubernetes/kubecfg/admin.kubeconfig",
								"--mysql=root:rainbond@tcp(rbd-db:3306)/region",
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
