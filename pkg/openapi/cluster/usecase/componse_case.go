package usecase

import (
	"github.com/GLYASAI/rainbond-operator/cmd/openapi/option"
	"github.com/GLYASAI/rainbond-operator/pkg/openapi/model"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ComponseCase cluster componse case
type ComponseCase interface { // TODO: loop call
	Get(name string) (*model.ComponseInfo, error)
	List() ([]*model.ComponseInfo, error)
}

// ComponseCaseImpl cluster
type ComponseCaseImpl struct {
	cfg option.Config
}

// NewComponseCase new componse case impl
func NewComponseCase(cfg option.Config) *ComponseCaseImpl {
	return &ComponseCaseImpl{cfg: cfg}
}

// Get get
func (cc *ComponseCaseImpl) Get(name string) (*model.ComponseInfo, error) {
	componse, err := cc.cfg.RainbondKubeClient.RainbondV1alpha1().RbdComponents(cc.cfg.Namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	info := &model.ComponseInfo{
		Name:    componse.Name,
		Version: componse.Spec.Version,
		// Status:      string(componse.Status.Phase),// TODO fanyangyang not ready
		// HealthCount: len(componse.Status.PodStatus.Healthy),
		// TotalCount:  len(componse.Status.PodStatus.Healthy) + len(componse.Status.PodStatus.UnHealthy),
		// Message:     componse.Status.Message,
	}
	return info, nil
}

// List list
func (cc *ComponseCaseImpl) List() ([]*model.ComponseInfo, error) {
	componseList, err := cc.cfg.RainbondKubeClient.RainbondV1alpha1().RbdComponents(cc.cfg.Namespace).List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	componseInfos := make([]*model.ComponseInfo, 0)
	for _, item := range componseList.Items {
		info := &model.ComponseInfo{
			Name:    item.Name,
			Version: item.Spec.Version,
		}
		if item.Status != nil {
			// info.Status = string(item.Status.Phase)// TODO fanyangyang not ready
			// info.Message = string(item.Status.Message)
			// if item.Status.PodStatus != nil {
			// 	info.HealthCount = len(item.Status.PodStatus.Ready)
			// 	info.TotalCount = len(item.Status.PodStatus.Ready) + len(item.Status.PodStatus.Unready)
			// }
		}
		componseInfos = append(componseInfos, info)
	}
	return componseInfos, nil
}
