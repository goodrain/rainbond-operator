package prepare

import (
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	rainbondv1alpha1 "github.com/GLYASAI/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	rainbondv1alpha1client "github.com/GLYASAI/rainbond-operator/pkg/generated/clientset/versioned"
	"github.com/GLYASAI/rainbond-operator/pkg/util/commonutil"
	"github.com/GLYASAI/rainbond-operator/pkg/util/constants"
	"github.com/GLYASAI/rainbond-operator/pkg/util/k8sutil"
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
	// if err := m.initNamespace(); err != nil {
	// 	return err
	// }
	if err := m.prepareRainbondCluster(); err != nil {
		return err
	}
	if err := m.grdataPersistentVolumeClaim(); err != nil {
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

func (m *Manager) grdataPersistentVolumeClaim() error {
	storageRequest := resource.NewQuantity(10, resource.BinarySI)
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: constants.Namespace,
			Name:      "grdata",
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteMany,
			},
			Resources: corev1.ResourceRequirements{
				Requests: map[corev1.ResourceName]resource.Quantity{
					corev1.ResourceStorage: *storageRequest,
				},
			},
			StorageClassName: commonutil.String("rbd-nfs"), // TODO: do not hard code
		},
	}

	reqLogger := log.WithValues("Namespace", pvc.Namespace, "Name", pvc.Name)

	_, err := m.clientset.CoreV1().PersistentVolumeClaims(pvc.Namespace).Get(pvc.Name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			reqLogger.Info("PersistentVolumeClaim not found, create a new one")
			if _, err := m.clientset.CoreV1().PersistentVolumeClaims(pvc.Namespace).Create(pvc); err != nil {
				reqLogger.Error(err, "Create a new PersistentVolumeClaim")
				return err
			}
			return nil
		}

		reqLogger.Error(err, "Find a PersistentVolumeClaim")
		return err
	}

	// Forbidden: is immutable after creation except resources.requests for bound claims

	return nil
}

func (m *Manager) prepareRainbondCluster() error {
	log.Info("prepare rainbond cluster")

	rainbondCluster := &rainbondv1alpha1.RainbondCluster{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "rbd-system",
			Name:      "rbd-rainbondcluster",
		},
		Status: &rainbondv1alpha1.RainbondClusterStatus{
			Phase: rainbondv1alpha1.RainbondClusterPending,
		},
	}

	_, err := m.rainbondv1aphal1clientset.RainbondV1alpha1().RainbondClusters(rainbondCluster.Namespace).Get(rainbondCluster.GetName(), metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("RainbondCluster not found, create a new one.")
			_, err := m.rainbondv1aphal1clientset.RainbondV1alpha1().RainbondClusters(rainbondCluster.Namespace).Create(rainbondCluster)
			if err != nil {
				log.Error(err, "Create rainbondcluster.")
				return err
			}
			return nil
		}
		return err
	}

	return nil
}
