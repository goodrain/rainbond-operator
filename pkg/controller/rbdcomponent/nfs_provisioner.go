package rbdcomponent

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	rainbondv1alpha1 "github.com/GLYASAI/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	"github.com/GLYASAI/rainbond-operator/pkg/util/commonutil"
	"github.com/GLYASAI/rainbond-operator/pkg/util/k8sutil"
)

var rbdNFSProvisionerName = "rbd-nfs-provisioner"

func resourcesForNFSProvisioner(r *rainbondv1alpha1.RbdComponent) []interface{} {
	return []interface{}{
		statefulsetForNFSProvisioner(r),
		serviceForNFSProvisioner(r),
	}
}

func statefulsetForNFSProvisioner(r *rainbondv1alpha1.RbdComponent) interface{} {
	labels := r.Labels()

	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rbdNFSProvisionerName,
			Namespace: r.Namespace, // TODO: can use custom namespace?
			Labels:    labels,
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas:    commonutil.Int32(1),
			ServiceName: rbdNFSProvisionerName,
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
							Image:           r.Spec.Image,
							ImagePullPolicy: corev1.PullIfNotPresent, // TODO: custom
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
									MountPath: "/export",
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
									Type: k8sutil.HostPath(corev1.HostPathDirectoryOrCreate),
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
	labels := rc.Labels()

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
					Port: 2049,
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

func storageClassForNFSProvisioner() *storagev1.StorageClass {
	sc := &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: "rbd-nfs",
		},
		Provisioner: "rainbond.io/nfs",
		MountOptions: []string{
			"vers=4.1",
		},
	}

	return sc
}
