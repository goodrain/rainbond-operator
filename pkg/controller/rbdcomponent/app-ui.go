package rbdcomponent

import (
	rainbondv1alpha1 "github.com/GLYASAI/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	"github.com/GLYASAI/rainbond-operator/pkg/util/commonutil"
	extensions "k8s.io/api/extensions/v1beta1"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var rbdAppUIName = "rbd-app-ui"

func resourcesForAppUI(r *rainbondv1alpha1.RbdComponent) []interface{} {
	return []interface{}{
		deploymentForAppUI(r),
		serviceForAppUI(r),
		ingressForAppUI(r),
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
							Image:           r.Spec.Image,
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
								{
									Name:  "REGION_URL",
									Value: "http://region.goodrain.me",
								},
								{
									Name:  "REGION_WS_URL",
									Value: "ws://region.goodrain.me",
								},
								{
									Name:  "REGION_HTTP_DOMAIN",
									Value: "foo.bar.com",
								},
								{
									Name:  "REGION_TCP_DOMAIN",
									Value: "172.20.0.11",
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "ssl",
									MountPath: "/app/region/ssl",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "ssl",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: "rbd-api-ssl",
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

func serviceForAppUI(r *rainbondv1alpha1.RbdComponent) interface{} {
	labels := r.Labels()

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rbdAppUIName,
			Namespace: r.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name: "http",
					Port: 7070,
					TargetPort: intstr.IntOrString{
						IntVal: 7070,
					},
				},
			},
			Selector: labels,
		},
	}

	return svc
}

func ingressForAppUI(r *rainbondv1alpha1.RbdComponent) interface{} {
	labels := r.Labels()
	ing := &extensions.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rbdAppUIName,
			Namespace: r.Namespace,
			Annotations: map[string]string{
				"nginx.ingress.kubernetes.io/l4-enable":             "true",
				"nginx.ingress.kubernetes.io/l4-host":               "0.0.0.0",
				"nginx.ingress.kubernetes.io/l4-port":               "17070",
				"nginx.ingress.kubernetes.io/set-header-Connection": "\"Upgrade\"",
				"nginx.ingress.kubernetes.io/set-header-Upgrade":    "$http_upgrade",
			},
			Labels: labels,
		},
		Spec: extensions.IngressSpec{
			Backend: &extensions.IngressBackend{
				ServiceName: rbdAppUIName,
				ServicePort: intstr.FromString("http"),
			},
		},
	}
	return ing
}
