package handler

import (
	"context"

	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NFSName name for nfs provisioner
var NFSName = "nfs-provisioner"

type nfs struct {
	ctx    context.Context
	client client.Client

	component *rainbondv1alpha1.RbdComponent
	labels    map[string]string
}

var _ ComponentHandler = &nfs{}

// NewNFS creates a new rbd-nfs handler.
func NewNFS(ctx context.Context, client client.Client, component *rainbondv1alpha1.RbdComponent, cluster *rainbondv1alpha1.RainbondCluster) ComponentHandler {
	return &nfs{
		ctx:       ctx,
		client:    client,
		component: component,
		labels:    LabelsForRainbondComponent(component),
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
