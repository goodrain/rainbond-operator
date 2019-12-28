package rainbond

import (
	rainbondv1alpha1 "github.com/GLYASAI/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	"github.com/GLYASAI/rainbond-operator/pkg/util/commonutil"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var metricsServerName = "metrics-server"

func deploymentForMetricsServer(r *rainbondv1alpha1.Rainbond) interface{} {
	labels := labelsForRainbond(metricsServerName) // TODO: only on rainbond
	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      metricsServerName,
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
					Name:   metricsServerName,
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            metricsServerName,
							Image:           "rainbond/metrics-server:v0.3.6",
							ImagePullPolicy: corev1.PullIfNotPresent, // TODO: custom
							Ports: []corev1.ContainerPort{
								{
									Name:          "main-port",
									ContainerPort: 4443,
								},
							},
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
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "kubecfg",
									MountPath: "/opt/rainbond/etc/kubernetes/kubecfg",
								},
								{
									Name:      "tmp-dir",
									MountPath: "/tmp",
								},
							},
							Args: []string{
								"--secure-port=4443",
								"--kubelet-preferred-address-types=InternalIP",
								"--kubelet-insecure-tls",
								"--cert-dir=/tmp",
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
						{
							Name: "tmp-dir",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
					},
				},
			},
		},
	}

	return deploy
}
