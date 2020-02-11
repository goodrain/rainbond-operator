package handler

import (
	"context"
	"fmt"

	"github.com/goodrain/rainbond-operator/pkg/util/commonutil"
	"github.com/goodrain/rainbond-operator/pkg/util/k8sutil"

	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

//GrctlName install grctl
var GrctlName = "rbd-grctl"

type grctl struct {
	ctx       context.Context
	client    client.Client
	component *rainbondv1alpha1.RbdComponent
	cluster   *rainbondv1alpha1.RainbondCluster
	pkg       *rainbondv1alpha1.RainbondPackage
	labels    map[string]string
	apiSecret *corev1.Secret
}

//NewGrctl new grctl handle
func NewGrctl(ctx context.Context, client client.Client, component *rainbondv1alpha1.RbdComponent, cluster *rainbondv1alpha1.RainbondCluster, pkg *rainbondv1alpha1.RainbondPackage) ComponentHandler {
	return &grctl{
		ctx:       ctx,
		client:    client,
		component: component,
		cluster:   cluster,
		labels:    component.GetLabels(),
		pkg:       pkg,
	}
}

func (w *grctl) Before() error {
	secret, err := getSecret(w.ctx, w.client, w.component.Namespace, apiClientSecretName)
	if err != nil {
		return fmt.Errorf("failed to get api tls secret: %v", err)
	}
	if len(secret.Data) == 0 {
		return fmt.Errorf("failed to get api tls secret, waiting secret ready")
	}
	w.apiSecret = secret
	return nil
}

func (w *grctl) Resources() []interface{} {
	return []interface{}{
		w.daemonSetForAPI(),
	}
}

func (w *grctl) After() error {
	return nil
}

func (w *grctl) daemonSetForAPI() interface{} {
	volumeMounts := []corev1.VolumeMount{
		{
			Name:      "path",
			MountPath: "/rootfs/path",
		},
		{
			Name:      "root",
			MountPath: "/rootfs/root",
		},
		{
			Name:      "ssl",
			MountPath: "/ssl",
		},
	}
	volumes := []corev1.Volume{
		{
			Name: "path",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/usr/local/bin",
					Type: k8sutil.HostPath(corev1.HostPathDirectory),
				},
			},
		},
		{
			Name: "root",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/root",
					Type: k8sutil.HostPath(corev1.HostPathDirectory),
				},
			},
		},
		{
			Name: "ssl",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/opt/rainbond/etc/rbd-api/region.goodrain.me/ssl/",
					Type: k8sutil.HostPath(corev1.HostPathDirectory),
				},
			},
		},
	}
	args := []string{"install"}
	if w.apiSecret != nil {
		volume, mount := volumeByAPISecret(w.apiSecret)
		volumeMounts = append(volumeMounts, mount)
		volumes = append(volumes, volume)
	}
	w.labels["name"] = GrctlName
	ds := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      GrctlName,
			Namespace: w.component.Namespace,
			Labels:    w.labels,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: w.labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:   GrctlName,
					Labels: w.labels,
				},
				Spec: corev1.PodSpec{
					TerminationGracePeriodSeconds: commonutil.Int64(0),
					Tolerations: []corev1.Toleration{
						{
							Key:    w.cluster.Status.MasterRoleLabel,
							Effect: corev1.TaintEffectNoSchedule,
						},
					},
					NodeSelector: w.cluster.Status.MasterNodeLabel(),
					Containers: []corev1.Container{
						{
							Name:            GrctlName,
							Image:           w.component.Spec.Image,
							ImagePullPolicy: w.component.ImagePullPolicy(),
							Args:            args,
							VolumeMounts:    volumeMounts,
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceMemory: *resource.NewQuantity(64*1024*1024, resource.BinarySI),
									corev1.ResourceCPU:    *resource.NewQuantity(200, resource.DecimalSI),
								},
							},
						},
					},
					Volumes: volumes,
				},
			},
		},
	}

	return ds
}
