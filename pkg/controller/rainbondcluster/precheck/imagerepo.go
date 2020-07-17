package precheck

import (
	"context"
	"fmt"
	"github.com/docker/docker/client"
	"github.com/goodrain/rainbond-operator/pkg/util/imageutil"
	"github.com/goodrain/rainbond-operator/pkg/util/rbdutil"
	"path"
	"time"

	"github.com/go-logr/logr"
	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/pkg/apis/rainbond/v1alpha1"
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

	localImage := path.Join(d.cluster.Spec.RainbondImageRepository, "smallimage")
	remoteImage := path.Join(rbdutil.GetImageRepository(d.cluster), "smallimage")

	dockerClient, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return failConditoin(condition, err)
	}
	dockerClient.NegotiateAPIVersion(d.ctx)

	exists, err := imageutil.CheckIfImageExists(d.ctx, dockerClient, remoteImage)
	if err != nil {
		return failConditoin(condition, fmt.Errorf("check if image %s exists: %v", remoteImage, err))
	}

	if !exists {
		if err := imageutil.ImagePull(d.ctx, dockerClient, localImage); err != nil {
			return failConditoin(condition, fmt.Errorf("pull image %s: %v", localImage, err))
		}
		if err := dockerClient.ImageTag(d.ctx, localImage, remoteImage); err != nil {
			return failConditoin(condition, fmt.Errorf("tag image %s to %s: %v", localImage, remoteImage, err))
		}
	}

	// push a small image to check the given image repository
	d.log.V(6).Info("push image", "image", remoteImage, "repository", rbdutil.GetImageRepository(d.cluster),
		"user", d.cluster.Spec.ImageHub.Username, "pass", d.cluster.Spec.ImageHub.Password)
	if err := imageutil.ImagePush(d.ctx, dockerClient, remoteImage, rbdutil.GetImageRepository(d.cluster),
		d.cluster.Spec.ImageHub.Username, d.cluster.Spec.ImageHub.Password); err != nil {
		return failConditoin(condition, fmt.Errorf("push image: %v", err))
	}

	return condition
}

func failConditoin(condition rainbondv1alpha1.RainbondClusterCondition, err error) rainbondv1alpha1.RainbondClusterCondition {
	condition.Status = corev1.ConditionFalse
	condition.Reason = "ImageRepoFailed"
	condition.Message = err.Error()
	return condition
}
