package precheck

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-logr/logr"
	wutongv1alpha1 "github.com/wutong/wutong-operator/api/v1alpha1"
	"github.com/wutong/wutong-operator/util/k8sutil"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type k8sStatus struct {
	ctx     context.Context
	log     logr.Logger
	client  client.Client
	cluster *wutongv1alpha1.WutongCluster
}

// NewK8sStatusPrechecker creates a new kubernetes status prechecker.
func NewK8sStatusPrechecker(ctx context.Context, cluster *wutongv1alpha1.WutongCluster, client client.Client, log logr.Logger) PreChecker {
	l := log.WithName("k8sStatusPreChecker")
	return &k8sStatus{
		ctx:     ctx,
		log:     l,
		cluster: cluster,
		client:  client,
	}
}

func (k *k8sStatus) Check() wutongv1alpha1.WutongClusterCondition {
	condition := wutongv1alpha1.WutongClusterCondition{
		Type:              wutongv1alpha1.WutongClusterConditionTypeKubernetesStatus,
		Status:            corev1.ConditionTrue,
		LastHeartbeatTime: metav1.NewTime(time.Now()),
	}

	pods, err := k.listNotReadyPods()
	if err != nil {
		return k.failCondition(condition, err.Error())
	}

	if len(pods) != 0 {
		k.log.V(6).Info("unhealthy pods found", "numbers", len(pods))
		msg := notReadyPodsToString(pods)
		return k.failCondition(condition, msg)
	}

	return condition
}

func (k *k8sStatus) listNotReadyPods() ([]corev1.Pod, error) {
	clientSet := k8sutil.GetClientSet()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	podList, err := clientSet.CoreV1().Pods("kube-system").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var pods []corev1.Pod
	for _, item := range podList.Items {
		if k8sutil.IsPodReady(&item) || k8sutil.IsPodCompleted(&item) {
			k.log.V(6).Info("pod is ready", "pod", item.Name)
			continue
		}
		k.log.V(6).Info("pod is not ready", "pod", item.Name)
		pods = append(pods, item)
	}

	return pods, nil
}

func (k *k8sStatus) failCondition(condition wutongv1alpha1.WutongClusterCondition, msg string) wutongv1alpha1.WutongClusterCondition {
	return failConditoin(condition, "KubernetesStatusFailed", msg)
}

func notReadyPodsToString(pods []corev1.Pod) string {
	var podNames []string
	for _, pod := range pods {
		podNames = append(podNames, pod.GetName())
	}
	return fmt.Sprintf("Unhealthy pods found in kube-system: %s", strings.Join(podNames, ","))
}
