package nfs

import (
	"context"
	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	"github.com/goodrain/rainbond-operator/pkg/controller/rainbondvolume/plugin"
	"github.com/goodrain/rainbond-operator/pkg/util/commonutil"
	"github.com/goodrain/rainbond-operator/pkg/util/k8sutil"
	rbdutil "github.com/goodrain/rainbond-operator/pkg/util/rbduitl"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var log = logf.Log.WithName("nfs_plugin")

const (
	provisioner = "rainbond.io/nfs"
)

// CSIPlugins is the primary entrypoint for csi plugins.
func CSIPlugins(ctx context.Context, cli client.Client, volume *rainbondv1alpha1.RainbondVolume) plugin.CSIPlugin {
	name := "nfs-provisioner"
	labels := rbdutil.LabelsForRainbond(map[string]string{
		"name": name,
	})
	return &nfsPlugin{
		ctx:    ctx,
		cli:    cli,
		name:   name,
		volume: volume,
		labels: labels,
	}
}

type nfsPlugin struct {
	ctx    context.Context
	cli    client.Client
	name   string
	volume *rainbondv1alpha1.RainbondVolume
	labels map[string]string
}

var _ plugin.CSIPlugin = &nfsPlugin{}

func (p *nfsPlugin) IsPluginReady() bool {
	sts := &appsv1.StatefulSet{}
	err := p.cli.Get(p.ctx, types.NamespacedName{Namespace: p.volume.Namespace, Name: p.name}, sts)
	if err != nil {
		if !errors.IsNotFound(err) {
			log.Error(err, "get statefulset for nfs plugin")
			return false
		}
		return false
	}

	return sts.Status.ReadyReplicas == sts.Status.Replicas
}

func (p *nfsPlugin) GetProvisioner() string {
	return provisioner
}

func (p *nfsPlugin) GetClusterScopedResources() []interface{} {
	return nil
}

func (p *nfsPlugin) GetSubResources() []interface{} {
	return []interface{}{
		p.statefulset(),
		p.service(),
	}
}

func (p *nfsPlugin) statefulset() interface{} {
	labels := p.labels
	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      p.name,
			Namespace: p.volume.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas:    commonutil.Int32(1),
			ServiceName: p.name,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:   p.name,
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: "rainbond-operator", // TODO: do not hard code, get sa from configuration.
					Containers: []corev1.Container{
						{
							Name:            p.name,
							Image:           "registry.cn-hangzhou.aliyuncs.com/goodrain/nfs-provisioner:v2.2.1-k8s1.12", // TODO: do not hard code, get sa from configuration.
							ImagePullPolicy: corev1.PullIfNotPresent,
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
									Value: p.name,
								},
							},
							Args: []string{
								"-provisioner=" + provisioner,
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

func (p *nfsPlugin) service() *corev1.Service {
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      p.name,
			Namespace: p.volume.Namespace,
			Labels:    p.labels,
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
			Selector: p.labels,
		},
	}

	return svc
}
