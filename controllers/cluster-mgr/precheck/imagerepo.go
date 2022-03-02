package precheck

import (
	"context"
	"fmt"
	"path"
	"time"

	"github.com/docker/docker/client"
	"github.com/wutong/wutong-operator/util/constants"
	"github.com/wutong/wutong-operator/util/imageutil"
	"github.com/wutong/wutong-operator/util/wtutil"

	"github.com/go-logr/logr"
	wutongv1alpha1 "github.com/wutong/wutong-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type imagerepo struct {
	ctx     context.Context
	log     logr.Logger
	cluster *wutongv1alpha1.WutongCluster
}

// NewImageRepoPrechecker creates a new prechecker.
func NewImageRepoPrechecker(ctx context.Context, log logr.Logger, cluster *wutongv1alpha1.WutongCluster) PreChecker {
	l := log.WithName("ImageRepoPreChecker")
	return &imagerepo{
		ctx:     ctx,
		log:     l,
		cluster: cluster,
	}
}

func (d *imagerepo) Check() wutongv1alpha1.WutongClusterCondition {
	condition := wutongv1alpha1.WutongClusterCondition{
		Type:              wutongv1alpha1.WutongClusterConditionTypeImageRepository,
		Status:            corev1.ConditionTrue,
		LastHeartbeatTime: metav1.NewTime(time.Now()),
	}

	imageRepo := wtutil.GetImageRepository(d.cluster)

	if idx, cdt := d.cluster.Status.GetCondition(wutongv1alpha1.WutongClusterConditionTypeImageRepository); (idx == -1 || cdt.Reason == "DefaultImageRepoFailed") && imageRepo != constants.DefImageRepository {
		condition.Status = corev1.ConditionFalse
		condition.Reason = "InProgress"
		condition.Message =
			fmt.Sprintf("precheck for %s is in progress", wutongv1alpha1.WutongClusterConditionTypeImageRepository)
	}

	localImage := path.Join(d.cluster.Spec.WutongImageRepository, "smallimage")
	remoteImage := path.Join(imageRepo, "smallimage")

	dockerClient, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return d.failConditoin(condition, err)
	}
	dockerClient.NegotiateAPIVersion(d.ctx)

	exists, err := imageutil.CheckIfImageExists(d.ctx, dockerClient, localImage)
	if err != nil {
		return d.failConditoin(condition, fmt.Errorf("check if image %s exists: %v", remoteImage, err))
	}

	if !exists {
		if err := imageutil.ImagePull(d.ctx, dockerClient, localImage); err != nil {
			return d.failConditoin(condition, fmt.Errorf("pull image %s: %v", localImage, err))
		}
	}
	if err := dockerClient.ImageTag(d.ctx, localImage, remoteImage); err != nil {
		return d.failConditoin(condition, fmt.Errorf("tag image %s to %s: %v", localImage, remoteImage, err))
	}

	// push a small image to check the given image repository
	d.log.V(6).Info("push image", "image", remoteImage, "repository", imageRepo, "user", d.cluster.Spec.ImageHub.Username)
	if err := imageutil.ImagePush(d.ctx, dockerClient, remoteImage, imageRepo,
		d.cluster.Spec.ImageHub.Username, d.cluster.Spec.ImageHub.Password); err != nil {
		condition = d.failConditoin(condition, fmt.Errorf("push image: %v", err))
		if imageRepo == constants.DefImageRepository {
			condition.Reason = "DefaultImageRepoFailed"
		}
		return condition
	}

	return condition
}

func (d *imagerepo) failConditoin(condition wutongv1alpha1.WutongClusterCondition, err error) wutongv1alpha1.WutongClusterCondition {
	return failConditoin(condition, "ImageRepoFailed", err.Error())
}
