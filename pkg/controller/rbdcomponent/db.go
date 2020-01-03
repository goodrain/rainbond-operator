package rbdcomponent

import (
	rainbondv1alpha1 "github.com/GLYASAI/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	"github.com/GLYASAI/rainbond-operator/pkg/util/commonutil"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var rbdDBName = "rbd-db"

func statefulsetForRainbondDB(r *rainbondv1alpha1.RbdComponent) interface{} {
	labels := labelsForRbdComponent(rbdDBName) // TODO: only on rainbond
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
							Image:           "rainbond/rbd-db:5.0",
							ImagePullPolicy: corev1.PullIfNotPresent, // TODO: custom
							Env: []corev1.EnvVar{
								{
									Name:  "MYSQL_ROOT_PASSWORD",
									Value: "rainbond",
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

func serviceForDB(rc *rainbondv1alpha1.RbdComponent) interface{} {
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rbd-db",
			Namespace: rc.Namespace,
			Labels: map[string]string{
				"name": "rbd-db",
			},
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name: "main",
					Port: 3306,
				},
			},
			Selector: map[string]string{
				"name": "rbd-db",
			},
		},
	}

	return svc
}
