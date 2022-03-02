package handler

import (
	"context"

	wutongv1alpha1 "github.com/wutong/wutong-operator/api/v1alpha1"
	"github.com/wutong/wutong-operator/util/commonutil"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// DashboardMetricsScraperName -
var DashboardMetricsScraperName = "dashboard-metrics-scraper"

type dashboardMetricsScraper struct {
	ctx       context.Context
	client    client.Client
	db        *wutongv1alpha1.Database
	labels    map[string]string
	component *wutongv1alpha1.WutongComponent
	cluster   *wutongv1alpha1.WutongCluster
}

var _ ComponentHandler = &dashboardMetricsScraper{}

// NewDashboardMetricsScraper -
func NewDashboardMetricsScraper(ctx context.Context, client client.Client, component *wutongv1alpha1.WutongComponent, cluster *wutongv1alpha1.WutongCluster) ComponentHandler {
	return &dashboardMetricsScraper{
		ctx:       ctx,
		client:    client,
		component: component,
		cluster:   cluster,
		labels:    LabelsForWutongComponent(component),
	}
}

func (k *dashboardMetricsScraper) Before() error {
	return nil
}

func (k *dashboardMetricsScraper) Resources() []client.Object {
	return []client.Object{
		k.deploymentForDashboardMetricsScraper(),
		k.serviceForDashboardMetricsScraper(),
	}
}

func (k *dashboardMetricsScraper) After() error {
	return nil
}

func (k *dashboardMetricsScraper) ListPods() ([]corev1.Pod, error) {
	return listPods(k.ctx, k.client, k.component.Namespace, k.labels)
}

func (k *dashboardMetricsScraper) deploymentForDashboardMetricsScraper() client.Object {
	labels := copyLabels(k.labels)
	labels["name"] = DashboardMetricsScraperName

	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      DashboardMetricsScraperName,
			Namespace: k.component.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: k.component.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:   DashboardMetricsScraperName,
					Labels: labels,
					Annotations: map[string]string{
						"seccomp.security.alpha.kubernetes.io/pod": "runtime/default",
					},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: "wutong-operator",
					Tolerations: []corev1.Toleration{
						{
							Operator: corev1.TolerationOpExists, // tolerate everything.
						},
					},
					ImagePullSecrets:              imagePullSecrets(k.component, k.cluster),
					TerminationGracePeriodSeconds: commonutil.Int64(0),
					Containers: []corev1.Container{
						{
							Name:            DashboardMetricsScraperName,
							Image:           k.component.Spec.Image,
							ImagePullPolicy: k.component.ImagePullPolicy(),
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 8000,
									Protocol:      corev1.ProtocolTCP,
								},
							},
							SecurityContext: &corev1.SecurityContext{
								Privileged: commonutil.Bool(true),
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "tmp-volume",
									MountPath: "/tmp",
								},
							},
							Resources: k.component.Spec.Resources,
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "tmp-volume",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
					},
				},
			},
		},
	}

	return deploy
}

func (k *dashboardMetricsScraper) serviceForDashboardMetricsScraper() client.Object {
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      DashboardMetricsScraperName,
			Namespace: k.component.Namespace,
			Labels:    k.labels,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Port: 8000,
					TargetPort: intstr.IntOrString{
						IntVal: 8000,
					},
				},
			},
			Selector: k.labels,
		},
	}

	return svc
}
