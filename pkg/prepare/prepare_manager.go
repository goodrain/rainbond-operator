package prepare

import (
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	rainbondv1alpha1 "github.com/GLYASAI/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	"github.com/GLYASAI/rainbond-operator/pkg/generated/clientset/versioned"
)

var log = logf.Log.WithName("prepare_manager") // TODO

type PrepareManager struct {
}

// CreatePrepareManager creates a new PrepareController.
func CreatePrepareManager() *PrepareManager {
	return &PrepareManager{}
}

func PrepareGlobalConfig(cfg *rest.Config) error {
	reqLogger := log.WithValues("Prepare", "Global Config")

	clientset := versioned.NewForConfigOrDie(cfg)

	globalConfig := &rainbondv1alpha1.GlobalConfig{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "rbd-system",
			Name:      "rbd-globalconfig",
		},
		Status: rainbondv1alpha1.GlobalConfigStatus{
			Phase: rainbondv1alpha1.GlobalConfigPhasePending,
		},
	}

	_, err := clientset.RainbondV1alpha1().GlobalConfigs(globalConfig.Namespace).Get(globalConfig.GetName(), metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			reqLogger.Info("GlobalConfig not found. will create a new one.")
			_, err := clientset.RainbondV1alpha1().GlobalConfigs(globalConfig.Namespace).Create(globalConfig)
			if err != nil {
				reqLogger.Error(err, "create global config")
				return err
			}
			return nil
		}
		return err
	}

	return nil
}
