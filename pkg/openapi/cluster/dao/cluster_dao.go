package dao

import (
	"github.com/GLYASAI/rainbond-operator/pkg/openapi/model"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/GLYASAI/rainbond-operator/cmd/openapi/option"
	"github.com/GLYASAI/rainbond-operator/pkg/util/suffixdomain"
)

// SuffixHTTPHostDao suffix http host dao
type SuffixHTTPHostDao interface {
	Get() (*model.SuffixHTTPHost, error)
	Generate(iip string) (*model.SuffixHTTPHost, error)
	Update(data *model.SuffixHTTPHost) error
}

// SuffixHTTPHostDaoImpl impl
type SuffixHTTPHostDaoImpl struct {
	cfg *option.Config
}

//NewSuffixHTTPHostDao new dao
func NewSuffixHTTPHostDao(cfg *option.Config) *SuffixHTTPHostDaoImpl {
	return &SuffixHTTPHostDaoImpl{cfg: cfg}
}

// Get get suffix http host
func (i *SuffixHTTPHostDaoImpl) Get() (*model.SuffixHTTPHost, error) {
	cm, err := i.cfg.KubeClient.CoreV1().ConfigMaps(i.cfg.Namespace).Get(i.cfg.SuffixHTTPHost, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	if domain, ok := cm.Data["domain"]; ok {
		return &model.SuffixHTTPHost{Domain: domain}, nil
	}
	return nil, nil
}

// Generate generate suffi http host
func (i *SuffixHTTPHostDaoImpl) Generate(iip string) (*model.SuffixHTTPHost, error) {
	suffix, err := i.Get()
	if err != nil && !k8sErrors.IsNotFound(err) {
		return nil, err
	}
	if k8sErrors.IsNotFound(err) {
		// create new configmap
		cm := &corev1.ConfigMap{}
		data := make(map[string]string)
		data["iip"] = iip
		data["uuid"] = ""
		data["secretkey"] = ""
		data["domain"] = ""
		suffixdomain.GenerateDomain(iip)
		cm.Data = data
		_, err = i.cfg.KubeClient.CoreV1().ConfigMaps(i.cfg.Namespace).Create(cm)
		if err != nil {
			return nil, err
		}
	}

	return suffix, nil
}

// Update update
func (i *SuffixHTTPHostDaoImpl) Update(data *model.SuffixHTTPHost) error {
	return nil
}
