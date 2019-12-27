package rainbond

import (
	rainbondv1alpha1 "github.com/GLYASAI/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	"github.com/GLYASAI/rainbond-operator/pkg/util/commonutil"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var rbdDBName = "rbd-db"

func statefulsetForRainbondDB(r *rainbondv1alpha1.Rainbond) interface{} {
	hostPathDir := corev1.HostPathDirectory
	labels := labelsForRainbond(rbdDBName) // TODO: only on rainbond
	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rbdDBName,
			Namespace: r.Namespace, // TODO: can use custom namespace?
			Labels:    labels,
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: commonutil.Int32(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:   rbdDBName,
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            rbdDBName,
							Image:           "rainbond/rbd-db:" + r.Spec.Version,
							ImagePullPolicy: corev1.PullIfNotPresent, // TODO: custom
							Env: []corev1.EnvVar{
								{
									Name:  "MYSQL_ROOT_PASSWORD",
									Value: "rainbond",
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "data",
									MountPath: "/data",
								},
								{
									Name:      "etcmysql",
									MountPath: "/etc/mysql",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "data",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{ // TODO: use pvc
									Path: "/opt/rainbond/data/rbd-db",
									Type: &hostPathDir,
								},
							},
						},
						{
							Name: "etcmysql",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{ // TODO: use pvc
									Path: "/opt/rainbond/etc/rbd-db",
									Type: &hostPathDir,
								},
							},
						},
					},
				},
			},
		},
	}

	return sts
}
