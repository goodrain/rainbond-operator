package rbdcomponent

import (
	rainbondv1alpha1 "github.com/GLYASAI/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	"github.com/GLYASAI/rainbond-operator/pkg/util/k8sutil"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var rbdRepoName = "rbd-repo"

func resourcesForRepo(r *rainbondv1alpha1.RbdComponent) []interface{} {
	return []interface{}{
		daemonSetForRepo(r),
		serviceForRepo(r),
	}
}

func daemonSetForRepo(r *rainbondv1alpha1.RbdComponent) interface{} {
	labels := r.Labels()

	ds := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rbdRepoName,
			Namespace: r.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:   rbdRepoName,
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Tolerations: []corev1.Toleration{
						{
							Key:    "node-role.kubernetes.io/master",
							Effect: corev1.TaintEffectNoSchedule,
						},
					},
					NodeSelector: map[string]string{
						"node-role.kubernetes.io/master": "",
					},
					Containers: []corev1.Container{
						{
							Name:            rbdRepoName,
							Image:           r.Spec.Image,
							ImagePullPolicy: corev1.PullIfNotPresent,
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "repo-data",
									MountPath: "/var/opt/jfrog/artifactory",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "repo-data",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/opt/rainbond/data/repo",
									Type: k8sutil.HostPath(corev1.HostPathDirectoryOrCreate),
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

func serviceForRepo(r *rainbondv1alpha1.RbdComponent) interface{} {
	labels := r.Labels()
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rbdRepoName,
			Namespace: r.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name: "http",
					Port: 80,
				},
			},
			Selector: labels,
		},
	}
	return svc
}
