package handler

import (
	"context"
	"fmt"
	"github.com/goodrain/rainbond-operator/pkg/util/commonutil"
	"strings"

	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/pkg/apis/rainbond/v1alpha1"
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

	component *rainbondv1alpha1.RbdComponent
	cluster   *rainbondv1alpha1.RainbondCluster
	pkg       *rainbondv1alpha1.RainbondPackage
	labels    map[string]string
}

var _ ComponentHandler = &gateway{}

func NewGateway(ctx context.Context, client client.Client, component *rainbondv1alpha1.RbdComponent, cluster *rainbondv1alpha1.RainbondCluster, pkg *rainbondv1alpha1.RainbondPackage) ComponentHandler {
	return &gateway{
		ctx:       ctx,
		client:    client,
		component: component,
		cluster:   cluster,
		pkg:       pkg,
		labels:    LabelsForRainbondComponent(component),
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
		g.deployment(),
	}
}

func (g *gateway) After() error {
	return nil
}

func (g *gateway) deployment() interface{} {
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

	var nodeNames []string
	for _, node := range g.cluster.Spec.NodesForGateway {
		nodeNames = append(nodeNames, node.Name)
	}

	ds := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      GatewayName,
			Namespace: g.component.Namespace,
			Labels:    g.labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: g.component.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: g.labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:   GatewayName,
					Labels: g.labels,
				},
				Spec: corev1.PodSpec{
					TerminationGracePeriodSeconds: commonutil.Int64(0),
					ServiceAccountName:            "rainbond-operator",
					HostNetwork:                   true,
					DNSPolicy:                     corev1.DNSClusterFirstWithHostNet,
					Tolerations: []corev1.Toleration{
						{
							Operator: corev1.TolerationOpExists, // tolerate everything.
						},
					},
					Affinity: &corev1.Affinity{
						NodeAffinity: &corev1.NodeAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
								NodeSelectorTerms: []corev1.NodeSelectorTerm{
									{
										MatchFields: []corev1.NodeSelectorRequirement{
											{
												Key:      "metadata.name",
												Operator: corev1.NodeSelectorOpIn,
												Values:   nodeNames,
											},
										},
									},
								},
							},
						},
					},
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
