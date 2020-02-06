package crutil

import (
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	"github.com/goodrain/rainbond-operator/pkg/generated/clientset/versioned"
)

// HandleClusterError handler cluster rror
func HandleClusterError(client versioned.Interface, err error) error {
	if k8sErrors.IsNotFound(err) {
		rainbondCluster := &rainbondv1alpha1.RainbondCluster{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "rbd-system",
				Name:      "rainbondcluster",
			},
			Status: &rainbondv1alpha1.RainbondClusterStatus{
				Phase: rainbondv1alpha1.RainbondClusterWaiting,
			},
		}
		_, err = client.RainbondV1alpha1().RainbondClusters(rainbondCluster.Namespace).Create(rainbondCluster)
		return err
	}
	return err
}
