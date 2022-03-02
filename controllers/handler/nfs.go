package handler

import (
	"context"

	wutongv1alpha1 "github.com/wutong/wutong-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NFSName name for nfs provisioner
var NFSName = "nfs-provisioner"

type nfs struct {
	ctx    context.Context
	client client.Client

	component *wutongv1alpha1.WutongComponent
	labels    map[string]string
}

var _ ComponentHandler = &nfs{}

// NewNFS creates a new wt-nfs handler.
func NewNFS(ctx context.Context, client client.Client, component *wutongv1alpha1.WutongComponent, cluster *wutongv1alpha1.WutongCluster) ComponentHandler {
	return &nfs{
		ctx:       ctx,
		client:    client,
		component: component,
		labels:    LabelsForWutongComponent(component),
	}
}

func (h *nfs) Before() error {
	return nil
}

func (h *nfs) Resources() []client.Object {
	return nil
}

func (h *nfs) After() error {
	return nil
}

func (h *nfs) ListPods() ([]corev1.Pod, error) {
	return listPods(h.ctx, h.client, h.component.Namespace, h.labels)
}
