package usecase

import (
	"github.com/goodrain/rainbond-operator/pkg/openapi/upgrade"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"regexp"
)

var validVersion = regexp.MustCompile(`^v[0-9]+.[0-9]+.[0-9]+$`)

type upgradeUsecase struct {
	versionDir string
}

// NewUpgradeUsecase creates a new user.Usecase.
func NewUpgradeUsecase(versionDir string) upgrade.Usecase {
	ucase := &upgradeUsecase{
		versionDir: versionDir,
	}

	return ucase
}

func (u *upgradeUsecase) Versions() ([]string, error) {
	// list all versions
	versions, err := u.listAllVersions()
	if err != nil {
		return versions, nil
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
