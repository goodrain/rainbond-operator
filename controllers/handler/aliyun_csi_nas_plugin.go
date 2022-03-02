package handler

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	wutongv1alpha1 "github.com/wutong/wutong-operator/api/v1alpha1"
	"github.com/wutong/wutong-operator/util/commonutil"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type aliyunCsiNasPlugin struct {
	ctx    context.Context
	client client.Client
	log    logr.Logger

	component *wutongv1alpha1.WutongComponent
	labels    map[string]string
}

var _ ComponentHandler = &aliyunCsiNasPlugin{}
var _ Replicaser = &aliyunCsiNasPlugin{}

// NewAliyunCSINasPlugin creates a new aliyun csi nas plugin handler.
func NewAliyunCSINasPlugin(ctx context.Context, client client.Client, component *wutongv1alpha1.WutongComponent, cluster *wutongv1alpha1.WutongCluster) ComponentHandler {
	return &aliyunCsiNasPlugin{
		ctx:       ctx,
		client:    client,
		log:       log.WithValues("Name", component.Name),
		component: component,
		labels:    LabelsForWutongComponent(component),
	}
}

func (h *aliyunCsiNasPlugin) Before() error {
	return nil
}

func (h *aliyunCsiNasPlugin) Resources() []client.Object {
	return nil
}

func (h *aliyunCsiNasPlugin) After() error {
	return nil
}

func (h *aliyunCsiNasPlugin) ListPods() ([]corev1.Pod, error) {
	return listPods(h.ctx, h.client, h.component.Namespace, h.labels)
}

func (h *aliyunCsiNasPlugin) Replicas() *int32 {
	nodeList := &corev1.NodeList{}
	if err := h.client.List(h.ctx, nodeList); err != nil {
		h.log.V(6).Info(fmt.Sprintf("list nodes: %v", err))
		return nil
	}
	return commonutil.Int32(int32(len(nodeList.Items)))
}
