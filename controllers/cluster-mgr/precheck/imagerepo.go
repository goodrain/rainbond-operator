package precheck

import (
	"context"
	"crypto/tls"
	"fmt"
	"path"
	"time"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/errdefs"
	"github.com/containerd/containerd/remotes/docker"
	"github.com/containerd/containerd/remotes/docker/config"
	"github.com/goodrain/rainbond-operator/util/initcontainerd"

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

	localImage := path.Join(d.cluster.Spec.RainbondImageRepository, "smallimage:latest")
	remoteImage := path.Join(imageRepo, "smallimage:latest")

	containerdCli, err := initcontainerd.InitContainerd()
	if err != nil {
		return d.failConditoin(condition, err)
	}
	image, err := containerdCli.ImageService.Get(containerdCli.CCtx, localImage)
	checkExsit := true
	if err != nil {
		if errdefs.IsNotFound(err) {
			checkExsit = false
		} else {
			return d.failConditoin(condition, fmt.Errorf("get image %v err：%v", localImage, err))
		}
	}
	if !checkExsit {
		defaultTLS := &tls.Config{
			InsecureSkipVerify: true,
		}
		hostOpt := config.HostOptions{}
		hostOpt.Credentials = func(host string) (string, string, error) {
			return d.cluster.Spec.ImageHub.Username, d.cluster.Spec.ImageHub.Password, nil
		}
		hostOpt.DefaultTLS = defaultTLS
		options := docker.ResolverOptions{
			Tracker: docker.NewInMemoryTracker(),
			Hosts:   config.ConfigureHosts(containerdCli.CCtx, hostOpt),
		}

		pullOpts := []containerd.RemoteOpt{
			containerd.WithPullUnpack,
			containerd.WithResolver(docker.NewResolver(options)),
		}
		_, err := containerdCli.ContainerdClient.Pull(containerdCli.CCtx, localImage, pullOpts...)
		if err != nil {
			return d.failConditoin(condition, fmt.Errorf("pull image %v err:%v", localImage, err))
		}
	}
	image.Name = remoteImage
	if _, err = containerdCli.ImageService.Create(containerdCli.CCtx, image); err != nil {
		// If user has specified force and the image already exists then
		// delete the original image and attempt to create the new one
		if errdefs.IsAlreadyExists(err) {
			if err = containerdCli.ImageService.Delete(containerdCli.CCtx, remoteImage); err != nil {
				return d.failConditoin(condition, fmt.Errorf("delete image %v err:%v", remoteImage, err))
			}
			if _, err = containerdCli.ImageService.Create(containerdCli.CCtx, image); err != nil {
				return d.failConditoin(condition, fmt.Errorf("create image %v err:%v", remoteImage, err))
			}
		} else {
			return d.failConditoin(condition, fmt.Errorf("create image %v err:%v", remoteImage, err))
		}
	}

	// push a small image to check the given image repository
	d.log.V(6).Info("push image", "image", remoteImage, "repository", imageRepo, "user", d.cluster.Spec.ImageHub.Username)
	defaultTLS := &tls.Config{
		InsecureSkipVerify: true,
	}

	hostOpt := config.HostOptions{}
	hostOpt.DefaultTLS = defaultTLS
	hostOpt.Credentials = func(host string) (string, string, error) {
		return d.cluster.Spec.ImageHub.Username, d.cluster.Spec.ImageHub.Password, nil
	}
	options := docker.ResolverOptions{
		Tracker: docker.NewInMemoryTracker(),
		Hosts:   config.ConfigureHosts(containerdCli.CCtx, hostOpt),
	}
	err = containerdCli.ContainerdClient.Push(containerdCli.CCtx, image.Name, image.Target, containerd.WithResolver(docker.NewResolver(options)))
	if err != nil {
		return d.failConditoin(condition, fmt.Errorf("push image %v err:%v", image.Name, err))
	}

	return condition
}

func (d *imagerepo) failConditoin(condition rainbondv1alpha1.RainbondClusterCondition, err error) rainbondv1alpha1.RainbondClusterCondition {
	return failConditoin(condition, "ImageRepoFailed", err.Error())
}
