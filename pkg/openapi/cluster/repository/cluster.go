package repository

import (
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"github.com/goodrain/rainbond-operator/pkg/openapi/cluster"

	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/goodrain/rainbond-operator/pkg/util/uuidutil"
)

var clusterRepoLog = logf.Log.WithName("cluster repo ")

// ClusterInit cluster init info
type ClusterInit struct {
	InitPath  string
	installID string
}

// NewClusterRepo new cluster repository
func NewClusterRepo(initPath string) cluster.Repository {
	return &ClusterInit{InitPath: initPath}
}

// EnterpriseID get enterprise
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

// InstallID get install id
func (ci *ClusterInit) InstallID() string {
	if ci.installID == "" { // TODO fanyangyang atomic
		ci.installID = uuidutil.NewUUID()
	}
	return ci.installID
}

// ResetClusterInfo reset install id
func (ci *ClusterInit) ResetClusterInfo() {
	ci.installID = ""
}
