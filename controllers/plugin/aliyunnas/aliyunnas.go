package aliyunnas

import (
	"context"
	"path"

	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/api/v1alpha1"
	"github.com/goodrain/rainbond-operator/controllers/plugin"
	"github.com/goodrain/rainbond-operator/util/commonutil"
	"github.com/goodrain/rainbond-operator/util/constants"
	"github.com/goodrain/rainbond-operator/util/k8sutil"
	"github.com/goodrain/rainbond-operator/util/rbdutil"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	storagev1beta1 "k8s.io/api/storage/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var log = logf.Log.WithName("aliyunnas_plugin")

const (
	provisioner = "nasplugin.csi.alibabacloud.com"
)

// CSIPlugins is the primary entrypoint for csi plugins.
func CSIPlugins(ctx context.Context, cli client.Client, volume *rainbondv1alpha1.RainbondVolume) plugin.CSIPlugin {
	labels := rbdutil.LabelsForRainbond(nil)
	return &aliyunnasPlugin{
		ctx:             ctx,
		cli:             cli,
		volume:          volume,
		labels:          labels,
		pluginName:      constants.AliyunCSINasPlugin,
		provisionerName: constants.AliyunCSINasProvisioner,
	}
}

type aliyunnasPlugin struct {
	ctx                         context.Context
	cli                         client.Client
	volume                      *rainbondv1alpha1.RainbondVolume
	labels                      map[string]string
	pluginName, provisionerName string
}

var _ plugin.CSIPlugin = &aliyunnasPlugin{}

func (p *aliyunnasPlugin) IsPluginReady() bool {
	sts := &appsv1.StatefulSet{}
	err := p.cli.Get(p.ctx, types.NamespacedName{Namespace: p.volume.Namespace, Name: p.provisionerName}, sts)
	if err != nil {
		if !errors.IsNotFound(err) {
			log.Error(err, "get statefulset for nfs plugin")
			return false
		}
		return false
	}

	return sts.Status.ReadyReplicas == sts.Status.Replicas
}

func (p *aliyunnasPlugin) GetProvisioner() string {
	return provisioner
}

func (p *aliyunnasPlugin) GetClusterScopedResources() []client.Object {
	return []client.Object{
		p.csiDriver(),
	}
}

func (p *aliyunnasPlugin) GetSubResources() []client.Object {
	return []client.Object{
		p.daemonset(),
		p.serviceForProvisioner(),
		p.statefulset(),
	}
}

func (p *aliyunnasPlugin) csiDriver() *storagev1beta1.CSIDriver {
	return &storagev1beta1.CSIDriver{
		ObjectMeta: metav1.ObjectMeta{
			Name: provisioner,
			Labels: rbdutil.LabelsForRainbond(map[string]string{
				"name": provisioner,
			}),
		},
		Spec: storagev1beta1.CSIDriverSpec{
			AttachRequired: commonutil.Bool(false),
		},
	}
}

func (p *aliyunnasPlugin) daemonset() *appsv1.DaemonSet {
	labels := commonutil.CopyLabels(p.labels)
	labels["name"] = p.pluginName
	ds := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      p.pluginName,
			Namespace: p.volume.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					TerminationGracePeriodSeconds: commonutil.Int64(0),
					ServiceAccountName:            "rainbond-operator",
					HostNetwork:                   true,
					HostPID:                       true,
					Tolerations: []corev1.Toleration{
						{
							Operator: corev1.TolerationOpExists,
						},
					},
					Containers: []corev1.Container{
						{
							Name:            "driver-registrar",
							Image:           path.Join(p.volume.Spec.ImageRepository, "csi-node-driver-registrar:v1.1.0"),
							ImagePullPolicy: "IfNotPresent",
							Lifecycle: &corev1.Lifecycle{
								PreStop: &corev1.Handler{
									Exec: &corev1.ExecAction{
										Command: []string{
											"/bin/sh",
											"-c",
											"rm -rf /registration/nasplugin.csi.alibabacloud.com /registration/nasplugin.csi.alibabacloud.com-reg.sock",
										},
									},
								},
							},
							Args: []string{
								"--v=5",
								"--csi-address=/var/lib/kubelet/plugins/nasplugin.csi.alibabacloud.com/csi.sock",
								"--kubelet-registration-path=/var/lib/kubelet/plugins/nasplugin.csi.alibabacloud.com/csi.sock",
							},
							Env: []corev1.EnvVar{
								{
									Name: "KUBE_NODE_NAME",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "spec.nodeName",
										},
									},
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "kubelet-dir",
									MountPath: "/var/lib/kubelet/",
								},
								{
									Name:      "registration-dir",
									MountPath: "/registration",
								},
							},
						},
						{
							Name:            "csi-nasplugin",
							Image:           path.Join(p.volume.Spec.ImageRepository, "csi-plugin:v1.14.8.32-aliyun"),
							ImagePullPolicy: "IfNotPresent",
							SecurityContext: &corev1.SecurityContext{
								Privileged: commonutil.Bool(true),
								Capabilities: &corev1.Capabilities{
									Add: []corev1.Capability{
										"SYS_ADMIN",
									},
								},
								AllowPrivilegeEscalation: commonutil.Bool(true),
							},
							Args: []string{
								"--endpoint=$(CSI_ENDPOINT)",
								"--v=5",
								"--driver=nasplugin.csi.alibabacloud.com",
							},
							Env: []corev1.EnvVar{
								{
									Name:  "CSI_ENDPOINT",
									Value: "unix://var/lib/kubelet/plugins/nasplugin.csi.alibabacloud.com/csi.sock",
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:             "kubelet-dir",
									MountPath:        "/var/lib/kubelet/",
									MountPropagation: k8sutil.MountPropagationMode(corev1.MountPropagationBidirectional),
								},
								{
									Name:             "host-log",
									MountPath:        "/var/log/",
									MountPropagation: k8sutil.MountPropagationMode(corev1.MountPropagationBidirectional),
								},
								{
									Name:      "etc",
									MountPath: "/host/etc",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "kubelet-dir",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/var/lib/kubelet/",
									Type: k8sutil.HostPath(corev1.HostPathDirectory),
								},
							},
						},
						{
							Name: "registration-dir",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/var/lib/kubelet/plugins_registry",
									Type: k8sutil.HostPath(corev1.HostPathDirectoryOrCreate),
								},
							},
						},
						{
							Name: "host-log",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/var/log/",
									Type: k8sutil.HostPath(corev1.HostPathDirectoryOrCreate),
								},
							},
						},
						{
							Name: "etc",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/etc",
									Type: k8sutil.HostPath(corev1.HostPathDirectoryOrCreate),
								},
							},
						},
					},
				},
			},
		},
	}

	return ds
}

func (p *aliyunnasPlugin) serviceForProvisioner() *corev1.Service {
	labels := commonutil.CopyLabels(p.labels)
	labels["name"] = p.provisionerName
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      p.provisionerName,
			Namespace: p.volume.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name: "dummy",
					Port: 12345,
				},
			},
			Selector: labels,
		},
	}

	return svc
}

func (p *aliyunnasPlugin) statefulset() client.Object {
	labels := commonutil.CopyLabels(p.labels)
	labels["name"] = p.provisionerName
	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      p.provisionerName,
			Namespace: p.volume.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.StatefulSetSpec{
			ServiceName: p.provisionerName,
			Replicas:    commonutil.Int32(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Tolerations: []corev1.Toleration{
						{
							Operator: corev1.TolerationOpExists,
						},
					},
					TerminationGracePeriodSeconds: commonutil.Int64(0),
					Affinity: &corev1.Affinity{
						NodeAffinity: &corev1.NodeAffinity{
							PreferredDuringSchedulingIgnoredDuringExecution: []corev1.PreferredSchedulingTerm{
								{
									Weight: 1,
									Preference: corev1.NodeSelectorTerm{
										MatchExpressions: []corev1.NodeSelectorRequirement{
											{
												Key:      "node-role.kubernetes.io/master",
												Operator: corev1.NodeSelectorOpExists,
											},
										},
									},
								},
							},
						},
					},
					ServiceAccountName: "rainbond-operator",
					HostNetwork:        true,
					Containers: []corev1.Container{
						{
							Name:            "csi-nas-external-provisioner",
							Image:           path.Join(p.volume.Spec.ImageRepository, "csi-provisioner:v1.2.2-aliyun"),
							ImagePullPolicy: "Always",
							Args: []string{
								"--provisioner=nasplugin.csi.alibabacloud.com",
								"--csi-address=$(ADDRESS)",
								"--volume-name-prefix=nas",
								"--v=5",
							},
							Env: []corev1.EnvVar{
								{
									Name:  "ADDRESS",
									Value: "/socketDir/csi.sock",
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "disk-provisioner-dir",
									MountPath: "/socketDir",
								},
							},
						},
						{
							Name:            "csi-nasprovisioner",
							Image:           path.Join(p.volume.Spec.ImageRepository, "csi-plugin:v1.14.8.32-aliyun"),
							ImagePullPolicy: "Always",
							SecurityContext: &corev1.SecurityContext{
								Privileged: commonutil.Bool(true),
								Capabilities: &corev1.Capabilities{
									Add: []corev1.Capability{
										"SYS_ADMIN",
									},
								},
								AllowPrivilegeEscalation: commonutil.Bool(true),
							},
							Args: []string{
								"--endpoint=$(CSI_ENDPOINT)",
								"--v=5",
								"--driver=nasplugin.csi.alibabacloud.com",
							},
							Env: []corev1.EnvVar{
								{
									Name:  "CSI_ENDPOINT",
									Value: "unix://socketDir/csi.sock",
								},
								{
									Name:  "ACCESS_KEY_ID",
									Value: p.volume.Spec.CSIPlugin.AliyunNas.AccessKeyID,
								},
								{
									Name:  "ACCESS_KEY_SECRET",
									Value: p.volume.Spec.CSIPlugin.AliyunNas.AccessKeySecret,
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "host-log",
									MountPath: "/var/log/",
								},
								{
									Name:      "disk-provisioner-dir",
									MountPath: "/socketDir/",
								},
								{
									Name:      "etc",
									MountPath: "/host/etc",
								},
							},
							LivenessProbe: &corev1.Probe{
								Handler: corev1.Handler{
									Exec: &corev1.ExecAction{Command: []string{"sh", "-c", "ps -ef | grep plugin.csi.alibabacloud.com | grep nasplugin.csi.alibabacloud.com | grep -v grep"}},
								},
								FailureThreshold:    8,
								InitialDelaySeconds: 15,
								PeriodSeconds:       10,
								SuccessThreshold:    1,
								TimeoutSeconds:      15,
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "disk-provisioner-dir",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
						{
							Name: "host-log",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/var/log/",
									Type: k8sutil.HostPath(corev1.HostPathDirectory),
								},
							},
						},
						{
							Name: "etc",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/etc",
									Type: k8sutil.HostPath(corev1.HostPathDirectory),
								},
							},
						},
					},
				},
			},
			UpdateStrategy: appsv1.StatefulSetUpdateStrategy{
				Type: appsv1.RollingUpdateStatefulSetStrategyType,
			},
		},
	}

	return sts
}
