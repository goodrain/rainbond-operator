package handler

import (
	"context"

	wutongv1alpha1 "github.com/wutong-paas/wutong-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type aliyunCSINasProvisioner struct {
	ctx    context.Context
	client client.Client

	component *wutongv1alpha1.WutongComponent
	labels    map[string]string
}

var _ ComponentHandler = &aliyunCSINasProvisioner{}

// NewAliyunCSINasProvisioner creates a new aliyun csi nas provisioner handler.
func NewAliyunCSINasProvisioner(ctx context.Context, client client.Client, component *wutongv1alpha1.WutongComponent, cluster *wutongv1alpha1.WutongCluster) ComponentHandler {
	return &aliyunCSINasProvisioner{
		ctx:       ctx,
		client:    client,
		component: component,
		labels:    LabelsForWutongComponent(component),
	}
}

func (h *aliyunCSINasProvisioner) Before() error {
	return nil
}

func (h *aliyunCSINasProvisioner) Resources() []client.Object {
	return nil
}

func (h *aliyunCSINasProvisioner) After() error {
	return nil
}

func (h *aliyunCSINasProvisioner) ListPods() ([]corev1.Pod, error) {
	return listPods(h.ctx, h.client, h.component.Namespace, h.labels)
}
