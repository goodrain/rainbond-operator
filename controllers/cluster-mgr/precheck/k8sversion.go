package precheck

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	wutongv1alpha1 "github.com/wutong-paas/wutong-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type k8sversion struct {
	ctx    context.Context
	log    logr.Logger
	client client.Client
}

// NewK8sVersionPrechecker creates a new kubernetes version prechecker.
func NewK8sVersionPrechecker(ctx context.Context, log logr.Logger, client client.Client) PreChecker {
	l := log.WithName("K8sVersionPreChecker")
	return &k8sversion{
		ctx:    ctx,
		log:    l,
		client: client,
	}
}

func (k *k8sversion) Check() wutongv1alpha1.WutongClusterCondition {
	condition := wutongv1alpha1.WutongClusterCondition{
		Type:              wutongv1alpha1.WutongClusterConditionTypeKubernetesVersion,
		Status:            corev1.ConditionTrue,
		LastHeartbeatTime: metav1.NewTime(time.Now()),
	}

	version, err := k.getKubernetesVersion()
	if err != nil {
		condition.Status = corev1.ConditionFalse
		condition.Reason = "KubernetesVersionFailed"
		condition.Message = err.Error()
		return condition
	}

	if version < "v1.13.0" {
		condition.Status = corev1.ConditionFalse
		condition.Reason = "UnsupportedKubernetesVersion"
		condition.Message = "expect the version of k8s to be greater than or equal to 1.13.0, but got " + version
		return condition
	}

	return condition
}

func (k *k8sversion) getKubernetesVersion() (string, error) {
	nodeList := &corev1.NodeList{}
	var listOpts []client.ListOption
	if err := k.client.List(k.ctx, nodeList, listOpts...); err != nil {
		k.log.Error(err, "list nodes")
		return "", fmt.Errorf("list nodes: %v", err)
	}

	var version string
	for _, node := range nodeList.Items {
		if node.Status.NodeInfo.KubeletVersion == "" {
			continue
		}
		version = node.Status.NodeInfo.KubeletVersion
		break
	}

	if version == "" {
		return "", fmt.Errorf("failed to get kubernetes version")
	}

	return version, nil
}
