package repository

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"github.com/goodrain/rainbond-operator/pkg/openapi/cluster"
	"github.com/goodrain/rainbond-operator/pkg/util/uuidutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var clusterRepoLog = logf.Log.WithName("cluster repo ")

// ClusterInit cluster init info
type ClusterInit struct {
	InitPath string
}

// NewClusterRepo new cluster repository
func NewClusterRepo(initPath string) cluster.Repository {
	return &ClusterInit{InitPath: initPath}
}

// EnterpriseID get enterprise
func (ci *ClusterInit) EnterpriseID(eid string) string {
	enterprise := path.Join(ci.InitPath, "enterprise")
	bs, err := ioutil.ReadFile(enterprise)

	if err != nil {
		clusterRepoLog.V(4).Info(fmt.Sprintf("read enterprise id from file: %s failed: %s, ignore it, rewrite it", enterprise, err.Error()))
		enterpriseID := uuidutil.NewUUID()
		if eid != "" {
			enterpriseID = eid
		}
		if _, err := os.Stat(ci.InitPath); os.IsNotExist(err) {
			if err := os.MkdirAll(ci.InitPath, os.ModePerm); err != nil {
				clusterRepoLog.Error(err, fmt.Sprintf("mkdir path [%s] failed: %s", enterpriseID, err.Error()))
				return ""
			}
		}
		if err := ioutil.WriteFile(enterprise, []byte(enterpriseID), os.ModePerm); err != nil {
			clusterRepoLog.Error(err, fmt.Sprintf("write enterprise id[%s] failed: %s", enterpriseID, err.Error()))
			return ""
		}
		return enterpriseID
	}

	return string(bs)
}
