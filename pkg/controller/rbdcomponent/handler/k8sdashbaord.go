package handler

import (
	"context"
	"fmt"

	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	"github.com/goodrain/rainbond-operator/pkg/util/commonutil"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// KubernetesDashboardName -
var KubernetesDashboardName = "kubernetes-dashboard"

type k8sdashbaord struct {
	ctx       context.Context
	client    client.Client
	db        *rainbondv1alpha1.Database
	labels    map[string]string
	component *rainbondv1alpha1.RbdComponent
	cluster   *rainbondv1alpha1.RainbondCluster
}

var _ ComponentHandler = &k8sdashbaord{}

// NewK8sDashboard -
func NewK8sDashboard(ctx context.Context, client client.Client, component *rainbondv1alpha1.RbdComponent, cluster *rainbondv1alpha1.RainbondCluster) ComponentHandler {
	return &k8sdashbaord{
		ctx:       ctx,
		client:    client,
		component: component,
		cluster:   cluster,
		labels:    LabelsForRainbondComponent(component),
	}
}

func (k *k8sdashbaord) Before() error {
	return nil
}

func (k *k8sdashbaord) Resources() []interface{} {
	return []interface{}{
		k.serviceForKubernetesDashboardCerts(),
		k.serviceForKubernetesDashboardCsrf(),
		k.serviceForKubernetesDashboardKeyHolder(),
		k.serviceForDashbaord(),
		k.deploymentForKubernetesDashboard(),
	}
}

func (k *k8sdashbaord) After() error {
	return nil
}

func (k *k8sdashbaord) ListPods() ([]corev1.Pod, error) {
	return listPods(k.ctx, k.client, k.component.Namespace, k.labels)
}

func (k *k8sdashbaord) deploymentForKubernetesDashboard() interface{} {
	args := []string{
		"--insecure-bind-address=0.0.0.0",
		fmt.Sprintf("--namespace=%s", k.component.Namespace),
	}
	labels := copyLabels(k.labels)
	labels["name"] = KubernetesDashboardName

	volumeMounts := []corev1.VolumeMount{
		{
			Name:      "tmp-volume",
			MountPath: "/tmp",
		},
		{
			Name:      "kubernetes-dashboard-certs",
			MountPath: "/certs",
		},
	}
	volumes := []corev1.Volume{
		{
			Name: "tmp-volume",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
		{
			Name: "kubernetes-dashboard-certs",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: "kubernetes-dashboard-certs",
				},
			},
		},
	}

	volumeMounts = mergeVolumeMounts(volumeMounts, k.component.Spec.VolumeMounts)
	volumes = mergeVolumes(volumes, k.component.Spec.Volumes)

	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      KubernetesDashboardName,
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
					Name:   KubernetesDashboardName,
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: "rainbond-operator",
					Tolerations: []corev1.Toleration{
						{
							Operator: corev1.TolerationOpExists, // tolerate everything.
						},
					},
					ImagePullSecrets:              imagePullSecrets(k.component, k.cluster),
					TerminationGracePeriodSeconds: commonutil.Int64(0),
					Containers: []corev1.Container{
						{
							Name:            KubernetesDashboardName,
							Image:           k.component.Spec.Image,
							ImagePullPolicy: k.component.ImagePullPolicy(),
							Args:            args,
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 8443,
									Protocol:      corev1.ProtocolTCP,
								},
								{
									ContainerPort: 9090,
									Protocol:      corev1.ProtocolTCP,
								},
							},
							SecurityContext: &corev1.SecurityContext{
								Privileged: commonutil.Bool(true),
							},
							VolumeMounts: volumeMounts,
							Resources:    k.component.Spec.Resources,
						},
					},
					Volumes:  volumes,
					Affinity: k.component.Spec.Affinity,
				},
			},
		},
	}

	return deploy
}

func (k *k8sdashbaord) serviceForDashbaord() interface{} {
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      KubernetesDashboardName,
			Namespace: k.component.Namespace,
			Labels:    k.labels,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Port: 443,
					TargetPort: intstr.IntOrString{
						IntVal: 9090,
					},
				},
			},
			Selector: k.labels,
		},
	}

	return svc
}

func (k *k8sdashbaord) serviceForKubernetesDashboardCerts() interface{} {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kubernetes-dashboard-certs",
			Namespace: k.component.Namespace,
			Labels:    k.labels,
		},
	}
	return secret
}

func (k *k8sdashbaord) serviceForKubernetesDashboardCsrf() interface{} {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kubernetes-dashboard-csrf",
			Namespace: k.component.Namespace,
			Labels:    k.labels,
		},
		Data: map[string][]byte{
			"csrf": []byte(""),
		},
	}
	return secret
}

func (k *k8sdashbaord) serviceForKubernetesDashboardKeyHolder() interface{} {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kubernetes-dashboard-key-holder",
			Namespace: k.component.Namespace,
			Labels:    k.labels,
		},
	}
	return secret
}
