package handler

import (
	"context"

	rainbondv1alpha1 "github.com/GLYASAI/rainbond-operator/pkg/apis/rainbond/v1alpha1"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var DNSName = "rbd-dns"

type dns struct {
	component *rainbondv1alpha1.RbdComponent
	cluster   *rainbondv1alpha1.RainbondCluster
	labels    map[string]string
}

func NewDNS(ctx context.Context, client client.Client, component *rainbondv1alpha1.RbdComponent, cluster *rainbondv1alpha1.RainbondCluster) ComponentHandler {
	return &dns{
		component: component,
		cluster:   cluster,
		labels:    component.Labels(),
	}
}

func (d *dns) Before() error {
	return isPhaseOK(d.cluster)
}

func (d *dns) Resources() []interface{} {
	return []interface{}{
		d.daemonSetForDNS(),
	}
}

func (d *dns) After() error {
	return nil
}

func (d *dns) daemonSetForDNS() interface{} {
	ds := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      DNSName,
			Namespace: d.component.Namespace,
			Labels:    d.labels,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: d.labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:   DNSName,
					Labels: d.labels,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: "rainbond-operator",
					HostNetwork:        true,
					DNSPolicy:          corev1.DNSClusterFirstWithHostNet,
					Tolerations: []corev1.Toleration{
						{
							Key:    "node-role.kubernetes.io/master",
							Effect: corev1.TaintEffectNoSchedule,
						},
					},
					NodeSelector: map[string]string{
						"node-role.kubernetes.io/master": "",
					},
					Containers: []corev1.Container{
						{
							Name:            DNSName,
							Image:           d.component.Spec.Image,
							ImagePullPolicy: d.component.ImagePullPolicy(),
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
									Name: "HOST_IP",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "status.hostIP",
										},
									},
								},
								{
									Name:  "EX_DOMAIN",
									Value: d.cluster.Spec.SuffixHTTPHost,
								},
							},
							Args: []string{
								"--v=2",
								"--healthz-port=8089",
								"--dns-bind-address=$(POD_IP)",
								"--nameservers=202.106.0.22,1.2.4.8",
								"--recoders=goodrain.me=$(HOST_IP),*.goodrain.me=$(HOST_IP),rainbond.kubernetes.apiserver=$(HOST_IP)", // TODO: goodrain.me
							},
						},
					},
				},
			},
		},
	}

	return ds
}
