package handler

import (
	"context"

	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type aliyunCSINasProvisioner struct {
	ctx    context.Context
	client client.Client

	component *rainbondv1alpha1.RbdComponent
	labels    map[string]string
}

var _ ComponentHandler = &aliyunCSINasProvisioner{}

// NewAliyunCSINasProvisioner creates a new aliyun csi nas provisioner handler.
func NewAliyunCSINasProvisioner(ctx context.Context, client client.Client, component *rainbondv1alpha1.RbdComponent, cluster *rainbondv1alpha1.RainbondCluster) ComponentHandler {
	return &aliyunCSINasProvisioner{
		ctx:       ctx,
		client:    client,
		component: component,
		labels:    LabelsForRainbondComponent(component),
	}
}

func (h *aliyunCSINasProvisioner) Before() error {
	return nil
}

func (h *aliyunCSINasProvisioner) Resources() []interface{} {
	return nil
}

func (h *aliyunCSINasProvisioner) After() error {
	return nil
}

func (h *aliyunCSINasProvisioner) ListPods() ([]corev1.Pod, error) {
	return listPods(h.ctx, h.client, h.component.Namespace, h.labels)
}
