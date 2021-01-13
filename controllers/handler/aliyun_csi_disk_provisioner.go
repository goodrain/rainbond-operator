package handler

import (
	"context"

	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type aliyunCSIDiskProvisioner struct {
	ctx    context.Context
	client client.Client

	component *rainbondv1alpha1.RbdComponent
	labels    map[string]string
}

var _ ComponentHandler = &aliyunCSIDiskProvisioner{}

// NewaliyunCSIDiskProvisioner creates a new aliyun csi disk provisioner handler.
func NewaliyunCSIDiskProvisioner(ctx context.Context, client client.Client, component *rainbondv1alpha1.RbdComponent, cluster *rainbondv1alpha1.RainbondCluster) ComponentHandler {
	return &aliyunCSIDiskProvisioner{
		ctx:       ctx,
		client:    client,
		component: component,
		labels:    LabelsForRainbondComponent(component),
	}
}

func (h *aliyunCSIDiskProvisioner) Before() error {
	return nil
}

func (h *aliyunCSIDiskProvisioner) Resources() []client.Object {
	return nil
}

func (h *aliyunCSIDiskProvisioner) After() error {
	return nil
}

func (h *aliyunCSIDiskProvisioner) ListPods() ([]corev1.Pod, error) {
	return listPods(h.ctx, h.client, h.component.Namespace, h.labels)
}
