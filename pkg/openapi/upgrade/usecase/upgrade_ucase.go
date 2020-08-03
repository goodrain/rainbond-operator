package usecase

import (
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/ghodss/yaml"
	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	rainbondversiond "github.com/goodrain/rainbond-operator/pkg/generated/clientset/versioned"
	"github.com/goodrain/rainbond-operator/pkg/library/bcode"
	v1 "github.com/goodrain/rainbond-operator/pkg/openapi/types/v1"
	"github.com/goodrain/rainbond-operator/pkg/openapi/upgrade"
	"github.com/goodrain/rainbond-operator/pkg/util/k8sutil"
	"github.com/goodrain/rainbond-operator/pkg/util/retryutil"
	"github.com/sirupsen/logrus"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var validVersion = regexp.MustCompile(`^v[0-9]+.[0-9]+.[0-9]+-release$`)

type upgradeUsecase struct {
	rainbondClient rainbondversiond.Interface
	versionDir     string
	namespace      string
	clusterName    string
}

// NewUpgradeUsecase creates a new user.Usecase.
func NewUpgradeUsecase(rainbondClient rainbondversiond.Interface, versionDir, namespace, clusterName string) upgrade.Usecase {
	ucase := &upgradeUsecase{
		rainbondClient: rainbondClient,
		versionDir:     versionDir,
		namespace:      namespace,
		clusterName:    clusterName,
	}

	return ucase
}

func (u *upgradeUsecase) Versions() (*v1.UpgradeVersionsResp, error) {
	currentVersion, err := u.currentVersion()
	if err != nil {
		if err == bcode.ErrCurrentVersionNotFound {
			return nil, nil
		}
		if err == bcode.ErrInvalidCurrentVersion {
			return &v1.UpgradeVersionsResp{
				CurrentVersion: currentVersion,
			}, nil
		}
		return nil, err
	}

	// list all versions
	allVersions, err := u.listAllVersions()
	if err != nil {
		return nil, err
	}

	// filter out lower versions
	var versions []string
	for _, version := range allVersions {
		if higherVersion(currentVersion, version) {
			versions = append(versions, version)
		}
	}

	return &v1.UpgradeVersionsResp{
		CurrentVersion:      currentVersion,
		UpgradeableVersions: versions,
	}, nil
}

func (u *upgradeUsecase) Upgrade(req *v1.UpgradeReq) error {
	if err := u.checkVersionForUpgrade(req.Version); err != nil {
		return err
	}

	// read rbdcomponents from files
	components, err := u.readRbdcomponents(req.Version)
	if err != nil {
		logrus.Errorf("read rbdcomponents: %v", err)
		return bcode.ErrReadRbdComponent
	}

	// update rbdcomponents
	logrus.Infof("[Upgrade] update rbdcomponents")
	for _, cpt := range components {
		cpt.Namespace = u.namespace
		err := retryutil.Retry(time.Second*2, 3, func() (bool, error) {
			if err := k8sutil.CreateOrUpdateRbdComponent(u.rainbondClient, cpt); err != nil {
				return false, err
			}
			return true, nil
		})
		if err != nil {
			return err
		}
	}

	// update rainbondcluster version
	cluster, err := u.rainbondClient.RainbondV1alpha1().RainbondClusters(u.namespace).Get(u.clusterName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	cluster.Spec.InstallVersion = req.Version
	logrus.Infof("[Upgrade] update version for rainbondcluster")
	if _, err := u.rainbondClient.RainbondV1alpha1().RainbondClusters(u.namespace).Update(cluster); err != nil {
		return err
	}

	return nil
}

func (u *upgradeUsecase) readRbdcomponents(version string) ([]*rainbondv1alpha1.RbdComponent, error) {
	dir := path.Join(u.versionDir, version)
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var components []*rainbondv1alpha1.RbdComponent
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		bytes, err := ioutil.ReadFile(path.Join(dir, file.Name()))
		if err != nil {
			return nil, err
		}
		component := rainbondv1alpha1.RbdComponent{}
		if err = yaml.Unmarshal(bytes, &component); err != nil {
			return nil, err
		}
		components = append(components, &component)
	}

	return components, nil
}

func (u *upgradeUsecase) currentVersion() (string, error) {
	currentVersion, err := u.getCurrentVersion()
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			return "", bcode.ErrCurrentVersionNotFound
		}
		return "", err
	}
	// compatible with the old version
	if os.Getenv("RAINBOND_VERSION") == "v5.2.0-release" {
		currentVersion = "v5.2.0-release"
	}
	if currentVersion == "" {
		logrus.Warningf("[upgradeUsecase] [currentVersion] current version is empty")
		return "", bcode.ErrCurrentVersionNotFound
	}
	// check if the version is valid
	if !versionValid(currentVersion) {
		logrus.Warningf("current version(%s) is invalid", currentVersion)
		return currentVersion, bcode.ErrInvalidCurrentVersion
	}
	return currentVersion, nil
}

func (u *upgradeUsecase) listAllVersions() ([]string, error) {
	infos, err := ioutil.ReadDir(u.versionDir)
	if err != nil {
		return nil, err
	}

	var versions []string
	for _, info := range infos {
		if !info.IsDir() {
			continue
		}
		version := info.Name()
		// check if the version is valid
		if !versionValid(version) {
			logrus.Warningf("version(%s) is invalid", version)
			continue
		}
		versions = append(versions, version)
	}

	return versions, nil
}

func (u *upgradeUsecase) getCurrentVersion() (string, error) {
	cluster, err := u.rainbondClient.RainbondV1alpha1().RainbondClusters(u.namespace).Get(u.clusterName, metav1.GetOptions{})
	if err != nil {
		return "", err
	}

	return cluster.Spec.InstallVersion, nil
}

func (u *upgradeUsecase) checkVersionForUpgrade(version string) error {
	if !versionValid(version) {
		return bcode.ErrInvalidVersion
	}

	// check if the given version is exist
	allVersions, err := u.listAllVersions()
	if err != nil {
		return err
	}
	exists := false
	for _, ver := range allVersions {
		if ver == version {
			exists = true
			break
		}
	}
	if !exists {
		return bcode.ErrVersionNotFound
	}

	// do not support downgrade, version must be less than current version
	currentVersion, err := u.currentVersion()
	if err != nil {
		return err
	}
	if !higherVersion(currentVersion, version) {
		return bcode.ErrLowerVersion
	}

	return nil
}

func higherVersion(currentVersion, version string) bool {
	currentVersion, version = strings.ReplaceAll(currentVersion, "v", ""), strings.ReplaceAll(version, "v", "")
	current := strings.Split(currentVersion, ".")
	target := strings.Split(version, ".")
	return target[0] >= current[0] && target[1] >= current[1] && target[2] > current[2]
}

func versionValid(version string) bool {
	return validVersion.MatchString(version)
}
