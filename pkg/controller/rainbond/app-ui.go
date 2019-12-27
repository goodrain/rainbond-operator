package rainbond

import (
	rainbondv1alpha1 "github.com/GLYASAI/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	"github.com/GLYASAI/rainbond-operator/pkg/util/commonutil"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var rbdAppUIName = "rbd-app-ui"

func deploymentForRainbondAppUI(r *rainbondv1alpha1.Rainbond) interface{} {
	labels := labelsForRainbond(rbdAppUIName) // TODO: only on rainbond
	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rbdAppUIName,
			Namespace: r.Namespace, // TODO: can use custom namespace?
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: commonutil.Int32(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:   rbdAppUIName,
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            rbdAppUIName,
							Image:           "rainbond/rbd-app-ui:" + r.Spec.Version,
							ImagePullPolicy: corev1.PullIfNotPresent, // TODO: custom
							Env: []corev1.EnvVar{
								{
									Name:  "MYSQL_HOST",
									Value: "rbd-db-mysql.rbd-system.svc.cluster.local",
								},
								{
									Name:  "MYSQL_PORT",
									Value: "3306",
								},
								{
									Name:  "MYSQL_USER",
									Value: "root",
								},
								{
									Name:  "MYSQL_PASS",
									Value: "rainbond",
								},
								{
									Name:  "MYSQL_DB",
									Value: "console",
								},
							},
						},
					},
				},
			},
		},
	}

	return deploy
}
