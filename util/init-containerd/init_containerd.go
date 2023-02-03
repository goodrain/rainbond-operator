package init_containerd

import (
	"context"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/images"
	"github.com/containerd/containerd/namespaces"
)

// ContainerdAPI -
type ContainerdAPI struct {
	ImageService     images.Store
	CCtx             context.Context
	ContainerdClient *containerd.Client
}

func InitContainerd() (*ContainerdAPI, error) {
	containerdClient, err := containerd.New("/run/containerd/containerd.sock")
	if err != nil {
		containerdClient, err = containerd.New("/var/run/docker/containerd/containerd.sock")
		if err != nil {
			return nil, err
		}
	}
	cctx := namespaces.WithNamespace(context.Background(), "k8s.io")
	imageService := containerdClient.ImageService()
	return &ContainerdAPI{
		ImageService:     imageService,
		CCtx:             cctx,
		ContainerdClient: containerdClient,
	}, nil
}
