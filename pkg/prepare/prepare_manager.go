package prepare

import (
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	rainbondv1alpha1 "github.com/GLYASAI/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	rainbondv1alpha1client "github.com/GLYASAI/rainbond-operator/pkg/generated/clientset/versioned"
)

var log = logf.Log.WithName("prepare_manager")

type Manager struct {
	clientset kubernetes.Interface

	rainbondv1aphal1clientset rainbondv1alpha1client.Interface

	client client.Client
}

// NewPrepareManager creates a new PrepareController.
func NewPrepareManager(cfg *rest.Config, client client.Client) *Manager {
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
	if err := m.prepareRainbondCluster(); err != nil {
		return err
	}
	return nil
}

func (m *Manager) Stop() error {
	// TODO: create resources created by Start()
	return nil
}

func (m *Manager) prepareRainbondCluster() error {
	log.Info("prepare rainbond cluster")

	rainbondCluster := &rainbondv1alpha1.RainbondCluster{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "rbd-system",
			Name:      "rainbondcluster",
		},
		Status: &rainbondv1alpha1.RainbondClusterStatus{
			Phase: rainbondv1alpha1.RainbondClusterWaiting,
		},
	}

	_, err := m.rainbondv1aphal1clientset.RainbondV1alpha1().RainbondClusters(rainbondCluster.Namespace).Get(rainbondCluster.GetName(), metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("RainbondCluster not found, create a new one.")
			rc, err := m.rainbondv1aphal1clientset.RainbondV1alpha1().RainbondClusters(rainbondCluster.Namespace).Create(rainbondCluster)
			if err != nil {
				log.Error(err, "Error creating rainbondcluster.")
				return err
			}
			rainbondCluster.ResourceVersion = rc.ResourceVersion
			if _, err := m.rainbondv1aphal1clientset.RainbondV1alpha1().RainbondClusters(rainbondCluster.Namespace).UpdateStatus(rainbondCluster); err != nil {
				log.Error(err, "Error updating rainbondcluster status.")
				return err
			}
			return nil
		}
		return err
	}

	return nil
}
