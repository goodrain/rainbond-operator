package precheck

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
	"io"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/go-logr/logr"
	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	"github.com/goodrain/rainbond-operator/pkg/util/rbdutil"
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

	// push a small image to check the given image repository
	err := d.imagePush("k8s.gcr.io/pause:3.1")
	if err != nil {
		condition.Status = corev1.ConditionFalse
		condition.Reason = "ImageRepoFailed"
		condition.Message = err.Error()
	}

	return condition
}

// TODO: duplicated code
func (d *imagerepo) imagePush(image string) error {
	// TODO: duplicated code
	dockerClient, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return fmt.Errorf("create new docker client: %v", err)
	}
	dockerClient.NegotiateAPIVersion(d.ctx)

	d.log.Info("start push image", "image", image)
	var opts types.ImagePushOptions
	authConfig := types.AuthConfig{
		ServerAddress: rbdutil.GetImageRepository(d.cluster),
	}
	authConfig.Username = d.cluster.Spec.ImageHub.Username
	authConfig.Password = d.cluster.Spec.ImageHub.Password

	registryAuth, err := encodeAuthToBase64(authConfig)
	if err != nil {
		return fmt.Errorf("failed to encode auth config: %v", err)
	}
	opts.RegistryAuth = registryAuth
	ctx, cancel := context.WithTimeout(d.ctx, 5 * time.Second)
	defer cancel()
	var res io.ReadCloser
	res, err = dockerClient.ImagePush(ctx, image, opts)
	if err != nil {
		d.log.Error(err, "failed to push image", "image", image)
		return err
	}
	if res != nil {
		defer res.Close()

		dec := json.NewDecoder(res)
		for {
			select {
			case <-ctx.Done():
				d.log.Error(d.ctx.Err(), "error form context")
				return d.ctx.Err()
			default:
			}
			var jm jsonmessage.JSONMessage
			if err := dec.Decode(&jm); err != nil {
				if err == io.EOF {
					break
				}
				return fmt.Errorf("failed to decode json message: %v", err)
			}
			if jm.Error != nil {
				return fmt.Errorf("error detail: %v", jm.Error)
			}
			d.log.V(6).Info("response from image pushing", "msg", jm.Stream)
		}
	}
	d.log.V(4).Info("success push image", "image", image)
	return nil
}

func encodeAuthToBase64(authConfig types.AuthConfig) (string, error) {
	buf, err := json.Marshal(authConfig)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(buf), nil
}
