package usecase

import (
	"github.com/goodrain/rainbond-operator/cmd/openapi/option"
	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	"github.com/goodrain/rainbond-operator/pkg/generated/clientset/versioned/fake"
	v1 "github.com/goodrain/rainbond-operator/pkg/openapi/types/v1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"testing"
)

func TestInstallUseCaseImpl_createRainbondVolumes(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = rainbondv1alpha1.AddToScheme(scheme)
	rainbondClient := fake.NewSimpleClientset()

	installUcase := InstallUseCaseImpl{
		namespace:          "rbd-system",
		rainbondKubeClient: rainbondClient,
		cfg: &option.Config{
			RainbondImageRepository: "foobar",
		},
	}

	req := &v1.ClusterInstallReq{
		RainbondVolumes: &v1.RainbondVolumes{
			RWX: &v1.RainbondVolume{
				StorageClassParameters: &v1.StorageClassParameters{
					Parameters: map[string]string{
						"zoneId":    "cn-neimeng-a",
						"volumeAs":  "filesystem",
						"vpcId":     "vpc-xxxxxx",
						"vSwitchId": "vsw-xxxxxx",
					},
				},
				CSIPlugin: &v1.CSIPluginSource{
					AliyunNas: &v1.AliyunNasCSIPluginSource{
						AccessKeyID:     "accessKeyID",
						AccessKeySecret: "accessKeySecret",
					},
				},
			},
		},
	}
	err := installUcase.createRainbondVolumes(req)
	assert.Nil(t, err)

	volume, err := rainbondClient.RainbondV1alpha1().RainbondVolumes("rbd-system").Get("rainbondvolumerwx", metav1.GetOptions{})
	assert.Nil(t, err)
	assert.Equal(t, "rainbondvolumerwx", volume.Name)
	assert.NotNil(t, volume.Spec.StorageClassParameters)
	assert.NotNil(t, volume.Spec.StorageClassParameters.Parameters)
	assert.NotNil(t, volume.Spec.CSIPlugin)
	assert.NotNil(t, volume.Spec.CSIPlugin.AliyunNas)
}
