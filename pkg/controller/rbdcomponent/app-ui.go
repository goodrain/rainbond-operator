package rbdcomponent

import (
	rainbondv1alpha1 "github.com/GLYASAI/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	"github.com/GLYASAI/rainbond-operator/pkg/util/commonutil"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var rbdAppUIName = "rbd-app-ui"

func resourcesForAppUI(r *rainbondv1alpha1.RbdComponent) []interface{} {
	return []interface{}{
		deploymentForAppUI(r),
	}
}

func deploymentForAppUI(r *rainbondv1alpha1.RbdComponent) interface{} {
	labels := r.Labels()

	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rbdAppUIName,
			Namespace: r.Namespace,
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
							Image:           "goodrain.me/rbd-app-ui:" + r.Spec.Version,
							ImagePullPolicy: corev1.PullIfNotPresent, // TODO: custom
							Env: []corev1.EnvVar{
								{
									Name:  "MYSQL_HOST",
									Value: "rbd-db",
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
