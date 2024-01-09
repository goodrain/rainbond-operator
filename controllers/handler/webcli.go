package handler

import (
	"context"
	"fmt"
	"k8s.io/apimachinery/pkg/util/intstr"
	"strings"

	"github.com/goodrain/rainbond-operator/util/commonutil"

	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// WebCliName name for rbd-webcli.
var WebCliName = "rbd-webcli"

type webcli struct {
	ctx        context.Context
	client     client.Client
	component  *rainbondv1alpha1.RbdComponent
	cluster    *rainbondv1alpha1.RainbondCluster
	labels     map[string]string
	etcdSecret *corev1.Secret
}

var _ ComponentHandler = &webcli{}

// NewWebCli creates a new rbd-webcli handler.
func NewWebCli(ctx context.Context, client client.Client, component *rainbondv1alpha1.RbdComponent, cluster *rainbondv1alpha1.RainbondCluster) ComponentHandler {
	return &webcli{
		ctx:       ctx,
		client:    client,
		component: component,
		cluster:   cluster,
		labels:    LabelsForRainbondComponent(component),
	}
}

func (w *webcli) Before() error {
	secret, err := etcdSecret(w.ctx, w.client, w.cluster)
	if err != nil {
		return fmt.Errorf("failed to get etcd secret: %v", err)
	}
	w.etcdSecret = secret

	return nil
}

func (w *webcli) Resources() []client.Object {
	return []client.Object{
		w.deployment(),
		w.service(),
	}
}

func (w *webcli) After() error {
	return nil
}

func (w *webcli) ListPods() ([]corev1.Pod, error) {
	return listPods(w.ctx, w.client, w.component.Namespace, w.labels)
}

func (w *webcli) deployment() client.Object {
	volumeMounts := []corev1.VolumeMount{}
	volumes := []corev1.Volume{}
	args := []string{
		"--hostIP=$(POD_IP)",
		"--etcd-endpoints=" + strings.Join(etcdEndpoints(w.cluster), ","),
	}
	if w.etcdSecret != nil {
		volume, mount := volumeByEtcd(w.etcdSecret)
		volumeMounts = append(volumeMounts, mount)
		volumes = append(volumes, volume)
		args = append(args, etcdSSLArgs()...)
	}

	volumeMounts = mergeVolumeMounts(volumeMounts, w.component.Spec.VolumeMounts)
	volumes = mergeVolumes(volumes, w.component.Spec.Volumes)
	args = mergeArgs(args, w.component.Spec.Args)

	ds := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      WebCliName,
			Namespace: w.component.Namespace,
			Labels:    w.labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: w.component.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: w.labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:   WebCliName,
					Labels: w.labels,
				},
				Spec: corev1.PodSpec{
					ImagePullSecrets:              imagePullSecrets(w.component, w.cluster),
					ServiceAccountName:            "rainbond-operator",
					TerminationGracePeriodSeconds: commonutil.Int64(0),
					Containers: []corev1.Container{
						{
							Name:            WebCliName,
							Image:           w.component.Spec.Image,
							ImagePullPolicy: w.component.ImagePullPolicy(),
							Env: []corev1.EnvVar{
								{
									Name: "POD_IP",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "status.podIP",
										},
									},
								},
							},
							Args:         args,
							VolumeMounts: volumeMounts,
							Resources:    w.component.Spec.Resources,
						},
					},
					Volumes: volumes,
				},
			},
		},
	}

	return ds
}

func (w *webcli) service() client.Object {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      WorkerName,
			Namespace: w.component.Namespace,
			Labels:    w.labels,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:       WorkerName,
					Port:       7171,
					TargetPort: intstr.FromInt(7171),
				},
			},
			Selector: w.labels,
		},
	}
}
