package rbdcomponent

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	rainbondv1alpha1 "github.com/GLYASAI/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	"github.com/GLYASAI/rainbond-operator/pkg/util/commonutil"
	"github.com/GLYASAI/rainbond-operator/pkg/util/rbduitl"
)

var rbdNFSProvisionerName = "rbd-nfs-provisioner"

func statefulsetForNFSProvisioner(p *rainbondv1alpha1.RbdComponent) interface{} {
	l := map[string]string{
		"name": rbdNFSProvisionerName,
	}
	labels := rbdutil.Labels(l).WithRainbondLabels()

	hostPathDir := corev1.HostPathDirectory

	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rbdNFSProvisionerName,
			Namespace: p.Namespace, // TODO: can use custom namespace?
			Labels:    labels,
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: commonutil.Int32(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:   rbdNFSProvisionerName,
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: "rainbond-operator", // TODO: do not hard code
					Containers: []corev1.Container{
						{
							Name:            rbdNFSProvisionerName,
							Image:           "abewang/nfs-provisioner:v2.2.1-k8s1.12", // TODO: do not hard code
							ImagePullPolicy: corev1.PullIfNotPresent,                  // TODO: custom
							Ports: []corev1.ContainerPort{
								{
									Name:          "nfs",
									ContainerPort: 2049,
								},
								{
									Name:          "mountd",
									ContainerPort: 20048,
								},
								{
									Name:          "rpcbind",
									ContainerPort: 111,
								},
								{
									Name:          "rpcbind-udp",
									ContainerPort: 111,
									Protocol:      corev1.ProtocolUDP,
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
									Name: "POD_NAMESPACE",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.namespace",
										},
									},
								},
								{
									Name:  "SERVICE_NAME",
									Value: rbdNFSProvisionerName,
								},
							},
							Args: []string{
								"-provisioner=rainbond.io/nfs",
							},
							SecurityContext: &corev1.SecurityContext{
								Capabilities: &corev1.Capabilities{
									Add: []corev1.Capability{
										"DAC_READ_SEARCH",
										"SYS_RESOURCE",
									},
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "export-volume",
									MountPath: " /export",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "export-volume",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/srv",
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

func serviceForNFSProvisioner(rc *rainbondv1alpha1.RbdComponent) interface{} {
	l := map[string]string{
		"name": rbdNFSProvisionerName,
	}
	labels := rbdutil.Labels(l).WithRainbondLabels()

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rbdNFSProvisionerName,
			Namespace: rc.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name: "name",
					Port: 2048,
				},
				{
					Name: "mountd",
					Port: 20048,
				},
				{
					Name: "rpcbind",
					Port: 111,
				},
				{
					Name:     "rpcbind-udp",
					Port:     111,
					Protocol: corev1.ProtocolUDP,
				},
			},
			Selector: labels,
		},
	}

	return svc
}
