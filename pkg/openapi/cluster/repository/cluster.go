package repository

import (
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"path"

	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/goodrain/rainbond-operator/pkg/util/uuidutil"
)

var clusterRepoLog = logf.Log.WithName("cluster repo ")

type ClusterInit struct {
	InitPath string
}

// NewClusterRepo new cluster repository
func NewClusterRepo(initPath string) Repository {
	return &ClusterInit{InitPath: initPath}
}

func (ci *ClusterInit) EnterpriseID() string {
	enterprise := path.Join(ci.InitPath, "enterprise")
	bs, err := ioutil.ReadFile(enterprise)

	if err != nil {
		clusterRepoLog.Error(err, fmt.Sprintf("read enterprise id from file: %s failed: %s, ignore it, rewrite it", enterprise, err.Error()))
		enterpriseID := uuidutil.NewUUID()
		enterpriseIDBytes, err := hex.DecodeString(enterpriseID)
		if err != nil {
			clusterRepoLog.Error(err, "decode enterprise id[%s] failed: %s", enterpriseID, err.Error())
			return ""
		}
		if _, err := os.Stat(ci.InitPath); os.IsNotExist(err) {
			if err := os.MkdirAll(ci.InitPath, os.ModePerm); err != nil {
				clusterRepoLog.Error(err, fmt.Sprintf("mkdir path [%s] failed: %s", enterpriseID, err.Error()))
				return ""
			}
		}
		if err := ioutil.WriteFile(enterprise, enterpriseIDBytes, os.ModePerm); err != nil {
			clusterRepoLog.Error(err, fmt.Sprintf("write enterprise id[%s] failed: %s", enterpriseID, err.Error()))
			return ""
		}
		return enterpriseID
	}

	return hex.EncodeToString(bs)
}

func (ci *ClusterInit) InstallID() string {
	return uuidutil.NewUUID()
}
