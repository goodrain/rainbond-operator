package handler

import (
	"context"
	"fmt"
	"k8s.io/apimachinery/pkg/util/intstr"
	"strings"

	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/api/v1alpha1"
	"github.com/goodrain/rainbond-operator/util/commonutil"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// MQName name for rbd-mq
var MQName = "rbd-mq"

type mq struct {
	ctx        context.Context
	client     client.Client
	component  *rainbondv1alpha1.RbdComponent
	cluster    *rainbondv1alpha1.RainbondCluster
	labels     map[string]string
	etcdSecret *corev1.Secret
}

var _ ComponentHandler = &mq{}

// NewMQ creates a new rbd-mq handler.
func NewMQ(ctx context.Context, client client.Client, component *rainbondv1alpha1.RbdComponent, cluster *rainbondv1alpha1.RainbondCluster) ComponentHandler {
	return &mq{
		ctx:       ctx,
		client:    client,
		component: component,
		cluster:   cluster,
		labels:    LabelsForRainbondComponent(component),
	}
}

func (m *mq) Before() error {
	secret, err := etcdSecret(m.ctx, m.client, m.cluster)
	if err != nil {
		return fmt.Errorf("failed to get etcd secret: %v", err)
	}
	m.etcdSecret = secret
	return nil
}

func (m *mq) Resources() []client.Object {
	return []client.Object{
		m.deployment(),
		m.service(),
	}
}

func (m *mq) After() error {
	return nil
}

func (m *mq) ListPods() ([]corev1.Pod, error) {
	return listPods(m.ctx, m.client, m.component.Namespace, m.labels)
}

func (m *mq) deployment() client.Object {
	args := []string{
		"--etcd-endpoints=" + strings.Join(etcdEndpoints(m.cluster), ","),
		"--hostIP=$(POD_IP)",
	}
	var volumeMounts []corev1.VolumeMount
	var volumes []corev1.Volume
	if m.etcdSecret != nil {
		volume, mount := volumeByEtcd(m.etcdSecret)
		volumeMounts = append(volumeMounts, mount)
		volumes = append(volumes, volume)
		args = append(args, etcdSSLArgs()...)
	}
	env := []corev1.EnvVar{
		{
			Name: "POD_IP",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "status.podIP",
				},
			},
		},
	}

	env = mergeEnvs(env, m.component.Spec.Env)
	args = mergeArgs(args, m.component.Spec.Args)
	volumeMounts = mergeVolumeMounts(volumeMounts, m.component.Spec.VolumeMounts)
	volumes = mergeVolumes(volumes, m.component.Spec.Volumes)

	ds := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      MQName,
			Namespace: m.component.Namespace,
			Labels:    m.labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: m.component.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: m.labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:   MQName,
					Labels: m.labels,
				},
				Spec: corev1.PodSpec{
					TerminationGracePeriodSeconds: commonutil.Int64(0),
					ImagePullSecrets:              imagePullSecrets(m.component, m.cluster),
					Containers: []corev1.Container{
						{
							Name:            MQName,
							Image:           m.component.Spec.Image,
							ImagePullPolicy: m.component.ImagePullPolicy(),
							Env:             env,
							Args:            args,
							VolumeMounts:    volumeMounts,
							Resources:       m.component.Spec.Resources,
						},
					},
					Volumes: volumes,
				},
			},
		},
	}

	return ds
}

func (m *mq) service() client.Object {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      MQName,
			Namespace: m.component.Namespace,
			Labels:    m.labels,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:       MQName,
					Port:       6300,
					TargetPort: intstr.FromInt(6300),
				},
			},
			Selector: m.labels,
		},
	}
}
