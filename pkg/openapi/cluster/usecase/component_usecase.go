package usecase

import (
	"fmt"

	"github.com/goodrain/rainbond-operator/pkg/library/bcode"

	"github.com/goodrain/rainbond-operator/cmd/openapi/option"
	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	"github.com/goodrain/rainbond-operator/pkg/openapi/cluster"
	v1 "github.com/goodrain/rainbond-operator/pkg/openapi/types/v1"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/reference"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var log = logf.Log.WithName("usecase_cluster")

type rbdComponentStatusFromSubObject func(cpn *rainbondv1alpha1.RbdComponent) (*v1.RbdComponentStatus, error)

type componentUsecase struct {
	cfg *option.Config
}

// NewComponentUsecase new component usecase.
func NewComponentUsecase(cfg *option.Config) cluster.ComponentUsecase {
	return &componentUsecase{cfg: cfg}
}

// Get get
func (cc *componentUsecase) Get(name string) (*v1.RbdComponentStatus, error) {
	cpn, err := cc.cfg.RainbondKubeClient.RainbondV1alpha1().RbdComponents(cc.cfg.Namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			return nil, bcode.ErrRbdComponentNotFound
		}
		return nil, err
	}

	pods, err := cc.listPods(cpn)
	if err != nil {
		return nil, fmt.Errorf("list pods: %v", err)
	}

	status := cc.convertRbdComponent(cpn, pods)

	return status, nil
}

func (cc *componentUsecase) listPods(cpn *rainbondv1alpha1.RbdComponent) ([]*corev1.Pod, error) {
	if cpn.Status == nil {
		return nil, nil
	}
	var pods []*corev1.Pod
	for _, ref := range cpn.Status.Pods {
		pod, err := cc.cfg.KubeClient.CoreV1().Pods(cc.cfg.Namespace).Get(ref.Name, metav1.GetOptions{})
		if err != nil {
			if k8sErrors.IsNotFound(err) {
				log.V(3).Info("pod reated to cpn not found", "namespace", cc.cfg.Namespace, "name", ref.Name)
				continue
			}
			return nil, fmt.Errorf("get pod: %v", err)
		}
		pods = append(pods, pod)
	}
	return pods, nil
}

// List list
func (cc *componentUsecase) List(isInit bool) ([]*v1.RbdComponentStatus, error) {
	listOption := metav1.ListOptions{}
	if isInit {
		log.Info("get init component status list")
		listOption.LabelSelector = "priorityComponent=true"
	}
	components, err := cc.cfg.RainbondKubeClient.RainbondV1alpha1().RbdComponents(cc.cfg.Namespace).List(listOption)
	if err != nil {
		return nil, fmt.Errorf("list rainbond components: %v", err)
	}

	var statues []*v1.RbdComponentStatus
	for _, cpn := range components.Items {
		pods, err := cc.listPods(&cpn)
		if err != nil {
			return nil, fmt.Errorf("list pods: %v", err)
		}

		status := cc.convertRbdComponent(&cpn, pods)

		statues = append(statues, status)
	}

	return statues, nil
}

func (cc *componentUsecase) convertRbdComponent(cpn *rainbondv1alpha1.RbdComponent, pods []*corev1.Pod) *v1.RbdComponentStatus {
	var replicas int32 = 1 // defualt replicas is 1
	if cpn.Status != nil {
		replicas = cpn.Status.Replicas
	}

	result := &v1.RbdComponentStatus{
		Name:            cpn.Name,
		ISInitComponent: cpn.Spec.PriorityComponent,
	}
	result.Replicas = replicas

	if cpn.Status != nil {
		result.ReadyReplicas = cpn.Status.ReadyReplicas
	}

	result.Status = v1.ComponentStatusCreating
	if result.Replicas == result.ReadyReplicas && result.Replicas > 0 {
		result.Status = v1.ComponentStatusRunning
	}

	podStatuses := cc.convertPodStatues(pods)
	for index := range podStatuses {
		if podStatuses[index].Phase == "NotReady" {
			result.Status = v1.ComponentStatusCreating // if pod not ready, component status can't be running, even nor replicas equals to ready replicas
		}
	}
	result.PodStatuses = podStatuses

	return result
}

func (cc *componentUsecase) convertPodStatues(pods []*corev1.Pod) []v1.PodStatus {
	var podStatuses []v1.PodStatus
	for _, pod := range pods {
		podStatus := v1.PodStatus{
			Name:    pod.Name,
			Phase:   "NotReady", // default phase NotReady, util PodReady condition is true
			HostIP:  pod.Status.HostIP,
			Reason:  pod.Status.Reason,
			Message: pod.Status.Message,
		}
		for _, condition := range pod.Status.Conditions {
			if condition.Type == corev1.PodReady && condition.Status == "True" {
				podStatus.Phase = "Ready"
				break
			}
		}
		if podStatus.Phase != "Ready" {
			podStatus.Reason, podStatus.Message = cc.getPodEvents(pod)
		}
		podStatuses = append(podStatuses, podStatus)
	}

	return podStatuses
}

func (cc *componentUsecase) getPodEvents(pod *corev1.Pod) (string, string) {
	ref, err := reference.GetReference(scheme.Scheme, pod)
	if err != nil {
		log.V(3).Info("get pod[%s] event list failed: %s", pod.Name, err.Error())
		return "", ""
	}
	ref.Kind = ""
	if _, isMirrorPod := pod.Annotations[corev1.MirrorPodAnnotationKey]; isMirrorPod {
		ref.UID = types.UID(pod.Annotations[corev1.MirrorPodAnnotationKey])
	}
	events, err := cc.cfg.KubeClient.CoreV1().Events(pod.GetNamespace()).Search(scheme.Scheme, ref)
	if err != nil {
		log.V(3).Info("search pod[%s] event list failed: %s", pod.Name, err.Error())
		return "", ""
	}
	if events == nil {
		return "", ""
	}
	return cc.convertEventMessage(events.Items)
}

func (cc *componentUsecase) convertEventMessage(events []corev1.Event) (string, string) {
	warnings := []corev1.Event{}
	for _, event := range events {
		switch event.Type {
		case corev1.EventTypeWarning:
			warnings = append(warnings, event)
		}
	}
	if len(warnings) == 0 {
		return "", ""
	}
	// get the latest event
	latest := warnings[0]
	for _, event := range warnings {
		if event.LastTimestamp.Time.After(latest.LastTimestamp.Time) {
			latest = event
		}
	}
	return latest.Reason, latest.Message
}
