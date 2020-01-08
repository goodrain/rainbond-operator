package rbdcomponent

import (
	rainbondv1alpha1 "github.com/GLYASAI/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	"github.com/GLYASAI/rainbond-operator/pkg/util/k8sutil"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var rbdNodeName = "rbd-node"

func resourcesForNode(r *rainbondv1alpha1.RbdComponent) []interface{} {
	return []interface{}{
		daemonSetForRainbondNode(r),
	}
}

func daemonSetForRainbondNode(r *rainbondv1alpha1.RbdComponent) interface{} {
	labels := r.Labels()
	ds := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rbdNodeName,
			Namespace: r.Namespace, // TODO: can use custom namespace?
			Labels:    labels,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:   rbdNodeName,
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: "rainbond-operator",
					HostNetwork:        true,
					HostPID:            true,
					DNSPolicy:          corev1.DNSClusterFirstWithHostNet,
					Tolerations: []corev1.Toleration{
						{
							Key:    "node-role.kubernetes.io/master",
							Effect: corev1.TaintEffectNoSchedule,
						},
					},
					Containers: []corev1.Container{
						{
							Name:            rbdNodeName,
							Image:           r.Spec.Image,
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
									Name: "NODE_NAME",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "spec.nodeName",
										},
									},
								},
								{
									Name:  "RBD_DOCKER_SECRET",
									Value: "rbd-hub-goodrain.me",
								},
							},
							Args: []string{ // TODO: huangrh
								"--log-level=debug",
								"--etcd=http://etcd0:2379",
								"--hostIP=$(POD_IP)",
								"--run-mode master",
								"--noderule manage",
								"--nodeid=$(NODE_NAME)",
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "grdata",
									MountPath: "/grdata",
								},
								{
									Name:      "proc",
									MountPath: "/proc",
								},
								{
									Name:      "sys",
									MountPath: "/sys",
								},
								{
									Name:      "docker",
									MountPath: "/etc/docker",
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
						{
							Name: "proc",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/proc",
									Type: k8sutil.HostPath(corev1.HostPathDirectory),
								},
							},
						},
						{
							Name: "sys",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/sys",
									Type: k8sutil.HostPath(corev1.HostPathDirectory),
								},
							},
						},
						{
							Name: "docker",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/etc/docker",
									Type: k8sutil.HostPath(corev1.HostPathDirectory),
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
