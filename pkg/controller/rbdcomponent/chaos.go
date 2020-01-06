package rbdcomponent

import (
	rainbondv1alpha1 "github.com/GLYASAI/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	"github.com/GLYASAI/rainbond-operator/pkg/util/k8sutil"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var rbdChaosName = "rbd-chaos"

func resourcesForChaos(r *rainbondv1alpha1.RbdComponent) []interface{} {
	return []interface{}{
		daemonSetForChaos(r),
	}
}

func daemonSetForChaos(r *rainbondv1alpha1.RbdComponent) interface{} {
	labels := r.Labels()

	ds := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rbdChaosName,
			Namespace: r.Namespace,
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
							Image:           "goodrain.me/rbd-chaos:" + r.Spec.Version,
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
								"--etcd-endpoints=http://etcd0:2379",
								"--hostIP=$(POD_IP)",
								"--log-level=debug",
								"--mysql=root:rainbond@tcp(rbd-db:3306)/region",
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "grdata",
									MountPath: "/grdata",
								},
								{
									Name:      "dockersock",
									MountPath: "/var/run/docker.sock",
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
							Name: "dockersock",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/var/run/docker.sock",
									Type: k8sutil.HostPath(corev1.HostPathFile),
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
