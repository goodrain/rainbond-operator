package handler

import (
	"context"
	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var AppStoreUser = "appstore-user-ui"

type user struct {
	ctx       context.Context
	labels    map[string]string
	client    client.Client
	component *rainbondv1alpha1.RbdComponent
	cluster   *rainbondv1alpha1.RainbondCluster
}

// NewUser -
func NewUser(ctx context.Context, client client.Client, component *rainbondv1alpha1.RbdComponent, cluster *rainbondv1alpha1.RainbondCluster) ComponentHandler {
	return &user{
		ctx:       ctx,
		client:    client,
		cluster:   cluster,
		component: component,
		labels:    LabelsForRainbondComponent(component),
	}
}

func (a *user) Before() error {
	return nil
}

func (a *user) Resources() []client.Object {
	return []client.Object{
		a.deploy(),
		a.service(),
	}
}

func (a *user) After() error {
	return nil
}

func (a *user) ListPods() ([]corev1.Pod, error) {
	return listPods(a.ctx, a.client, a.component.Namespace, a.labels)
}

func (a *user) deploy() client.Object {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      AppStoreUser,
			Namespace: a.component.Namespace,
			Labels:    a.labels,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: a.labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:      AppStoreUser,
					Namespace: a.component.Namespace,
					Labels:    a.labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            AppStoreUser,
							Image:           a.component.Spec.Image,
							ImagePullPolicy: a.component.ImagePullPolicy(),
						},
					},
				},
			},
		},
	}
}

func (a *user) service() client.Object {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      AppStoreUser,
			Namespace: a.component.Namespace,
			Labels:    a.labels,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name: AppStoreUser,
					Port: int32(80),
					TargetPort: intstr.IntOrString{
						IntVal: int32(80),
					},
				},
			},
			Selector: a.labels,
		},
	}
}
