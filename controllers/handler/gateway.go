package handler

import (
	"context"
	"fmt"
	"strings"

	wutongv1alpha1 "github.com/wutong-paas/wutong-operator/api/v1alpha1"
	"github.com/wutong-paas/wutong-operator/util/commonutil"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GatewayName name for wt-gateway.
var GatewayName = "wt-gateway"

type gateway struct {
	ctx        context.Context
	client     client.Client
	etcdSecret *corev1.Secret

	component *wutongv1alpha1.WutongComponent
	cluster   *wutongv1alpha1.WutongCluster
	labels    map[string]string
}

var _ ComponentHandler = &gateway{}
var _ Replicaser = &gateway{}

// NewGateway returns a new wt-gateway handler.
func NewGateway(ctx context.Context, client client.Client, component *wutongv1alpha1.WutongComponent, cluster *wutongv1alpha1.WutongCluster) ComponentHandler {
	return &gateway{
		ctx:       ctx,
		client:    client,
		component: component,
		cluster:   cluster,
		labels:    LabelsForWutongComponent(component),
	}
}

func (g *gateway) Before() error {
	secret, err := etcdSecret(g.ctx, g.client, g.cluster)
	if err != nil {
		return fmt.Errorf("failed to get etcd secret: %v", err)
	}
	g.etcdSecret = secret

	if err := isEtcdAvailable(g.ctx, g.client, g.component, g.cluster); err != nil {
		return fmt.Errorf("etcd not available: %v", err)
	}

	return nil
}

func (g *gateway) Resources() []client.Object {
	return []client.Object{
		g.daemonset(),
	}
}

func (g *gateway) After() error {
	return nil
}

func (g *gateway) ListPods() ([]corev1.Pod, error) {
	return listPods(g.ctx, g.client, g.component.Namespace, g.labels)
}

func (g *gateway) Replicas() *int32 {
	return commonutil.Int32(int32(len(g.cluster.Spec.NodesForGateway)))
}

func (g *gateway) daemonset() client.Object {
	args := []string{
		"--error-log=/dev/stderr",
		"--errlog-level=error",
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

	var nodeNames []string
	for _, node := range g.cluster.Spec.NodesForGateway {
		nodeNames = append(nodeNames, node.Name)
	}
	var affinity *corev1.Affinity
	if len(nodeNames) > 0 {
		affinity = affinityForRequiredNodes(nodeNames)
	}
	if affinity == nil {
		// TODO: make sure nodeNames not empty
		return nil
	}

	// merge attributes
	volumeMounts = mergeVolumeMounts(volumeMounts, g.component.Spec.VolumeMounts)
	volumes = mergeVolumes(volumes, g.component.Spec.Volumes)
	args = mergeArgs(args, g.component.Spec.Args)

	ds := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      GatewayName,
			Namespace: g.component.Namespace,
			Labels:    g.labels,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: g.labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:   GatewayName,
					Labels: g.labels,
				},
				Spec: corev1.PodSpec{
					ImagePullSecrets:              imagePullSecrets(g.component, g.cluster),
					TerminationGracePeriodSeconds: commonutil.Int64(0),
					ServiceAccountName:            "wutong-operator",
					HostNetwork:                   true,
					DNSPolicy:                     corev1.DNSClusterFirstWithHostNet,
					Tolerations: []corev1.Toleration{
						{
							Operator: corev1.TolerationOpExists, // tolerate everything.
						},
					},
					Affinity: affinity,
					Containers: []corev1.Container{
						{
							Name:            GatewayName,
							Image:           g.component.Spec.Image,
							ImagePullPolicy: g.component.ImagePullPolicy(),
							Args:            args,
							VolumeMounts:    volumeMounts,
							SecurityContext: &corev1.SecurityContext{
								Privileged: commonutil.Bool(true),
							},
							Env: g.component.Spec.Env,
						},
					},
					Volumes: volumes,
				},
			},
		},
	}

	return ds
}
