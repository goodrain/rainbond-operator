package prepare

import (
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	rainbondv1alpha1 "github.com/GLYASAI/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	rainbondv1alpha1client "github.com/GLYASAI/rainbond-operator/pkg/generated/clientset/versioned"
	"github.com/GLYASAI/rainbond-operator/pkg/util/constants"
	"github.com/GLYASAI/rainbond-operator/pkg/util/k8sutil"
)

var log = logf.Log.WithName("prepare_manager")

type Manager struct {
	clientset kubernetes.Interface

	rainbondv1aphal1clientset rainbondv1alpha1client.Interface

	client client.Client
}

// CreatePrepareManager creates a new PrepareController.
func CreatePrepareManager(cfg *rest.Config, client client.Client) *Manager {
	log.Info("create prepare manager")

	clientset := kubernetes.NewForConfigOrDie(cfg)
	rainbondv1aphal1clientset := rainbondv1alpha1client.NewForConfigOrDie(cfg)

	return &Manager{
		clientset:                 clientset,
		client:                    client,
		rainbondv1aphal1clientset: rainbondv1aphal1clientset,
	}
}

func (m *Manager) Start() error {
	log.Info("start prepare manager")
	// if err := m.initNamespace(); err != nil {
	// 	return err
	// }
	if err := m.prepareGlobalConfig(); err != nil {
		return err
	}
	return nil
}

func (m *Manager) Stop() error {
	// TODO: create resources created by Start()
	return nil
}

func (m *Manager) initNamespace() error {
	log.Info("Initiate namepace", "Namespace", constants.Namespace)
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: constants.Namespace,
		},
	}
	_, err := k8sutil.UpdateOrCreateResource(log, m.client, ns, ns.GetObjectMeta()) // TODO: no need reconcile
	if err != nil {
		log.Error(err, "update or create namespace")
		return fmt.Errorf("update or create namespace: %v", err)
	}
	return err
}

func (m *Manager) prepareGlobalConfig() error {
	log.Info("prepare global config")

	globalConfig := &rainbondv1alpha1.GlobalConfig{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "rbd-system",
			Name:      "rbd-globalconfig",
		},
		Status: rainbondv1alpha1.GlobalConfigStatus{
			Phase: rainbondv1alpha1.GlobalConfigPhasePending,
		},
	}

	_, err := m.rainbondv1aphal1clientset.RainbondV1alpha1().GlobalConfigs(globalConfig.Namespace).Get(globalConfig.GetName(), metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("GlobalConfig not found. will create a new one.")
			_, err := m.rainbondv1aphal1clientset.RainbondV1alpha1().GlobalConfigs(globalConfig.Namespace).Create(globalConfig)
			if err != nil {
				log.Error(err, "create global config")
				return err
			}
			return nil
		}
		return err
	}

	return nil
}
