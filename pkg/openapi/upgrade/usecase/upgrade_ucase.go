package usecase

import (
	"io/ioutil"
	"regexp"
	"strings"

	rainbondversiond "github.com/goodrain/rainbond-operator/pkg/generated/clientset/versioned"
	"github.com/goodrain/rainbond-operator/pkg/openapi/upgrade"
	"github.com/sirupsen/logrus"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var validVersion = regexp.MustCompile(`^v[0-9]+.[0-9]+.[0-9]+$`)

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

func (u *upgradeUsecase) Versions() ([]string, error) {
	currentVersion, err := u.getCurrentVersion()
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			return nil, nil
		}
	}
	if currentVersion == "" {
		logrus.Warningf("[upgradeUsecase] [Versions] current version is empty")
		return nil, nil
	}
	// check if the version is valid
	if !validVersion.MatchString(currentVersion) {
		logrus.Warningf("current version(%s) is invalid", currentVersion)
		return nil, nil
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

	return versions, nil
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
		if !validVersion.MatchString(version) {
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

func higherVersion(currentVersion, version string) bool {
	currentVersion, version = strings.ReplaceAll(currentVersion, "v", ""), strings.ReplaceAll(version, "v", "")
	current := strings.Split(currentVersion, ".")
	target := strings.Split(version, ".")
	return target[0] >= current[0] && target[1] >= current[1] && target[2] > current[2]
}
