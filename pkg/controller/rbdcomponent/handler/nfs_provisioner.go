package handler

import (
	"context"
	rainbondv1alpha1 "github.com/GLYASAI/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	"github.com/GLYASAI/rainbond-operator/pkg/util/commonutil"
	"github.com/GLYASAI/rainbond-operator/pkg/util/constants"
	"github.com/GLYASAI/rainbond-operator/pkg/util/k8sutil"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var NFSName = constants.DefStorageClass
var nfsProvisionerName = "rainbond.io/nfs"

type nfsProvisioner struct {
	component *rainbondv1alpha1.RbdComponent
	cluster   *rainbondv1alpha1.RainbondCluster
}

func NewNFSProvisioner(ctx context.Context, client client.Client, component *rainbondv1alpha1.RbdComponent, cluster *rainbondv1alpha1.RainbondCluster) ComponentHandler {
	return &nfsProvisioner{
		component: component,
		cluster:   cluster,
	}
}

func (n *nfsProvisioner) Before() error {
	// No prerequisites, if no gateway-installed node is specified, install on all nodes that meet the conditions
	return nil
}

func (n *nfsProvisioner) Resources() []interface{} {
	return []interface{}{
		n.statefulsetForNFSProvisioner(),
		n.serviceForNFSProvisioner(),
	}
}

func (n *nfsProvisioner) After() error {
	return nil
}

func (n *nfsProvisioner) statefulsetForNFSProvisioner() interface{} {
	labels := n.component.Labels()
	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      NFSName,
			Namespace: n.component.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas:    commonutil.Int32(1),
			ServiceName: NFSName,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:   NFSName,
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: "rainbond-operator", // TODO: do not hard code, get sa from configuration.
					Containers: []corev1.Container{
						{
							Name:            NFSName,
							Image:           n.component.Spec.Image,
							ImagePullPolicy: n.component.ImagePullPolicy(),
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
									Value: NFSName,
								},
							},
							Args: []string{
								"-provisioner=" + nfsProvisionerName,
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
									Path: "/opt/rainbond/data/nfs",
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

func (n *nfsProvisioner) serviceForNFSProvisioner() interface{} {
	labels := n.component.Labels()

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      NFSName,
			Namespace: n.component.Namespace,
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
