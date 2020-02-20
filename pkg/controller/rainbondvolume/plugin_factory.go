package rainbondvolume

import (
	"context"
	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	"github.com/goodrain/rainbond-operator/pkg/controller/rainbondvolume/plugin"
	"github.com/goodrain/rainbond-operator/pkg/controller/rainbondvolume/plugin/aliyunclouddisk"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewCSIPlugin(ctx context.Context, cli client.Client, volume *rainbondv1alpha1.RainbondVolume) plugin.CSIPlugin {
	cp := volume.Spec.CSIPlugin
	switch {
	case cp.AliyunCloudDisk != nil:
		return aliyunclouddisk.CSIPlugins(ctx, cli, volume)
	case cp.AliyunNas != nil:

	}
	return nil
}
