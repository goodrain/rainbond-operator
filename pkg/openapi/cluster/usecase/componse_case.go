package usecase

import (
	"github.com/GLYASAI/rainbond-operator/pkg/generated/clientset/versioned"
	"github.com/GLYASAI/rainbond-operator/pkg/openapi/model"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// CompnseCaseGetter componse case getter
type CompnseCaseGetter interface {
	Componses() ComponseCase
}

// ComponseCase cluster componse case
type ComponseCase interface {
	Get(name string) (*model.ComponseInfo, error)
	List() ([]*model.ComponseInfo, error)
}

// ComponseCaseImpl cluster
type ComponseCaseImpl struct {
	normalClientset *kubernetes.Clientset
	rbdClientset    *versioned.Clientset
	namespace       string
}

// NewComponseCase new componse case impl
func NewComponseCase(namespace string, normalClientset *kubernetes.Clientset, rbdClientset *versioned.Clientset) *ComponseCaseImpl {
	return &ComponseCaseImpl{normalClientset: normalClientset, rbdClientset: rbdClientset, namespace: namespace}
}

// Get get
func (cc *ComponseCaseImpl) Get(name string) (*model.ComponseInfo, error) {
	componse, err := cc.rbdClientset.RainbondV1alpha1().RbdComponents(cc.namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	info := &model.ComponseInfo{
		Name:        componse.Name,
		Version:     componse.Spec.Version,
		Status:      string(componse.Status.Phase),
		HealthCount: len(componse.Status.PodStatus.Healthy),
		TotalCount:  len(componse.Status.PodStatus.Healthy) + len(componse.Status.PodStatus.UnHealthy),
		Message:     componse.Status.Message,
	}
	return info, nil
}

// List list
func (cc *ComponseCaseImpl) List() ([]*model.ComponseInfo, error) {
	componseList, err := cc.rbdClientset.RainbondV1alpha1().RbdComponents(cc.namespace).List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	componseInfos := make([]*model.ComponseInfo, 0)
	for _, item := range componseList.Items {
		info := &model.ComponseInfo{
			Name:        item.Name,
			Version:     item.Spec.Version,
			Status:      string(item.Status.Phase),
			HealthCount: len(item.Status.PodStatus.Healthy),
			TotalCount:  len(item.Status.PodStatus.Healthy) + len(item.Status.PodStatus.UnHealthy),
			Message:     item.Status.Message,
		}
		componseInfos = append(componseInfos, info)
	}
	return componseInfos, nil
}
