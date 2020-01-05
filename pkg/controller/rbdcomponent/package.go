package rbdcomponent

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"

	"github.com/GLYASAI/rainbond-operator/pkg/util/commonutil"
)

func handleRainbondPackage(pkgFile, dst string) error {
	reqLogger := log.WithValues("Package file", pkgFile, "Destination", dst)
	reqLogger.Info("Handle rainbond images package")

	pkgDir := path.Join(dst, strings.Replace(path.Base(pkgFile), ".tgz", "", -1))
	if commonutil.DirExists(pkgDir) {
		reqLogger.Info("Package directory already exists", "package directory", pkgDir)
		return nil
	}

	reqLogger.Info("Extract installation package")
	if err := extractInstallationPackage(pkgFile, dst); err != nil {
		return err
	}

	reqLogger.Info("Load rainbond images")
	if err := loadRainbondImages(pkgDir); err != nil {
		return err
	}

	reqLogger.Info("Push rainbond images")
	if err := pushRainbondImages(pkgDir); err != nil {
		return err
	}

	return nil
}

// Extract the installation package
func extractInstallationPackage(pkgFile, dst string) error {
	pkgf, err := os.Open(pkgFile)
	if err != nil {
		return fmt.Errorf("open file: %v", err)
	}

	if err := commonutil.Untar(pkgf, dst); err != nil {
		return fmt.Errorf("untar %s: %v", pkgFile, err)
	}

	return nil
}

func loadRainbondImages(imageDir string) error {
	reqLogger := log.WithValues("Image package directory", imageDir)
	reqLogger.Info("Load rainbond images")

	// TODO: bad situation
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		reqLogger.Error(err, "create new docker client")
		return fmt.Errorf("create new docker client: %v", err)
	}
	cli.NegotiateAPIVersion(ctx)

	return filepath.Walk(imageDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Printf("prevent panic by handling failure accessing a path %q: %v\n", path, err)
			return err
		}

		if !commonutil.IsFile(path) {
			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("open file %s: %v", path, err)
		}

		reqLogger.Info("Start loading image", "file", path)
		_, err = cli.ImageLoad(ctx, f, true)
		if err != nil {
			return fmt.Errorf("file: %s; load image: %v", path, err)
		}
		// TODO: print response
		reqLogger.Info("Finish loading image", "file", path)

		return nil
	})
}

func pushRainbondImages(imageDir string) error {
	reqLogger := log.WithValues("Image package directory", imageDir)
	reqLogger.Info("Push rainbond images")

	mf := path.Join(imageDir, "metadata.json")
	metadata, err := ioutil.ReadFile(mf)
	if err != nil {
		return fmt.Errorf("read medadata.json: %v", err)
	}
	var images []string
	if err := json.Unmarshal(metadata, &images); err != nil {
		return fmt.Errorf("unmarshalling metadata %s: %v", string(metadata), err)
	}

	// TODO: duplicate code
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		reqLogger.Error(err, "create new docker client")
		return fmt.Errorf("create new docker client: %v", err)
	}
	cli.NegotiateAPIVersion(ctx)

	var opts types.ImagePushOptions
	registryAuth, err := EncodeAuthToBase64(types.AuthConfig{
		ServerAddress: "goodrain.me",
	})
	opts.RegistryAuth = registryAuth
	for _, image := range images {
		newImage := strings.Replace(image, "rainbond", "goodrain.me", -1)
		if err := cli.ImageTag(ctx, image, newImage); err != nil {
			reqLogger.Error(err, fmt.Sprintf("rename image %s", image))
			return fmt.Errorf("rename image %s: %v", image, err)
		}

		_, err := cli.ImagePush(ctx, newImage, opts) // TODO: print response
		if err != nil {
			reqLogger.Error(err, fmt.Sprintf("push image %s", newImage))
			return fmt.Errorf("push image %s: %v", newImage, err)
		}
	}

	return nil
}

// EncodeAuthToBase64 serializes the auth configuration as JSON base64 payload
func EncodeAuthToBase64(authConfig types.AuthConfig) (string, error) {
	buf, err := json.Marshal(authConfig)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(buf), nil
}
