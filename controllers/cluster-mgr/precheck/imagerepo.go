package precheck

import (
	"context"
	"fmt"
	"github.com/goodrain/rainbond-operator/util/repositoryutil"
	"time"

	"github.com/goodrain/rainbond-operator/util/constants"
	"github.com/goodrain/rainbond-operator/util/rbdutil"

	"github.com/go-logr/logr"
	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type imagerepo struct {
	ctx     context.Context
	log     logr.Logger
	cluster *rainbondv1alpha1.RainbondCluster
}

// NewImageRepoPrechecker creates a new prechecker.
func NewImageRepoPrechecker(ctx context.Context, log logr.Logger, cluster *rainbondv1alpha1.RainbondCluster) PreChecker {
	l := log.WithName("ImageRepoPreChecker")
	return &imagerepo{
		ctx:     ctx,
		log:     l,
		cluster: cluster,
	}
}

func (d *imagerepo) Check() rainbondv1alpha1.RainbondClusterCondition {
	condition := rainbondv1alpha1.RainbondClusterCondition{
		Type:              rainbondv1alpha1.RainbondClusterConditionTypeImageRepository,
		Status:            corev1.ConditionTrue,
		LastHeartbeatTime: metav1.NewTime(time.Now()),
	}

	imageRepo := rbdutil.GetImageRepository(d.cluster)

	if idx, cdt := d.cluster.Status.GetCondition(rainbondv1alpha1.RainbondClusterConditionTypeImageRepository); (idx == -1 || cdt.Reason == "DefaultImageRepoFailed") && imageRepo != constants.DefImageRepository {
		condition.Status = corev1.ConditionFalse
		condition.Reason = "InProgress"
		condition.Message =
			fmt.Sprintf("precheck for %s is in progress", rainbondv1alpha1.RainbondClusterConditionTypeImageRepository)
	}

	// Verify that the image repository is available
	d.log.V(6).Info("login repository", "repository", rbdutil.GetImageRepositoryDomain(d.cluster), "user", d.cluster.Spec.ImageHub.Username)

	if err := repositoryutil.LoginRepository(rbdutil.GetImageRepositoryDomain(d.cluster), d.cluster.Spec.ImageHub.Username, d.cluster.Spec.ImageHub.Password); err != nil {
		return d.failConditoin(condition, err)
	}
	return condition
}

func (d *imagerepo) failConditoin(condition rainbondv1alpha1.RainbondClusterCondition, err error) rainbondv1alpha1.RainbondClusterCondition {
	return failConditoin(condition, "ImageRepoFailed", err.Error())
}
