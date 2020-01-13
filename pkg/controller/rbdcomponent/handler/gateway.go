package handler

import (
	"fmt"

	rainbondv1alpha1 "github.com/GLYASAI/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var GatewayName = "rbd-gateway"

type gateway struct {
	component *rainbondv1alpha1.RbdComponent
	cluster   *rainbondv1alpha1.RainbondCluster
}

func NewGateway(component *rainbondv1alpha1.RbdComponent, cluster *rainbondv1alpha1.RainbondCluster) ComponentHandler {
	return &gateway{
		component: component,
		cluster:   cluster,
	}
}

func (g *gateway) Before() error {
	// No prerequisites, if no gateway-installed node is specified, install on all nodes that meet the conditions
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
	labels := g.component.Labels()
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
					ServiceAccountName: "rainbond-operator",
					HostNetwork:        true,
					DNSPolicy:          corev1.DNSClusterFirstWithHostNet,
					Tolerations: []corev1.Toleration{
						{
							Key:    "node-role.kubernetes.io/master", // TODO: There are other labels used to identify the master
							Effect: corev1.TaintEffectNoSchedule,
						},
					},
					NodeSelector: map[string]string{
						"node-role.kubernetes.io/master": "", // TODO: There are other labels used to identify the master
					},
					Containers: []corev1.Container{
						{
							Name:            GatewayName,
							Image:           g.component.Spec.Image,
							ImagePullPolicy: g.component.ImagePullPolicy(),
							Args: []string{
								fmt.Sprintf("--log-level=%s", g.component.LogLevel()),
								"--error-log=/dev/stderr error",
								"--enable-kubeapi=false",
								"--etcd-endpoints=http://etcd0:2379", // TODO: use rainbondcluster
							},
						},
					},
				},
			},
		},
	}

	return ds
}
