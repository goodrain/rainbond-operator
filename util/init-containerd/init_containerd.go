package init_containerd

import (
	"context"
	"github.com/containerd/containerd"
	"github.com/containerd/containerd/images"
	"github.com/containerd/containerd/namespaces"
	"os"
)

//ContainerdAPI -
type ContainerdAPI struct {
	ImageService     images.Store
	CCtx             context.Context
	ContainerdClient *containerd.Client
}

func InitContainerd() (*ContainerdAPI, error) {
	sock := os.Getenv("CONTAINERD_SOCK")
	if sock == "" {
		sock = "/run/containerd/containerd.sock"
	}
	containerdClient, err := containerd.New(sock)
	if err != nil {
		return nil, err
	}
	cctx := namespaces.WithNamespace(context.Background(), "rainbond")
	imageService := containerdClient.ImageService()
	return &ContainerdAPI{
		ImageService:     imageService,
		CCtx:             cctx,
		ContainerdClient: containerdClient,
	}, nil
}
