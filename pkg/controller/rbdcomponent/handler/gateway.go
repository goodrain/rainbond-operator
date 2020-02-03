package handler

import (
	"context"
	"fmt"
	"github.com/GLYASAI/rainbond-operator/pkg/util/commonutil"
	"strings"

	rainbondv1alpha1 "github.com/GLYASAI/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var GatewayName = "rbd-gateway"

type gateway struct {
	ctx        context.Context
	client     client.Client
	etcdSecret *corev1.Secret

	component  *rainbondv1alpha1.RbdComponent
	cluster    *rainbondv1alpha1.RainbondCluster
}

func NewGateway(ctx context.Context, client client.Client, component *rainbondv1alpha1.RbdComponent, cluster *rainbondv1alpha1.RainbondCluster) ComponentHandler {
	return &gateway{
		ctx:       ctx,
		client:    client,
		component: component,
		cluster:   cluster,
	}
}

func (g *gateway) Before() error {
	secret, err := etcdSecret(g.ctx, g.client, g.cluster)
	if err != nil {
		return fmt.Errorf("failed to get etcd secret: %v", err)
	}
	g.etcdSecret = secret
	return nil
}

func (g *gateway) Resources() []interface{} {
	return []interface{}{
		g.daemonSetForGateway(),
	}
}

func (g *gateway) After() error {
	return nil
}

func (g *gateway) daemonSetForGateway() interface{} {
	args := []string{
		fmt.Sprintf("--log-level=%s", g.component.LogLevel()),
		"--error-log=/dev/stderr error",
		"--enable-kubeapi=false",
		"--etcd-endpoints=" + strings.Join(etcdEndpoints(g.cluster), ","),
	}
	var volumeMounts []corev1.VolumeMount
	var volumes []corev1.Volume
	if g.etcdSecret != nil {
		volume, mount := volumeByEtcd(g.etcdSecret)
		volumeMounts = append(volumeMounts, mount)
		volumes = append(volumes, volume)
		args = append(args, etcdSSLArgs()...)
	}

	labels := g.component.GetLabels()
	ds := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      GatewayName,
			Namespace: g.component.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:   GatewayName,
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					TerminationGracePeriodSeconds: commonutil.Int64(0),
					ServiceAccountName: "rainbond-operator",
					HostNetwork:        true,
					DNSPolicy:          corev1.DNSClusterFirstWithHostNet,
					Tolerations: []corev1.Toleration{
						{
							Key:    g.cluster.Status.MasterRoleLabel,
							Effect: corev1.TaintEffectNoSchedule,
						},
					},
					NodeSelector: g.cluster.Status.MasterNodeLabel(),
					Containers: []corev1.Container{
						{
							Name:            GatewayName,
							Image:           g.component.Spec.Image,
							ImagePullPolicy: g.component.ImagePullPolicy(),
							Args:            args,
							VolumeMounts:    volumeMounts,
						},
					},
					Volumes: volumes,
				},
			},
		},
	}

	return ds
}
