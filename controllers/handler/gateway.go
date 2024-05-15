package handler

import (
	"context"
	"fmt"
	"github.com/goodrain/rainbond-operator/util/rbdutil"

	"github.com/goodrain/rainbond-operator/util/k8sutil"
	"github.com/sirupsen/logrus"

	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/api/v1alpha1"
	"github.com/goodrain/rainbond-operator/util/commonutil"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GatewayName name for rbd-gateway.
var GatewayName = "rbd-gateway"

type gateway struct {
	ctx        context.Context
	client     client.Client
	etcdSecret *corev1.Secret

	component *rainbondv1alpha1.RbdComponent
	cluster   *rainbondv1alpha1.RainbondCluster
	labels    map[string]string
}

var _ ComponentHandler = &gateway{}
var _ Replicaser = &gateway{}

// NewGateway returns a new rbd-gateway handler.
func NewGateway(ctx context.Context, client client.Client, component *rainbondv1alpha1.RbdComponent, cluster *rainbondv1alpha1.RainbondCluster) ComponentHandler {
	return &gateway{
		ctx:       ctx,
		client:    client,
		component: component,
		cluster:   cluster,
		labels:    LabelsForRainbondComponent(component),
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
	k8sNodes, err := k8sutil.ListNodes(context.Background(), g.client)
	if err != nil {
		logrus.Error("get cluster node list error:", err)
	}
	k8sNodeNames := make(map[string]struct{})
	nodeLabels := make(map[string]struct{})
	for _, k8sNode := range k8sNodes {
		if hostName, ok := k8sNode.Labels["kubernetes.io/hostname"]; ok {
			nodeLabels[hostName] = struct{}{}
		}
		if hostName, ok := k8sNode.Labels["k3s.io/hostname"]; ok {
			nodeLabels[hostName] = struct{}{}
		}
		k8sNodeNames[k8sNode.Name] = struct{}{}
	}
	nodeForGateway := g.cluster.Spec.NodesForGateway
	if nodeForGateway != nil {
		for _, currentNode := range nodeForGateway {
			if _, ok := k8sNodeNames[currentNode.Name]; !ok {
				fmt.Printf("\033[1;31;40m%s\033[0m\n", fmt.Sprintf("Node %v cannot be found in the cluster", currentNode.Name))
			}
			if _, ok := nodeLabels[currentNode.Name]; !ok {
				fmt.Printf("\033[1;31;40m%s\033[0m\n", fmt.Sprintf("Node name %v is not bound to the label of a cluster node", currentNode.Name))
			}
		}
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
	}
	envs := []corev1.EnvVar{
		{
			Name:  "SERVICE_ID",
			Value: "rbd-gateway",
		},
		{
			Name:  "LOGGER_DRIVER_NAME",
			Value: "streamlog",
		},
	}
	envs = append(envs, g.component.Spec.Env...)

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
					ServiceAccountName:            rbdutil.GetenvDefault("SERVICE_ACCOUNT_NAME", "rainbond-operator"),
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
							Env: envs,
						},
					},
					Volumes: volumes,
				},
			},
		},
	}

	return ds
}
