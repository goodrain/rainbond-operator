package usecase

import (
	"fmt"
	"github.com/goodrain/rainbond-operator/cmd/openapi/option"
	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	v1 "github.com/goodrain/rainbond-operator/pkg/openapi/types/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	plabels "k8s.io/apimachinery/pkg/labels"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"strings"
)

var log = logf.Log.WithName("usecase_cluster")

type rbdComponentStatusFromSubObject func(cpn *rainbondv1alpha1.RbdComponent) (*v1.RbdComponentStatus, error)

// ComponentUseCase cluster componse case
type ComponentUseCase interface { // TODO: loop call
	Get(name string) (*v1.RbdComponentStatus, error)
	List() ([]*v1.RbdComponentStatus, error)
}

// ComponentUsecaseImpl cluster
type ComponentUsecaseImpl struct {
	cfg *option.Config
}

// NewComponentUsecase new componse case impl
func NewComponentUsecase(cfg *option.Config) *ComponentUsecaseImpl {
	return &ComponentUsecaseImpl{cfg: cfg}
}

// Get get
func (cc *ComponentUsecaseImpl) Get(name string) (*v1.RbdComponentStatus, error) {
	component, err := cc.cfg.RainbondKubeClient.RainbondV1alpha1().RbdComponents(cc.cfg.Namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return cc.typeRbdComponentStatus(component)
}

// List list
func (cc *ComponentUsecaseImpl) List() ([]*v1.RbdComponentStatus, error) {
	reqLogger := log.WithValues("Namespace", cc.cfg.Namespace)
	reqLogger.Info("Start listing RbdComponent associated controller")

	components, err := cc.cfg.RainbondKubeClient.RainbondV1alpha1().RbdComponents(cc.cfg.Namespace).List(metav1.ListOptions{})
	if err != nil {
		reqLogger.Error(err, "Listing RbdComponents")
		return nil, err
	}

	var statues []*v1.RbdComponentStatus
	for _, component := range components.Items {
		var status *v1.RbdComponentStatus
		if component.Status == nil {
			// Initially, status may be nil
			status = &v1.RbdComponentStatus{Name: component.Name}
			continue
		} else {
			status, err = cc.typeRbdComponentStatus(&component)
			if err != nil {
				reqLogger.Error(err, "Get RbdComponent status", "Name", component.Name)
				status = &v1.RbdComponentStatus{Name: component.Name}
			}
		}
		statues = append(statues, status)
	}

	return statues, nil
}

func (cc *ComponentUsecaseImpl) typeRbdComponentStatus(cpn *rainbondv1alpha1.RbdComponent) (*v1.RbdComponentStatus, error) {
	reqLogger := log.WithValues("Namespace", cpn.Namespace, "Name", cpn.Name, "ControllerType", cpn.Status.ControllerType)
	reqLogger.Info("Start getting RbdComponent associated controller")

	k2fn := map[string]rbdComponentStatusFromSubObject{
		rainbondv1alpha1.ControllerTypeDeployment.String():  cc.rbdComponentStatusFromDeployment,
		rainbondv1alpha1.ControllerTypeStatefulSet.String(): cc.rbdComponentStatusFromStatefulSet,
		rainbondv1alpha1.ControllerTypeDaemonSet.String():   cc.rbdComponentStatusFromDaemonSet,
	}
	fn, ok := k2fn[cpn.Status.ControllerType.String()]
	if !ok {
		return nil, fmt.Errorf("unsupportted controller type: %s", cpn.Status.ControllerType.String())
	}

	status, err := fn(cpn)
	if err != nil {
		log.Error(err, "get RbdComponent associated controller")
		return nil, err
	}

	return status, nil
}

func (cc *ComponentUsecaseImpl) rbdComponentStatusFromDeployment(cpn *rainbondv1alpha1.RbdComponent) (*v1.RbdComponentStatus, error) {
	reqLogger := log.WithValues("Namespace", cpn.Namespace, "Name", cpn.Name, "ControllerType", cpn.Status.ControllerType)
	reqLogger.Info("Start getting RbdComponent associated deployment")

	deploy, err := cc.cfg.KubeClient.AppsV1().Deployments(cpn.Namespace).Get(cpn.Status.ControllerName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	status := &v1.RbdComponentStatus{
		Name:          cpn.Name,
		Replicas:      deploy.Status.Replicas,
		ReadyReplicas: deploy.Status.ReadyReplicas,
	}

	labels := deploy.Spec.Template.Labels
	podStatuses, err := cc.listPodStatues(deploy.Namespace, labels)
	if err != nil {
		reqLogger.Error(err, "List deployment associated pods", "labels", labels)
	}
	status.PodStatuses = podStatuses

	return status, nil
}

func (cc *ComponentUsecaseImpl) rbdComponentStatusFromStatefulSet(cpn *rainbondv1alpha1.RbdComponent) (*v1.RbdComponentStatus, error) {
	reqLogger := log.WithValues("Namespace", cpn.Namespace, "Name", cpn.Name, "ControllerType", cpn.Status.ControllerType)
	reqLogger.Info("Start getting RbdComponent associated statefulset")

	sts, err := cc.cfg.KubeClient.AppsV1().StatefulSets(cpn.Namespace).Get(cpn.Status.ControllerName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	status := &v1.RbdComponentStatus{
		Name:          cpn.Name,
		Replicas:      sts.Status.Replicas,
		ReadyReplicas: sts.Status.ReadyReplicas,
	}

	labels := sts.Spec.Template.Labels
	podStatuses, err := cc.listPodStatues(sts.Namespace, labels)
	if err != nil {
		reqLogger.Error(err, "List deployment associated pods", "labels", labels)
	}
	status.PodStatuses = podStatuses

	return status, nil
}

func (cc *ComponentUsecaseImpl) rbdComponentStatusFromDaemonSet(cpn *rainbondv1alpha1.RbdComponent) (*v1.RbdComponentStatus, error) {
	reqLogger := log.WithValues("Namespace", cpn.Namespace, "Name", cpn.Name, "ControllerType", cpn.Status.ControllerType)
	reqLogger.Info("Start getting RbdComponent associated daemonset")

	ds, err := cc.cfg.KubeClient.AppsV1().DaemonSets(cpn.Namespace).Get(cpn.Status.ControllerName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	status := &v1.RbdComponentStatus{
		Name:          cpn.Name,
		Replicas:      ds.Status.DesiredNumberScheduled,
		ReadyReplicas: ds.Status.NumberAvailable,
	}

	labels := ds.Spec.Template.Labels
	podStatuses, err := cc.listPodStatues(ds.Namespace, labels)
	if err != nil {
		reqLogger.Error(err, "List deployment associated pods", "labels", labels)
	}
	status.PodStatuses = podStatuses

	return status, nil
}

func (cc *ComponentUsecaseImpl) listPodStatues(namespace string, labels map[string]string) ([]v1.PodStatus, error) {
	selector := plabels.SelectorFromSet(labels)
	opts := metav1.ListOptions{
		LabelSelector: selector.String(),
	}
	podList, err := cc.cfg.KubeClient.CoreV1().Pods(namespace).List(opts)
	if err != nil {
		return nil, err
	}

	var podStatuses []v1.PodStatus
	for _, pod := range podList.Items {
		podStatus := v1.PodStatus{
			Name:    pod.Name,
			Phase:   string(pod.Status.Phase),
			HostIP:  pod.Status.HostIP,
			Reason:  pod.Status.Reason,
			Message: pod.Status.Message,
		}
		var containerStatuses []v1.PodContainerStatus
		for _, cs := range pod.Status.ContainerStatuses {
			containerStatus := v1.PodContainerStatus{
				Image: cs.Image,
				Ready: cs.Ready,
			}
			if cs.ContainerID != "" {
				containerStatus.ContainerID = strings.Replace(cs.ContainerID, "docker://", "", -1)[0:8]
			}

			// TODO: move out
			if cs.State.Running != nil {
				containerStatus.State = "Running"
			}
			if cs.State.Waiting != nil {
				containerStatus.State = "Waiting"
				containerStatus.Reason = cs.State.Waiting.Reason
				containerStatus.Message = cs.State.Waiting.Message
			}
			if cs.State.Terminated != nil {
				containerStatus.State = "Terminated"
				containerStatus.Reason = cs.State.Terminated.Reason
				containerStatus.Message = cs.State.Terminated.Message
			}

			containerStatuses = append(containerStatuses, containerStatus)
		}
		podStatus.ContainerStatuses = containerStatuses
		podStatuses = append(podStatuses, podStatus)
	}

	return podStatuses, nil
}
