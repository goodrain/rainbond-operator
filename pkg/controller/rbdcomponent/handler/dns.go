package handler

import (
	"context"

	"github.com/goodrain/rainbond-operator/pkg/util/commonutil"
	"k8s.io/apimachinery/pkg/util/intstr"

	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/pkg/apis/rainbond/v1alpha1"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// DNSName name for rbd-dns
var DNSName = "rbd-dns"

type dns struct {
	ctx       context.Context
	client    client.Client
	component *rainbondv1alpha1.RbdComponent
	cluster   *rainbondv1alpha1.RainbondCluster
	labels    map[string]string
}

var _ ComponentHandler = &dns{}

// NewDNS creates a new rbd-dns handler.
func NewDNS(ctx context.Context, client client.Client, component *rainbondv1alpha1.RbdComponent, cluster *rainbondv1alpha1.RainbondCluster) ComponentHandler {
	return &dns{
		ctx:       ctx,
		client:    client,
		component: component,
		cluster:   cluster,
		labels:    LabelsForRainbondComponent(component),
	}
}

func (d *dns) Before() error {
	return nil
}

func (d *dns) ListPods() ([]corev1.Pod, error) {
	return listPods(d.ctx, d.client, d.component.Namespace, d.labels)
}

func (d *dns) Resources() []interface{} {
	return []interface{}{
		d.deployment(),
		d.serviceForDNS(),
	}
}

func (d *dns) After() error {
	return nil
}

func (d *dns) deployment() interface{} {
	ds := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      DNSName,
			Namespace: d.component.Namespace,
			Labels:    d.labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: d.component.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: d.labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:   DNSName,
					Labels: d.labels,
				},
				Spec: corev1.PodSpec{
					TerminationGracePeriodSeconds: commonutil.Int64(0),
					ServiceAccountName:            "rainbond-operator",
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

func (d *dns) serviceForDNS() interface{} {
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      DNSName,
			Namespace: d.component.Namespace,
			Labels:    d.labels,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name: "dns",
					Port: 53,
					TargetPort: intstr.IntOrString{
						IntVal: 53,
					},
				},
			},
			Selector: d.labels,
		},
	}

	return svc
}
