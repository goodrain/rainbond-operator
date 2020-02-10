package handler

import (
	"context"

	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	"github.com/goodrain/rainbond-operator/pkg/util/commonutil"
	"github.com/goodrain/rainbond-operator/pkg/util/constants"
	"github.com/goodrain/rainbond-operator/pkg/util/k8sutil"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

//NFSName nfs provider name
var NFSName = constants.DefStorageClass
var nfsProvisionerName = "rainbond.io/nfs"

type nfsProvisioner struct {
	ctx       context.Context
	client    client.Client
	component *rainbondv1alpha1.RbdComponent
	cluster   *rainbondv1alpha1.RainbondCluster
	pkg       *rainbondv1alpha1.RainbondPackage
}

//NewNFSProvisioner new nfs provider
func NewNFSProvisioner(ctx context.Context, client client.Client, component *rainbondv1alpha1.RbdComponent, cluster *rainbondv1alpha1.RainbondCluster, pkg *rainbondv1alpha1.RainbondPackage) ComponentHandler {
	return &nfsProvisioner{
		ctx:       ctx,
		client:    client,
		component: component,
		cluster:   cluster,
		pkg:       pkg,
	}
}

func (n *nfsProvisioner) Before() error {
	withPackage := n.cluster.Spec.InstallMode == rainbondv1alpha1.InstallationModeWithPackage
	if withPackage {
		// in InstallationModeWithPackage mode, no need to wait until rainbondpackage is completed.
		return nil
	}
	// in InstallationModeWithoutPackage mode, we have to make sure rainbondpackage is completed before we create the resource.
	return checkPackageStatus(n.pkg)
}

func (n *nfsProvisioner) Resources() []interface{} {
	return []interface{}{
		n.statefulsetForNFSProvisioner(),
		n.serviceForNFSProvisioner(),
	}
}

func (n *nfsProvisioner) After() error {
	class := n.storageClassForNFSProvisioner()
	oldClass := &storagev1.StorageClass{}
	if err := n.client.Get(n.ctx, types.NamespacedName{Name: class.Name}, oldClass); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		if err := n.client.Create(n.ctx, class); err != nil {
			return err
		}
	}
	return nil
}

func (n *nfsProvisioner) statefulsetForNFSProvisioner() interface{} {
	labels := n.component.GetLabels()
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
					DNSPolicy:          corev1.DNSClusterFirstWithHostNet,
					ServiceAccountName: "rainbond-operator", // TODO: do not hard code, get sa from configuration.
					NodeSelector:       n.cluster.Status.FirstMasterNodeLabel(),
					Tolerations: []corev1.Toleration{
						{
							Key:    n.cluster.Status.MasterRoleLabel,
							Effect: corev1.TaintEffectNoSchedule,
						},
					},
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
	labels := n.component.GetLabels()

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

func (n *nfsProvisioner) storageClassForNFSProvisioner() *storagev1.StorageClass {
	sc := &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{
			Name:   NFSName,
			Labels: n.component.GetLabels(),
		},
		Provisioner: nfsProvisionerName,
		MountOptions: []string{
			"vers=4.1",
		},
	}

	return sc
}
